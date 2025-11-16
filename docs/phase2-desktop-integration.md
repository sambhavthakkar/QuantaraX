# Phase 2 — Desktop Frontend Integration

Owner: Frontend
Status: Planned

## Goals
Wire Flutter desktop app to the REST API and events.

## Tasks
1) Configuration
- [ ] Add ConfigService with base URL (http://127.0.0.1:8080) and env overrides
- [ ] Provide dev/prod toggles and CORS/localhost notes

2) Clients
- [ ] Add Dio-based API client (or http) with interceptors and retry
- [ ] Add SSE client using package:eventsource or equivalent

3) API Methods
- [ ] createTransfer(filePath)
- [ ] acceptTransfer(token, outputPath)
- [ ] getStatus(sessionId)
- [ ] listTransfers(filter?, limit?, offset?)
- [ ] getKeys()

4) UI Wiring
- [ ] Generate Token: trigger createTransfer and render QR (qr_code_data)
- [ ] Scan QR (clipboard/input): parse token and call acceptTransfer
- [ ] Monitoring: subscribe to SSE events and poll status fallback
- [ ] State mapping: TransferState -> UI; show speed/ETA/progress

5) Error Handling & UX
- [ ] Toasts/dialogs for API errors
- [ ] File path checks and permissions guidance

## Deliverables
- Desktop app can control a local daemon and perform transfers locally or across peers.

## Success Criteria
- Manual QA passes on macOS, Windows, and Linux with local daemon.
