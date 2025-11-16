# QuantaraX

Advanced decentralized high-speed file transfer and collaboration framework.

## Overview

QuantaraX achieves reliable, resumable, secure, and cross-platform file transfers under unreliable networks using QUIC protocol, end-to-end encryption, forward error correction (FEC), and distributed architecture. It supports direct peer-to-peer transfers with relay fallbacks, real-time monitoring, and a Flutter-based cross-platform client.

## Features

- **QUIC-based Transport**: High-performance, multiplexed, UDP-based protocol with built-in reliability and security.
- **End-to-End Encryption**: TLS 1.3 with AES-256-GCM and ECDH key exchange.
- **Resumable Transfers**: Automatic resume from interruption points with session persistence.
- **Forward Error Correction**: Reed-Solomon encoding for packet loss recovery without retransmission.
- **NAT Traversal**: Built-in relay support for firewall/NAT scenarios.
- **Cross-Platform**: Native support for Linux, macOS, Windows, iOS, Android, and Web via Flutter.
- **Observability**: OpenTelemetry tracing, Prometheus metrics, and Grafana dashboards.
- **Distributed Architecture**: Daemon, bootstrap, and relay services for scalable deployments.
- **Rust Bridge (Planned)**: Native mobile performance via flutter_rust_bridge for Android/iOS.

## Architecture

This is a monorepo containing:
- **backend/daemon**: Main file transfer daemon service with QUIC transport and FEC.
- **backend/bootstrap**: Discovery and token registration service.
- **backend/relay**: Relay fallback service for NAT traversal.
- **frontend/quantarax**: Flutter-based cross-platform client (desktop, mobile, web).
- **monitoring/**: Grafana dashboards and Prometheus configuration.
- **scripts/**: Build and packaging scripts.

## Prerequisites

- Go 1.24.0 or higher
- Flutter SDK ^3.9.2 (for frontend development)
- Make (for build automation)
- Docker (for containerized builds)
- Git

## Installation

### Backend (Go)

1. Install Go 1.24.0+ from [golang.org](https://golang.org/dl/).
2. Clone and build:
   ```bash
   git clone https://github.com/quantarax/quantarax.git
   cd quantarax
   make build  # Builds all binaries to bin/
   ```
3. For cross-platform releases:
   ```bash
   make release  # Builds for Windows, Linux, macOS (AMD64/ARM64)
   ```

### Frontend (Flutter)

1. Install Flutter SDK ^3.9.2 from [flutter.dev](https://flutter.dev/docs/get-started/install).
2. Build and run:
   ```bash
   cd frontend/quantarax
   flutter pub get
   flutter run  # Select platform (linux, windows, macos, android, ios, web)
   ```
3. For releases:
   ```bash
   flutter build linux   # Or android, ios, web, etc.
   ```

### Monitoring (Optional)

1. Install Docker.
2. Run Prometheus and Grafana:
   ```bash
   docker-compose up -d  # Uses docker-compose.yml
   ```
3. Access Grafana at `http://localhost:3000` (admin/admin).

## Quick Start

### Run Full Stack Locally

```bash
# Backend services
make run-daemon &
make run-bootstrap &
make run-relay &

# Frontend
cd frontend/quantarax && flutter run -d linux  # Or your platform
```

### CLI Usage Examples

- **Direct Transfer**:
  ```bash
  # Receiver
  ./bin/quic_recv --listen :4433 --output-dir ./received

  # Sender
  ./bin/quic_send --addr localhost:4433 --file /path/to/file.bin
  ```

- **Relayed Transfer**:
  ```bash
  # Relay
  ./bin/relay --listen :4434

  # Receiver
  ./bin/quic_recv --listen :4436 --output-dir ./received

  # Sender
  ./bin/quic_send --relay localhost:4434 --target localhost:4436 --file /path/to/file.bin
  ```

### API Examples

See [docs/api-examples.md](docs/api-examples.md) for curl commands to interact with the daemon API.

### Development Workflow

```bash
# Run all tests
make test

# Run linter
make lint

# Clean build artifacts
make clean

# Build release binaries
make release
```

## Project Structure

```
quantarax/
├── backend/              # Go backend services
│   ├── daemon/           # Main transfer daemon
│   ├── bootstrap/        # Discovery service
│   ├── relay/            # Relay service
│   ├── cmd/              # CLI tools (chunker, keygen, quic_send/recv)
│   └── internal/         # Shared packages (chunker, crypto, fec, etc.)
├── frontend/quantarax/   # Flutter cross-platform app
│   ├── lib/              # Dart source
│   ├── android/          # Android-specific
│   ├── ios/              # iOS-specific
│   └── web/              # Web-specific
├── monitoring/           # Grafana/Prometheus configs
├── docs/                 # Documentation
├── scripts/              # Build/packaging scripts
├── Dockerfile.*          # Container builds
└── Makefile              # Build automation
```

## Technology Stack

### Backend
- **Language**: Go 1.24.0
- **Protocol**: QUIC (via quic-go)
- **Encryption**: TLS 1.3, AES-256-GCM, X25519 ECDH
- **Hashing**: BLAKE3
- **FEC**: Reed-Solomon (via klauspost/reedsolomon)
- **Database**: SQLite (via modernc.org/sqlite), BoltDB
- **Observability**: OpenTelemetry (tracing), Prometheus (metrics), Jaeger
- **Containerization**: Docker

### Frontend
- **Framework**: Flutter (Dart SDK ^3.9.2)
- **Platforms**: iOS, Android, Linux, macOS, Windows, Web
- **UI Libraries**: fl_chart, google_fonts, shimmer, qr_flutter
- **Networking**: http package

### Monitoring
- **Metrics**: Prometheus
- **Visualization**: Grafana dashboards
- **Tracing**: Jaeger

### Build & Deployment
- **Build System**: Makefile with cross-compilation
- **Packaging**: Docker images, tarballs for releases
- **CI/CD**: Docker-based builds

### Future (Rust Bridge)
- **Framework**: flutter_rust_bridge (FRB) for Dart-Rust bindings
- **Purpose**: Native mobile performance for daemon APIs on Android/iOS
- **Features**: Streaming events, background transfers, platform-compliant storage

## Contributing

Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Status

Nearly complete. Core transfer functionality implemented with QUIC, FEC, encryption, observability, and cross-platform support. Flutter frontend deployed across all targets. Rust bridge planned for mobile optimization. Ready for final testing and production deployment.
