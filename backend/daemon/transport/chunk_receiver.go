package transport

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
	"encoding/json"
	"encoding/base64"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/quantarax/backend/internal/crypto"
	"github.com/quantarax/backend/internal/chunker"
	"github.com/quantarax/backend/internal/crypto/identity"
	"github.com/quantarax/backend/internal/fec"
	"github.com/quantarax/backend/daemon/manager"
	"github.com/quantarax/backend/internal/observability"
	"github.com/zeebo/blake3"
)



const (
	ChunkMagic      = 0x514E5458 // "QNTX"
	ChunkVersion    = 0x01
	ChunkHeaderSize = 32
)

var (
	ErrInvalidMagic   = errors.New("invalid chunk magic")
	ErrInvalidVersion = errors.New("unsupported chunk version")
)

// ChunkReceiver handles incoming chunk streams
type ChunkReceiver struct {
	connection  *quic.Conn
	sessionKeys *crypto.SessionKeys
	sessionID   uuid.UUID
	logger      *observability.Logger
	metrics     *observability.Metrics
	outputPath  string
	chunkSize   int64
	onChunkReceived func(chunkIndex int64)
	control     *ControlStream
	ackComp     ChunkRangeCompressor
	receivedCnt int64
	manifest    *chunker.Manifest
	fecDec      *fec.Decoder
	lastFECUpdate time.Time
}

// NewChunkReceiver creates a new chunk receiver
func NewChunkReceiver(
	connection *quic.Conn,
	sessionKeys *crypto.SessionKeys,
	sessionID uuid.UUID,
	outputPath string,
	chunkSize int64,
	onChunkReceived func(chunkIndex int64),
	control *ControlStream,
	manifest *chunker.Manifest,
	logger *observability.Logger,
	metrics *observability.Metrics,
) *ChunkReceiver {
	cr := &ChunkReceiver{
		connection:      connection,
		sessionKeys:     sessionKeys,
		sessionID:       sessionID,
		outputPath:      outputPath,
		chunkSize:       chunkSize,
		onChunkReceived: onChunkReceived,
		control:         control,
		manifest:        manifest,
		logger:          logger,
		metrics:         metrics,
	}
	if manifest != nil && manifest.FEC != nil {
		if dec, err := fec.NewDecoder(manifest.FEC.K, manifest.FEC.R); err == nil { cr.fecDec = dec }
	}
	return cr
}

// AcceptAndProcessStreams accepts incoming chunk streams and processes them
func (r *ChunkReceiver) AcceptAndProcessStreams() error {
	for {
		stream, err := r.connection.AcceptStream(r.connection.Context())
		if err != nil {
			return err
		}
		
		// Process stream in goroutine
		go r.processChunkStream(stream)
	}
}

