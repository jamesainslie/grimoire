# Grimoire

An AI-powered knowledge base for programming best practices. Grimoire provides semantic search over curated documentation for Go, Rust, Python, and more.

## Features

- **Vector Search**: Semantic similarity search using local embeddings (Ollama + snowflake-arctic-embed)
- **Hybrid Search**: Combines vector search with full-text search for better results
- **MCP Server**: Expose the knowledge base to AI assistants via Model Context Protocol
- **Language Packs**: Modular documentation sources organized by programming language
- **Local-First**: All data stored locally in SQLite with sqlite-vec for vectors

## Requirements

- Go 1.21+
- [Ollama](https://ollama.ai/) running locally
- GCC (for CGO/sqlite-vec)

## Installation

### Claude Plugin (Recommended)

Install as a Claude Code plugin for automatic MCP server integration:

```bash
# Install the embedding model first
ollama pull snowflake-arctic-embed:l

# Add the marketplace and install
/plugin marketplace add jamesainslie/grimoire
/plugin install grimoire@grimoire

# Or install from local clone
git clone https://github.com/jamesainslie/grimoire.git
cd grimoire
/plugin marketplace add .
/plugin install grimoire@grimoire
```

The plugin automatically:
- Builds the CLI and MCP server
- Configures the MCP server in Claude
- Creates the data directory at `~/.grimoire/`

### Manual Installation

```bash
# Clone the repository
git clone https://github.com/jamesainslie/grimoire.git
cd grimoire

# Install the embedding model
ollama pull snowflake-arctic-embed:l

# Build
make build

# Or use the install script
./scripts/install.sh
```

## Quick Start

### 1. Ingest Documentation

Ingest a language pack to populate the knowledge base:

```bash
# Ingest the Go language pack
grimoire ingest langpacks/go/sources.yaml
```

This will:
- Fetch documentation from configured git repositories
- Parse markdown files
- Chunk content hierarchically
- Generate embeddings via Ollama
- Store everything in the local database

### 2. Query the Knowledge Base

```bash
# Search for information
grimoire query "how to handle errors in Go"

# Filter by language
grimoire query --lang go "interface design patterns"

# Limit results
grimoire query --limit 10 "concurrency best practices"

# Use vector-only search (skip full-text)
grimoire query --vector-only "mutex vs channels"
```

### 3. View Statistics

```bash
grimoire stats
```

## MCP Server

Grimoire includes an MCP server for integration with AI assistants like Claude.

### Configuration

Add to your Claude configuration:

```json
{
  "mcpServers": {
    "grimoire": {
      "command": "/path/to/grimoire-mcp",
      "env": {
        "GRIMOIRE_DB": "~/.grimoire/grimoire.db",
        "OLLAMA_URL": "http://localhost:11434"
      }
    }
  }
}
```

### Available Tools

- **query**: Search the knowledge base for programming best practices
- **list_languages**: List installed programming languages
- **list_sources**: List documentation sources

## CLI Reference

### Global Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--db` | Database path | `~/.grimoire/grimoire.db` |
| `--ollama-url` | Ollama API URL | `http://localhost:11434` |

### Commands

#### `grimoire ingest <language-pack>`

Fetch and index documentation from a language pack.

```bash
grimoire ingest langpacks/go/sources.yaml
grimoire ingest --source go-wiki langpacks/go/sources.yaml  # Single source
```

#### `grimoire query <text>`

Search the knowledge base.

| Flag | Description | Default |
|------|-------------|---------|
| `--lang` | Filter by language | (all) |
| `--limit` | Max results | 5 |
| `--vector-only` | Skip full-text search | false |

#### `grimoire stats`

Show knowledge base statistics.

#### `grimoire languages list`

List installed languages.

#### `grimoire languages install <language>`

Install a language pack (placeholder).

#### `grimoire sources list`

List documentation sources.

| Flag | Description |
|------|-------------|
| `--lang` | Filter by language |

#### `grimoire sources add <language> <type> <url>`

Add a custom documentation source (placeholder).

## Language Packs

Language packs are YAML files that define documentation sources:

```yaml
language: go
display_name: Go

sources:
  - name: go-wiki
    type: git
    url: https://github.com/golang/wiki
    paths:
      - "*.md"
    tier: 1

  - name: uber-style-guide
    type: git
    url: https://github.com/uber-go/guide
    paths:
      - style.md
    tier: 2
```

### Source Tiers

| Tier | Description |
|------|-------------|
| 1 | Official documentation |
| 2 | Industry style guides |
| 3 | Books and tutorials |
| 4 | Blog posts and articles |
| 5 | Curated lists |

### Included Language Packs

- **Go** (`langpacks/go/sources.yaml`): Official wiki, Uber style guide, learn-go-with-tests, and more

## Architecture

```
grimoire/
├── .claude-plugin/    # Claude plugin configuration
│   ├── plugin.json    # Plugin manifest
│   └── marketplace.json
├── bin/               # Built binaries (after install)
├── cmd/
│   ├── grimoire/      # CLI application
│   └── grimoire-mcp/  # MCP server
├── internal/
│   ├── chunk/         # Document chunking
│   ├── embed/         # Ollama embeddings client
│   ├── ingest/        # Language pack loading
│   ├── parse/         # Markdown parsing
│   ├── source/git/    # Git repository fetcher
│   └── store/         # SQLite + sqlite-vec storage
├── langpacks/         # Language pack definitions
│   └── go/
├── scripts/
│   └── install.sh     # Installation script
└── skills/            # Claude skills
    └── grimoire-query/
```

## Development

```bash
# Run tests
make test

# Build binaries
make build

# Run linter
make lint
```

## License

MIT
