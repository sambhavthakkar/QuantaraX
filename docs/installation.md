# Installation Guide

This guide covers prerequisites, building, and running QuantaraX components with observability.

Prerequisites:
- Go 1.22+
- Docker (for containers) and docker-compose (for demo)
- Optional: Prometheus, Grafana, and Jaeger (docker-compose provided)

Build binaries:
- Daemon: `cd backend/daemon && go build -o ../../bin/daemon .`
- Bootstrap: `cd backend/bootstrap && go build -o ../../bin/bootstrap .`
- Relay: `cd backend/relay && go build -o ../../bin/relay .`
- CLI tools: `cd backend/cmd/quic_recv && go build -o ../../../bin/quic_recv .` and `cd backend/cmd/quic_send && go build -o ../../../bin/quic_send .`

Run observability locally:
- Set `OTEL_EXPORTER_JAEGER_ENDPOINT=http://localhost:14268/api/traces` to enable tracing.
- Daemon exposes metrics/pprof on `:8081`, Relay on `:8083`, Bootstrap pprof on `/debug/pprof/`.

Run demo with Docker Compose:
- `docker-compose up --build`
- Jaeger UI: http://localhost:16686
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000

