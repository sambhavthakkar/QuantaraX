# QuantaraX

Advanced decentralized high-speed file transfer and collaboration framework.

## Overview

QuantaraX achieves reliable, resumable, secure, and cross-platform file transfers under unreliable networks using QUIC protocol, end-to-end encryption, and forward error correction (FEC).

## Features

- **QUIC-based Transport**: High-performance, multiplexed, UDP-based protocol
- **End-to-End Encryption**: AES-256-GCM with ECDH key exchange
- **Resumable Transfers**: Automatic resume from interruption points
- **Forward Error Correction**: Reed-Solomon encoding for packet loss recovery
- **NAT Traversal**: Built-in relay support for firewall/NAT scenarios
- **Cross-Platform**: Linux, macOS, Windows support

## Architecture

This is a monorepo containing:
- **backend/daemon**: Main file transfer daemon service
- **backend/bootstrap**: Discovery and token service
- **backend/relay**: Relay fallback service
- **frontend/flutter_app**: Cross-platform client (future)

## Prerequisites

- Go 1.22 or higher
- Make (for build automation)
- Git

## Quick Start

### Clone and Build

```bash
git clone https://github.com/quantarax/quantarax.git
cd quantarax
make build
```

### Run Services

```bash
# Start daemon
make run-daemon

# Start bootstrap service
make run-bootstrap
```

### Development Workflow

```bash
# Run tests
make test

# Run linter
make lint

# Clean build artifacts
make clean
```

## Project Structure

```
quantarax/
├── backend/          # Go backend services
│   ├── daemon/       # Main transfer daemon
│   ├── bootstrap/    # Discovery service
│   └── internal/     # Shared internal packages
├── cmd/              # CLI tools
├── frontend/         # Flutter application
├── docs/             # Documentation
└── Makefile          # Build automation
```

## Technology Stack

- **Language**: Go 1.22+
- **Protocol**: QUIC (via quic-go)
- **Encryption**: X25519 + AES-256-GCM
- **Hashing**: BLAKE3
- **FEC**: Reed-Solomon
- **Database**: SQLite / BoltDB
- **Metrics**: Prometheus / Grafana

## Contributing

Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Status

Currently in development. Core transfer functionality implemented with end-to-end encryption, observability, and basic integration tests. Moving to testing phase for advanced features (FEC, relay, resume).
