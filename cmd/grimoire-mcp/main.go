package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jamesainslie/grimoire/internal/embed"
	"github.com/jamesainslie/grimoire/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Global configuration
var (
	dbPath    string
	ollamaURL string
)

// Tool argument types
type queryArgs struct {
	Query    string `json:"query" jsonschema_description:"The search query"`
	Language string `json:"language,omitempty" jsonschema_description:"Filter results by programming language (e.g. go, rust)"`
	Limit    int    `json:"limit,omitempty" jsonschema_description:"Maximum number of results (default 5, max 20)"`
}

type listLanguagesArgs struct{}

type listSourcesArgs struct {
	Language string `json:"language,omitempty" jsonschema_description:"Filter sources by programming language"`
}

func main() {
	// Get configuration from environment or use defaults
	dbPath = os.Getenv("GRIMOIRE_DB")
	if dbPath == "" {
		dbPath = getDefaultDBPath()
	}

	ollamaURL = os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func getDefaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "grimoire.db"
	}
	return filepath.Join(home, ".grimoire", "grimoire.db")
}

func run() error {
	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "grimoire",
		Version: "0.1.0",
	}, nil)

	// Add query tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "query",
		Description: "Search the grimoire knowledge base for programming best practices, patterns, and documentation. Returns relevant chunks from curated sources.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args queryArgs) (*mcp.CallToolResult, any, error) {
		return handleQuery(ctx, args)
	})

	// Add list_languages tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_languages",
		Description: "List all programming languages installed in the grimoire knowledge base.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listLanguagesArgs) (*mcp.CallToolResult, any, error) {
		return handleListLanguages(ctx)
	})

	// Add list_sources tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_sources",
		Description: "List documentation sources in the grimoire knowledge base, optionally filtered by language.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listSourcesArgs) (*mcp.CallToolResult, any, error) {
		return handleListSources(ctx, args)
	})

	// Run server over stdio
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

func handleQuery(ctx context.Context, args queryArgs) (*mcp.CallToolResult, any, error) {
	// Validate and set defaults
	if args.Query == "" {
		return nil, nil, fmt.Errorf("query is required")
	}
	if args.Limit <= 0 {
		args.Limit = 5
	}
	if args.Limit > 20 {
		args.Limit = 20
	}

	// Open database
	db, err := store.New(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Get language ID if specified
	var languageID int64
	if args.Language != "" {
		lang, err := db.GetLanguage(ctx, args.Language)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Language %q not found in knowledge base.", args.Language)},
				},
			}, nil, nil
		}
		languageID = lang.ID
	}

	// Get query embedding
	client := embed.New(ollamaURL, "snowflake-arctic-embed:l")
	queryVec, err := client.Embed(ctx, args.Query)
	if err != nil {
		return nil, nil, fmt.Errorf("get embedding: %w", err)
	}

	// Perform hybrid search
	results, err := db.SearchChunksHybrid(ctx, queryVec, args.Query, languageID, args.Limit)
	if err != nil {
		return nil, nil, fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "No results found for the query."},
			},
		}, nil, nil
	}

	// Format results
	// Distance is 0-1 where 0=perfect match, so relevance = 1 - distance
	var text string
	for i, r := range results {
		relevance := 1.0 - r.Distance
		text += fmt.Sprintf("## Result %d (relevance: %.0f%%)\n", i+1, relevance*100)
		text += fmt.Sprintf("**Title:** %s\n", r.Chunk.Title)
		text += fmt.Sprintf("**Level:** %s\n\n", r.Chunk.Level)
		text += r.Chunk.Content + "\n\n---\n\n"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, nil, nil
}

func handleListLanguages(ctx context.Context) (*mcp.CallToolResult, any, error) {
	db, err := store.New(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	languages, err := db.ListLanguages(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list languages: %w", err)
	}

	if len(languages) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "No languages installed. Use `grimoire languages install <language>` to add languages."},
			},
		}, nil, nil
	}

	var text string
	text += "Installed languages:\n\n"
	for _, lang := range languages {
		text += fmt.Sprintf("- **%s** (%s)\n", lang.DisplayName, lang.Name)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, nil, nil
}

func handleListSources(ctx context.Context, args listSourcesArgs) (*mcp.CallToolResult, any, error) {
	db, err := store.New(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	var languageID int64
	if args.Language != "" {
		lang, err := db.GetLanguage(ctx, args.Language)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Language %q not found.", args.Language)},
				},
			}, nil, nil
		}
		languageID = lang.ID
	}

	sources, err := db.ListSources(ctx, languageID)
	if err != nil {
		return nil, nil, fmt.Errorf("list sources: %w", err)
	}

	if len(sources) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "No sources found."},
			},
		}, nil, nil
	}

	var text string
	if args.Language != "" {
		text += fmt.Sprintf("Sources for %s:\n\n", args.Language)
	} else {
		text += "All sources:\n\n"
	}
	for _, src := range sources {
		text += fmt.Sprintf("- **%s** (%s): %s\n", src.Name, src.Type, src.URL)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, nil, nil
}
