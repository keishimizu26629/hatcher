#!/bin/bash

# Hatcher Installation Script
# Usage: curl -fsSL https://keishimizu26629.github.io/hatcher/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="keishimizu26629/hatcher"
BINARY_NAME="hatcher"
INSTALL_DIR="/usr/local/bin"
TEMP_DIR=$(mktemp -d)

# Functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

cleanup() {
    rm -rf "$TEMP_DIR"
}

trap cleanup EXIT

detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$os" in
        darwin)
            OS="darwin"
            ;;
        linux)
            OS="linux"
            ;;
        *)
            log_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac

    case "$arch" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac

    log_info "Detected platform: $OS/$ARCH"
}

get_latest_release() {
    log_info "Fetching latest release information..."

    # Get latest release info from GitHub API
    local release_info
    if command -v curl >/dev/null 2>&1; then
        release_info=$(curl -s "https://api.github.com/repos/$REPO/releases/latest")
    else
        log_error "curl is required but not installed"
        exit 1
    fi

    # Extract version and download URL
    VERSION=$(echo "$release_info" | grep '"tag_name"' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')

    if [ -z "$VERSION" ]; then
        log_error "Failed to get latest release version"
        exit 1
    fi

    log_info "Latest version: $VERSION"
}

download_binary() {
    log_info "Downloading Hatcher $VERSION for $OS/$ARCH..."

    # Construct download URL
    local download_url="https://github.com/$REPO/releases/download/$VERSION/hatcher-$OS-$ARCH"
    if [ "$OS" = "windows" ]; then
        download_url="$download_url.exe"
    fi

    # Download binary
    local binary_path="$TEMP_DIR/$BINARY_NAME"
    if ! curl -fsSL "$download_url" -o "$binary_path"; then
        # Fallback: try to build from source
        log_warning "Pre-built binary not available, building from source..."
        build_from_source
        return
    fi

    # Make executable
    chmod +x "$binary_path"

    # Verify binary works
    if ! "$binary_path" --version >/dev/null 2>&1; then
        log_error "Downloaded binary is not working correctly"
        exit 1
    fi

    log_success "Binary downloaded and verified"
}

build_from_source() {
    log_info "Building Hatcher from source..."

    # Check if Go is installed
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is required to build from source but not installed"
        log_info "Please install Go from https://golang.org/dl/"
        exit 1
    fi

    # Download source code
    local source_url="https://github.com/$REPO/archive/$VERSION.tar.gz"
    local source_archive="$TEMP_DIR/source.tar.gz"

    curl -fsSL "$source_url" -o "$source_archive"

    # Extract and build
    cd "$TEMP_DIR"
    tar -xzf "$source_archive"
    cd "hatcher-${VERSION#v}"

    # Build binary
    go build -ldflags "-s -w -X main.Version=$VERSION" -o "$TEMP_DIR/$BINARY_NAME" ./main.go

    log_success "Built from source successfully"
}

install_binary() {
    log_info "Installing Hatcher to $INSTALL_DIR..."

    local binary_path="$TEMP_DIR/$BINARY_NAME"
    local install_path="$INSTALL_DIR/$BINARY_NAME"
    local hch_path="$INSTALL_DIR/hch"

    # Check if install directory is writable
    if [ ! -w "$INSTALL_DIR" ]; then
        log_warning "Need sudo privileges to install to $INSTALL_DIR"
        sudo cp "$binary_path" "$install_path"
        sudo chmod +x "$install_path"

        # Create hch alias
        sudo ln -sf "$install_path" "$hch_path"
    else
        cp "$binary_path" "$install_path"
        chmod +x "$install_path"

        # Create hch alias
        ln -sf "$install_path" "$hch_path"
    fi

    log_success "Hatcher installed to $install_path"
    log_success "hch alias created at $hch_path"
}

verify_installation() {
    log_info "Verifying installation..."

    if command -v hatcher >/dev/null 2>&1; then
        local installed_version=$(hatcher --version | grep -o 'version [0-9.]*' | cut -d' ' -f2)
        log_success "hatcher command is available (version: $installed_version)"
    else
        log_error "hatcher command not found in PATH"
        exit 1
    fi

    if command -v hch >/dev/null 2>&1; then
        log_success "hch alias is available"
    else
        log_warning "hch alias not found in PATH"
    fi

    # Test basic functionality
    if hatcher doctor >/dev/null 2>&1; then
        log_success "Basic functionality test passed"
    else
        log_warning "Basic functionality test failed (this may be normal outside a Git repository)"
    fi
}

show_usage() {
    echo
    log_success "🎉 Hatcher installation completed!"
    echo
    echo "Usage:"
    echo "  hatcher create feature/new-feature    # Create new worktree"
    echo "  hch -v list                          # List worktrees (verbose)"
    echo "  hatcher doctor                       # System diagnostics"
    echo "  hatcher --help                       # Show all commands"
    echo
    echo "Documentation: https://github.com/$REPO"
    echo
}

# Main installation process
main() {
    echo "🥇 Hatcher Installation Script"
    echo "=============================="
    echo

    detect_platform
    get_latest_release
    download_binary
    install_binary
    verify_installation
    show_usage
}

# Run main function
main "$@"
