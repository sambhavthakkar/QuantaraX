# Runtime Topology and Ports

Processes:
- Daemon (Go):
  - QUIC listener: 0.0.0.0:4433/udp
  - REST API (grpc-gateway or native): 127.0.0.1:8080
  - Observability (health/metrics/pprof): 127.0.0.1:8081
  - Optional gRPC (native): 127.0.0.1:9090 (future)
- Relay (Go): public relay for NAT traversal (optional in local LAN)
- Bootstrap (Go): token/registration service (optional in local-only dev)

Desktop scenario:
- UI (Flutter) -> REST 127.0.0.1:8080
- UI subscribes SSE at /api/v1/events
- Daemon performs QUIC connections to peer via LAN or relay.

Mobile scenario (via Rust bridge):
- UI (Flutter) -> Dart FFI -> Rust bridge -> local library APIs
- Rust performs control/data operations; no open localhost port required on device.
- Optional: bridge exposes an in-process SSE-like stream mapped to Dart Streams.

Configuration knobs:
- QUIC_ADDR (default :4433)
- REST_ADDR (default 127.0.0.1:8080)
- OBS_ADDR (default 127.0.0.1:8081)
- GRPC_ADDR (optional, default disabled)
