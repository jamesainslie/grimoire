package embed_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jamesainslie/grimoire/internal/embed"
)

func TestClient_Embed(t *testing.T) {
	t.Parallel()

	// Create mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		var req struct {
			Model string `json:"model"`
			Input string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Model != "snowflake-arctic-embed:l" {
			t.Errorf("unexpected model: %s", req.Model)
		}

		// Return mock embedding (1024 dimensions)
		resp := struct {
			Embeddings [][]float32 `json:"embeddings"`
		}{
			Embeddings: [][]float32{make([]float32, 1024)},
		}
		resp.Embeddings[0][0] = 0.5 // Set a value we can verify

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := embed.New(server.URL, "snowflake-arctic-embed:l")

	embedding, err := client.Embed(context.Background(), "test text")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if len(embedding) != 1024 {
		t.Errorf("Embed() returned %d dimensions, want 1024", len(embedding))
	}

	if embedding[0] != 0.5 {
		t.Errorf("Embed() first value = %v, want 0.5", embedding[0])
	}
}

func TestClient_EmbedBatch(t *testing.T) {
	t.Parallel()

	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Return embeddings for each input
		resp := struct {
			Embeddings [][]float32 `json:"embeddings"`
		}{
			Embeddings: make([][]float32, len(req.Input)),
		}
		for i := range req.Input {
			resp.Embeddings[i] = make([]float32, 1024)
			resp.Embeddings[i][0] = float32(i + 1) // Different value for each
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := embed.New(server.URL, "snowflake-arctic-embed:l")

	texts := []string{"text one", "text two", "text three"}
	embeddings, err := client.EmbedBatch(context.Background(), texts)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}

	if len(embeddings) != 3 {
		t.Errorf("EmbedBatch() returned %d embeddings, want 3", len(embeddings))
	}

	// Verify each embedding has correct value
	for i, emb := range embeddings {
		if emb[0] != float32(i+1) {
			t.Errorf("EmbedBatch()[%d][0] = %v, want %v", i, emb[0], float32(i+1))
		}
	}

	// Should be a single API call for batch
	if callCount != 1 {
		t.Errorf("EmbedBatch() made %d API calls, want 1", callCount)
	}
}

func TestClient_EmbedServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := embed.New(server.URL, "snowflake-arctic-embed:l")

	_, err := client.Embed(context.Background(), "test")
	if err == nil {
		t.Error("Embed() expected error for server error, got nil")
	}
}
