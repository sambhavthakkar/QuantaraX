.PHONY: build test lint run-daemon run-bootstrap run-relay clean build-chunker test-chunker build-keygen test-crypto bench-crypto build-quic build-daemon test-daemon test-bootstrap test-relay build-bootstrap build-relay proto-gen

# Build all binaries
build:
	@echo "Building binaries..."
	@mkdir -p bin
	cd backend/daemon && go build -o ../../bin/daemon.exe .
	cd backend/bootstrap && go build -o ../../bin/bootstrap.exe .
	cd backend/relay && go build -o ../../bin/relay.exe .
	cd backend && go build -o ../bin/chunker.exe ./cmd/chunker
	cd backend && go build -o ../bin/keygen.exe ./cmd/keygen
	cd backend && go build -o ../bin/quic_send.exe ./cmd/quic_send
	cd backend && go build -o ../bin/quic_recv.exe ./cmd/quic_recv
	@echo "Build complete. Binaries in bin/"

# Build daemon service (Phase 5)
build-daemon:
	@echo "Building daemon..."
	@mkdir -p bin
	cd backend/daemon && go build -o ../../bin/daemon.exe .
	@echo "Daemon built: bin/daemon.exe"

# Build chunker CLI
build-chunker:
	@echo "Building chunker..."
	@mkdir -p bin
	cd backend && go build -o ../bin/chunker.exe ./cmd/chunker
	@echo "Chunker built: bin/chunker.exe"

# Build keygen CLI
build-keygen:
	@echo "Building keygen..."
	@mkdir -p bin
	cd backend && go build -o ../bin/keygen.exe ./cmd/keygen
	@echo "Keygen built: bin/keygen.exe"

# Build bootstrap service (Phase 9)
build-bootstrap:
	@echo "Building bootstrap service..."
	@mkdir -p bin
	cd backend/bootstrap && go build -o ../../bin/bootstrap.exe .
	@echo "Bootstrap built: bin/bootstrap.exe"

# Build relay service (Phase 10)
build-relay:
	@echo "Building relay service..."
	@mkdir -p bin
	cd backend/relay && go build -o ../../bin/relay.exe .
	@echo "Relay built: bin/relay.exe"

# Run all tests
test:
	@echo "Running tests..."
	cd backend/daemon && go test ./...
	cd backend/bootstrap && go test ./...
	cd backend/relay && go test ./...
	cd backend && go test ./internal/chunker/...
	cd backend && go test ./internal/crypto/...
	@echo "All tests passed"

# Test chunker package
test-chunker:
	@echo "Testing chunker..."
	cd backend && go test -v ./internal/chunker/...
	@echo "Chunker tests passed"

# Test daemon (Phase 5)
test-daemon:
	@echo "Testing daemon..."
	cd backend/daemon && go test -v ./...
	@echo "Daemon tests passed"

# Test crypto package
test-crypto:
	@echo "Testing crypto..."
	cd backend && go test -v ./internal/crypto/...
	@echo "Crypto tests passed"

# Test bootstrap service (Phase 9)
test-bootstrap:
	@echo "Testing bootstrap..."
	cd backend/bootstrap && go test -v ./...
	@echo "Bootstrap tests passed"

# Test relay service (Phase 10)
test-relay:
	@echo "Testing relay..."
	cd backend/relay && go test -v ./...
	@echo "Relay tests passed"

# Benchmark crypto operations
bench-crypto:
	@echo "Benchmarking crypto..."
	cd backend && go test -bench=. ./internal/crypto/...

# Build QUIC POC tools
build-quic:
	@echo "Building QUIC POC tools..."
	@mkdir -p bin
	cd backend && go build -o ../bin/quic_send.exe ./cmd/quic_send
	cd backend && go build -o ../bin/quic_recv.exe ./cmd/quic_recv
	@echo "QUIC tools built: bin/quic_send.exe, bin/quic_recv.exe"

# Generate Protocol Buffer code (Phase 5)
proto-gen:
	@echo "Generating Protocol Buffer code..."
	@echo "Note: protoc must be installed with go plugins"
	cd backend/daemon/api/v1 && \
		protoc -I . -I ./google -I ../../.. \
		--go_out=. --go-grpc_out=. \
		--grpc-gateway_out=. \
		daemon.proto
	@echo "Protocol Buffer code generated"

# Lint code
lint:
	@echo "Running linters..."
	cd backend/daemon && go vet ./...
	cd backend/bootstrap && go vet ./...
	cd backend/relay && go vet ./...
	cd backend && go vet ./internal/...
	cd backend && go vet ./cmd/...
	@echo "Lint complete"

# Run daemon in dev mode
run-daemon:
	cd backend/daemon && go run main.go

# Run bootstrap service in dev mode
run-bootstrap:
	cd backend/bootstrap && go run main.go

# Run relay service in dev mode
run-relay:
	cd backend/relay && go run main.go

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	@echo "Clean complete"

# Integration tests (Phase 11)
test-integration:
	@echo "Running integration tests..."
	cd backend/tests/integration && go test -v ./scenarios/
	@echo "Integration tests complete"

# Release builds (Phase 14)
release:
	@echo "Building release binaries..."
	@mkdir -p bin/release
	GOOS=windows GOARCH=amd64 go build -o bin/release/daemon-windows-amd64.exe ./backend/daemon
	GOOS=linux GOARCH=amd64 go build -o bin/release/daemon-linux-amd64 ./backend/daemon
	GOOS=darwin GOARCH=amd64 go build -o bin/release/daemon-darwin-amd64 ./backend/daemon
	GOOS=darwin GOARCH=arm64 go build -o bin/release/daemon-darwin-arm64 ./backend/daemon
	@echo "Release binaries built"

package: release
	@echo "Creating release packages..."
	cd bin/release && tar -czf quantarax-daemon-1.0.0-linux-amd64.tar.gz daemon-linux-amd64
	cd bin/release && tar -czf quantarax-daemon-1.0.0-darwin-amd64.tar.gz daemon-darwin-amd64
	cd bin/release && tar -czf quantarax-daemon-1.0.0-darwin-arm64.tar.gz daemon-darwin-arm64
	@echo "Packages created"

# Run all validations
validate: lint test test-integration
	@echo "All validation passed"
