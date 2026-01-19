package store_test

import (
	"context"
	"testing"

	"github.com/jamesainslie/grimoire/internal/store"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "creates in-memory database",
			path:    ":memory:",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := store.New(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			defer s.Close()

			if s == nil {
				t.Error("New() returned nil store")
			}
		})
	}
}

func TestStore_CreateLanguage(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)

	ctx := context.Background()

	tests := []struct {
		name        string
		langName    string
		displayName string
		wantErr     bool
	}{
		{
			name:        "creates new language",
			langName:    "go",
			displayName: "Go",
			wantErr:     false,
		},
		{
			name:        "duplicate language fails",
			langName:    "go",
			displayName: "Go",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang, err := s.CreateLanguage(ctx, tt.langName, tt.displayName)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateLanguage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if lang.ID == 0 {
				t.Error("CreateLanguage() returned language with zero ID")
			}
			if lang.Name != tt.langName {
				t.Errorf("CreateLanguage() Name = %v, want %v", lang.Name, tt.langName)
			}
			if lang.DisplayName != tt.displayName {
				t.Errorf("CreateLanguage() DisplayName = %v, want %v", lang.DisplayName, tt.displayName)
			}
		})
	}
}

func TestStore_GetLanguage(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	// Create a language first
	created, err := s.CreateLanguage(ctx, "rust", "Rust")
	if err != nil {
		t.Fatalf("setup: CreateLanguage() error = %v", err)
	}

	tests := []struct {
		name     string
		langName string
		want     *store.Language
		wantErr  bool
	}{
		{
			name:     "finds existing language",
			langName: "rust",
			want:     created,
			wantErr:  false,
		},
		{
			name:     "returns error for non-existent language",
			langName: "python",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.GetLanguage(ctx, tt.langName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLanguage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if got.ID != tt.want.ID {
				t.Errorf("GetLanguage() ID = %v, want %v", got.ID, tt.want.ID)
			}
			if got.Name != tt.want.Name {
				t.Errorf("GetLanguage() Name = %v, want %v", got.Name, tt.want.Name)
			}
		})
	}
}

func TestStore_ListLanguages(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	// Initially empty
	langs, err := s.ListLanguages(ctx)
	if err != nil {
		t.Fatalf("ListLanguages() error = %v", err)
	}
	if len(langs) != 0 {
		t.Errorf("ListLanguages() on empty store = %d languages, want 0", len(langs))
	}

	// Add languages
	_, err = s.CreateLanguage(ctx, "go", "Go")
	if err != nil {
		t.Fatalf("CreateLanguage(go) error = %v", err)
	}
	_, err = s.CreateLanguage(ctx, "rust", "Rust")
	if err != nil {
		t.Fatalf("CreateLanguage(rust) error = %v", err)
	}

	langs, err = s.ListLanguages(ctx)
	if err != nil {
		t.Fatalf("ListLanguages() error = %v", err)
	}
	if len(langs) != 2 {
		t.Errorf("ListLanguages() = %d languages, want 2", len(langs))
	}
}

func TestStore_CreateSource(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	// Create a language first
	lang, err := s.CreateLanguage(ctx, "go", "Go")
	if err != nil {
		t.Fatalf("setup: CreateLanguage() error = %v", err)
	}

	tests := []struct {
		name       string
		langID     int64
		sourceName string
		sourceType string
		url        string
		wantErr    bool
	}{
		{
			name:       "creates git source",
			langID:     lang.ID,
			sourceName: "uber-guide",
			sourceType: "git",
			url:        "https://github.com/uber-go/guide",
			wantErr:    false,
		},
		{
			name:       "creates web source",
			langID:     lang.ID,
			sourceName: "dave-cheney",
			sourceType: "web",
			url:        "https://dave.cheney.net",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := s.CreateSource(ctx, tt.langID, tt.sourceName, tt.sourceType, tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if src.ID == 0 {
				t.Error("CreateSource() returned source with zero ID")
			}
			if src.Name != tt.sourceName {
				t.Errorf("CreateSource() Name = %v, want %v", src.Name, tt.sourceName)
			}
			if src.Type != tt.sourceType {
				t.Errorf("CreateSource() Type = %v, want %v", src.Type, tt.sourceType)
			}
		})
	}
}

