# Grimoire - AI Knowledge Base for Programming Languages

A vector database of programming best practices, queryable via MCP server and CLI. Language-agnostic design, starting with Go.

## Overview

**Problem:** AI agents need language best practices context, but loading full style guides bloats context windows.

**Solution:** A local vector database containing curated documentation per language. Agents query semantically, receive only relevant chunks.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     grimoire                            │
├─────────────────────────────────────────────────────────┤
│  Interfaces                                             │
│  ├── cmd/grimoire/         CLI tool                     │
│  └── cmd/grimoire-mcp/     MCP server                   │
├─────────────────────────────────────────────────────────┤
│  Core Library                                           │
│  ├── internal/store/       SQLite + sqlite-vec storage  │
│  ├── internal/embed/       Ollama embedding client      │
│  ├── internal/ingest/      Document ingestion pipeline  │
│  └── internal/query/       Search and retrieval logic   │
├─────────────────────────────────────────────────────────┤
│  Ingestion Sources                                      │
│  ├── internal/source/git/  Git repo cloner/parser       │
│  └── internal/source/web/  Web scraper for blogs        │
├─────────────────────────────────────────────────────────┤
│  Language Packs (sources manifests)                     │
│  ├── languages/go/sources.yaml                          │
│  ├── languages/rust/sources.yaml    (future)            │
│  ├── languages/python/sources.yaml  (future)            │
│  └── languages/typescript/sources.yaml (future)         │
└─────────────────────────────────────────────────────────┘
```

**Key decisions:**
- **Vector DB:** SQLite + sqlite-vec (single file, no dependencies)
- **Embeddings:** Ollama local (snowflake-arctic-embed:l, 1024 dimensions)
- **Interfaces:** MCP server + CLI tool
- **Ingestion:** Hybrid git repos + web scraping
- **Chunking:** Hierarchical (summary → section → paragraph)
- **RAG:** Retrieval-only (agent does synthesis)
- **Multi-language:** Separate source manifests per language, single DB with language tags

## Data Model

```sql
-- Languages supported
CREATE TABLE languages (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,  -- 'go', 'rust', 'python'
    display_name TEXT NOT NULL  -- 'Go', 'Rust', 'Python'
);

-- Sources metadata
CREATE TABLE sources (
    id INTEGER PRIMARY KEY,
    language_id INTEGER REFERENCES languages(id),
    name TEXT NOT NULL,
    type TEXT NOT NULL,  -- 'git' | 'web'
    url TEXT NOT NULL,
    last_fetched DATETIME,
    etag TEXT
);

-- Documents (a single page/file)
CREATE TABLE documents (
    id INTEGER PRIMARY KEY,
    source_id INTEGER REFERENCES sources(id),
    path TEXT NOT NULL,
    title TEXT,
    content_hash TEXT,
    fetched_at DATETIME
);

-- Hierarchical chunks
CREATE TABLE chunks (
    id INTEGER PRIMARY KEY,
    document_id INTEGER REFERENCES documents(id),
    parent_chunk_id INTEGER REFERENCES chunks(id),
    level TEXT NOT NULL,  -- 'summary' | 'section' | 'paragraph'
    title TEXT,
    content TEXT NOT NULL,
    token_count INTEGER,
    embedding BLOB
);

-- Vector search index
CREATE VIRTUAL TABLE chunks_vec USING vec0(
    chunk_id INTEGER PRIMARY KEY,
    embedding FLOAT[1024]
);

-- Full-text search for hybrid retrieval
CREATE VIRTUAL TABLE chunks_fts USING fts5(content, title);
```

## CLI Interface

```bash
# Language management
grimoire languages list              # Show available languages
grimoire languages install go        # Install Go sources manifest
grimoire languages install rust      # (future)

# Ingestion
grimoire ingest                      # Fetch all installed languages
grimoire ingest --lang=go            # Specific language only
grimoire ingest --source=uber-guide  # Specific source
grimoire ingest --refresh            # Force re-fetch

# Query
grimoire query "error handling best practices"           # Search all
grimoire query "error handling" --lang=go                # Go only
grimoire query "context cancellation" --limit=5

