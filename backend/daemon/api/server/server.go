package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/quantarax/backend/daemon/manager"
	"github.com/quantarax/backend/daemon/service"
	"github.com/quantarax/backend/internal/chunker"
)

// HTTP contract types (mirror docs/api-integration-contract.md)

type (
	CreateTransferRequest struct {
		FilePath          string            `json:"file_path"`
		RecipientID       string            `json:"recipient_id"`
		ChunkSizeOverride int64             `json:"chunk_size_override"`
		Metadata          map[string]string `json:"metadata"`
	}
	CreateTransferResponse struct {
		SessionID         string        `json:"session_id"`
		TransferToken     string        `json:"transfer_token"`
		Manifest          *ManifestJSON `json:"manifest,omitempty"`
		QRCodeData        string        `json:"qr_code_data"`
		EstimatedDuration int64         `json:"estimated_duration"`
	}

	AcceptTransferRequest struct {
		TransferToken   string `json:"transfer_token"`
		OutputPath      string `json:"output_path"`
		ResumeSessionID string `json:"resume_session_id"`
	}
	AcceptTransferResponse struct {
		SessionID       string        `json:"session_id"`
		Manifest        *ManifestJSON `json:"manifest,omitempty"`
		SenderPublicKey string        `json:"sender_public_key"`
		TotalChunks     int64         `json:"total_chunks"`
		EstimatedSize   int64         `json:"estimated_size"`
	}

	GetTransferStatusResponse struct {
		State                  string  `json:"state"`
		ProgressPercent        float64 `json:"progress_percent"`
		ChunksTransferred      int64   `json:"chunks_transferred"`
		TotalChunks            int64   `json:"total_chunks"`
		BytesTransferred       int64   `json:"bytes_transferred"`
		TransferRateMbps       float64 `json:"transfer_rate_mbps"`
		EstimatedTimeRemaining int64   `json:"estimated_time_remaining"`
		RttMs                  float64 `json:"rtt_ms,omitempty"`
		Streams                int     `json:"streams,omitempty"`
		LossRatePct            float64 `json:"loss_rate_pct,omitempty"`
		ErrorMessage           string  `json:"error_message,omitempty"`
	}

	TransferSummary struct {
		SessionID       string  `json:"session_id"`
		FileName        string  `json:"file_name"`
		State           string  `json:"state"`
		ProgressPercent float64 `json:"progress_percent"`
		StartTime       int64   `json:"start_time"`
		Direction       string  `json:"direction"`
	}
	ListTransfersResponse struct {
		Transfers  []*TransferSummary `json:"transfers"`
		TotalCount int32              `json:"total_count"`
		HasMore    bool               `json:"has_more"`
	}

	GetKeysResponse struct {
		PublicKeyBase64 string `json:"public_key_base64"`
		Fingerprint     string `json:"fingerprint"`
	}

	TransferEventJSON struct {
		SessionID       string            `json:"session_id"`
		EventType       string            `json:"event_type"`
		Timestamp       int64             `json:"timestamp"`
		ProgressPercent float64           `json:"progress_percent"`
		Message         string            `json:"message,omitempty"`
		Metadata        map[string]string `json:"metadata,omitempty"`
	}

	ManifestJSON struct {
		FileName    string   `json:"file_name"`
		FileSize    int64    `json:"file_size"`
		ChunkSize   int64    `json:"chunk_size"`
		TotalChunks int64    `json:"total_chunks"`
		MerkleRoot  string   `json:"merkle_root"`
		ChunkHashes []string `json:"chunk_hashes,omitempty"`
	}
)

// DaemonAPIServer wires services to HTTP handlers

type DaemonAPIServer struct {
	transfer *service.TransferService
	sessions *manager.SessionStore
	events   *service.EventPublisher
}

func NewDaemonAPIServer(ts *service.TransferService, store *manager.SessionStore, events *service.EventPublisher) *DaemonAPIServer {
	return &DaemonAPIServer{transfer: ts, sessions: store, events: events}
}