func TestStore_ListSources(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	// Create languages
	goLang, err := s.CreateLanguage(ctx, "go", "Go")
	if err != nil {
		t.Fatalf("CreateLanguage(go) error = %v", err)
	}
	rustLang, err := s.CreateLanguage(ctx, "rust", "Rust")
	if err != nil {
		t.Fatalf("CreateLanguage(rust) error = %v", err)
	}

	// Create sources
	_, err = s.CreateSource(ctx, goLang.ID, "uber-guide", "git", "https://github.com/uber-go/guide")
	if err != nil {
		t.Fatalf("CreateSource(uber-guide) error = %v", err)
	}
	_, err = s.CreateSource(ctx, goLang.ID, "go-wiki", "git", "https://github.com/golang/go")
	if err != nil {
		t.Fatalf("CreateSource(go-wiki) error = %v", err)
	}
	_, err = s.CreateSource(ctx, rustLang.ID, "rust-book", "web", "https://doc.rust-lang.org/book")
	if err != nil {
		t.Fatalf("CreateSource(rust-book) error = %v", err)
	}

	// List all sources
	allSources, err := s.ListSources(ctx, 0)
	if err != nil {
		t.Fatalf("ListSources(all) error = %v", err)
	}
	if len(allSources) != 3 {
		t.Errorf("ListSources(all) = %d sources, want 3", len(allSources))
	}

	// List Go sources only
	goSources, err := s.ListSources(ctx, goLang.ID)
	if err != nil {
		t.Fatalf("ListSources(go) error = %v", err)
	}
	if len(goSources) != 2 {
		t.Errorf("ListSources(go) = %d sources, want 2", len(goSources))
	}
}

func TestStore_CreateDocument(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	// Setup
	lang, _ := s.CreateLanguage(ctx, "go", "Go")
	src, _ := s.CreateSource(ctx, lang.ID, "uber-guide", "git", "https://github.com/uber-go/guide")

	doc, err := s.CreateDocument(ctx, src.ID, "style.md", "Uber Go Style Guide")
	if err != nil {
		t.Fatalf("CreateDocument() error = %v", err)
	}

	if doc.ID == 0 {
		t.Error("CreateDocument() returned document with zero ID")
	}
	if doc.Path != "style.md" {
		t.Errorf("CreateDocument() Path = %v, want %v", doc.Path, "style.md")
	}
	if doc.Title != "Uber Go Style Guide" {
		t.Errorf("CreateDocument() Title = %v, want %v", doc.Title, "Uber Go Style Guide")
	}
}

func TestStore_CreateChunk(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	// Setup
	lang, _ := s.CreateLanguage(ctx, "go", "Go")
	src, _ := s.CreateSource(ctx, lang.ID, "uber-guide", "git", "https://github.com/uber-go/guide")
	doc, _ := s.CreateDocument(ctx, src.ID, "style.md", "Uber Go Style Guide")

	// Create summary chunk (no parent)
	summary, err := s.CreateChunk(ctx, doc.ID, nil, "summary", "Introduction", "This is the summary content", 50)
	if err != nil {
		t.Fatalf("CreateChunk(summary) error = %v", err)
	}
	if summary.Level != "summary" {
		t.Errorf("CreateChunk() Level = %v, want summary", summary.Level)
	}

	// Create section chunk (parent = summary)
	section, err := s.CreateChunk(ctx, doc.ID, &summary.ID, "section", "Error Handling", "Always handle errors", 20)
	if err != nil {
		t.Fatalf("CreateChunk(section) error = %v", err)
	}
	if section.ParentChunkID == nil || *section.ParentChunkID != summary.ID {
		t.Errorf("CreateChunk() ParentChunkID = %v, want %v", section.ParentChunkID, summary.ID)
	}
}

