#!/usr/bin/env bash
# Build a single-file AppImage for the Quantarax Linux desktop app.
# Requirements on the build machine:
#  - Go toolchain
#  - Flutter with linux-desktop enabled and GTK build deps
#  - appimagetool available in PATH (https://github.com/AppImage/AppImageKit/releases)
#
# Usage:
#  ./scripts/package_linux_appimage.sh
#
# Output:
#  dist/Quantarax-x86_64.AppImage

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
BUNDLE_DIR="$ROOT_DIR/frontend/quantarax/build/linux/x64/release/bundle"
DIST_DIR="$ROOT_DIR/dist"
APPDIR="$DIST_DIR/AppDir"
APPIMAGE_OUT="$DIST_DIR/Quantarax-x86_64.AppImage"

mkdir -p "$DIST_DIR"

# 1) Build components
pushd "$ROOT_DIR" >/dev/null
  echo "[1/4] Building daemon"
  make -s build-daemon
  echo "[2/4] Building Flutter linux release"
  pushd frontend/quantarax >/dev/null
    flutter pub get
    flutter build linux --release
  popd >/dev/null
popd >/dev/null

# 2) Assemble AppDir structure
rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/share/applications" "$APPDIR/usr/share/icons/hicolor/256x256/apps"

# Copy binaries
install -m 0755 "$BUNDLE_DIR/quantarax" "$APPDIR/usr/bin/quantarax"
install -m 0755 "$ROOT_DIR/backend/daemon/quantarax-daemon" "$APPDIR/usr/bin/daemon"

# AppRun launcher
cat > "$APPDIR/AppRun" << 'LAUNCH'
#!/usr/bin/env bash
set -euo pipefail
HERE="$(dirname "$(readlink -f "$0")")"
cd "$HERE"

# Prefer env or default
echo "Starting Quantarax Daemon..."
: "${QUANTARAX_AUTH_TOKEN:=demo}"
"$HERE/usr/bin/daemon" > "$HERE/daemon.log" 2>&1 &
DAEMON_PID=$!
trap 'kill "$DAEMON_PID" 2>/dev/null || true' EXIT

exec "$HERE/usr/bin/quantarax"
LAUNCH
chmod +x "$APPDIR/AppRun"

# Desktop file
cat > "$APPDIR/quantarax.desktop" << 'DESKTOP'
[Desktop Entry]
Type=Application
Name=Quantarax
Exec=AppRun
Icon=quantarax
Categories=Network;Utility;
Comment=Secure peer-to-peer file transfer
Terminal=false
DESKTOP

# Icon: use existing app icon from Flutter project if available, else generate a placeholder
ICON_SRC="$ROOT_DIR/frontend/quantarax/linux/runner/resources/app_icon.ico"
PNG_OUT="$APPDIR/usr/share/icons/hicolor/256x256/apps/quantarax.png"
if command -v convert >/dev/null 2>&1 && [ -f "$ICON_SRC" ]; then
  # Use ImageMagick to convert ICO to PNG
  convert "$ICON_SRC[0]" -resize 256x256 "$PNG_OUT" || true
fi
# Fallback: copy any available PNG asset
if [ ! -f "$PNG_OUT" ]; then
  if [ -f "$ROOT_DIR/frontend/quantarax/macos/Runner/Assets.xcassets/AppIcon.appiconset/app_icon_256.png" ]; then
    cp "$ROOT_DIR/frontend/quantarax/macos/Runner/Assets.xcassets/AppIcon.appiconset/app_icon_256.png" "$PNG_OUT"
  else
    # Placeholder single-pixel to satisfy Icon reference
    printf "\x89PNG\r\n\x1a\n" > "$PNG_OUT"
  fi
fi

# 3) Validate appimagetool
if ! command -v appimagetool >/dev/null 2>&1; then
  echo "ERROR: appimagetool not found in PATH. Please install from:"
  echo "  https://github.com/AppImage/AppImageKit/releases"
  echo "Then re-run this script."
  exit 2
fi

# 4) Build AppImage
pushd "$DIST_DIR" >/dev/null
  echo "[4/4] Building AppImage -> $APPIMAGE_OUT"
  appimagetool "$APPDIR" "$APPIMAGE_OUT"
  chmod +x "$APPIMAGE_OUT"
  echo "Done: $APPIMAGE_OUT"
  echo "Run: chmod +x $APPIMAGE_OUT && QUANTARAX_AUTH_TOKEN=demo ./$APPIMAGE_OUT"
popd >/dev/null
