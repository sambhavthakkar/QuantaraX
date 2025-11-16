# CLI Reference

## quic_recv
- `--identity-key` Path to Ed25519 private key (defaults to `~/.quantarax/id_ed25519`)


- `--listen` (host:port) Listen address (default `:4433`)
- `--output-dir` Directory to store received chunks (default `./received`)
- ALPN: Uses "quantarax-quic" internally for direct transfers

## quic_send
- `--identity-key` Path to Ed25519 private key (defaults to `~/.quantarax/id_ed25519`)


- `--file` Path to file to send (required)
- Direct mode: `--addr` Receiver address (host:port) [uses ALPN "quantarax-quic"]
- Relay mode: `--relay` Relay address (host:port) and `--target` Target receiver address [relay uses ALPN "quic-relay", target uses "quantarax-quic"]
- `--chunk-index` Chunk index (default 0)
- `--chunk-size` Chunk size bytes (default 1 MiB)
- `--offset` Byte offset in file (default 0)

Examples:
- Direct: `./bin/quic_send --addr localhost:4433 --file ./f.bin --chunk-index 0 --chunk-size 65536`
- Relay: `./bin/quic_send --relay localhost:4434 --target localhost:4436 --file ./f.bin --chunk-index 0`
