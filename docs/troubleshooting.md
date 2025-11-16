# Troubleshooting Guide

- Ports in use: ensure no other process is listening on relevant QUIC or HTTP ports.
- TLS issues: CLI tools use self-signed TLS for QUIC in dev mode; ensure client/server versions match and ALPNs are compatible.
- Tracing: set `OTEL_EXPORTER_JAEGER_ENDPOINT` and check Jaeger UI.
- pprof: access `/debug/pprof/` on Relay `:8083` and Daemon `:8081`.
- Network loss tests (Linux): tc/netem requires sudo; GitHub Actions workflow enables it only on Linux.

QUIC receive buffer warning
- You may see: `failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 8192 kiB, got: 416 kiB). See https://github.com/quic-go/quic-go/wiki/UDP-Buffer-Sizes`.
- This is typically benign for local tests, but for production/high-throughput you should increase UDP buffers.
- Follow the quic-go wiki instructions to adjust OS UDP buffer sizes:
  - Linux: increase `net.core.rmem_max`/`net.core.wmem_max` and per-socket limits via sysctl.
  - macOS: tune `net.inet.udp.recvspace`/`sendspace`.
  - Windows: adjust relevant registry or rely on autotuning.

Relay readiness and health
- Tests now wait for the relay to be ready by polling `/health` instead of fixed sleeps.
- You can manually check relay health at `http://<relay-host>:8083/health`.
