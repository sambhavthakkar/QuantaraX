# API Integration Contract (Daemon REST + SSE)

Base URL: http://localhost:8080
Observability: http://localhost:8081 (health, metrics)
QUIC: udp://0.0.0.0:4433

Auth: none (local-only dev). Optional header `X-Auth-Token` for future use.
Content-Type: application/json

Endpoints:
- POST /api/v1/transfer/create
  - Request: { file_path: string, recipient_id: string, chunk_size_override?: number, metadata?: { [k: string]: string } }
  - Response: { session_id: string, transfer_token: string, manifest: Manifest, qr_code_data: string, estimated_duration: number }
- POST /api/v1/transfer/accept
  - Request: { transfer_token: string, output_path: string, resume_session_id?: string }
  - Response: { session_id: string, manifest: Manifest, sender_public_key: string(base64), total_chunks: number, estimated_size: number }
- GET /api/v1/transfer/{session_id}/status
  - Response: { state: TransferState, progress_percent: number, chunks_transferred: number, total_chunks: number, bytes_transferred: number, transfer_rate_mbps: number, estimated_time_remaining: number, error_message?: string }
- GET /api/v1/transfers?state=&limit=&offset=
  - Response: { transfers: TransferSummary[], total_count: number, has_more: boolean }
- GET /api/v1/keys
  - Response: { public_key_base64: string, fingerprint: string }
- GET /api/v1/events[?session_id=]
  - Response: SSE stream of `TransferEvent` JSON objects, one per line.

Types:
- Manifest { file_name: string, file_size: number, chunk_size: number, total_chunks: number, merkle_root: string(base64), chunk_hashes: string[](base64) }
- TransferState: PENDING | ACTIVE | PAUSED | COMPLETED | FAILED
- TransferDirection: SEND | RECEIVE
- TransferSummary { session_id: string, file_name: string, state: TransferState, progress_percent: number, start_time: number(epoch ms), direction: TransferDirection }
- TransferEvent { session_id: string, event_type: string, timestamp: number(epoch ms), progress_percent?: number, message?: string, metadata?: { [k: string]: string } }

Error model:
- HTTP errors return { code: string, message: string, details?: object } with appropriate status codes.
- Common codes: INVALID_ARGUMENT, NOT_FOUND, FAILED_PRECONDITION, INTERNAL.

Versioning:
- Prefix with /api/v1. Breaking changes bump version.

Compatibility:
- Desktop Flutter apps consume REST + SSE directly.
- Mobile apps (Android/iOS) will use a Rust bridge to provide equivalent APIs locally, mapping to the same shapes.
