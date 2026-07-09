#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
MACOS_DIR="$PROJECT_DIR/macos-ai-critic"

APP_NAME="${APP_NAME:-ai-critic-macos}"
BUNDLE_ID="${BUNDLE_ID:-com.xhd2015.ai-critic-macos}"
SWIFT_BUILD_CONFIG="${SWIFT_BUILD_CONFIG:-release}"
SWIFT_EXECUTABLE="${SWIFT_EXECUTABLE:-ai-critic-macos}"
# MODE=remote / REMOTE_ONLY=1 / SKIP_SERVER=1: remote menu-bar product;
# does not embed the ai-critic server binary (no local daemon).
MODE="${MODE:-local}"
SKIP_SERVER="${SKIP_SERVER:-0}"
REMOTE_ONLY="${REMOTE_ONLY:-0}"
if [[ "$MODE" == "remote" || "$REMOTE_ONLY" == "1" ]]; then
    SKIP_SERVER=1
fi
BUNDLE_DIR="$MACOS_DIR/$APP_NAME.app"
CONTENTS="$BUNDLE_DIR/Contents"
MACOS_BIN="$CONTENTS/MacOS"

cd "$PROJECT_DIR"

if [[ "$SKIP_SERVER" != "1" ]]; then
    echo "==> Building frontend (ai-critic-react/dist)"
    go run ./script/vite/build

    echo "==> Building ai-critic CLI"
    mkdir -p "$MACOS_DIR/.build"
    go run ./script/server/build -o "$MACOS_DIR/.build/ai-critic"
else
    echo "==> Skipping frontend + server binary embed (remote / SKIP_SERVER)"
fi

echo "==> Building $APP_NAME ($SWIFT_BUILD_CONFIG, bundle: $BUNDLE_ID)"
cd "$MACOS_DIR"
if [[ "$SWIFT_EXECUTABLE" != "ai-critic-macos" ]]; then
    swift build -c "$SWIFT_BUILD_CONFIG" --product "$SWIFT_EXECUTABLE"
else
    swift build -c "$SWIFT_BUILD_CONFIG"
fi

echo "==> Creating .app bundle at $BUNDLE_DIR"
rm -rf "$BUNDLE_DIR"
mkdir -p "$MACOS_BIN"

BIN_PATH="$(swift build -c "$SWIFT_BUILD_CONFIG" --show-bin-path)/$SWIFT_EXECUTABLE"
cp "$BIN_PATH" "$MACOS_BIN/$SWIFT_EXECUTABLE"
if [[ "$SKIP_SERVER" != "1" ]]; then
    cp "$MACOS_DIR/.build/ai-critic" "$MACOS_BIN/ai-critic"
    chmod +x "$MACOS_BIN/ai-critic"
fi

cat > "$CONTENTS/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>$SWIFT_EXECUTABLE</string>
    <key>CFBundleIdentifier</key>
    <string>$BUNDLE_ID</string>
    <key>CFBundleName</key>
    <string>$APP_NAME</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>LSMinimumSystemVersion</key>
    <string>15.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
PLIST

echo "==> Ad-hoc code signing"
codesign --force --deep -s - "$BUNDLE_DIR" 2>/dev/null || true

echo ""
echo "==> App bundle ready: $BUNDLE_DIR"