# Mobile Rust Bridge Strategy

Objective:
Provide Android/iOS support for the daemon capabilities via an in-app Rust library exposing an API equivalent to the daemon REST contract.

Technology choice: flutter_rust_bridge (FRB)
- Rationale: First-class Dart bindings generation, good streaming support, widely used in Flutter ecosystems.
- Alternative: UniFFI (Rust) + Platform Channels (Dart) if FRB is insufficient for advanced cases.

Packaging:
- Android: Build .so via Cargo + Gradle. Add to app/libs and configure Gradle. Support arm64-v8a, armeabi-v7a, x86_64.
- iOS: Build static lib or XCFramework via Cargo + Xcode. Integrate via CocoaPods or direct linking.

API surface (Rust):
- create_transfer(file_path, recipient_id, chunk_size_override?, metadata?) -> CreateTransferResponse
- accept_transfer(transfer_token, output_path, resume_session_id?) -> AcceptTransferResponse
- get_transfer_status(session_id) -> GetTransferStatusResponse
- list_transfers(filter_state?, limit?, offset?) -> ListTransfersResponse
- get_keys() -> GetKeysResponse
- stream_events(session_id_filter?) -> Stream<TransferEvent>

Streams & background:
- FRB provides Stream support; wire internal events to Dart via channels.
- Background transfers: initial version supports foreground execution only; later add Android foreground service and iOS background modes.

Security & permissions:
- Scoped storage on Android; Files app on iOS. Ensure path and file handle handling are platform-compliant.

Testing:
- Device tests for start/accept/status/events.
