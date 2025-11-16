package scenarios

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quantarax/backend/tests/integration/helpers"
)

type createResp struct {
	SessionID         string                 `json:"session_id"`
	TransferToken     string                 `json:"transfer_token"`
	EstimatedDuration int64                  `json:"estimated_duration"`
	Manifest          map[string]interface{} `json:"manifest"`
}

type acceptResp struct {
	SessionID string `json:"session_id"`
}

// TestDaemonREST_E2E performs create -> accept -> observe events via SSE
func TestDaemonREST_E2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start daemon via helper
	daemon := helpers.NewDaemonRunner("../../../../bin/daemon", "127.0.0.1:9090", "127.0.0.1:8080", "127.0.0.1:8081")
	if err := daemon.Start(); err != nil {
		t.Fatalf("daemon start: %v", err)
	}
	defer daemon.Stop()
	os.Setenv("QUANTARAX_AUTH_TOKEN", "testtoken")

	// Generate a temp file to send
	fg, err := helpers.NewFileGenerator()
	if err != nil { t.Fatalf("filegen: %v", err) }
	defer fg.Cleanup()
	filePath, _, err := fg.GenerateSmallFile("api-e2e.bin")
	if err != nil { t.Fatalf("gen: %v", err) }
	outDir := fg.MakeTempDir("recv-api")
	outPath := filepath.Join(outDir, "received.bin")

	base := "http://127.0.0.1:8080"
	h := map[string]string{"Content-Type":"application/json", "X-Auth-Token":"testtoken"}

	// Create transfer
	cbody := map[string]interface{}{"file_path": filePath, "recipient_id":"peer-local"}
	cjs, _ := json.Marshal(cbody)
	creq, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/transfer/create", bytes.NewReader(cjs))
	for k,v := range h { creq.Header.Set(k,v) }
	cres, err := http.DefaultClient.Do(creq)
	if err != nil { t.Fatalf("create: %v", err) }
	defer cres.Body.Close()
	if cres.StatusCode != 200 { b,_:=io.ReadAll(cres.Body); t.Fatalf("create status=%d body=%s", cres.StatusCode, string(b)) }
	var c createResp
	_ = json.NewDecoder(cres.Body).Decode(&c)
	if c.SessionID == "" || c.TransferToken == "" { t.Fatalf("bad create resp: %+v", c) }

	// Accept transfer
	abody := map[string]interface{}{"transfer_token": c.TransferToken, "output_path": outPath}
	ajs, _ := json.Marshal(abody)
	areq, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/transfer/accept", bytes.NewReader(ajs))
	for k,v := range h { areq.Header.Set(k,v) }
	ares, err := http.DefaultClient.Do(areq)
	if err != nil { t.Fatalf("accept: %v", err) }
	defer ares.Body.Close()
	if ares.StatusCode != 200 { b,_:=io.ReadAll(ares.Body); t.Fatalf("accept status=%d body=%s", ares.StatusCode, string(b)) }
	var a acceptResp
	_ = json.NewDecoder(ares.Body).Decode(&a)
	if a.SessionID == "" { t.Fatalf("bad accept resp: %+v", a) }

	// Observe SSE for a few lines
	esreq, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/events?session_id="+c.SessionID, nil)
	esreq.Header.Set("X-Auth-Token", "testtoken")
	esres, err := http.DefaultClient.Do(esreq)
	if err != nil { t.Fatalf("sse: %v", err) }
	defer esres.Body.Close()
	if esres.StatusCode != 200 { b,_:=io.ReadAll(esres.Body); t.Fatalf("sse status=%d body=%s", esres.StatusCode, string(b)) }
	reader := bufio.NewReader(esres.Body)
	lines := 0
	deadline := time.Now().Add(5*time.Second)
	for time.Now().Before(deadline) && lines < 2 {
		b, err := reader.ReadBytes('\n')
		if err != nil { break }
		if len(bytes.TrimSpace(b)) == 0 { continue }
		lines++
	}
	if lines == 0 { t.Fatalf("no SSE lines observed") }
}
