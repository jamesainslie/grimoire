// Package store provides SQLite-based storage for the grimoire knowledge base.
package store

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// Language represents a programming language in the knowledge base.
type Language struct {
	ID          int64
	Name        string
	DisplayName string
}

// Source represents a documentation source (git repo or web page).
type Source struct {
	ID         int64
	LanguageID int64
	Name       string
	Type       string // "git" or "web"
	URL        string
}

// Document represents a single document (file or page) from a source.
type Document struct {
	ID       int64
	SourceID int64
	Path     string
	Title    string
}

// Chunk represents a piece of content from a document.
type Chunk struct {
	ID            int64
	DocumentID    int64
	ParentChunkID *int64
	Level         string // "summary", "section", "paragraph"
	Title         string
	Content       string
	TokenCount    int
}

// SearchResult represents a chunk with its similarity score.
type SearchResult struct {
	Chunk    *Chunk
	Distance float64 // Lower distance = more similar
}

// Store provides access to the grimoire knowledge base.
type Store struct {
	db *sql.DB
}

func init() {
	sqlite_vec.Auto()
}

// New creates a new Store with the given database path.
// Use ":memory:" for an in-memory database.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	s := &Store{db: db}

	if err := s.init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// init creates the database schema.
func (s *Store) init() error {
	schema := `
		CREATE TABLE IF NOT EXISTS languages (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS sources (
			id INTEGER PRIMARY KEY,
			language_id INTEGER NOT NULL REFERENCES languages(id),
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			url TEXT NOT NULL,
			UNIQUE(language_id, name)
		);

		CREATE TABLE IF NOT EXISTS documents (
			id INTEGER PRIMARY KEY,
			source_id INTEGER NOT NULL REFERENCES sources(id),
			path TEXT NOT NULL,
			title TEXT,
			UNIQUE(source_id, path)
		);

		CREATE TABLE IF NOT EXISTS chunks (
			id INTEGER PRIMARY KEY,
			document_id INTEGER NOT NULL REFERENCES documents(id),
			parent_chunk_id INTEGER REFERENCES chunks(id),
			level TEXT NOT NULL,
			title TEXT,
			content TEXT NOT NULL,
			token_count INTEGER
		);

		CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
			title,
			content,
			content='chunks',
			content_rowid='id'
		);

		CREATE TRIGGER IF NOT EXISTS chunks_ai AFTER INSERT ON chunks BEGIN
			INSERT INTO chunks_fts(rowid, title, content)
			VALUES (new.id, new.title, new.content);
		END;

		CREATE TRIGGER IF NOT EXISTS chunks_ad AFTER DELETE ON chunks BEGIN
			INSERT INTO chunks_fts(chunks_fts, rowid, title, content)
			VALUES ('delete', old.id, old.title, old.content);
		END;

		CREATE TRIGGER IF NOT EXISTS chunks_au AFTER UPDATE ON chunks BEGIN
			INSERT INTO chunks_fts(chunks_fts, rowid, title, content)
			VALUES ('delete', old.id, old.title, old.content);
			INSERT INTO chunks_fts(rowid, title, content)
			VALUES (new.id, new.title, new.content);
		END;
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	// Create vector table (requires separate statement)
	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS chunks_vec USING vec0(
			chunk_id INTEGER PRIMARY KEY,
			embedding FLOAT[1024]
		)
	`)
	if err != nil {
		return fmt.Errorf("create vector table: %w", err)
	}

	return nil
}

// CreateLanguage creates a new language in the store.
func (s *Store) CreateLanguage(ctx context.Context, name, displayName string) (*Language, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO languages (name, display_name) VALUES (?, ?)",
		name, displayName,
	)
	if err != nil {
		return nil, fmt.Errorf("insert language: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	return &Language{
		ID:          id,
		Name:        name,
		DisplayName: displayName,
	}, nil
}

// GetLanguage retrieves a language by name.
func (s *Store) GetLanguage(ctx context.Context, name string) (*Language, error) {
	var lang Language
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, display_name FROM languages WHERE name = ?",
		name,
	).Scan(&lang.ID, &lang.Name, &lang.DisplayName)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("language %q: %w", name, ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query language: %w", err)
	}

	return &lang, nil
}

