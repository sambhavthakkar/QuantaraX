# ADR 0001: API Transport Choice (REST + SSE)

Status: Accepted
Date: 2025-11-16

Context:
- The daemon exposes operations used by desktop and mobile clients (Flutter), plus potential web UI.
- gRPC provides efficient streaming and typed contracts but requires grpc-web or proxies to support browsers.
- REST + SSE (Server-Sent Events) is widely compatible across platforms, including the web.

Decision:
- Use REST for request/response endpoints and SSE for realtime event streaming.
- Keep gRPC service definitions (daemon.proto) as source of truth for types and service shape.
- Expose REST via grpc-gateway or native HTTP handlers mapped to the same protobuf messages.
- For events:
  - Preferred: native SSE endpoint emitting JSON TransferEvent lines.
  - Alternative: grpc-gateway websocket proxy if SSE proves insufficient.

Consequences:
- Pros: Broad client compatibility, simpler browser integration, easier debugging with curl/Postman.
- Cons: Slightly more boilerplate than pure gRPC; SSE limits (one-way server push, single stream per connection).

Notes:
- Keep option to expose native gRPC on localhost for advanced/CLI clients.
- Ensure CORS and localhost-only binding for REST in desktop scenarios.