// RegisterHTTP registers REST routes on mux
func (s *DaemonAPIServer) RegisterHTTP(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/transfer/create", s.handleCreateTransfer)
	mux.HandleFunc("/api/v1/transfer/accept", s.handleAcceptTransfer)
	mux.HandleFunc("/api/v1/transfer/", s.handleTransferPrefix)
	mux.HandleFunc("/api/v1/transfers", s.handleListTransfers)
	mux.HandleFunc("/api/v1/keys", s.handleGetKeys)
}

func (s *DaemonAPIServer) handleCreateTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req CreateTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid JSON body")
		return
	}
	sessionID, token, manifest, err := s.transfer.CreateTransfer(req.FilePath, req.RecipientID, req.ChunkSizeOverride, req.Metadata)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	estimated := int64(0)
	if manifest != nil && manifest.FileSize > 0 {
		bps := 50.0 * 1024 * 1024
		estimated = int64(float64(manifest.FileSize) / bps)
		if manifest.Network != nil && manifest.Network.Bandwidth > 0 {
			bps = manifest.Network.Bandwidth * 1024 * 1024
			estimated = int64(float64(manifest.FileSize) / bps)
		}
	}
	resp := &CreateTransferResponse{SessionID: sessionID, TransferToken: token, Manifest: toHTTPManifest(manifest), QRCodeData: token, EstimatedDuration: estimated}
	writeJSON(w, http.StatusOK, resp)
}

func (s *DaemonAPIServer) handleAcceptTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AcceptTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid JSON body")
		return
	}
	sid, manifest, err := s.transfer.AcceptTransfer(req.TransferToken, req.OutputPath, req.ResumeSessionID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	pub, _ := s.transfer.GetPublicKey()
	// keep as base64 string
	resp := &AcceptTransferResponse{SessionID: sid, Manifest: toHTTPManifest(manifest), SenderPublicKey: pub, TotalChunks: int64(manifest.ChunkCount), EstimatedSize: manifest.FileSize}
	writeJSON(w, http.StatusOK, resp)
}

func (s *DaemonAPIServer) handleTransferPrefix(w http.ResponseWriter, r *http.Request) {
	// Expect /api/v1/transfer/{session_id}/status
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/transfer/"), "/")
	sessionID := parts[0]
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	action := parts[1]
	if action == "status" {
		st, err := s.transfer.GetTransferStatus(sessionID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		resp := &GetTransferStatusResponse{
			State:                  toHTTPState(st.State),
			ProgressPercent:        st.ProgressPercent,
			ChunksTransferred:      st.ChunksTransferred,
			TotalChunks:            st.TotalChunks,
			BytesTransferred:       st.BytesTransferred,
			TransferRateMbps:       st.TransferRateMbps,
			EstimatedTimeRemaining: st.EstimatedTimeRemaining,
			ErrorMessage:           st.ErrorMessage,
		}
		// Optional diagnostics from session metadata
		if sess, err2 := s.sessions.Get(sessionID); err2 == nil {
			if v, ok := sess.Metadata["rtt_ms"]; ok {
				if f, errp := strconv.ParseFloat(v, 64); errp == nil {
					resp.RttMs = f
				}
			}
			if v, ok := sess.Metadata["streams"]; ok {
				if n, errp := strconv.Atoi(v); errp == nil {
					resp.Streams = n
				}
			}
			if v, ok := sess.Metadata["loss_rate_pct"]; ok {
				if f, errp := strconv.ParseFloat(v, 64); errp == nil {
					resp.LossRatePct = f
				}
			}
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}
}

func (s *DaemonAPIServer) handleListTransfers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var filter *manager.TransferState
	if v := q.Get("state"); v != "" {
		s := fromHTTPState(v)
		filter = &s
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	sessions, total := s.sessions.List(filter, limit, offset)
	resp := &ListTransfersResponse{Transfers: make([]*TransferSummary, 0, len(sessions)), TotalCount: int32(total)}
	for _, se := range sessions {
		resp.Transfers = append(resp.Transfers, &TransferSummary{
			SessionID:       se.ID,
			FileName:        se.FileName,
			State:           toHTTPState(se.State),
			ProgressPercent: se.GetProgressPercent(),
			StartTime:       se.StartTime.UnixMilli(),
			Direction:       toHTTPDirection(se.Direction),
		})
	}
	resp.HasMore = offset+len(resp.Transfers) < total
	writeJSON(w, http.StatusOK, resp)
}

func (s *DaemonAPIServer) handleGetKeys(w http.ResponseWriter, r *http.Request) {
	pub, fp := s.transfer.GetPublicKey()
	writeJSON(w, http.StatusOK, &GetKeysResponse{PublicKeyBase64: pub, Fingerprint: fp})
}

// SSE handler remains for event streaming
func SSEHandler(events *service.EventPublisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}
		filter := r.URL.Query().Get("session_id")
		sub := events.Subscribe(filter)
		defer events.Unsubscribe(sub.ID)
		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub.Channel:
				if !ok {
					return
				}
				line := toJSONLine(ev)
				_, _ = w.Write([]byte("data: "))
				_, _ = w.Write(line)
				_, _ = w.Write([]byte("\n\n"))
				flusher.Flush()
			}
		}
	}
}

