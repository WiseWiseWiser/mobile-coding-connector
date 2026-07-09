#!/usr/bin/env bash
# install-remote.sh — build and install the remote menu-bar app
# (ai-critic-remote-macos.app / AI Critic(Remote)).
#
# REMOTE_ONLY: this install does not embed the ai-critic server binary.
# The remote app talks to an already-running remote-agent server over HTTP
# with Bearer auth; there is no local keep-alive daemon or bundled server.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

APP_NAME="${APP_NAME:-ai-critic-remote-macos}"
BUNDLE_ID="${BUNDLE_ID:-com.xhd2015.ai-critic-remote-macos}"
SWIFT_EXECUTABLE="${SWIFT_EXECUTABLE:-ai-critic-remote-macos}"
SWIFT_BUILD_CONFIG="${SWIFT_BUILD_CONFIG:-release}"
SOURCE_APP="$PROJECT_DIR/macos-ai-critic/$APP_NAME.app"
INSTALL_ROOT="${INSTALL_ROOT:-/Applications}"
TARGET_APP="$INSTALL_ROOT/$APP_NAME.app"
OPEN_AFTER_INSTALL=1
# SKIP_SERVER=1: no server binary embed (remote product).
SKIP_SERVER=1
REMOTE_ONLY=1

usage() {
    cat <<EOF
Usage: $(basename "$0") [options]

Build $APP_NAME.app (remote menu bar) and install to $INSTALL_ROOT.

The remote app does not embed a server binary and does not spawn a local
keep-alive daemon. Configure a remote server + token via Configure… in the app
(or ~/.ai-critic/remote-agent-config.json).

Options:
  --no-open       Skip launching $APP_NAME after install
  --open          Launch $APP_NAME after install (default)
  --install-root  Override install directory (default: /Applications)
  -h, --help      Show this help
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --no-open) OPEN_AFTER_INSTALL=0; shift ;;
        --open) OPEN_AFTER_INSTALL=1; shift ;;
        --install-root)
            INSTALL_ROOT="$2"
            TARGET_APP="$INSTALL_ROOT/$APP_NAME.app"
            shift 2
            ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
    esac
done

if [[ ! -w "$INSTALL_ROOT" ]]; then
    echo "error: cannot write to $INSTALL_ROOT" >&2
    exit 1
fi

echo "==> Building $APP_NAME.app ($SWIFT_BUILD_CONFIG) [REMOTE_ONLY, no server binary]"
BUNDLE_SKIP_DMG=1 \
    SKIP_SERVER="$SKIP_SERVER" \
    REMOTE_ONLY="$REMOTE_ONLY" \
    APP_NAME="$APP_NAME" \
    BUNDLE_ID="$BUNDLE_ID" \
    SWIFT_EXECUTABLE="$SWIFT_EXECUTABLE" \
    SWIFT_BUILD_CONFIG="$SWIFT_BUILD_CONFIG" \
    MODE=remote \
    "$SCRIPT_DIR/bundle.sh"

if [[ ! -d "$SOURCE_APP" ]]; then
    echo "error: expected app bundle at $SOURCE_APP" >&2
    exit 1
fi

echo "==> Stopping running $APP_NAME (if any)"
osascript -e "tell application \"$APP_NAME\" to quit" 2>/dev/null || true
pkill -f "${TARGET_APP}/Contents/MacOS/" 2>/dev/null || true
sleep 0.5

echo "==> Installing to $TARGET_APP"
rm -rf "$TARGET_APP"
ditto "$SOURCE_APP" "$TARGET_APP"
xattr -dr com.apple.quarantine "$TARGET_APP" 2>/dev/null || true

echo "==> Installed: $TARGET_APP"

if [[ "$OPEN_AFTER_INSTALL" -eq 1 ]]; then
    echo "==> Opening $APP_NAME"
    open "$TARGET_APP"
fi
