# User Guide

This guide shows how to run direct and relayed transfers using the CLI.

Direct transfer:
1. Start receiver (choose any free UDP port): `./bin/quic_recv --listen localhost:4433 --output-dir ./received`
   - Note: The QUIC ALPN used by the receiver is set internally to "quantarax-quic" and is handled automatically by the sender.
2. Send a chunk: `./bin/quic_send --addr localhost:4433 --file ./path/to/file --chunk-index 0 --chunk-size 65536`
3. Check output: `ls ./received` (expect `chunk_0000.bin`)

Relay-mediated transfer:
1. Start relay (choose any free UDP port): `./bin/relay --listen :4434`
   - The relay listens using ALPN "quic-relay" and accepts a control stream containing the target address.
2. Start receiver on another port: `./bin/quic_recv --listen localhost:4436 --output-dir ./received`
3. Send via relay (sender connects to relay and specifies the receiver target):
   `./bin/quic_send --relay localhost:4434 --target localhost:4436 --file ./path/to/file --chunk-index 0`
   - The sender opens a control stream to the relay, writes the target address, waits for an "OK" response, then opens a data stream for the payload.

Bootstrap service (discovery placeholder):
- Start: `./bin/bootstrap --listen :8082`
- Interact with API: `/api/v1/token`, `/api/v1/lookup/`, etc.

Tracing and profiling:
- Set `OTEL_EXPORTER_JAEGER_ENDPOINT` to export spans to Jaeger.
- Access pprof: Relay `:8083/debug/pprof/`, Daemon `:8081/debug/pprof/`.
