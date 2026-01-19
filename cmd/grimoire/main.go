package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jamesainslie/grimoire/internal/chunk"
	"github.com/jamesainslie/grimoire/internal/embed"
	"github.com/jamesainslie/grimoire/internal/ingest"
	"github.com/jamesainslie/grimoire/internal/parse"
	"github.com/jamesainslie/grimoire/internal/source/git"
	"github.com/jamesainslie/grimoire/internal/store"
	"github.com/spf13/cobra"
)

// Global flags
var (
	dbPath    string
	ollamaURL string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "grimoire",
	Short: "AI knowledge base for programming languages",
	Long: `Grimoire is a vector database of programming best practices,
queryable via CLI and MCP server. It provides semantic search
over curated documentation for Go, Rust, Python, and more.`,
}

// Language commands
var languagesCmd = &cobra.Command{
	Use:   "languages",
	Short: "Manage installed languages",
}

var languagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed languages",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Installed languages:")
		fmt.Println("  (none installed yet)")
		return nil
	},
}

var languagesInstallCmd = &cobra.Command{
	Use:   "install <language>",
	Short: "Install a language pack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Installing language pack: %s\n", args[0])
		return nil
	},
}

// Ingest command
var ingestCmd = &cobra.Command{
	Use:   "ingest <language-pack-path>",
	Short: "Fetch and index documentation sources from a language pack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		packPath := args[0]
		sourceFilter, _ := cmd.Flags().GetString("source")

		// Load language pack
		fmt.Printf("Loading language pack from %s...\n", packPath)
		pack, err := ingest.LoadLanguagePack(packPath)
		if err != nil {
			return fmt.Errorf("load language pack: %w", err)
		}
		if err := pack.Validate(); err != nil {
			return fmt.Errorf("invalid language pack: %w", err)
		}
		fmt.Printf("Language: %s (%s)\n", pack.DisplayName, pack.Language)
		fmt.Printf("Sources: %d\n\n", len(pack.Sources))

		// Open database
		db, err := store.New(getDBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		// Create or get language
		lang, err := db.GetLanguage(ctx, pack.Language)
		if err != nil {
			lang, err = db.CreateLanguage(ctx, pack.Language, pack.DisplayName)
			if err != nil {
				return fmt.Errorf("create language: %w", err)
			}
			fmt.Printf("Created language: %s\n", pack.Language)
		}

		// Set up git fetcher and embeddings client
		cacheDir := filepath.Join(filepath.Dir(getDBPath()), "cache", "git")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return fmt.Errorf("create cache dir: %w", err)
		}
		fetcher := git.NewFetcher(cacheDir)
		embedClient := embed.New(ollamaURL, "snowflake-arctic-embed:l")
		chunker := chunk.NewChunker(512) // ~512 tokens per chunk

		// Process each source
		for _, srcDef := range pack.Sources {
			// Filter by source if specified
			if sourceFilter != "" && srcDef.Name != sourceFilter {
				continue
			}

			fmt.Printf("\n--- Processing: %s ---\n", srcDef.Name)

			// Only handle git sources for now
			if srcDef.Type != "git" {
				fmt.Printf("  Skipping (type %s not yet supported)\n", srcDef.Type)
				continue
			}

			// Fetch/update repository
			fmt.Printf("  Fetching %s...\n", srcDef.URL)
			repoPath, err := fetcher.Fetch(ctx, srcDef.URL)
			if err != nil {
				fmt.Printf("  Error fetching: %v\n", err)
				continue
			}

			// Create or get source
			src, err := db.CreateSource(ctx, lang.ID, srcDef.Name, srcDef.Type, srcDef.URL)
			if err != nil {
				// Source might already exist, try to find it
				sources, _ := db.ListSources(ctx, lang.ID)
				for _, s := range sources {
					if s.Name == srcDef.Name {
						src = s
						break
					}
				}
				if src == nil {
					fmt.Printf("  Error creating source: %v\n", err)
					continue
				}
			}

			// List files matching patterns
			files, err := fetcher.ListFiles(repoPath, srcDef.Paths)
			if err != nil {
				fmt.Printf("  Error listing files: %v\n", err)
				continue
			}
			fmt.Printf("  Found %d files\n", len(files))

			// Process each file
			for _, relPath := range files {
				fullPath := filepath.Join(repoPath, relPath)
				fmt.Printf("    Processing: %s\n", relPath)

				// Read and parse file
				content, err := os.ReadFile(fullPath)
				if err != nil {
					fmt.Printf("      Error reading: %v\n", err)
					continue
				}

				doc, err := parse.Parse(content)
				if err != nil {
					fmt.Printf("      Error parsing: %v\n", err)
					continue
				}

				// Create or get document
				dbDoc, err := db.CreateDocument(ctx, src.ID, relPath, doc.Title)
				if err != nil {
					// Document might already exist, try to find it
					dbDoc, err = db.GetDocumentByPath(ctx, src.ID, relPath)
					if err != nil {
						fmt.Printf("      Error creating document: %v\n", err)
						continue
					}
					fmt.Printf("      (already indexed, skipping)\n")
					continue
				}

				// Chunk the document
				chunks, err := chunker.Chunk(doc)
				if err != nil {
					fmt.Printf("      Error chunking: %v\n", err)
					continue
				}
				fmt.Printf("      %d chunks\n", len(chunks))

				// Store chunks and embeddings
				chunkIDs := make(map[int]int64) // chunk index -> db ID
				for i, c := range chunks {
					var parentID *int64
					if c.ParentIndex != nil {
						if pid, ok := chunkIDs[*c.ParentIndex]; ok {
							parentID = &pid
						}
					}

					dbChunk, err := db.CreateChunk(ctx, dbDoc.ID, parentID, c.Level, c.Title, c.Content, c.TokenCount)
					if err != nil {
						fmt.Printf("      Error creating chunk: %v\n", err)
						continue
					}
					chunkIDs[i] = dbChunk.ID

					// Generate and store embedding (skip if content too short)
					if len(strings.TrimSpace(c.Content)) < 10 {
						continue
					}
					embedding, err := embedClient.Embed(ctx, c.Content)
					if err != nil {
						fmt.Printf("      Error embedding: %v\n", err)
						continue
					}
					if err := db.StoreEmbedding(ctx, dbChunk.ID, embedding); err != nil {
						fmt.Printf("      Error storing embedding: %v\n", err)
						continue
					}
				}
			}
		}

		fmt.Println("\nIngest complete!")
		return nil
	},
}

