#!/bin/bash
# Grimoire Plugin Setup Script
# Runs on SessionStart - builds binaries if not present or outdated

set -e

# Read stdin (required by hook protocol)
input=$(cat)

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(dirname $(dirname $(dirname "$0")))}"
BIN_DIR="$PLUGIN_ROOT/bin"
GRIMOIRE_HOME="$HOME/.grimoire"
BUILD_MARKER="$BIN_DIR/.built-with-fts5"

# CGO flags for SQLite FTS5 support (required for full-text search)
export CGO_ENABLED=1
export CGO_CFLAGS="-DSQLITE_ENABLE_FTS5"
BUILD_TAGS="-tags fts5"

# Check if binaries already exist AND were built with FTS5 support
if [ -f "$BIN_DIR/grimoire-mcp" ] && [ -f "$BIN_DIR/grimoire" ] && [ -f "$BUILD_MARKER" ]; then
    # Already built with FTS5, nothing to do
    exit 0
fi

# Check for Go
if ! command -v go &> /dev/null; then
    echo '{"systemMessage": "WARNING: Grimoire plugin requires Go to build. Please install Go and run: cd '"$PLUGIN_ROOT"' && make build"}'
    exit 0
fi

# Create directories
mkdir -p "$BIN_DIR"
mkdir -p "$GRIMOIRE_HOME"
mkdir -p "$GRIMOIRE_HOME/cache/git"

# Build binaries with FTS5 support
cd "$PLUGIN_ROOT"

if go build $BUILD_TAGS -o "$BIN_DIR/grimoire" ./cmd/grimoire 2>/dev/null; then
    if go build $BUILD_TAGS -o "$BIN_DIR/grimoire-mcp" ./cmd/grimoire-mcp 2>/dev/null; then
        # Mark that we built with FTS5 support
        touch "$BUILD_MARKER"
        echo '{"systemMessage": "Grimoire plugin built successfully with FTS5 support. MCP server ready."}'
        exit 0
    fi
fi

echo '{"systemMessage": "Grimoire plugin build failed. Run manually: cd '"$PLUGIN_ROOT"' && make build"}'
exit 0
