// Package embed provides an embedding client for generating vector embeddings.
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client generates embeddings using an Ollama-compatible API.
type Client struct {
	baseURL string
	model   string
	http    *http.Client
}

// New creates a new embedding client.
// baseURL is the Ollama API URL (e.g., "http://localhost:11434").
// model is the embedding model name (e.g., "snowflake-arctic-embed:l").
func New(baseURL, model string) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{},
	}
}

// embedRequest is the request body for the Ollama embed API.
type embedRequest struct {
	Model string `json:"model"`
	Input any    `json:"input"` // string for single, []string for batch
}

// embedResponse is the response from the Ollama embed API.
type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// Embed generates an embedding for a single text.
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := c.embed(ctx, text)
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts in a single API call.
func (c *Client) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	return c.embed(ctx, texts)
}

// embed sends an embedding request to the Ollama API.
func (c *Client) embed(ctx context.Context, input any) ([][]float32, error) {
	reqBody := embedRequest{
		Model: c.model,
		Input: input,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API error: status %d", resp.StatusCode)
	}

	var embedResp embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return embedResp.Embeddings, nil
}
