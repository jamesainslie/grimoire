.PHONY: build test clean install

# CGO flags for SQLite extensions
export CGO_ENABLED := 1
export CGO_CFLAGS := -DSQLITE_ENABLE_FTS5

# Build tags for go-sqlite3 FTS5 support
BUILD_TAGS := -tags "fts5"

build:
	@mkdir -p bin
	go build $(BUILD_TAGS) -o bin/grimoire ./cmd/grimoire
	go build $(BUILD_TAGS) -o bin/grimoire-mcp ./cmd/grimoire-mcp
	@touch bin/.built-with-fts5

install: build
	@echo "Binary built to bin/grimoire-mcp"
	@echo "Restart Claude Code to use the updated MCP server"

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f bin/grimoire bin/grimoire-mcp bin/.built-with-fts5 coverage.out coverage.html

lint:
	golangci-lint run

.DEFAULT_GOAL := build
