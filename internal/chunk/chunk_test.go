package chunk_test

import (
	"testing"

	"github.com/jamesainslie/grimoire/internal/chunk"
	"github.com/jamesainslie/grimoire/internal/parse"
)

func TestChunker_Chunk(t *testing.T) {
	t.Parallel()

	input := `# Main Title

This is the introduction paragraph that explains the document.

## Section One

Content for section one. This has multiple sentences to test paragraph handling.

### Subsection 1.1

Detailed content in subsection.

## Section Two

Content for section two with code:

` + "```go\n" + `func main() {
    fmt.Println("Hello")
}
` + "```\n"

	doc, err := parse.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	chunker := chunk.NewChunker(512) // 512 token limit
	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	// Should have at least summary + sections
	if len(chunks) < 4 {
		t.Errorf("Chunk() = %d chunks, want at least 4", len(chunks))
	}

	// First chunk should be summary
	if chunks[0].Level != "summary" {
		t.Errorf("chunks[0].Level = %q, want %q", chunks[0].Level, "summary")
	}

	// Summary should have title
	if chunks[0].Title != "Main Title" {
		t.Errorf("chunks[0].Title = %q, want %q", chunks[0].Title, "Main Title")
	}

	// Check that we have sections
	sectionCount := 0
	for _, c := range chunks {
		if c.Level == "section" {
			sectionCount++
		}
	}
	if sectionCount < 2 {
		t.Errorf("section count = %d, want at least 2", sectionCount)
	}
}

func TestChunker_TokenCounting(t *testing.T) {
	t.Parallel()

	// A simple approximation: ~4 chars per token
	chunker := chunk.NewChunker(100)

	text := "This is a test sentence with about twenty tokens or so in it."
	tokens := chunker.CountTokens(text)

	// Should be roughly 15-20 tokens
	if tokens < 10 || tokens > 30 {
		t.Errorf("CountTokens() = %d, expected ~15", tokens)
	}
}

func TestChunker_LargeSection(t *testing.T) {
	t.Parallel()

	// Create a document with a very large section that should be split
	input := `# Title

` + "## Large Section\n\n" + generateLargeContent(2000) // ~500 tokens

	doc, err := parse.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	chunker := chunk.NewChunker(200) // Small limit to force splitting
	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	// Large section should be split into paragraphs
	paragraphCount := 0
	for _, c := range chunks {
		if c.Level == "paragraph" {
			paragraphCount++
		}
	}
	if paragraphCount == 0 {
		t.Error("expected large section to be split into paragraphs")
	}

	// All chunks should be under token limit (with some tolerance)
	for i, c := range chunks {
		if c.TokenCount > 250 { // Allow some tolerance
			t.Errorf("chunks[%d].TokenCount = %d, exceeds limit", i, c.TokenCount)
		}
	}
}

func TestChunker_Breadcrumbs(t *testing.T) {
	t.Parallel()

	input := `# Main Doc

## Section A

### Subsection A1

Content here.
`

	doc, err := parse.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	chunker := chunk.NewChunker(512)
	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk() error = %v", err)
	}

	// Find the subsection chunk
	var subsectionChunk *chunk.Chunk
	for i := range chunks {
		if chunks[i].Title == "Subsection A1" {
			subsectionChunk = &chunks[i]
			break
		}
	}

	if subsectionChunk == nil {
		t.Fatal("could not find Subsection A1 chunk")
	}

	// Breadcrumbs should include parent headings
	if len(subsectionChunk.Breadcrumbs) < 2 {
		t.Errorf("Breadcrumbs = %v, want at least 2 entries", subsectionChunk.Breadcrumbs)
	}
}

func generateLargeContent(words int) string {
	content := ""
	for i := 0; i < words; i++ {
		content += "word "
		if i%20 == 19 {
			content += "\n\n" // Paragraph breaks
		}
	}
	return content
}