// processChunkStream reads and processes a single chunk stream
func (r *ChunkReceiver) processChunkStream(stream *quic.Stream) {
	defer stream.Close()
	
	// Read chunk header
	header := make([]byte, ChunkHeaderSize)
	if _, err := io.ReadFull(stream, header); err != nil {
		fmt.Printf("Failed to read chunk header: %v\n", err)
		return
	}
	
	// Parse header
	chunkIndex, payloadLen, err := r.parseChunkHeader(header)
	if err != nil {
		fmt.Printf("Failed to parse chunk header: %v\n", err)
		return
	}
	
	// Read encrypted payload
	encryptedPayload := make([]byte, payloadLen)
	if _, err := io.ReadFull(stream, encryptedPayload); err != nil {
		fmt.Printf("Failed to read chunk payload: %v\n", err)
		return
	}
	
	// Decrypt chunk
	plaintext, err := r.decryptChunk(chunkIndex, encryptedPayload)
	if err != nil {
		fmt.Printf("Failed to decrypt chunk %d: %v\n", chunkIndex, err)
// Metrics and logging for decrypt failure
		// Metrics and logging for decrypt failure
		if r.metrics != nil { r.metrics.RecordChunkRetransmit("decrypt_failed") }
		if r.logger != nil { r.logger.ChunkDecryptFailed(r.sessionID.String(), int(chunkIndex), "decrypt_failed", err.Error(), 0) }
		// Send NACK to request retransmission
		if r.control != nil {
			var comp ChunkRangeCompressor
			rangeStr := comp.Compress([]int64{chunkIndex})
			_ = r.control.SendNack(&NackMessage{MissingRanges: rangeStr, Reason: "decrypt_failed", SessionID: r.sessionID.String(), Timestamp: time.Now().Unix()})
		}
		return
	}
	// Per-chunk integrity: verify against manifest hash if available
	if r.manifest != nil && chunkIndex >= 0 && int(chunkIndex) < len(r.manifest.Chunks) {
		expected := r.manifest.Chunks[chunkIndex].Hash
		h := blake3.Sum256(plaintext)
		computed := base64.StdEncoding.EncodeToString(h[:])
		if computed != expected {
			fmt.Printf("Chunk %d hash mismatch: expected %s got %s\n", chunkIndex, expected, computed)
			// Metrics and logging for hash mismatch
			if r.metrics != nil { r.metrics.RecordChunkRetransmit("hash_mismatch") }
			if r.logger != nil {
				r.logger.WithSession(r.sessionID.String()).
					Error(fmt.Errorf("hash mismatch"), fmt.Sprintf("chunk %d hash mismatch: expected %s got %s", chunkIndex, expected, computed))
			}
			// Send NACK to request retransmission
			if r.control != nil {
				var comp ChunkRangeCompressor
				rangeStr := comp.Compress([]int64{chunkIndex})
				_ = r.control.SendNack(&NackMessage{MissingRanges: rangeStr, Reason: "hash_mismatch", SessionID: r.sessionID.String(), Timestamp: time.Now().Unix()})
			}
			return
		}
	}
	// Compute and store CAS entry (after validation)
	h := blake3.Sum256(plaintext)
	chunkHash := base64.StdEncoding.EncodeToString(h[:])
	casPut(chunkHash, len(plaintext))
	// Write chunk to file
	if err := r.writeChunkToFile(chunkIndex, plaintext); err != nil {
		fmt.Printf("Failed to write chunk %d to file: %v\n", chunkIndex, err)
		return
	}
	
	// Notify chunk received
	if r.onChunkReceived != nil {
		r.onChunkReceived(chunkIndex)
	}
	// Send immediate ACK for this chunk (simple per-chunk ACK). In production, batch every ~250ms.
	if r.control != nil {
		r.receivedCnt++
		ranges := r.ackComp.Compress([]int64{chunkIndex})
		_ = r.control.SendAck(&AckMessage{ChunkRanges: ranges, TotalReceived: r.receivedCnt, Timestamp: time.Now().Unix(), SessionID: r.sessionID.String()})
		// If transfer complete, compute Merkle root and send VerificationMessage
		if r.manifest != nil && r.receivedCnt >= int64(r.manifest.ChunkCount) {
			computedRoot, _ := r.computeFileMerkleRoot()
			mv := manager.NewMerkleVerifier()
			vr := mv.CreateVerificationResult(r.sessionID.String(), []byte(computedRoot), []byte(r.manifest.MerkleRoot))
			// Record metrics for Merkle verification
			if r.metrics != nil { r.metrics.RecordMerkleVerification(vr.Status == manager.VerificationSuccess) }
			// Structured log for verification outcome
			if r.logger != nil {
				l := r.logger.WithSession(r.sessionID.String())
				msg := fmt.Sprintf("verification completed: status=%s", vr.Status.String())
				if vr.Status == manager.VerificationSuccess { l.Info(msg) } else { l.Warn(msg) }
			}
			// Sign the verification result using local identity keys
			if priv, pub, err := identity.LoadOrCreate("", ""); err == nil {
				if err := mv.SignVerificationResult(vr, priv); err == nil {
					fmt.Printf("Verification signed (pub=%d bytes)\n", len(pub))
				} else {
					fmt.Printf("Verification signing failed: %v\n", err)
				}
			} else {
				fmt.Printf("Identity load failed: %v\n", err)
			}
			_ = r.control.SendVerification(&VerificationMessage{SessionID: r.sessionID.String(), Status: vr.Status.String(), MerkleRootComputed: []byte(computedRoot), MerkleRootExpected: []byte(r.manifest.MerkleRoot), Timestamp: time.Now().Unix(), Signature: vr.Signature, PublicKey: vr.PublicKey})
		}
	}
}

// extractChunkHashes returns the list of chunk hashes from manifest in index order.
func extractChunkHashes(m *chunker.Manifest) []string {
	if m == nil || len(m.Chunks) == 0 { return nil }
	h := make([]string, len(m.Chunks))
	for i, ch := range m.Chunks { h[i] = ch.Hash }
	return h
}

// computeFileMerkleRoot computes the Merkle root from the actual file bytes on disk in chunk order.
func (r *ChunkReceiver) computeFileMerkleRoot() (string, error) {
	if r.manifest == nil { return "", nil }
	f, err := os.Open(r.outputPath)
	if err != nil { return "", err }
	defer f.Close()
	// Build base64 BLAKE3 per-chunk hashes compatible with manifest
	hashes := make([]string, 0, r.manifest.ChunkCount)
	buf := make([]byte, r.chunkSize)
	for i := 0; i < int(r.manifest.ChunkCount); i++ {
		// Seek and read exact chunk length from manifest
		if _, err := f.Seek(int64(i)*r.chunkSize, 0); err != nil { return "", err }
		n := r.manifest.Chunks[i].Length
		if n <= 0 || int64(n) > int64(len(buf)) { n = int(r.chunkSize) }
		b := buf[:n]
		if _, err := io.ReadFull(f, b); err != nil && err != io.EOF && err != io.ErrUnexpectedEOF { return "", err }
		// Hash and encode base64
		h := blake3.Sum256(b)
		hashes = append(hashes, base64.StdEncoding.EncodeToString(h[:]))
	}
	return chunker.ComputeMerkleRoot(hashes)
}