// ListLanguages returns all languages in the store.
func (s *Store) ListLanguages(ctx context.Context) ([]*Language, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, display_name FROM languages ORDER BY name",
	)
	if err != nil {
		return nil, fmt.Errorf("query languages: %w", err)
	}
	defer rows.Close()

	var langs []*Language
	for rows.Next() {
		var lang Language
		if err := rows.Scan(&lang.ID, &lang.Name, &lang.DisplayName); err != nil {
			return nil, fmt.Errorf("scan language: %w", err)
		}
		langs = append(langs, &lang)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate languages: %w", err)
	}

	return langs, nil
}

// CreateSource creates a new source in the store.
func (s *Store) CreateSource(ctx context.Context, languageID int64, name, sourceType, url string) (*Source, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO sources (language_id, name, type, url) VALUES (?, ?, ?, ?)",
		languageID, name, sourceType, url,
	)
	if err != nil {
		return nil, fmt.Errorf("insert source: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	return &Source{
		ID:         id,
		LanguageID: languageID,
		Name:       name,
		Type:       sourceType,
		URL:        url,
	}, nil
}

// ListSources returns sources, optionally filtered by language ID.
// Pass 0 for languageID to list all sources.
func (s *Store) ListSources(ctx context.Context, languageID int64) ([]*Source, error) {
	var rows *sql.Rows
	var err error

	if languageID == 0 {
		rows, err = s.db.QueryContext(ctx,
			"SELECT id, language_id, name, type, url FROM sources ORDER BY name",
		)
	} else {
		rows, err = s.db.QueryContext(ctx,
			"SELECT id, language_id, name, type, url FROM sources WHERE language_id = ? ORDER BY name",
			languageID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("query sources: %w", err)
	}
	defer rows.Close()

	var sources []*Source
	for rows.Next() {
		var src Source
		if err := rows.Scan(&src.ID, &src.LanguageID, &src.Name, &src.Type, &src.URL); err != nil {
			return nil, fmt.Errorf("scan source: %w", err)
		}
		sources = append(sources, &src)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sources: %w", err)
	}

	return sources, nil
}

// GetDocumentByPath returns a document by its source ID and path.
func (s *Store) GetDocumentByPath(ctx context.Context, sourceID int64, path string) (*Document, error) {
	var doc Document
	err := s.db.QueryRowContext(ctx,
		"SELECT id, source_id, path, title FROM documents WHERE source_id = ? AND path = ?",
		sourceID, path,
	).Scan(&doc.ID, &doc.SourceID, &doc.Path, &doc.Title)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// CreateDocument creates a new document in the store.
func (s *Store) CreateDocument(ctx context.Context, sourceID int64, path, title string) (*Document, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO documents (source_id, path, title) VALUES (?, ?, ?)",
		sourceID, path, title,
	)
	if err != nil {
		return nil, fmt.Errorf("insert document: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	return &Document{
		ID:       id,
		SourceID: sourceID,
		Path:     path,
		Title:    title,
	}, nil
}

// CreateChunk creates a new chunk in the store.
func (s *Store) CreateChunk(ctx context.Context, documentID int64, parentChunkID *int64, level, title, content string, tokenCount int) (*Chunk, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO chunks (document_id, parent_chunk_id, level, title, content, token_count) VALUES (?, ?, ?, ?, ?, ?)",
		documentID, parentChunkID, level, title, content, tokenCount,
	)
	if err != nil {
		return nil, fmt.Errorf("insert chunk: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	return &Chunk{
		ID:            id,
		DocumentID:    documentID,
		ParentChunkID: parentChunkID,
		Level:         level,
		Title:         title,
		Content:       content,
		TokenCount:    tokenCount,
	}, nil
}

// SearchChunksFTS searches chunks using full-text search.
// Pass languageID=0 to search all languages.
func (s *Store) SearchChunksFTS(ctx context.Context, query string, languageID int64, limit int) ([]*Chunk, error) {
	var rows *sql.Rows
	var err error

	if languageID == 0 {
		rows, err = s.db.QueryContext(ctx, `
			SELECT c.id, c.document_id, c.parent_chunk_id, c.level, c.title, c.content, c.token_count
			FROM chunks c
			JOIN chunks_fts fts ON c.id = fts.rowid
			WHERE chunks_fts MATCH ?
			ORDER BY rank
			LIMIT ?
		`, query, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT c.id, c.document_id, c.parent_chunk_id, c.level, c.title, c.content, c.token_count
			FROM chunks c
			JOIN chunks_fts fts ON c.id = fts.rowid
			JOIN documents d ON c.document_id = d.id
			JOIN sources s ON d.source_id = s.id
			WHERE chunks_fts MATCH ? AND s.language_id = ?
			ORDER BY rank
			LIMIT ?
		`, query, languageID, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}
	defer rows.Close()

	var chunks []*Chunk
	for rows.Next() {
		var chunk Chunk
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.ParentChunkID, &chunk.Level, &chunk.Title, &chunk.Content, &chunk.TokenCount); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		chunks = append(chunks, &chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chunks: %w", err)
	}

	return chunks, nil
}

// StoreEmbedding stores a vector embedding for a chunk.
func (s *Store) StoreEmbedding(ctx context.Context, chunkID int64, embedding []float32) error {
	blob := float32ToBytes(embedding)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO chunks_vec (chunk_id, embedding) VALUES (?, ?)",
		chunkID, blob,
	)
	if err != nil {
		return fmt.Errorf("insert embedding: %w", err)
	}
	return nil
}

// SearchChunksVector searches chunks using vector similarity.
// Pass languageID=0 to search all languages.
func (s *Store) SearchChunksVector(ctx context.Context, queryVec []float32, languageID int64, limit int) ([]*Chunk, error) {
	blob := float32ToBytes(queryVec)

	var rows *sql.Rows
	var err error

	if languageID == 0 {
		rows, err = s.db.QueryContext(ctx, `
			SELECT c.id, c.document_id, c.parent_chunk_id, c.level, c.title, c.content, c.token_count
			FROM chunks c
			JOIN chunks_vec v ON c.id = v.chunk_id
			WHERE v.embedding MATCH ? AND k = ?
			ORDER BY distance
		`, blob, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT c.id, c.document_id, c.parent_chunk_id, c.level, c.title, c.content, c.token_count
			FROM chunks c
			JOIN (
				SELECT chunk_id, distance
				FROM chunks_vec
				WHERE embedding MATCH ? AND k = ?
			) v ON c.id = v.chunk_id
			JOIN documents d ON c.document_id = d.id
			JOIN sources s ON d.source_id = s.id
			WHERE s.language_id = ?
			ORDER BY v.distance
		`, blob, limit, languageID)
	}
	if err != nil {
		return nil, fmt.Errorf("search chunks vector: %w", err)
	}
	defer rows.Close()

	var chunks []*Chunk
	for rows.Next() {
		var chunk Chunk
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.ParentChunkID, &chunk.Level, &chunk.Title, &chunk.Content, &chunk.TokenCount); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		chunks = append(chunks, &chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chunks: %w", err)
	}

	return chunks, nil
}

// SearchChunksVectorWithScore searches chunks using vector similarity and returns distances.
// Pass languageID=0 to search all languages.
func (s *Store) SearchChunksVectorWithScore(ctx context.Context, queryVec []float32, languageID int64, limit int) ([]*SearchResult, error) {
	blob := float32ToBytes(queryVec)

	var rows *sql.Rows
	var err error

	if languageID == 0 {
		rows, err = s.db.QueryContext(ctx, `
			SELECT c.id, c.document_id, c.parent_chunk_id, c.level, c.title, c.content, c.token_count, v.distance
			FROM chunks c
			JOIN chunks_vec v ON c.id = v.chunk_id
			WHERE v.embedding MATCH ? AND k = ?
			ORDER BY v.distance
		`, blob, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT c.id, c.document_id, c.parent_chunk_id, c.level, c.title, c.content, c.token_count, v.distance
			FROM chunks c
			JOIN (
				SELECT chunk_id, distance
				FROM chunks_vec
				WHERE embedding MATCH ? AND k = ?
			) v ON c.id = v.chunk_id
			JOIN documents d ON c.document_id = d.id
			JOIN sources src ON d.source_id = src.id
			WHERE src.language_id = ?
			ORDER BY v.distance
		`, blob, limit*2, languageID) // Request more results to account for filtering
	}
	if err != nil {
		return nil, fmt.Errorf("search chunks vector with score: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var chunk Chunk
		var distance float64
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.ParentChunkID, &chunk.Level, &chunk.Title, &chunk.Content, &chunk.TokenCount, &distance); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		results = append(results, &SearchResult{
			Chunk:    &chunk,
			Distance: distance,
		})
		if len(results) >= limit {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}

	return results, nil
}

// SearchChunksHybrid combines vector similarity and FTS5 search using Reciprocal Rank Fusion.
// Pass languageID=0 to search all languages. If textQuery is empty, only vector search is used.
func (s *Store) SearchChunksHybrid(ctx context.Context, queryVec []float32, textQuery string, languageID int64, limit int) ([]*SearchResult, error) {
	// Get vector results
	vectorResults, err := s.SearchChunksVectorWithScore(ctx, queryVec, languageID, limit)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	// If no text query, return vector results only
	if textQuery == "" {
		return vectorResults, nil
	}

	// Get FTS results
	ftsChunks, err := s.SearchChunksFTS(ctx, textQuery, languageID, limit)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}

	// Use Reciprocal Rank Fusion to combine results
	// RRF score = sum(1 / (k + rank)) for each ranking list where the document appears
	// k is a constant (typically 60) to prevent high scores dominating
	const k = 60.0

	// Build a map of chunk IDs to RRF scores
	scores := make(map[int64]float64)
	chunks := make(map[int64]*Chunk)

	// Add vector results
	for i, r := range vectorResults {
		rank := float64(i + 1)
		scores[r.Chunk.ID] += 1.0 / (k + rank)
		chunks[r.Chunk.ID] = r.Chunk
	}

	// Add FTS results
	for i, c := range ftsChunks {
		rank := float64(i + 1)
		scores[c.ID] += 1.0 / (k + rank)
		if _, exists := chunks[c.ID]; !exists {
			chunks[c.ID] = c
		}
	}

	// Convert to results slice and sort by RRF score (higher is better)
	var results []*SearchResult
	for id, score := range scores {
		results = append(results, &SearchResult{
			Chunk:    chunks[id],
			Distance: 1.0 / score, // Convert to distance (lower = better)
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// Stats represents statistics about the knowledge base.
type Stats struct {
	Languages  int64
	Sources    int64
	Documents  int64
	Chunks     int64
	Embeddings int64
}

// GetStats returns statistics about the knowledge base.
func (s *Store) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{}

	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM languages").Scan(&stats.Languages)
	if err != nil {
		return nil, fmt.Errorf("count languages: %w", err)
	}

	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sources").Scan(&stats.Sources)
	if err != nil {
		return nil, fmt.Errorf("count sources: %w", err)
	}

	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM documents").Scan(&stats.Documents)
	if err != nil {
		return nil, fmt.Errorf("count documents: %w", err)
	}

	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunks").Scan(&stats.Chunks)
	if err != nil {
		return nil, fmt.Errorf("count chunks: %w", err)
	}

	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chunks_vec").Scan(&stats.Embeddings)
	if err != nil {
		return nil, fmt.Errorf("count embeddings: %w", err)
	}

	return stats, nil
}

// float32ToBytes converts a slice of float32 to a byte slice for storage.
func float32ToBytes(floats []float32) []byte {
	buf := make([]byte, len(floats)*4)
	for i, f := range floats {
		bits := math.Float32bits(f)
		binary.LittleEndian.PutUint32(buf[i*4:], bits)
	}
	return buf
}
