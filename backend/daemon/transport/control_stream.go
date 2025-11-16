package transport

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/quic-go/quic-go"
)

var (
	ErrInvalidSignature       = errors.New("invalid manifest signature")
	ErrInvalidProtocolVersion = errors.New("unsupported protocol version")
)

const (
	ProtocolVersion = 1
	ControlStreamID = 0
)

// ControlMessageType represents control message types
type ControlMessageType uint8

const (
	MessageTypeManifest ControlMessageType = iota + 1
	MessageTypeAck
	MessageTypeNack
	MessageTypeStatus
	MessageTypeVerification
	MessageTypeFECUpdate
	MessageTypeChunkHaveRequest
	MessageTypeChunkHaveResponse
)

// SignedManifest represents a cryptographically signed file manifest
type SignedManifest struct {
	ManifestJSON    []byte
	Signature       []byte
	PublicKey       []byte
	ProtocolVersion int32
}

// AckMessage represents chunk acknowledgment
type AckMessage struct {
	ChunkRanges   string
	TotalReceived int64
	Timestamp     int64
	SessionID     string
}

// NackMessage represents missing chunk request
type NackMessage struct {
	MissingRanges string
	Reason        string
	SessionID     string
	Timestamp     int64
}

// StatusMessage represents transfer status update
type StatusMessage struct {
	CurrentState    int32
	ProgressPercent float64
	Message         string
	Timestamp       int64
}

// VerificationMessage represents Merkle root verification result
type VerificationMessage struct {
	SessionID          string
	Status             string
	MerkleRootComputed []byte
	MerkleRootExpected []byte
	Timestamp          int64
	Signature          []byte
	PublicKey          []byte
}

// FECUpdateMessage updates FEC parameters during a session.
type FECUpdateMessage struct {
	SessionID string
	K         int
	R         int
	Reason    string
	Timestamp int64
}

// ChunkHaveRequest asks the receiver to provide a bitmap of chunks present in CAS.
type ChunkHaveRequest struct {
	SessionID  string
	ChunkCount int
}

// ChunkHaveResponse contains a range-compressed bitmap of chunks present.
type ChunkHaveResponse struct {
	SessionID  string
	HaveRanges string
	ChunkCount int
	Timestamp  int64
}

// ControlStream manages the control protocol stream
type ControlStream struct {
	stream *quic.Stream
}

// NewControlStream creates a new control stream wrapper
func NewControlStream(stream *quic.Stream) *ControlStream {
	return &ControlStream{
		stream: stream,
	}
}

// SendSignedManifest sends a signed manifest over the control stream
func (cs *ControlStream) SendSignedManifest(manifestJSON []byte, privateKey ed25519.PrivateKey) error {
	signature := ed25519.Sign(privateKey, manifestJSON)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	sm := &SignedManifest{
		ManifestJSON:    manifestJSON,
		Signature:       signature,
		PublicKey:       publicKey,
		ProtocolVersion: ProtocolVersion,
	}

	return cs.sendControlMessage(MessageTypeManifest, sm)
}

// ReceiveSignedManifest receives and verifies a signed manifest
func (cs *ControlStream) ReceiveSignedManifest() (*SignedManifest, error) {
	msgType, data, err := cs.receiveControlMessage()
	if err != nil {
		return nil, err
	}

	if msgType != MessageTypeManifest {
		return nil, fmt.Errorf("expected manifest message, got %d", msgType)
	}

	var sm SignedManifest
	if err := json.Unmarshal(data, &sm); err != nil {
		return nil, err
	}

	if sm.ProtocolVersion != ProtocolVersion {
		return nil, ErrInvalidProtocolVersion
	}

	if !ed25519.Verify(sm.PublicKey, sm.ManifestJSON, sm.Signature) {
		return nil, ErrInvalidSignature
	}

	return &sm, nil
}

