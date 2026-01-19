---
name: grimoire-query
description: >-
  Use when you need programming best practices, patterns, or documentation.
  Queries the Grimoire knowledge base for curated guidance on Go, Rust, Python, and more.
  Invoke before implementing features, debugging, or reviewing code to get relevant context.
---

# Grimoire Knowledge Base Query

## Overview

Grimoire is a semantic search knowledge base containing curated programming best practices, style guides, and documentation. Use this skill when you need authoritative guidance on:

- Error handling patterns
- Interface design
- Concurrency patterns
- Testing strategies
- Code organization
- Language-specific idioms

## When to Use

**Always query Grimoire when:**
- Implementing new features in a supported language
- Reviewing code for best practices
- Debugging unfamiliar patterns
- Making architectural decisions
- Writing tests

**Supported languages:**
- Go (comprehensive: official wiki, Uber style guide, learn-go-with-tests, OWASP security)
- More languages coming soon

## How to Query

Use the `grimoire.query` MCP tool:

```
Tool: grimoire.query
Arguments:
  query: "your search query"
  language: "go"  (optional - filter by language)
  limit: 5        (optional - max results, default 5)
```

### Example Queries

**Error handling:**
```
grimoire.query("how to wrap errors with context in Go")
```

**Interface design:**
```
grimoire.query("when to use interfaces vs concrete types")
```

**Concurrency:**
```
grimoire.query("channel vs mutex for shared state")
```

**Testing:**
```
grimoire.query("table-driven tests best practices")
```

## Query Tips

1. **Be specific** - "error handling" is too broad; "wrapping errors with context" is better
2. **Include context** - "interface design for dependency injection" beats "interfaces"
3. **Use language filter** - If you know the language, filter for more relevant results
4. **Read multiple results** - Different sources offer complementary perspectives

## Available MCP Tools

### grimoire.query
Search the knowledge base for programming guidance.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| query | string | Yes | The search query |
| language | string | No | Filter by language (e.g., "go") |
| limit | int | No | Max results (default 5, max 20) |

### grimoire.list_languages
List all installed programming languages.

### grimoire.list_sources
List documentation sources, optionally filtered by language.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| language | string | No | Filter sources by language |

## Understanding Results

Results include:
- **Title** - The section or topic heading
- **Level** - Hierarchy level (summary, section, paragraph)
- **Content** - The actual guidance text
- **Relevance** - Higher scores indicate better matches

## Workflow Integration

### Before Implementation
```
1. Query Grimoire for relevant patterns
2. Review top 3-5 results
3. Apply guidance to your implementation
4. Query again if you hit edge cases
```

### During Code Review
```
1. Identify areas of concern
2. Query Grimoire for best practices
3. Compare code against guidance
4. Suggest improvements with citations
```

### When Debugging
```
1. Identify the problematic pattern
2. Query Grimoire for correct approach
3. Compare against current implementation
4. Refactor following guidance
```

## CLI Alternative

For quick lookups outside Claude, use the CLI:

```bash
# Basic query
grimoire query "error handling best practices"

# Filter by language
grimoire query --lang go "interface design"

# More results
grimoire query --limit 10 "testing patterns"

# Vector-only search (faster, less precise)
grimoire query --vector-only "concurrency"
```

## Adding Documentation

To ingest more documentation:

```bash
# Ingest a language pack
grimoire ingest langpacks/go/sources.yaml

# Check what's indexed
grimoire stats

# List sources
grimoire sources list --lang go
```

## Best Practices

1. **Query early, query often** - Don't wait until you're stuck
2. **Cross-reference sources** - Different guides offer different perspectives
3. **Apply with context** - Guidance is general; adapt to your specific situation
4. **Update regularly** - Re-ingest to get the latest documentation
