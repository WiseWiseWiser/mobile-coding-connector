#!/usr/bin/env bash
set -euo pipefail

REPO="WiseWiseWiser/mobile-coding-connector"
BINARY_NAME="ai-critic-server"

# Check if curl is available
if ! command -v curl &> /dev/null; then
    echo "Error: curl is required but not installed."
    echo ""
    echo "Install curl with one of the following commands:"
    echo "  Ubuntu/Debian:  sudo apt-get install -y curl"
    echo "  CentOS/RHEL:    sudo yum install -y curl"
    echo "  Fedora:         sudo dnf install -y curl"
    echo "  Alpine:         apk add curl"
    echo "  macOS:          brew install curl"
    exit 1
fi

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
    linux)   OS="linux" ;;
    darwin)  OS="darwin" ;;
    *)
        echo "Error: unsupported OS: $OS"
        echo "This installer only supports Linux."
        exit 1
        ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)   ARCH="amd64" ;;
    aarch64|arm64)   ARCH="arm64" ;;
    *)
        echo "Error: unsupported architecture: $ARCH"
        echo "This installer supports amd64 and arm64."
        exit 1
        ;;
esac

# Only linux binaries are provided in releases
if [ "$OS" != "linux" ]; then
    echo "Error: pre-built binaries are only available for Linux."
    echo "Detected OS: $OS ($ARCH)"
    echo "Please build from source: go run ./script/build"
    exit 1
fi

ASSET_NAME="${BINARY_NAME}-${OS}-${ARCH}"

echo "Detected: ${OS}/${ARCH}"
echo "Downloading: ${ASSET_NAME}"

# Get the latest release download URL from GitHub API
RELEASE_URL="https://api.github.com/repos/${REPO}/releases/latest"
echo "Fetching latest release from ${RELEASE_URL}..."

DOWNLOAD_URL=$(curl -sL "$RELEASE_URL" | grep -o "\"browser_download_url\": *\"[^\"]*${ASSET_NAME}\"" | head -1 | sed 's/.*"browser_download_url": *"\(.*\)"/\1/')

if [ -z "$DOWNLOAD_URL" ]; then
    echo "Error: could not find ${ASSET_NAME} in the latest release."
    echo "Available assets:"
    curl -sL "$RELEASE_URL" | grep '"browser_download_url"' | sed 's/.*"browser_download_url": *"\(.*\)"/  \1/' | head -20
    exit 1
fi

echo "Downloading from: ${DOWNLOAD_URL}"

# Download to current directory
OUTPUT="./${BINARY_NAME}"
curl -fSL -o "$OUTPUT" "$DOWNLOAD_URL"
chmod +x "$OUTPUT"

echo ""
echo "Downloaded: ${OUTPUT}"
echo ""
echo "To start the server, run:"
echo "  ./${BINARY_NAME} keep-alive"
