Demo: Linux Desktop Bundle (Daemon + App)

Prereqs:
- Go toolchain
- Flutter with linux desktop enabled: `flutter config --enable-linux-desktop`
- Build deps per Flutter docs (GTK, clang, cmake, ninja, pkg-config)

Build and assemble:
- ./scripts/package_linux_bundle.sh

Run:
- QUANTARAX_AUTH_TOKEN=demo frontend/quantarax/build/linux/x64/release/bundle/run_quantarax.sh

Flow:
1) Click "Generate Token" and choose a file.
2) A token and QR are shown. Copy the token if needed.
3) Click "Accept Transfer", paste the token, select an output directory.
4) Watch progress, speed (Mbps), and ETA update live.

Troubleshooting:
- If Unauthorized, ensure QUANTARAX_AUTH_TOKEN matches between launcher and app.
- If dialogs look off, check desktop permissions and run from a user-writable directory.
