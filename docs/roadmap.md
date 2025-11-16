# Quantarax Delivery Roadmap (6 Phases)

This document tracks the multi-phase delivery plan, tasks, deliverables, and success criteria.

## Phase 0 — Planning and Foundations (Done)
- Status: Completed
- Deliverables:
  - ADR 0001: API transport choice (REST + SSE)
  - Cleaned daemon.proto (removed duplication, aligned go_package)
  - API Integration Contract (endpoints, payloads, events, errors)
  - Runtime topology (ports/processes) and Rust bridge strategy
- Links:
  - docs/adr/0001-api-transport.md
  - docs/api-integration-contract.md
  - docs/runtime-topology.md
  - docs/mobile-rust-bridge-strategy.md

## Phase 1 — Backend API Enablement (Done)
- Goals: Implement Daemon API and expose HTTP endpoints and event streams.
- Status: Completed
- Implementation highlights:
  - gRPC handlers, grpc-gateway REST server, and native SSE implemented.
  - main.go starts gRPC (127.0.0.1:9090), REST (127.0.0.1:8080), and observability (127.0.0.1:8081).
- Notes:
  - Ensure `protoc` and plugins are installed locally; run `make proto-gen` to generate stubs.
- Deliverables:
  - Running daemon with REST API and SSE
  - Basic API usage docs (curl/Postman) and E2E smoke test
- Success criteria:
  - E2E local test: create -> accept -> observe events -> complete

## Phase 2 — Desktop Frontend Integration (1–2 sprints)
- Goals: Wire Flutter desktop app to REST API and SSE.
- Tasks:
  1) Config service:
     - Base URL/ports, environment overrides (dev/prod)
  2) REST client and SSE client:
     - Choose Dio or http (recommend Dio) and SSE package
  3) API methods:
     - createTransfer(filePath)
     - acceptTransfer(token, outputPath)
     - getStatus(sessionId)
     - listTransfers(filter?, limit?, offset?)
     - getKeys()
  4) UI wiring:
     - Generate Token -> createTransfer -> render QR from qr_code_data
     - Scan QR (clipboard/input) -> acceptTransfer
     - Monitoring/Progress -> subscribe to events + poll fallback
     - State mapping: daemon TransferState -> UI; show speed/ETA/progress
  5) Error handling & UX polish:
     - Display API errors in toasts/dialogs
     - File path validation; permissions/edge cases
- Deliverables:
  - Desktop app controlling local daemon; transfers locally/across peers
- Success criteria:
  - Manual QA on macOS/Windows/Linux: full user flow works

## Phase 3 — Rust Bridge for Android/iOS (2–3 sprints)
- Goals: Provide mobile integration via Rust, mirroring daemon APIs.
- Tasks:
  - Select flutter_rust_bridge; set up Cargo + Gradle/CocoaPods builds
  - Implement Rust APIs: create/accept/getStatus/list/getKeys/stream_events
  - Map streams to Dart Streams; handle file I/O permissions
  - Foreground execution initially; plan background modes for later
- Deliverables: Rust library integrated, Flutter bindings, basic mobile flows
- Success criteria: Android/iOS can send/receive with visible progress

## Phase 4 — UX Hardening and Advanced Features (1–2 sprints)
- Goals: Improve resilience and UX.
- Tasks: Resume/Retry flows (DTN), QR scanning flow, error UX, permission flows, optional tuning controls
- Deliverables: Polished UX and robust error handling
- Success criteria: Usability tests pass; retry/resume works

## Phase 5 — Observability, Security, Packaging (1–2 sprints)
- Goals: Production-readiness.
- Tasks: Metrics/tracing/logging, health diagnostics, identity key UI, auth toggles, installers/packaging
- Deliverables: Operator docs, packages for desktop
- Success criteria: CI release artifacts, basic security review

## Phase 6 — QA, E2E Tests, Documentation (1 sprint)
- Goals: Confidence and documentation.
- Tasks: E2E tests, device tests, Confluence docs, runbooks
- Deliverables: CI pipelines and comprehensive docs
- Success criteria: Green CI across targets; documented onboarding