// SendAck sends an acknowledgment message
func (cs *ControlStream) SendAck(ack *AckMessage) error {
	return cs.sendControlMessage(MessageTypeAck, ack)
}

// ReceiveAck receives an acknowledgment message
func (cs *ControlStream) ReceiveAck() (*AckMessage, error) {
	msgType, data, err := cs.receiveControlMessage()
	if err != nil {
		return nil, err
	}

	if msgType != MessageTypeAck {
		return nil, fmt.Errorf("expected ack message, got %d", msgType)
	}

	var ack AckMessage
	if err := json.Unmarshal(data, &ack); err != nil {
		return nil, err
	}

	return &ack, nil
}

// SendNack sends a negative acknowledgment message
func (cs *ControlStream) SendNack(nack *NackMessage) error {
	return cs.sendControlMessage(MessageTypeNack, nack)
}

// ReceiveNack receives a negative acknowledgment message
func (cs *ControlStream) ReceiveNack() (*NackMessage, error) {
	msgType, data, err := cs.receiveControlMessage()
	if err != nil {
		return nil, err
	}

	if msgType != MessageTypeNack {
		return nil, fmt.Errorf("expected nack message, got %d", msgType)
	}

	var nack NackMessage
	if err := json.Unmarshal(data, &nack); err != nil {
		return nil, err
	}

	return &nack, nil
}

// SendStatus sends a status update message
func (cs *ControlStream) SendStatus(status *StatusMessage) error {
	return cs.sendControlMessage(MessageTypeStatus, status)
}

