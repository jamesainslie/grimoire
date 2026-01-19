# goai - Go AI Knowledge Base

A vector database of modern Go best practices, queryable via MCP server and CLI.

## Overview

**Problem:** AI agents need Go best practices context, but loading full style guides bloats context windows.

**Solution:** A local vector database containing ~500+ Go documentation sources. Agents query semantically, receive only relevant chunks.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      goai                               │
├─────────────────────────────────────────────────────────┤
│  Interfaces                                             │
│  ├── cmd/goai/         CLI tool                         │
│  └── cmd/goai-mcp/     MCP server                       │
├─────────────────────────────────────────────────────────┤
│  Core Library                                           │
│  ├── internal/store/   SQLite + sqlite-vec storage      │
│  ├── internal/embed/   Ollama embedding client          │
│  ├── internal/ingest/  Document ingestion pipeline      │
│  └── internal/query/   Search and retrieval logic       │
├─────────────────────────────────────────────────────────┤
│  Ingestion Sources                                      │
│  ├── internal/source/git/    Git repo cloner/parser     │
│  └── internal/source/web/    Web scraper for blogs      │
└─────────────────────────────────────────────────────────┘
```

**Key decisions:**
- **Vector DB:** SQLite + sqlite-vec (single file, no dependencies)
- **Embeddings:** Ollama local (nomic-embed-text, 384 dimensions)
- **Interfaces:** MCP server + CLI tool
- **Ingestion:** Hybrid git repos + web scraping
- **Chunking:** Hierarchical (summary → section → paragraph)
- **RAG:** Retrieval-only (agent does synthesis)

## Documentation Sources

### Tier 1: Official Sources (Git-based)

| Source | Location | Content |
|--------|----------|---------|
| Go Wiki | `github.com/golang/wiki` | CodeReviewComments, CommonMistakes, TableDrivenTests |
| Effective Go | `go.dev/doc/effective_go` | Official style and idioms |
| Go Blog | `github.com/golang/blog` | Deep dives, release notes, generics, slog |
| Go Security | `go.dev/doc/security/best-practices` | govulncheck, fuzzing, race detection |
| Go Talks | `go.dev/talks` | Concurrency patterns, error handling |

### Tier 2: Industry Style Guides (Git-based)

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
| GoBooks | `github.com/dariubs/GoBooks` |

### Tier 6: Conference Talks

| Source | Repository |
|--------|------------|
| GopherCon 2022 | `github.com/gophercon/2022-talks` |
| GopherCon 2021 | `github.com/gophercon/2021-talks` |
| GopherCon 2020 | `github.com/gophercon/2020-talks` |
| GopherCon 2019 | `github.com/gophercon/2019-talks` |

## Data Model

```sql
-- Sources metadata
CREATE TABLE sources (
    id INTEGER PRIMARY KEY,
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
    embedding FLOAT[384]
);

-- Full-text search for hybrid retrieval
CREATE VIRTUAL TABLE chunks_fts USING fts5(content, title);
```

**Chunking hierarchy:**

```
Document: "Effective Go"
├── Summary (1 embedding)
├── Section: "Formatting"
│   └── Paragraph chunks (~300 tokens each)
├── Section: "Commentary"
│   └── Paragraph chunks
└── Section: "Names"
    └── Paragraph chunks
```

## Ingestion Pipeline

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Fetch     │───▶│   Parse     │───▶│   Chunk     │───▶│   Embed     │───▶│   Store     │
│  git/web    │    │  md/html    │    │ hierarchical│    │   ollama    │    │  sqlite-vec │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

**Source manifest (`sources.yaml`):**

```yaml
git_sources:
  - name: go-wiki
    repo: https://github.com/golang/wiki.git
    patterns: ["*.md"]

  - name: uber-guide
    repo: https://github.com/uber-go/guide.git
    patterns: ["*.md"]

  - name: learn-go-with-tests
    repo: https://github.com/quii/learn-go-with-tests.git
    patterns: ["*.md"]

web_sources:
  - name: dave-cheney
    urls:
      - https://dave.cheney.net/practical-go/presentations/qcon-china.html
    selector: "article"
```

**Incremental updates:**
- Content hash comparison - skip unchanged documents
- Only re-embed modified chunks
- Git pull for repos, ETag/Last-Modified for web

## CLI Interface

```bash
# Ingestion
goai ingest              # Fetch all sources, update DB
goai ingest --source=uber-guide
goai ingest --refresh    # Force re-fetch

# Query
goai query "error handling best practices"
goai query "context cancellation" --limit=5

# Management
goai sources list
goai sources add git <repo-url>
goai sources add web <url> --selector="article"
goai stats
```

## MCP Server Interface

**Tools:**

```json
{
  "tools": [
    {
      "name": "query_go_knowledge",
      "description": "Search Go best practices and documentation",
      "parameters": {
        "query": "string",
        "limit": "int (default 5)"
      }
    },
    {
      "name": "get_document",
      "description": "Retrieve full document by path/title",
      "parameters": {
        "path": "string"
      }
    },
    {
      "name": "list_topics",
      "description": "List available topic areas"
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
      "source": "Uber Style Guide",
      "section": "Error Handling > Wrapping Errors",
      "url": "https://github.com/uber-go/guide/blob/master/style.md",
      "relevance": 0.94
    }
  ],
  "query": "error handling best practices",
  "total_chunks_searched": 12847
}
```

## RAG Strategy

**Retrieval-only:** MCP server returns chunks, calling agent synthesizes.

```go
func (q *Querier) Query(question string, opts QueryOpts) (*QueryResult, error) {
    // 1. Embed question via Ollama
    embedding := q.embedder.Embed(question)

    // 2. Vector search - top 20 candidates
    candidates := q.store.SimilaritySearch(embedding, 20)

    // 3. Hybrid rerank (vector + keyword)
    //    final_score = (0.7 * semantic) + (0.3 * keyword)
    ranked := q.rerank(candidates, question)

    // 4. Expand with parent context
    results := q.expandContext(ranked[:opts.Limit])

    return results
}
```

## Data Location

```
~/.goai/
├── knowledge.db      # SQLite + vectors
├── sources.yaml      # User's manifest
├── cache/
│   └── git/          # Cloned repos
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
- [ ] Initialize Go module
- [ ] SQLite + sqlite-vec storage layer
- [ ] Ollama embedding client
- [ ] Cobra CLI skeleton

### Phase 2: Ingestion
- [ ] Git fetcher (clone/pull)
- [ ] Markdown parser with heading extraction
- [ ] Hierarchical chunker
- [ ] Source manifest loading

### Phase 3: Query
- [ ] Vector similarity search
- [ ] Hybrid search (vector + FTS5)
- [ ] Hierarchical retrieval
- [ ] CLI query command

### Phase 4: Web Scraping
- [ ] HTML fetcher with rate limiting
- [ ] HTML-to-markdown conversion
- [ ] Robots.txt, ETag caching

### Phase 5: MCP Server
- [ ] MCP protocol implementation
- [ ] Tool definitions
- [ ] Claude Code integration

### Phase 6: Polish
- [ ] Progress reporting
- [ ] Incremental updates
- [ ] Source management commands
- [ ] Stats and diagnostics

## File Structure

```
goai/
├── cmd/
│   ├── goai/main.go
│   └── goai-mcp/main.go
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
├── sources.yaml
├── go.mod
└── docs/
    └── plans/
```
