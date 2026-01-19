.PHONY: build test clean

# CGO flags for SQLite extensions
export CGO_CFLAGS := -DSQLITE_ENABLE_FTS5

build:
	go build -o grimoire ./cmd/grimoire
	go build -o grimoire-mcp ./cmd/grimoire-mcp

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f grimoire grimoire-mcp coverage.out coverage.html

lint:
	golangci-lint run

.DEFAULT_GOAL := build
