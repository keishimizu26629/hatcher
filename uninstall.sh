#!/bin/bash

# Hatcher Uninstallation Script
# Usage: curl -fsSL https://keishimizu26629.github.io/hatcher/uninstall.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="hatcher"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.config/hatcher"

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

confirm_uninstall() {
    echo "🗑️  Hatcher Uninstallation"
    echo "========================="
    echo
    log_warning "This will remove:"
    echo "  - $INSTALL_DIR/$BINARY_NAME"
    echo "  - $INSTALL_DIR/hch"
    echo "  - $CONFIG_DIR (configuration files)"
    echo

    read -p "Are you sure you want to uninstall Hatcher? (y/N): " -n 1 -r
    echo

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Uninstallation cancelled"
        exit 0
    fi
}

remove_binaries() {
    log_info "Removing Hatcher binaries..."

    local hatcher_path="$INSTALL_DIR/$BINARY_NAME"
    local hch_path="$INSTALL_DIR/hch"

    # Remove hatcher binary
    if [ -f "$hatcher_path" ]; then
        if [ -w "$INSTALL_DIR" ]; then
            rm -f "$hatcher_path"
        else
            sudo rm -f "$hatcher_path"
        fi
        log_success "Removed $hatcher_path"
    else
        log_warning "$hatcher_path not found"
    fi

    # Remove hch alias
    if [ -f "$hch_path" ] || [ -L "$hch_path" ]; then
        if [ -w "$INSTALL_DIR" ]; then
            rm -f "$hch_path"
        else
            sudo rm -f "$hch_path"
        fi
        log_success "Removed $hch_path"
    else
        log_warning "$hch_path not found"
    fi
}

remove_config() {
    log_info "Removing configuration files..."

    if [ -d "$CONFIG_DIR" ]; then
        read -p "Remove configuration directory $CONFIG_DIR? (y/N): " -n 1 -r
        echo

        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "$CONFIG_DIR"
            log_success "Removed $CONFIG_DIR"
        else
            log_info "Configuration directory preserved"
        fi
    else
        log_info "No configuration directory found"
    fi
}

verify_removal() {
    log_info "Verifying removal..."

    if command -v hatcher >/dev/null 2>&1; then
        log_warning "hatcher command still found in PATH"
        log_info "You may need to restart your shell or check other installation locations"
    else
        log_success "hatcher command removed from PATH"
    fi

    if command -v hch >/dev/null 2>&1; then
        log_warning "hch command still found in PATH"
    else
        log_success "hch command removed from PATH"
    fi
}

check_homebrew() {
    log_info "Checking for Homebrew installation..."

    if command -v brew >/dev/null 2>&1; then
        if brew list | grep -q "^hatcher$"; then
            log_warning "Hatcher is also installed via Homebrew"
            echo
            read -p "Remove Homebrew installation as well? (y/N): " -n 1 -r
            echo

            if [[ $REPLY =~ ^[Yy]$ ]]; then
                brew uninstall hatcher
                log_success "Removed Homebrew installation"

                # Check if tap should be removed
                if brew tap | grep -q "keishimizu26629/tap"; then
                    read -p "Remove keishimizu26629/tap as well? (y/N): " -n 1 -r
                    echo

                    if [[ $REPLY =~ ^[Yy]$ ]]; then
                        brew untap keishimizu26629/tap
                        log_success "Removed Homebrew tap"
                    fi
                fi
            fi
        fi
    fi
}

main() {
    confirm_uninstall
    check_homebrew
    remove_binaries
    remove_config
    verify_removal

    echo
    log_success "🎉 Hatcher uninstallation completed!"
    echo
    log_info "Thank you for using Hatcher!"
    echo
}

# Run main function
main "$@"