// ServeControlUpdates listens for FEC updates and CHUNK_HAVE requests and responds appropriately.
func (r *ChunkReceiver) ServeControlUpdates() {
	go func(){
		for {
			if r.control == nil { return }
			t, data, err := r.control.ReceiveAny()
			if err != nil { return }
			switch t {
			case MessageTypeFECUpdate:
				var u FECUpdateMessage
				if json.Unmarshal(data, &u) == nil {
					// Debounce quick changes: apply at most once per 500ms
					// and only at group boundaries (when receivedCnt % K == 0)
					if r.fecDec != nil && (u.K > 0 && u.R > 0) {
						k, _ := r.fecDec.GetParameters()
						apply := true
						if time.Since(r.lastFECUpdate) < 500*time.Millisecond { apply = false }
						if k > 0 && r.receivedCnt%int64(k) != 0 { apply = false }
						if apply {
							if dec, err := fec.NewDecoder(u.K, u.R); err == nil { r.fecDec = dec; r.lastFECUpdate = time.Now() }
						}
					}
				}
			case MessageTypeChunkHaveRequest:
				var req ChunkHaveRequest
				if json.Unmarshal(data, &req) == nil {
					// Build CAS bitmap from manifest chunk hashes
					var idxs []int64
					if r.manifest != nil {
						for _, ch := range r.manifest.Chunks {
							if casHas(ch.Hash) { idxs = append(idxs, int64(ch.Index)) }
						}
					}
					var comp ChunkRangeCompressor
					ranges := comp.Compress(idxs)
					_ = r.control.SendChunkHaveResponse(&ChunkHaveResponse{SessionID: req.SessionID, ChunkCount: req.ChunkCount, HaveRanges: ranges, Timestamp: time.Now().Unix()})
				}
			}
		}
	}()
}

// parseChunkHeader parses the chunk message header
func (r *ChunkReceiver) parseChunkHeader(header []byte) (int64, int, error) {
	// Verify magic
	magic := binary.BigEndian.Uint32(header[0:4])
	if magic != ChunkMagic {
		return 0, 0, ErrInvalidMagic
	}
	
	// Verify version
	version := header[4]
	if version != ChunkVersion {
		return 0, 0, ErrInvalidVersion
	}
	
	// Extract session ID
	receivedSessionID := uuid.UUID{}
	copy(receivedSessionID[:], header[8:24])
	
	// Verify session ID matches
	if receivedSessionID != r.sessionID {
		return 0, 0, fmt.Errorf("session ID mismatch: expected %s, got %s", r.sessionID, receivedSessionID)
	}
	
	// Extract chunk index
	chunkIndex := int64(binary.BigEndian.Uint32(header[24:28]))
	
	// Extract payload length
	payloadLen := int(binary.BigEndian.Uint32(header[28:32]))
	
	return chunkIndex, payloadLen, nil
}

// decryptChunk decrypts chunk data using session keys
func (r *ChunkReceiver) decryptChunk(chunkIndex int64, ciphertext []byte) ([]byte, error) {
	// Derive nonce from chunk index
	nonce := crypto.DeriveNonce(r.sessionKeys.IVBase, uint64(chunkIndex))
	
	// Construct AAD from session ID and chunk index
	aad := make([]byte, 16+8)
	copy(aad[0:16], r.sessionID[:])
	binary.BigEndian.PutUint64(aad[16:24], uint64(chunkIndex))
	
	// Decrypt using AES-256-GCM
	plaintext, err := crypto.Open(r.sessionKeys.PayloadKey[:], nonce[:], aad, ciphertext)
	if err != nil {
		return nil, err
	}
	
	return plaintext, nil
}

// writeChunkToFile writes chunk data to the output file
func (r *ChunkReceiver) writeChunkToFile(chunkIndex int64, data []byte) error {
	// Open or create output file
	file, err := os.OpenFile(r.outputPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Seek to chunk offset
	offset := chunkIndex * r.chunkSize
	if _, err := file.Seek(offset, 0); err != nil {
		return err
	}
	
	// Write chunk data
	if _, err := file.Write(data); err != nil {
		return err
	}
	
	return nil
}