func TestStore_SearchChunks(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	// Setup
	lang, _ := s.CreateLanguage(ctx, "go", "Go")
	src, _ := s.CreateSource(ctx, lang.ID, "uber-guide", "git", "https://github.com/uber-go/guide")
	doc, _ := s.CreateDocument(ctx, src.ID, "style.md", "Uber Go Style Guide")

	// Create chunks with searchable content
	_, _ = s.CreateChunk(ctx, doc.ID, nil, "section", "Error Handling", "Always wrap errors with context using fmt.Errorf", 30)
	_, _ = s.CreateChunk(ctx, doc.ID, nil, "section", "Naming", "Use short variable names in narrow scope", 25)

	// Search for "error"
	results, err := s.SearchChunksFTS(ctx, "error", 0, 10)
	if err != nil {
		t.Fatalf("SearchChunksFTS() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchChunksFTS('error') = %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].Title != "Error Handling" {
		t.Errorf("SearchChunksFTS() first result Title = %v, want Error Handling", results[0].Title)
	}
}

func TestStore_VectorSearch(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	// Setup
	lang, _ := s.CreateLanguage(ctx, "go", "Go")
	src, _ := s.CreateSource(ctx, lang.ID, "uber-guide", "git", "https://github.com/uber-go/guide")
	doc, _ := s.CreateDocument(ctx, src.ID, "style.md", "Uber Go Style Guide")

	// Create chunks with embeddings (1024-dim vectors)
	embedding1 := make([]float32, 1024)
	embedding1[0] = 1.0 // error handling direction
	embedding2 := make([]float32, 1024)
	embedding2[1] = 1.0 // naming direction

	chunk1, _ := s.CreateChunk(ctx, doc.ID, nil, "section", "Error Handling", "Always wrap errors", 20)
	chunk2, _ := s.CreateChunk(ctx, doc.ID, nil, "section", "Naming", "Use short names", 15)

	// Store embeddings
	err := s.StoreEmbedding(ctx, chunk1.ID, embedding1)
	if err != nil {
		t.Fatalf("StoreEmbedding(chunk1) error = %v", err)
	}
	err = s.StoreEmbedding(ctx, chunk2.ID, embedding2)
	if err != nil {
		t.Fatalf("StoreEmbedding(chunk2) error = %v", err)
	}

	// Search with query vector similar to chunk1
	queryVec := make([]float32, 1024)
	queryVec[0] = 0.9
	queryVec[1] = 0.1

	results, err := s.SearchChunksVector(ctx, queryVec, 0, 10)
	if err != nil {
		t.Fatalf("SearchChunksVector() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("SearchChunksVector() = %d results, want 2", len(results))
	}
	// First result should be chunk1 (more similar to query)
	if len(results) > 0 && results[0].ID != chunk1.ID {
		t.Errorf("SearchChunksVector() first result ID = %v, want %v", results[0].ID, chunk1.ID)
	}
}

func TestStore_VectorSearchWithLanguageFilter(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	// Setup two languages with chunks
	goLang, _ := s.CreateLanguage(ctx, "go", "Go")
	rustLang, _ := s.CreateLanguage(ctx, "rust", "Rust")

	goSrc, _ := s.CreateSource(ctx, goLang.ID, "uber-guide", "git", "https://github.com/uber-go/guide")
	rustSrc, _ := s.CreateSource(ctx, rustLang.ID, "rust-book", "web", "https://doc.rust-lang.org")

	goDoc, _ := s.CreateDocument(ctx, goSrc.ID, "style.md", "Go Style")
	rustDoc, _ := s.CreateDocument(ctx, rustSrc.ID, "ch01.md", "Rust Chapter 1")

	// Create chunks with similar embeddings
	embedding := make([]float32, 1024)
	embedding[0] = 1.0

	goChunk, _ := s.CreateChunk(ctx, goDoc.ID, nil, "section", "Go Errors", "Handle errors in Go", 20)
	rustChunk, _ := s.CreateChunk(ctx, rustDoc.ID, nil, "section", "Rust Errors", "Handle errors in Rust", 20)

	_ = s.StoreEmbedding(ctx, goChunk.ID, embedding)
	_ = s.StoreEmbedding(ctx, rustChunk.ID, embedding)

	// Search all languages - should find both
	allResults, err := s.SearchChunksVector(ctx, embedding, 0, 10)
	if err != nil {
		t.Fatalf("SearchChunksVector(all) error = %v", err)
	}
	if len(allResults) != 2 {
		t.Errorf("SearchChunksVector(all) = %d results, want 2", len(allResults))
	}

	// Search Go only - should find only Go chunk
	goResults, err := s.SearchChunksVector(ctx, embedding, goLang.ID, 10)
	if err != nil {
		t.Fatalf("SearchChunksVector(go) error = %v", err)
	}
	if len(goResults) != 1 {
		t.Errorf("SearchChunksVector(go) = %d results, want 1", len(goResults))
	}
	if len(goResults) > 0 && goResults[0].ID != goChunk.ID {
		t.Errorf("SearchChunksVector(go) result ID = %v, want %v", goResults[0].ID, goChunk.ID)
	}
}

func TestStore_SearchChunksVectorWithScore(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	// Setup
	lang, _ := s.CreateLanguage(ctx, "go", "Go")
	src, _ := s.CreateSource(ctx, lang.ID, "uber-guide", "git", "https://github.com/uber-go/guide")
	doc, _ := s.CreateDocument(ctx, src.ID, "style.md", "Uber Go Style Guide")

	// Create chunks with different embeddings
	embedding1 := make([]float32, 1024)
	embedding1[0] = 1.0
	embedding2 := make([]float32, 1024)
	embedding2[0] = 0.5
	embedding2[1] = 0.5

	chunk1, _ := s.CreateChunk(ctx, doc.ID, nil, "section", "Close Match", "Close content", 20)
	chunk2, _ := s.CreateChunk(ctx, doc.ID, nil, "section", "Far Match", "Far content", 20)

	_ = s.StoreEmbedding(ctx, chunk1.ID, embedding1)
	_ = s.StoreEmbedding(ctx, chunk2.ID, embedding2)

	// Query with embedding similar to chunk1
	queryVec := make([]float32, 1024)
	queryVec[0] = 0.95

	results, err := s.SearchChunksVectorWithScore(ctx, queryVec, 0, 10)
	if err != nil {
		t.Fatalf("SearchChunksVectorWithScore() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("SearchChunksVectorWithScore() = %d results, want 2", len(results))
	}

	// First result should be chunk1 with lower distance (more similar)
	if results[0].Chunk.ID != chunk1.ID {
		t.Errorf("first result ID = %v, want %v", results[0].Chunk.ID, chunk1.ID)
	}

	// Distance should be included and chunk1 should have lower distance
	if results[0].Distance >= results[1].Distance {
		t.Errorf("first result distance (%v) should be less than second (%v)", results[0].Distance, results[1].Distance)
	}

	// Distance should be non-negative
	if results[0].Distance < 0 || results[1].Distance < 0 {
		t.Error("distances should be non-negative")
	}
}

func TestStore_SearchChunksHybrid(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	// Setup
	lang, _ := s.CreateLanguage(ctx, "go", "Go")
	src, _ := s.CreateSource(ctx, lang.ID, "uber-guide", "git", "https://github.com/uber-go/guide")
	doc, _ := s.CreateDocument(ctx, src.ID, "style.md", "Uber Go Style Guide")

	// Create chunks:
	// 1. Semantically similar to "error handling" but doesn't contain "error" keyword
	// 2. Contains "error" keyword but semantically different
	// 3. Both semantically similar AND contains "error" keyword (best match)

	embedding1 := make([]float32, 1024)
	embedding1[0] = 0.8 // Semantic: somewhat similar

	embedding2 := make([]float32, 1024)
	embedding2[1] = 0.9 // Semantic: different direction (naming)

	embedding3 := make([]float32, 1024)
	embedding3[0] = 0.99 // Semantic: very close to query

	chunk1, _ := s.CreateChunk(ctx, doc.ID, nil, "section", "Fault Tolerance",
		"When something goes wrong, wrap the issue with context", 30)
	chunk2, _ := s.CreateChunk(ctx, doc.ID, nil, "section", "Error Messages",
		"Error messages should include error details", 25)
	chunk3, _ := s.CreateChunk(ctx, doc.ID, nil, "section", "Error Handling",
		"Always wrap errors with context using fmt.Errorf", 30)

	_ = s.StoreEmbedding(ctx, chunk1.ID, embedding1)
	_ = s.StoreEmbedding(ctx, chunk2.ID, embedding2)
	_ = s.StoreEmbedding(ctx, chunk3.ID, embedding3)

	// Query with vector very similar to chunk3's embedding and keyword "error"
	queryVec := make([]float32, 1024)
	queryVec[0] = 1.0

	results, err := s.SearchChunksHybrid(ctx, queryVec, "error", 0, 10)
	if err != nil {
		t.Fatalf("SearchChunksHybrid() error = %v", err)
	}

	if len(results) == 0 {
		t.Fatal("SearchChunksHybrid() returned no results")
	}

	// Best result should be chunk3 (both semantic AND keyword match)
	if results[0].Chunk.ID != chunk3.ID {
		t.Errorf("first result ID = %v, want %v (Error Handling)", results[0].Chunk.ID, chunk3.ID)
	}

	// Should return all 3 chunks (2 from FTS, 2 from vector with overlap)
	if len(results) < 2 {
		t.Errorf("SearchChunksHybrid() = %d results, want at least 2", len(results))
	}
}

func TestStore_SearchChunksHybrid_VectorOnly(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	lang, _ := s.CreateLanguage(ctx, "go", "Go")
	src, _ := s.CreateSource(ctx, lang.ID, "uber-guide", "git", "https://github.com/uber-go/guide")
	doc, _ := s.CreateDocument(ctx, src.ID, "style.md", "Uber Go Style Guide")

	embedding := make([]float32, 1024)
	embedding[0] = 1.0

	chunk, _ := s.CreateChunk(ctx, doc.ID, nil, "section", "Unique Content",
		"Some unique content without common keywords", 20)
	_ = s.StoreEmbedding(ctx, chunk.ID, embedding)

	// Search with vector only (empty text query)
	queryVec := make([]float32, 1024)
	queryVec[0] = 0.95

	results, err := s.SearchChunksHybrid(ctx, queryVec, "", 0, 10)
	if err != nil {
		t.Fatalf("SearchChunksHybrid() error = %v", err)
	}

	// Should still return vector results
	if len(results) != 1 {
		t.Errorf("SearchChunksHybrid(vector only) = %d results, want 1", len(results))
	}
}

// newTestStore creates an in-memory store for testing.
func newTestStore(t *testing.T) *store.Store {
	t.Helper()

	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	return s
}
