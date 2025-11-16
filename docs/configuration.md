# Configuration Reference

## Relay
- `--listen` QUIC listen address (default `:4433`)
- `--max-connections` Maximum concurrent connections
- `--auth-mode` Authentication mode (none|token)
- Observability: http `:8083` health/metrics/pprof

## Bootstrap
- `--listen` HTTP listen address (default `:8082`)
- `--token-ttl-max` Maximum token TTL
- Rate limiting: per-IP limits on endpoints

## Daemon
- QUIC address, chunk size, worker count (via config package)
- Observability: http `:8081` metrics/health/pprof

## Telemetry
- `OTEL_EXPORTER_JAEGER_ENDPOINT` Jaeger collector endpoint for traces