// Query command
var queryCmd = &cobra.Command{
	Use:   "query <text>",
	Short: "Search the knowledge base",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		query := args[0]
		lang, _ := cmd.Flags().GetString("lang")
		limit, _ := cmd.Flags().GetInt("limit")
		vectorOnly, _ := cmd.Flags().GetBool("vector-only")

		// Open database
		db, err := store.New(getDBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		// Get language ID if specified
		var languageID int64
		if lang != "" {
			language, err := db.GetLanguage(ctx, lang)
			if err != nil {
				return fmt.Errorf("language %q not found: %w", lang, err)
			}
			languageID = language.ID
		}

		// Get query embedding from Ollama
		client := embed.New(ollamaURL, "snowflake-arctic-embed:l")
		queryVec, err := client.Embed(ctx, query)
		if err != nil {
			return fmt.Errorf("get embedding: %w", err)
		}

		// Perform search
		var results []*store.SearchResult
		if vectorOnly {
			results, err = db.SearchChunksVectorWithScore(ctx, queryVec, languageID, limit)
		} else {
			results, err = db.SearchChunksHybrid(ctx, queryVec, query, languageID, limit)
		}
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		if len(results) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		// Display results
		// Distance is 0-1 where 0=perfect match, so relevance = 1 - distance
		fmt.Printf("Found %d results for %q:\n\n", len(results), query)
		for i, r := range results {
			relevance := 1.0 - r.Distance
			fmt.Printf("─── Result %d (relevance: %.0f%%) ───\n", i+1, relevance*100)
			fmt.Printf("Title: %s\n", r.Chunk.Title)
			fmt.Printf("Level: %s\n", r.Chunk.Level)
			fmt.Printf("\n%s\n\n", truncate(r.Chunk.Content, 500))
		}

		return nil
	},
}

// truncate truncates a string to maxLen characters with ellipsis.
func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// getDBPath returns the database path, using default if not specified.
func getDBPath() string {
	if dbPath != "" {
		return dbPath
	}
	// Default to ~/.grimoire/grimoire.db
	home, err := os.UserHomeDir()
	if err != nil {
		return "grimoire.db"
	}
	dir := filepath.Join(home, ".grimoire")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "grimoire.db"
	}
	return filepath.Join(dir, "grimoire.db")
}

// Sources commands
var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "Manage documentation sources",
}

var sourcesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sources",
	RunE: func(cmd *cobra.Command, args []string) error {
		lang, _ := cmd.Flags().GetString("lang")
		if lang != "" {
			fmt.Printf("Sources for %s:\n", lang)
		} else {
			fmt.Println("All sources:")
		}
		fmt.Println("  (none configured yet)")
		return nil
	},
}

var sourcesAddCmd = &cobra.Command{
	Use:   "add <language> <type> <url>",
	Short: "Add a custom source",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Adding source:\n  Language: %s\n  Type: %s\n  URL: %s\n", args[0], args[1], args[2])
		return nil
	},
}

// Stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show knowledge base statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		db, err := store.New(getDBPath())
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		stats, err := db.GetStats(ctx)
		if err != nil {
			return fmt.Errorf("get stats: %w", err)
		}

		fmt.Printf("Grimoire Knowledge Base Statistics\n")
		fmt.Printf("===================================\n")
		fmt.Printf("Database: %s\n\n", getDBPath())
		fmt.Printf("  Languages:  %d\n", stats.Languages)
		fmt.Printf("  Sources:    %d\n", stats.Sources)
		fmt.Printf("  Documents:  %d\n", stats.Documents)
		fmt.Printf("  Chunks:     %d\n", stats.Chunks)
		fmt.Printf("  Embeddings: %d\n", stats.Embeddings)
		return nil
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Database path (default: ~/.grimoire/grimoire.db)")
	rootCmd.PersistentFlags().StringVar(&ollamaURL, "ollama-url", "http://localhost:11434", "Ollama API URL")

	// Add language commands
	rootCmd.AddCommand(languagesCmd)
	languagesCmd.AddCommand(languagesListCmd)
	languagesCmd.AddCommand(languagesInstallCmd)

	// Add ingest command
	rootCmd.AddCommand(ingestCmd)
	ingestCmd.Flags().String("source", "", "Filter by source name")

	// Add query command
	rootCmd.AddCommand(queryCmd)
	queryCmd.Flags().String("lang", "", "Filter by language")
	queryCmd.Flags().Int("limit", 5, "Maximum number of results")
	queryCmd.Flags().Bool("vector-only", false, "Use vector search only (no FTS)")

	// Add sources commands
	rootCmd.AddCommand(sourcesCmd)
	sourcesCmd.AddCommand(sourcesListCmd)
	sourcesCmd.AddCommand(sourcesAddCmd)
	sourcesListCmd.Flags().String("lang", "", "Filter by language")

	// Add stats command
	rootCmd.AddCommand(statsCmd)
}
