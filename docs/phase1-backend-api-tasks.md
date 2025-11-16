# Phase 1 — Backend API Enablement: Task Breakdown

Owner: Backend
Status: In Progress

## Checklist
- [ ] Install protoc + plugins (protoc-gen-go, protoc-gen-go-grpc, protoc-gen-grpc-gateway)
- [ ] Run `make proto-gen` and commit generated stubs (requires protoc locally)
- [ ] Implement chunk hash mapping in Manifest (if required by UI)
- [ ] Improve CreateTransfer estimated_duration (leverage network/profile)
- [ ] Ensure session progress and events publication from orchestrator
- [ ] Expand SSE payloads: rate_mbps, eta_seconds, chunk_index (when applicable)
- [ ] Normalize error responses to {code,message,details}
- [x] Add curl/Postman examples for all endpoints (docs/api-examples.md)
- [x] Add tmp_rovodev_e2e.sh smoke script
- [x] Optional: X-Auth-Token enforcement when enabled (localhost only by default)

## Notes
- REST: 127.0.0.1:8080; Observability: 127.0.0.1:8081; gRPC: 127.0.0.1:9090
- SSE: /api/v1/events?session_id=...
