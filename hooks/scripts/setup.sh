#!/bin/bash
# Grimoire Plugin Setup Script
# Runs on SessionStart - builds binaries if not present

set -e

# Read stdin (required by hook protocol)
input=$(cat)

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(dirname $(dirname $(dirname "$0")))}"
BIN_DIR="$PLUGIN_ROOT/bin"
GRIMOIRE_HOME="$HOME/.grimoire"

# Check if binaries already exist
if [ -f "$BIN_DIR/grimoire-mcp" ] && [ -f "$BIN_DIR/grimoire" ]; then
    # Already built, nothing to do
    exit 0
fi

# Check for Go
if ! command -v go &> /dev/null; then
    echo '{"systemMessage": "WARNING: Grimoire plugin requires Go to build. Please install Go and run: cd '"$PLUGIN_ROOT"' && ./scripts/install.sh"}'
    exit 0
fi

# Create directories
mkdir -p "$BIN_DIR"
mkdir -p "$GRIMOIRE_HOME"
mkdir -p "$GRIMOIRE_HOME/cache/git"

# Build binaries
cd "$PLUGIN_ROOT"

if CGO_ENABLED=1 go build -o "$BIN_DIR/grimoire" ./cmd/grimoire 2>/dev/null; then
    if CGO_ENABLED=1 go build -o "$BIN_DIR/grimoire-mcp" ./cmd/grimoire-mcp 2>/dev/null; then
        echo '{"systemMessage": "Grimoire plugin built successfully. MCP server ready."}'
        exit 0
    fi
fi

echo '{"systemMessage": "Grimoire plugin build failed. Run manually: cd '"$PLUGIN_ROOT"' && ./scripts/install.sh"}'
exit 0