func toJSONLine(ev *service.TransferEvent) []byte {
	b := &strings.Builder{}
	b.WriteString("{")
	b.WriteString("\"session_id\":\"")
	b.WriteString(ev.SessionID)
	b.WriteString("\",")
	b.WriteString("\"event_type\":\"")
	b.WriteString(ev.EventType.String())
	b.WriteString("\",")
	b.WriteString("\"timestamp\":")
	b.WriteString(strconv.FormatInt(ev.Timestamp.UnixMilli(), 10))
	b.WriteString(",")
	b.WriteString("\"progress_percent\":")
	b.WriteString(strconv.FormatFloat(ev.ProgressPercent, 'f', 2, 64))
	if ev.Message != "" {
		b.WriteString(",\"message\":\"")
		b.WriteString(ev.Message)
		b.WriteString("\"")
	}
	if len(ev.Metadata) > 0 {
		b.WriteString(",\"metadata\":{")
		i := 0
		for k, v := range ev.Metadata {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString("\"")
			b.WriteString(k)
			b.WriteString("\":\"")
			b.WriteString(v)
			b.WriteString("\"")
			i++
		}
		b.WriteString("}")
	}
	b.WriteString("}")
	return []byte(b.String())
}

func toHTTPManifest(m *chunker.Manifest) *ManifestJSON {
	if m == nil {
		return nil
	}
	pm := &ManifestJSON{
		FileName:    m.FileName,
		FileSize:    m.FileSize,
		ChunkSize:   int64(m.ChunkSize),
		TotalChunks: int64(m.ChunkCount),
		MerkleRoot:  base64.StdEncoding.EncodeToString([]byte(m.MerkleRoot)),
	}
	// Optional: include chunk hashes
	// for _, ch := range m.Chunks { pm.ChunkHashes = append(pm.ChunkHashes, ch.Hash) }
	return pm
}

func toHTTPState(s manager.TransferState) string {
	switch s {
	case manager.StatePending:
		return "PENDING"
	case manager.StateActive:
		return "ACTIVE"
	case manager.StatePaused:
		return "PAUSED"
	case manager.StateCompleted:
		return "COMPLETED"
	case manager.StateFailed:
		return "FAILED"
	default:
		return "UNSPECIFIED"
	}
}

func fromHTTPState(s string) manager.TransferState {
	s = strings.ToUpper(s)
	switch s {
	case "PENDING":
		return manager.StatePending
	case "ACTIVE":
		return manager.StateActive
	case "PAUSED":
		return manager.StatePaused
	case "COMPLETED":
		return manager.StateCompleted
	case "FAILED":
		return manager.StateFailed
	default:
		return manager.StatePending
	}
}

func toHTTPDirection(d manager.TransferDirection) string {
	switch d {
	case manager.DirectionSend:
		return "SEND"
	case manager.DirectionReceive:
		return "RECEIVE"
	default:
		return "UNSPECIFIED"
	}
}

// JSON helpers

type JSONError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, JSONError{Code: code, Message: msg})
}
