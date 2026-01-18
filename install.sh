#!/bin/bash
set -e

# BrowserWing Installation Script
# Usage: curl -fsSL https://raw.githubusercontent.com/browserwing/browserwing/main/install.sh | bash

REPO="browserwing/browserwing"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.browserwing}"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored messages
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $OS in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="darwin"
            ;;
        msys*|mingw*|cygwin*)
            OS="windows"
            ;;
        *)
            print_error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac
    
    case $ARCH in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="armv7"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    print_info "Detected platform: ${OS}-${ARCH}"
}

# Get latest release version
get_latest_version() {
    print_info "Fetching latest release..."
    
    LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"
    VERSION=$(curl -sL "$LATEST_URL" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        print_error "Failed to fetch latest version"
        exit 1
    fi
    
    print_info "Latest version: $VERSION"
}

# Download binary
download_binary() {
    print_info "Downloading BrowserWing..."
    
    # Construct download URL and binary name
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="browserwing-${OS}-${ARCH}.exe"
    else
        BINARY_NAME="browserwing-${OS}-${ARCH}"
    fi
    
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}"
    
    print_info "Download URL: $DOWNLOAD_URL"
    
    # Create temp directory
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    # Download binary directly
    if ! curl -fsSL -o "$BINARY_NAME" "$DOWNLOAD_URL"; then
        print_error "Failed to download binary"
        rm -rf "$TMP_DIR"
        exit 1
    fi
    
    print_info "Binary downloaded successfully"
}

# Install binary
install_binary() {
    print_info "Installing BrowserWing..."
    
    # Create installation directory
    mkdir -p "$INSTALL_DIR"
    
    # Copy binary
    if [ "$OS" = "windows" ]; then
        BINARY_PATH="$INSTALL_DIR/browserwing.exe"
    else
        BINARY_PATH="$INSTALL_DIR/browserwing"
    fi
    
    cp "$BINARY_NAME" "$BINARY_PATH"
    chmod +x "$BINARY_PATH"
    
    # Try to create symlink in /usr/local/bin (requires sudo on some systems)
    if [ "$OS" != "windows" ]; then
        if [ -w "$BIN_DIR" ] || [ "$(id -u)" = "0" ]; then
            print_info "Creating symlink in $BIN_DIR..."
            ln -sf "$BINARY_PATH" "$BIN_DIR/browserwing"
        else
            print_warning "Cannot create symlink in $BIN_DIR (no write permission)"
            print_info "You can run: sudo ln -sf $BINARY_PATH $BIN_DIR/browserwing"
        fi
    fi
    
    # Cleanup
    cd - > /dev/null
    rm -rf "$TMP_DIR"
}

# Print success message
print_success() {
    echo ""
    print_info "${GREEN}BrowserWing installed successfully!${NC}"
    echo ""
    echo "Installation location: $BINARY_PATH"
    echo ""
    echo "Quick start:"
    echo "  1. Run: browserwing --port 8080"
    echo "  2. Open: http://localhost:8080"
    echo ""
    
    if [ "$OS" != "windows" ] && [ ! -L "$BIN_DIR/browserwing" ]; then
        print_warning "Binary not in PATH. Add to your shell profile:"
        echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    fi
    
    echo ""
    echo "Documentation: https://github.com/${REPO}"
    echo "Report issues: https://github.com/${REPO}/issues"
}

# Main installation flow
main() {
    echo ""
    echo "╔════════════════════════════════════════╗"
    echo "║   BrowserWing Installation Script     ║"
    echo "╚════════════════════════════════════════╝"
    echo ""
    
    # Check dependencies
    if ! command -v curl &> /dev/null; then
        print_error "curl is required but not installed"
        exit 1
    fi
    
    detect_platform
    get_latest_version
    download_binary
    install_binary
    print_success
}

# Run main function
main
