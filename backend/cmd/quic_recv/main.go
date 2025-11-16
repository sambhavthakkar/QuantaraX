package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/quantarax/backend/internal/crypto"
	"github.com/quantarax/backend/internal/observability"
	"github.com/quantarax/backend/internal/quicutil"
	"github.com/quic-go/quic-go"
	"go.opentelemetry.io/otel"
	"golang.org/x/crypto/curve25519"
)

const (
	magicBytes      = "QNTX"
	protocolVersion = 0x01
	headerSize      = 32
)

var (
	listen    string
	outputDir string
)

func main() {
	flag.StringVar(&listen, "listen", ":4433", "Listen address (host:port)")
	flag.StringVar(&outputDir, "output-dir", "./received", "Output directory for chunks")
	flag.Parse()

	// Init tracing if configured
	if shutdown, err := observability.InitTracing(context.Background(), "quantarax-quic-recv"); err == nil {
		defer shutdown(context.Background())
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output directory: %v\n", err)
		os.Exit(1)
	}
	if err := receiveChunks(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func receiveChunks() error {
	tr := otel.Tracer("quantarax-quic-recv")
	ctx, span := tr.Start(context.Background(), "receiveChunks")
	defer span.End()
	certPEM, keyPEM, err := quicutil.GenerateSelfSignedCert()
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %w", err)
	}
	tlsConfig, err := quicutil.MakeTLSConfig(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("failed to create TLS config: %w", err)
	}
	// Set ALPN for direct QuantaraX QUIC transfers
	tlsConfig.NextProtos = []string{"quantarax-quic"}

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
	sessionKeys, err := crypto.DeriveSessionKeys(&ourKeys.PrivateKey, &theirPubKey, manifestHash[:])
	if err != nil {
		return fmt.Errorf("failed to derive session keys: %w", err)
	}

	listener, err := quic.ListenAddr(listen, tlsConfig, &quic.Config{EnableDatagrams: false})
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()
	fmt.Printf("QUIC receiver listening on %s\n", listen)
	for {
		conn, err := listener.Accept(ctx)
		if err != nil {
			return fmt.Errorf("failed to accept connection: %w", err)
		}
		_, acspan := tr.Start(ctx, "acceptConnection")
		go func(c any) { defer acspan.End(); handleConnection(c, sessionKeys) }(conn)
	}
}

func handleConnection(conn any, sessionKeys *crypto.SessionKeys) {
	type connCloser interface {
		CloseWithError(code quic.ApplicationErrorCode, reason string) error
		AcceptStream(ctx context.Context) (*quic.Stream, error)
	}
	c := conn.(connCloser)
	defer c.CloseWithError(0, "done")
	stream, err := c.AcceptStream(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to accept stream: %v\n", err)
		return
	}
	defer stream.Close()
	// Set a reasonable read deadline so AcceptStream isn't blocked indefinitely
	// and to ensure timely failure if sender closes early.
	_ = stream.SetReadDeadline(time.Now().Add(10 * time.Second))

	_, hspan := otel.Tracer("quantarax-quic-recv").Start(context.Background(), "parseHeader")
	header := make([]byte, headerSize)
	if _, err = io.ReadFull(stream, header); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read header: %v\n", err)
		hspan.End()
		return
	}
	sessionID, chunkIdx, payloadLen, err := parseHeader(header)
	hspan.End()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse header: %v\n", err)
		return
	}
	fmt.Println("Received chunk header:")
	fmt.Printf("  SessionID: %s\n", sessionID.String())
	fmt.Printf("  ChunkIndex: %d\n", chunkIdx)
	fmt.Printf("  PayloadLength: %d bytes\n", payloadLen)

	encryptedPayload := make([]byte, payloadLen)
	if _, err = io.ReadFull(stream, encryptedPayload); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read payload: %v\n", err)
		return
	}
	nonce := crypto.DeriveChunkNonce(sessionKeys.IVBase, chunkIdx)
	aad := make([]byte, 20)
	copy(aad[0:16], sessionID[:])
	binary.LittleEndian.PutUint32(aad[16:20], chunkIdx)
	plaintext, err := crypto.Open(sessionKeys.PayloadKey[:], nonce[:], aad, encryptedPayload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Decryption failed: %v\n", err)
		return
	}
	fmt.Println("Decryption successful!")
	// Ensure output directory exists in case it was removed externally
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ensure output directory: %v\n", err)
		return
	}
	outputPath := filepath.Join(outputDir, fmt.Sprintf("chunk_%04d.bin", chunkIdx))
	if err := os.WriteFile(outputPath, plaintext, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write chunk: %v\n", err)
		return
	}
	fmt.Printf("Chunk saved to: %s\n", outputPath)
}

func parseHeader(header []byte) (sessionID uuid.UUID, chunkIdx uint32, payloadLen uint32, err error) {
	if string(header[0:4]) != magicBytes {
		return uuid.Nil, 0, 0, fmt.Errorf("invalid magic bytes")
	}
	if header[4] != protocolVersion {
		return uuid.Nil, 0, 0, fmt.Errorf("unsupported protocol version: %d", header[4])
	}
	copy(sessionID[:], header[8:24])
	chunkIdx = binary.LittleEndian.Uint32(header[24:28])
	payloadLen = binary.LittleEndian.Uint32(header[28:32])
	return sessionID, chunkIdx, payloadLen, nil
}