// ReceiveStatus receives a status update message
func (cs *ControlStream) ReceiveStatus() (*StatusMessage, error) {
	msgType, data, err := cs.receiveControlMessage()
	if err != nil {
		return nil, err
	}

	if msgType != MessageTypeStatus {
		return nil, fmt.Errorf("expected status message, got %d", msgType)
	}

	var status StatusMessage
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// SendVerification sends a verification result message
func (cs *ControlStream) SendVerification(verification *VerificationMessage) error {
	return cs.sendControlMessage(MessageTypeVerification, verification)
}

// SendFECUpdate sends FEC update
func (cs *ControlStream) SendFECUpdate(msg *FECUpdateMessage) error {
	return cs.sendControlMessage(MessageTypeFECUpdate, msg)
}

// ReceiveFECUpdate receives FEC update
func (cs *ControlStream) ReceiveFECUpdate() (*FECUpdateMessage, error) {
	msgType, data, err := cs.receiveControlMessage()
	if err != nil {
		return nil, err
	}
	if msgType != MessageTypeFECUpdate {
		return nil, fmt.Errorf("expected FEC_UPDATE, got %d", msgType)
	}
	var m FECUpdateMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// SendChunkHaveRequest sends a request for receiver CAS bitmap
func (cs *ControlStream) SendChunkHaveRequest(req *ChunkHaveRequest) error {
	return cs.sendControlMessage(MessageTypeChunkHaveRequest, req)
}

// ReceiveChunkHaveRequest receives a request
func (cs *ControlStream) ReceiveChunkHaveRequest() (*ChunkHaveRequest, error) {
	msgType, data, err := cs.receiveControlMessage()
	if err != nil {
		return nil, err
	}
	if msgType != MessageTypeChunkHaveRequest {
		return nil, fmt.Errorf("expected CHUNK_HAVE_REQUEST, got %d", msgType)
	}
	var req ChunkHaveRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// SendChunkHaveResponse sends CAS bitmap response
func (cs *ControlStream) SendChunkHaveResponse(resp *ChunkHaveResponse) error {
	return cs.sendControlMessage(MessageTypeChunkHaveResponse, resp)
}

// ReceiveChunkHaveResponse receives CAS bitmap response
func (cs *ControlStream) ReceiveChunkHaveResponse() (*ChunkHaveResponse, error) {
	msgType, data, err := cs.receiveControlMessage()
	if err != nil {
		return nil, err
	}
	if msgType != MessageTypeChunkHaveResponse {
		return nil, fmt.Errorf("expected CHUNK_HAVE_RESPONSE, got %d", msgType)
	}
	var resp ChunkHaveResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ReceiveVerification receives a verification result message
func (cs *ControlStream) ReceiveVerification() (*VerificationMessage, error) {
	msgType, data, err := cs.receiveControlMessage()
	if err != nil {
		return nil, err
	}

	if msgType != MessageTypeVerification {
		return nil, fmt.Errorf("expected verification message, got %d", msgType)
	}

	var verification VerificationMessage
	if err := json.Unmarshal(data, &verification); err != nil {
		return nil, err
	}

	return &verification, nil
}

// sendControlMessage sends a control message with type and payload
func (cs *ControlStream) sendControlMessage(msgType ControlMessageType, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if err := binary.Write(cs.stream, binary.BigEndian, msgType); err != nil {
		return err
	}

	length := uint32(len(data))
	if err := binary.Write(cs.stream, binary.BigEndian, length); err != nil {
		return err
	}

	_, err = cs.stream.Write(data)
	return err
}

// ReceiveAny receives any control message and returns its type and raw payload
func (cs *ControlStream) ReceiveAny() (ControlMessageType, []byte, error) {
	return cs.receiveControlMessage()
}

// receiveControlMessage receives a control message
func (cs *ControlStream) receiveControlMessage() (ControlMessageType, []byte, error) {
	var msgType ControlMessageType
	if err := binary.Read(cs.stream, binary.BigEndian, &msgType); err != nil {
		return 0, nil, err
	}

	var length uint32
	if err := binary.Read(cs.stream, binary.BigEndian, &length); err != nil {
		return 0, nil, err
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(cs.stream, data); err != nil {
		return 0, nil, err
	}

	return msgType, data, nil
}

// Close closes the control stream
func (cs *ControlStream) Close() error {
	return cs.stream.Close()
}

// ChunkRangeCompressor compresses chunk indices into range notation
type ChunkRangeCompressor struct{}

// Compress converts a slice of chunk indices to range string
func (c *ChunkRangeCompressor) Compress(chunks []int64) string {
	if len(chunks) == 0 {
		return ""
	}

	var buf bytes.Buffer
	start := chunks[0]
	prev := chunks[0]

	for i := 1; i < len(chunks); i++ {
		curr := chunks[i]

		if curr == prev+1 {
			prev = curr
		} else {
			if start == prev {
				fmt.Fprintf(&buf, "%d,", start)
			} else {
				fmt.Fprintf(&buf, "%d-%d,", start, prev)
			}
			start = curr
			prev = curr
		}
	}

	if start == prev {
		fmt.Fprintf(&buf, "%d", start)
	} else {
		fmt.Fprintf(&buf, "%d-%d", start, prev)
	}

	return buf.String()
}

// Decompress converts range string to slice of chunk indices
func (c *ChunkRangeCompressor) Decompress(rangeStr string) ([]int64, error) {
	if rangeStr == "" {
		return []int64{}, nil
	}

	var chunks []int64
	ranges := bytes.Split([]byte(rangeStr), []byte(","))

	for _, r := range ranges {
		parts := bytes.Split(r, []byte("-"))

		if len(parts) == 1 {
			var chunk int64
			if _, err := fmt.Sscanf(string(parts[0]), "%d", &chunk); err != nil {
				return nil, err
			}
			chunks = append(chunks, chunk)
		} else if len(parts) == 2 {
			var start, end int64
			if _, err := fmt.Sscanf(string(parts[0]), "%d", &start); err != nil {
				return nil, err
			}
			if _, err := fmt.Sscanf(string(parts[1]), "%d", &end); err != nil {
				return nil, err
			}
			for i := start; i <= end; i++ {
				chunks = append(chunks, i)
			}
		}
	}

	return chunks, nil
}
