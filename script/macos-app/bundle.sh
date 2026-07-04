#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
MACOS_DIR="$PROJECT_DIR/macos-ai-critic"

APP_NAME="${APP_NAME:-ai-critic-macos}"
BUNDLE_ID="${BUNDLE_ID:-com.xhd2015.ai-critic-macos}"
SWIFT_BUILD_CONFIG="${SWIFT_BUILD_CONFIG:-release}"
SWIFT_EXECUTABLE="${SWIFT_EXECUTABLE:-ai-critic-macos}"
BUNDLE_DIR="$MACOS_DIR/$APP_NAME.app"
CONTENTS="$BUNDLE_DIR/Contents"
MACOS_BIN="$CONTENTS/MacOS"

echo "==> Building ai-critic CLI"
cd "$PROJECT_DIR"
mkdir -p "$MACOS_DIR/.build"
go build -o "$MACOS_DIR/.build/ai-critic" .

echo "==> Building $APP_NAME ($SWIFT_BUILD_CONFIG, bundle: $BUNDLE_ID)"
cd "$MACOS_DIR"
swift build -c "$SWIFT_BUILD_CONFIG"

echo "==> Creating .app bundle at $BUNDLE_DIR"
rm -rf "$BUNDLE_DIR"
mkdir -p "$MACOS_BIN"

BIN_PATH="$(swift build -c "$SWIFT_BUILD_CONFIG" --show-bin-path)/$SWIFT_EXECUTABLE"
cp "$BIN_PATH" "$MACOS_BIN/$SWIFT_EXECUTABLE"
cp "$MACOS_DIR/.build/ai-critic" "$MACOS_BIN/ai-critic"
chmod +x "$MACOS_BIN/ai-critic"

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