package parse_test

import (
	"testing"

	"github.com/jamesainslie/grimoire/internal/parse"
)

func TestParse(t *testing.T) {
	t.Parallel()

	input := `# Main Title

This is the introduction.

## Section One

Content for section one.

### Subsection

More detailed content.

## Section Two

Content for section two.

` + "```go\n" + `func main() {
    fmt.Println("Hello")
}
` + "```\n"

	doc, err := parse.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check title extraction
	if doc.Title != "Main Title" {
		t.Errorf("Parse() Title = %q, want %q", doc.Title, "Main Title")
	}

	// Check heading count
	if len(doc.Headings) != 4 {
		t.Errorf("Parse() Headings = %d, want 4", len(doc.Headings))
	}

	// Check heading levels
	expectedLevels := []int{1, 2, 3, 2}
	for i, h := range doc.Headings {
		if h.Level != expectedLevels[i] {
			t.Errorf("Heading[%d].Level = %d, want %d", i, h.Level, expectedLevels[i])
		}
	}

	// Check sections
	if len(doc.Sections) != 4 {
		t.Errorf("Parse() Sections = %d, want 4", len(doc.Sections))
	}
}

func TestParse_CodeBlocks(t *testing.T) {
	t.Parallel()

	input := `# Code Examples

Here's some Go code:

` + "```go\n" + `package main

func main() {}
` + "```\n" + `

And some Python:

` + "```python\n" + `def main():
    pass
` + "```\n"

	doc, err := parse.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Should have code blocks identified
	if len(doc.CodeBlocks) != 2 {
		t.Errorf("Parse() CodeBlocks = %d, want 2", len(doc.CodeBlocks))
	}

	if len(doc.CodeBlocks) >= 2 {
		if doc.CodeBlocks[0].Language != "go" {
			t.Errorf("CodeBlocks[0].Language = %q, want %q", doc.CodeBlocks[0].Language, "go")
		}
		if doc.CodeBlocks[1].Language != "python" {
			t.Errorf("CodeBlocks[1].Language = %q, want %q", doc.CodeBlocks[1].Language, "python")
		}
	}
}

func TestParse_FrontMatter(t *testing.T) {
	t.Parallel()

	input := `---
title: My Document
author: Test Author
---

# Actual Title

Content here.
`

	doc, err := parse.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Title should come from frontmatter
	if doc.Title != "My Document" {
		t.Errorf("Parse() Title = %q, want %q", doc.Title, "My Document")
	}

	// Author should be in frontmatter
	if doc.FrontMatter["author"] != "Test Author" {
		t.Errorf("FrontMatter[author] = %q, want %q", doc.FrontMatter["author"], "Test Author")
	}
}

func TestParse_EmptyDocument(t *testing.T) {
	t.Parallel()

	doc, err := parse.Parse([]byte(""))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if doc.Title != "" {
		t.Errorf("Parse() Title = %q, want empty", doc.Title)
	}
}

func TestParse_NoHeadings(t *testing.T) {
	t.Parallel()

	input := `Just some plain text without any headings.

More text here.`

	doc, err := parse.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(doc.Headings) != 0 {
		t.Errorf("Parse() Headings = %d, want 0", len(doc.Headings))
	}

	// Should still have content in sections
	if len(doc.Sections) == 0 {
		t.Error("Parse() should have at least one section for content")
	}
}
