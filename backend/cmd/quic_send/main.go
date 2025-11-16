package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/quantarax/backend/internal/observability"
	"github.com/quantarax/backend/internal/quicutil"
	"github.com/quic-go/quic-go"
	"github.com/quantarax/backend/internal/chunker"
	"github.com/quantarax/backend/internal/crypto"
	"github.com/quantarax/backend/internal/introspect"
	"encoding/binary"
	"github.com/google/uuid"
	"golang.org/x/crypto/curve25519"
	"io"
)

const (
	magicBytes      = "QNTX"
	protocolVersion = 0x01
)

var (
	addr       string // direct receiver address
	relayAddr  string // optional relay address
	targetAddr string // target receiver address for relay mode
	filePath   string
	chunkIndex int
	chunkSize  int
	offset     int
)

func main() {
	flag.StringVar(&addr, "addr", "", "Receiver address (host:port); ignored if --relay is set")
	flag.StringVar(&relayAddr, "relay", "", "Relay address (host:port); if set, traffic goes via relay")
	flag.StringVar(&targetAddr, "target", "", "Target receiver address when using --relay (host:port)")
	flag.StringVar(&filePath, "file", "", "File path to send")
	flag.IntVar(&chunkIndex, "chunk-index", 0, "Chunk index to send")
	flag.IntVar(&chunkSize, "chunk-size", 1<<20, "Chunk size in bytes (default 1MiB)")
	flag.IntVar(&offset, "offset", 0, "Byte offset in file")
	flag.Parse()

	// Init tracing if configured
	if shutdown, err := observability.InitTracing(context.Background(), "quantarax-quic-send"); err == nil {
		defer shutdown(context.Background())
	}

	if filePath == "" {
		fmt.Fprintln(os.Stderr, "Usage: quic_send -file <path> [-addr host:port | -relay host:port -target host:port] [options]")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if relayAddr == "" && addr == "" {
		fmt.Fprintln(os.Stderr, "Either --addr or --relay + --target must be provided")
		os.Exit(1)
	}
	if relayAddr != "" && targetAddr == "" {
		fmt.Fprintln(os.Stderr, "--target is required when --relay is set")
		os.Exit(1)
	}

	if err := sendChunk(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func sendChunk() error {
	// Generate session UUID
	sessionID := uuid.New()
	fmt.Printf("Session ID: %s\n", sessionID.String())

	// Build a simple manifest for the file (single-file demo)
	mf, err := chunker.ComputeManifest(filePath, chunker.ChunkOptions{ChunkSize: chunkSize})
	if err != nil {
		return fmt.Errorf("failed to build manifest: %w", err)
	}
	// Auto-detect domain for demo sender
	dec := introspect.Decide(filePath)
	mf.Domain = dec.Domain

	// Fixed keys for E2E compatibility (no key exchange yet)
	var theirPubKey [32]byte
	var manifestHash [32]byte
	for i := range theirPubKey {
		theirPubKey[i] = 0x11
	}
	for i := range manifestHash {
		manifestHash[i] = 0x22
	}
	var ourKeys crypto.X25519KeyPair
	for i := range ourKeys.PrivateKey {
		ourKeys.PrivateKey[i] = 0x33
	}
	curve25519.ScalarBaseMult(&ourKeys.PublicKey, &ourKeys.PrivateKey)
	// Derive session keys
	sessionKeys, err := crypto.DeriveSessionKeys(&ourKeys.PrivateKey, &theirPubKey, manifestHash[:])
	if err != nil {
		return fmt.Errorf("failed to derive session keys: %w", err)
	}

	// Read chunk data
	chunkData, err := readChunkFromFile(filePath, offset, chunkSize)
	if err != nil {
		return fmt.Errorf("failed to read chunk: %w", err)
	}
	// Derive nonce for this chunk
	nonce := crypto.DeriveChunkNonce(sessionKeys.IVBase, uint32(chunkIndex))
	// Construct AAD (SessionID || ChunkIndex)
	aad := make([]byte, 20)
	copy(aad[0:16], sessionID[:])
	binary.LittleEndian.PutUint32(aad[16:20], uint32(chunkIndex))
	// Encrypt
	ciphertext, err := crypto.Seal(sessionKeys.PayloadKey[:], nonce[:], aad, chunkData)
	if err != nil {
		return fmt.Errorf("failed to encrypt chunk: %w", err)
	}

	_ = buildChunkMessage(sessionID, uint32(chunkIndex), ciphertext)

	// Connect
	tlsConfig := quicutil.MakeClientTLSConfig()
	var dialAddr string
	if relayAddr != "" {
		dialAddr = relayAddr
		tlsConfig.NextProtos = []string{"quic-relay"}
	} else {
		dialAddr = addr
		// Use direct transfer ALPN matching receiver
		tlsConfig.NextProtos = []string{"quantarax-quic"}
	}
	fmt.Printf("Connecting to %s...\n", dialAddr)
	conn, err := quic.DialAddr(context.Background(), dialAddr, tlsConfig, &quic.Config{EnableDatagrams: false})
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.CloseWithError(0, "done")
	fmt.Println("Connection established")

	// If relay mode, send control stream with target address
	if relayAddr != "" {
		ctrl, err := conn.OpenStreamSync(context.Background())
		if err != nil {
			return fmt.Errorf("failed to open control stream: %w", err)
		}
		defer ctrl.Close()
		if _, err := ctrl.Write([]byte(targetAddr)); err != nil {
			return fmt.Errorf("failed to write target: %w", err)
		}
		// Read response from relay
		respBuf := make([]byte, 256)
		n, err := ctrl.Read(respBuf)
		if err != nil {
			return fmt.Errorf("failed to read relay response: %w", err)
		}
		response := string(respBuf[:n])
		if response != "OK" {
			return fmt.Errorf("relay error: %s", response)
		}
	}

	// Send the chunk
	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		return fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	// Build and send chunk message
	message := buildChunkMessage(sessionID, uint32(chunkIndex), ciphertext)
	if _, err := stream.Write(message); err != nil {
		return fmt.Errorf("failed to send chunk: %w", err)
	}

	fmt.Printf("Chunk %d sent successfully (%d bytes)\n", chunkIndex, len(message))
	
	// Give the receiver a moment to process before closing connection
	time.Sleep(100 * time.Millisecond)
	return nil
}

func readChunkFromFile(path string, offset, size int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if _, err := file.Seek(int64(offset), 0); err != nil {
		return nil, err
	}
	buffer := make([]byte, size)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buffer[:n], nil
}

func buildChunkMessage(sessionID uuid.UUID, chunkIdx uint32, encryptedPayload []byte) []byte {
	header := make([]byte, 32)
	copy(header[0:4], magicBytes)
	header[4] = protocolVersion
	copy(header[8:24], sessionID[:])
	binary.LittleEndian.PutUint32(header[24:28], chunkIdx)
	binary.LittleEndian.PutUint32(header[28:32], uint32(len(encryptedPayload)))
	return append(header, encryptedPayload...)
}
