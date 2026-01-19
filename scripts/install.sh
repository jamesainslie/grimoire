#!/bin/bash
# Grimoire Claude Plugin Installation Script
# This script builds and installs the Grimoire CLI and MCP server

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGIN_DIR="$(dirname "$SCRIPT_DIR")"
BIN_DIR="$PLUGIN_DIR/bin"
GRIMOIRE_HOME="$HOME/.grimoire"

echo "=== Grimoire Plugin Installation ==="
echo "Plugin directory: $PLUGIN_DIR"
echo ""

# Check for Go
if ! command -v go &> /dev/null; then
    echo "ERROR: Go is required but not installed."
    echo "Please install Go from https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Found Go version: $GO_VERSION"

# Check for Ollama
echo ""
echo "Checking Ollama..."
if ! curl -s http://localhost:11434/api/tags &> /dev/null; then
    echo "WARNING: Ollama does not appear to be running."
    echo "Please install and start Ollama: https://ollama.ai/"
    echo "Then run: ollama pull snowflake-arctic-embed:l"
    echo ""
    echo "Continuing with installation anyway..."
else
    if curl -s http://localhost:11434/api/tags | grep -q "snowflake-arctic-embed"; then
        echo "Found snowflake-arctic-embed model"
    else
        echo "WARNING: snowflake-arctic-embed model not found."
        echo "Run: ollama pull snowflake-arctic-embed:l"
        echo ""
        echo "Continuing with installation anyway..."
    fi
fi

# Create directories
echo ""
echo "Creating directories..."
mkdir -p "$BIN_DIR"
mkdir -p "$GRIMOIRE_HOME"
mkdir -p "$GRIMOIRE_HOME/cache/git"

# Build binaries
echo ""
echo "Building Grimoire CLI..."
cd "$PLUGIN_DIR"
CGO_ENABLED=1 go build -o "$BIN_DIR/grimoire" ./cmd/grimoire
echo "Built: $BIN_DIR/grimoire"

echo ""
echo "Building Grimoire MCP Server..."
CGO_ENABLED=1 go build -o "$BIN_DIR/grimoire-mcp" ./cmd/grimoire-mcp
echo "Built: $BIN_DIR/grimoire-mcp"

# Create symlink in user's bin if desired
if [ -d "$HOME/.local/bin" ]; then
    echo ""
    echo "Creating symlink in ~/.local/bin..."
    ln -sf "$BIN_DIR/grimoire" "$HOME/.local/bin/grimoire"
    echo "Created: ~/.local/bin/grimoire"
fi

# Summary
echo ""
echo "=== Installation Complete ==="
echo ""
echo "Grimoire has been installed with the following components:"
echo "  - CLI:        $BIN_DIR/grimoire"
echo "  - MCP Server: $BIN_DIR/grimoire-mcp"
echo "  - Data:       $GRIMOIRE_HOME/"
echo ""
echo "The MCP server will be available as 'grimoire' in Claude."
echo ""
echo "Quick Start:"
echo "  1. Ingest documentation:"
echo "     $BIN_DIR/grimoire ingest langpacks/go/sources.yaml"
echo ""
echo "  2. Query from CLI:"
echo "     $BIN_DIR/grimoire query 'error handling best practices'"
echo ""
echo "  3. Query from Claude using the MCP tools:"
echo "     - grimoire.query"
echo "     - grimoire.list_languages"
echo "     - grimoire.list_sources"
echo ""
