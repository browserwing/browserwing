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
BLUE='\033[0;34m'
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

print_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
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
    
    # Try GitHub API first
    LATEST_URL="https://api.github.com/repos/${REPO}/releases/latest"
    VERSION=$(curl -sL --connect-timeout 5 "$LATEST_URL" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || echo "")
    
    # If GitHub fails, try Gitee API
    if [ -z "$VERSION" ]; then
        print_warning "GitHub API failed, trying Gitee..."
        LATEST_URL="https://gitee.com/api/v5/repos/browserwing/browserwing/releases/latest"
        VERSION=$(curl -sL --connect-timeout 5 "$LATEST_URL" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || echo "")
    fi
    
    # Fallback to default version
    if [ -z "$VERSION" ]; then
        print_warning "Failed to fetch latest version, using default: v0.0.1"
        VERSION="v0.0.1"
    else
        print_info "Latest version: $VERSION"
    fi
}

# Test download speed and select fastest mirror
select_mirror() {
    print_info "Testing download mirrors..."
    
    # Test GitHub
    GITHUB_TIME=$(curl -o /dev/null -s -w '%{time_total}' --connect-timeout 5 \
        "https://github.com/${REPO}/releases/download/${VERSION}/browserwing-${OS}-${ARCH}.tar.gz" 2>/dev/null || echo "999")
    print_debug "GitHub response time: ${GITHUB_TIME}s"
    
    # Test Gitee
    GITEE_TIME=$(curl -o /dev/null -s -w '%{time_total}' --connect-timeout 5 \
        "https://gitee.com/browserwing/browserwing/releases/download/${VERSION}/browserwing-${OS}-${ARCH}.tar.gz" 2>/dev/null || echo "999")
    print_debug "Gitee response time: ${GITEE_TIME}s"
    
    # Select fastest mirror
    if (( $(echo "$GITHUB_TIME < $GITEE_TIME" | bc -l 2>/dev/null || echo "0") )); then
        MIRROR="github"
        MIRROR_URL="https://github.com/${REPO}/releases/download"
        print_info "Using GitHub mirror (faster)"
    else
        MIRROR="gitee"
        MIRROR_URL="https://gitee.com/browserwing/browserwing/releases/download"
        print_info "Using Gitee mirror (faster)"
    fi
}

# Download and extract archive
download_binary() {
    print_info "Downloading BrowserWing..."
    
    # Determine archive name and format
    if [ "$OS" = "windows" ]; then
        ARCHIVE_NAME="browserwing-${OS}-${ARCH}.zip"
        BINARY_NAME="browserwing-${OS}-${ARCH}.exe"
    else
        ARCHIVE_NAME="browserwing-${OS}-${ARCH}.tar.gz"
        BINARY_NAME="browserwing-${OS}-${ARCH}"
    fi
    
    DOWNLOAD_URL="${MIRROR_URL}/${VERSION}/${ARCHIVE_NAME}"
    print_info "Download URL: $DOWNLOAD_URL"
    
    # Create temp directory
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    # Download archive with retry
    MAX_RETRIES=3
    RETRY_COUNT=0
    
    while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
        if curl -fsSL -o "$ARCHIVE_NAME" "$DOWNLOAD_URL"; then
            print_info "Download completed"
            break
        else
            RETRY_COUNT=$((RETRY_COUNT + 1))
            if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
                print_warning "Download failed, retrying ($RETRY_COUNT/$MAX_RETRIES)..."
                
                # Switch mirror on failure
                if [ "$MIRROR" = "github" ]; then
                    MIRROR="gitee"
                    MIRROR_URL="https://gitee.com/browserwing/browserwing/releases/download"
                    print_info "Switching to Gitee mirror..."
                else
                    MIRROR="github"
                    MIRROR_URL="https://github.com/${REPO}/releases/download"
                    print_info "Switching to GitHub mirror..."
                fi
                
                DOWNLOAD_URL="${MIRROR_URL}/${VERSION}/${ARCHIVE_NAME}"
                sleep 2
            else
                print_error "Failed to download after $MAX_RETRIES attempts"
                rm -rf "$TMP_DIR"
                exit 1
            fi
        fi
    done
    
    # Extract archive
    print_info "Extracting archive..."
    if [ "$OS" = "windows" ]; then
        unzip -q "$ARCHIVE_NAME"
    else
        tar -xzf "$ARCHIVE_NAME"
    fi
    
    # Verify binary exists
    if [ ! -f "$BINARY_NAME" ]; then
        print_error "Binary not found after extraction: $BINARY_NAME"
        rm -rf "$TMP_DIR"
        exit 1
    fi
    
    print_info "Extraction completed"
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
    echo "中文文档: https://gitee.com/browserwing/browserwing"
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
    
    if ! command -v tar &> /dev/null && [ "$OS" != "windows" ]; then
        print_error "tar is required but not installed"
        exit 1
    fi
    
    if ! command -v unzip &> /dev/null && [ "$OS" = "windows" ]; then
        print_error "unzip is required but not installed"
        exit 1
    fi
    
    detect_platform
    get_latest_version
    select_mirror
    download_binary
    install_binary
    print_success
}

# Run main function
main
