#!/usr/bin/env bash
set -euo pipefail
# Builds the Go daemon and Flutter Linux desktop bundle, then assembles a runnable bundle with a launcher.
# Requirements: Go toolchain, Flutter with linux-desktop enabled, GTK build deps per Flutter docs.

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
BUNDLE_DIR="$ROOT_DIR/frontend/quantarax/build/linux/x64/release/bundle"

pushd "$ROOT_DIR" >/dev/null

echo "[1/4] Build daemon"
make -s build-daemon

echo "[2/4] Build Flutter linux release"
pushd frontend/quantarax >/dev/null
flutter pub get
flutter build linux --release
popd >/dev/null

if [[ ! -d "$BUNDLE_DIR" ]]; then
  echo "Bundle directory not found: $BUNDLE_DIR" >&2
  exit 1
fi

echo "[3/4] Assemble bundle (daemon + launcher)"
install -m 0755 backend/daemon/quantarax-daemon "$BUNDLE_DIR/daemon" || cp backend/daemon/quantarax-daemon "$BUNDLE_DIR/daemon"
cat > "$BUNDLE_DIR/run_quantarax.sh" <<'LAUNCH'
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
: "${QUANTARAX_AUTH_TOKEN:=demo}"
./daemon > quanta_daemon.log 2>&1 &
DAEMON_PID=$!
trap 'kill "$DAEMON_PID" 2>/dev/null || true' EXIT
exec ./quantarax
LAUNCH
chmod +x "$BUNDLE_DIR/run_quantarax.sh"

echo "[4/4] Done. To run:"
echo "QUANTARAX_AUTH_TOKEN=demo \"$BUNDLE_DIR/run_quantarax.sh\""

popd >/dev/null