# Management
grimoire sources list                # All sources
grimoire sources list --lang=go      # Go sources only
grimoire sources add go git <repo>   # Add custom source
grimoire stats
grimoire stats --lang=go
```

## MCP Server Interface

**Tools:**

```json
{
  "tools": [
    {
      "name": "query_knowledge",
      "description": "Search programming best practices and documentation",
      "parameters": {
        "query": "string",
        "language": "string (optional - 'go', 'rust', etc.)",
        "limit": "int (default 5)"
      }
    },
    {
      "name": "get_document",
      "description": "Retrieve full document by path/title",
      "parameters": {
        "path": "string",
        "language": "string (optional)"
      }
    },
    {
      "name": "list_topics",
      "description": "List available topic areas",
      "parameters": {
        "language": "string (optional)"
      }
    },
    {
      "name": "list_languages",
      "description": "List installed languages"
    }
  ]
}
```

**Response format:**

```json
{
  "results": [
    {
      "content": "Prefer wrapping errors with fmt.Errorf and %w...",
      "language": "go",
      "source": "Uber Style Guide",
      "section": "Error Handling > Wrapping Errors",
      "url": "https://github.com/uber-go/guide/blob/master/style.md",
      "relevance": 0.94
    }
  ],
  "query": "error handling best practices",
  "language_filter": "go",
  "total_chunks_searched": 12847
}
```

## Language Pack: Go

**File:** `languages/go/sources.yaml`

### Tier 1: Official Sources

| Source | Location |
|--------|----------|
| Go Wiki | `github.com/golang/wiki` |
| Effective Go | `go.dev/doc/effective_go` |
| Go Blog | `github.com/golang/blog` |
| Go Security | `go.dev/doc/security/best-practices` |
| Go Talks | `go.dev/talks` |

### Tier 2: Industry Style Guides

| Source | Repository |
|--------|------------|
| Uber Style Guide | `github.com/uber-go/guide` |
| Google Style Guide | `google.github.io/styleguide/go` |
| CockroachDB Guidelines | Confluence (scrape) |
| Gruntwork Style Guide | `docs.gruntwork.io` |

### Tier 3: Books (Free/Open)

| Source | Location |
|--------|----------|
| Go 101 | `go101.org` |
| Go by Example | `gobyexample.com` |
| Learn Go with Tests | `quii.gitbook.io/learn-go-with-tests` |
| Practical Go Lessons | `practical-go-lessons.com` |
| Essential Go | `programming-books.io/essential/go` |
| Build Web App with Go | `github.com/astaxie/build-web-application-with-golang` |
| OWASP Go Secure Coding | `github.com/OWASP/Go-SCP` |

### Tier 4: Community Blogs

| Source | URL |
|--------|-----|
| Dave Cheney | `dave.cheney.net` |
| Three Dots Labs | `threedots.tech` |
| Ardan Labs | `ardanlabs.com/blog` |
| Bitfield Consulting | `bitfieldconsulting.com` |
| Alex Edwards | `alexedwards.net/blog` |
| Eli Bendersky | `eli.thegreenplace.net` |
| GolangBot | `golangbot.com` |

### Tier 5: Curated Lists

| Source | Repository |
|--------|------------|
| awesome-go | `github.com/avelino/awesome-go` |
| awesome-go-education | `mehdihadeli.github.io/awesome-go-education` |
| go-best-practices | `github.com/smallnest/go-best-practices` |
| awesome-grpc | `github.com/grpc-ecosystem/awesome-grpc` |

### Tier 6: Conference Talks

| Source | Repository |
|--------|------------|
| GopherCon 2022 | `github.com/gophercon/2022-talks` |
| GopherCon 2021 | `github.com/gophercon/2021-talks` |
| GopherCon 2020 | `github.com/gophercon/2020-talks` |

## Data Location

```
~/.grimoire/
├── knowledge.db           # SQLite + vectors (all languages)
├── cache/
│   └── git/               # Cloned repos
├── languages/
│   ├── go/
│   │   └── sources.yaml   # Installed Go manifest
│   └── rust/              # (future)
│       └── sources.yaml
└── logs/
    └── ingest.log
```

## Dependencies

```go
require (
    github.com/mattn/go-sqlite3
    github.com/asg017/sqlite-vec-go
    github.com/spf13/cobra
    github.com/yuin/goldmark
    github.com/PuerkitoBio/goquery
    github.com/go-git/go-git/v5
)
```

## Implementation Phases

### Phase 1: Core Foundation
- [ ] Initialize Go module (`grimoire`)
- [ ] SQLite + sqlite-vec storage layer with language support
- [ ] Ollama embedding client
- [ ] Cobra CLI skeleton

### Phase 2: Ingestion
- [ ] Git fetcher (clone/pull)
- [ ] Markdown parser with heading extraction
- [ ] Hierarchical chunker
- [ ] Language pack loading

### Phase 3: Query
- [ ] Vector similarity search with language filter
- [ ] Hybrid search (vector + FTS5)
- [ ] Hierarchical retrieval
- [ ] CLI query command

### Phase 4: Web Scraping
- [ ] HTML fetcher with rate limiting
- [ ] HTML-to-markdown conversion
- [ ] Robots.txt, ETag caching

### Phase 5: MCP Server
- [ ] MCP protocol implementation
- [ ] Tool definitions with language parameter
- [ ] Claude Code integration

### Phase 6: Go Language Pack
- [ ] Complete sources.yaml for Go
- [ ] Test full ingestion pipeline
- [ ] Validate retrieval quality

### Phase 7: Polish
- [ ] Progress reporting
- [ ] Incremental updates
- [ ] Language/source management commands
- [ ] Stats and diagnostics

### Future: Additional Languages
- [ ] Rust language pack
- [ ] Python language pack
- [ ] TypeScript language pack

## File Structure

```
grimoire/
├── cmd/
│   ├── grimoire/main.go
│   └── grimoire-mcp/main.go
├── internal/
│   ├── store/
│   ├── embed/
│   ├── ingest/
│   ├── source/
│   │   ├── git/
│   │   └── web/
│   ├── parse/
│   ├── chunk/
│   └── query/
├── languages/
│   └── go/
│       └── sources.yaml
├── go.mod
└── docs/
    └── plans/
```
