// Package chunk provides hierarchical document chunking.
package chunk

import (
	"strings"

	"github.com/jamesainslie/grimoire/internal/parse"
)

// Chunk represents a piece of content from a document.
type Chunk struct {
	Level       string   // "summary", "section", "paragraph"
	Title       string
	Content     string
	TokenCount  int
	ParentIndex *int
	Breadcrumbs []string
}

// Chunker splits documents into hierarchical chunks.
type Chunker struct {
	maxTokens int
}

// NewChunker creates a new chunker with the given token limit.
func NewChunker(maxTokens int) *Chunker {
	return &Chunker{maxTokens: maxTokens}
}

// Chunk splits a parsed document into chunks.
func (c *Chunker) Chunk(doc *parse.Document) ([]Chunk, error) {
	var chunks []Chunk

	// Create summary chunk (document overview)
	summaryContent := doc.Title
	if len(doc.Sections) > 0 && doc.Sections[0].Content != "" {
		// Add intro paragraph if available
		intro := strings.TrimSpace(doc.Sections[0].Content)
		if intro != "" && len(intro) < 500 {
			summaryContent += "\n\n" + intro
		}
	}

	summaryChunk := Chunk{
		Level:       "summary",
		Title:       doc.Title,
		Content:     summaryContent,
		TokenCount:  c.CountTokens(summaryContent),
		Breadcrumbs: []string{doc.Title},
	}
	chunks = append(chunks, summaryChunk)
	summaryIdx := 0

	// Track heading hierarchy for breadcrumbs
	headingStack := []string{doc.Title}

	// Process each section
	for _, section := range doc.Sections {
		if section.Heading == nil {
			continue
		}

		// Update heading stack based on level
		level := section.Heading.Level
		for len(headingStack) > level {
			headingStack = headingStack[:len(headingStack)-1]
		}
		if len(headingStack) < level {
			headingStack = append(headingStack, section.Heading.Text)
		} else {
			headingStack[level-1] = section.Heading.Text
		}

		// Build breadcrumbs
		breadcrumbs := make([]string, len(headingStack))
		copy(breadcrumbs, headingStack)

		content := strings.TrimSpace(section.Content)
		tokens := c.CountTokens(content)

		if tokens <= c.maxTokens {
			// Section fits in one chunk
			sectionChunk := Chunk{
				Level:       "section",
				Title:       section.Heading.Text,
				Content:     content,
				TokenCount:  tokens,
				ParentIndex: &summaryIdx,
				Breadcrumbs: breadcrumbs,
			}
			chunks = append(chunks, sectionChunk)
		} else {
			// Section too large, split into paragraphs
			sectionIdx := len(chunks)

			// Add section header chunk
			headerContent := section.Heading.Text
			sectionChunk := Chunk{
				Level:       "section",
				Title:       section.Heading.Text,
				Content:     headerContent,
				TokenCount:  c.CountTokens(headerContent),
				ParentIndex: &summaryIdx,
				Breadcrumbs: breadcrumbs,
			}
			chunks = append(chunks, sectionChunk)

			// Split content into paragraphs
			paragraphs := splitIntoParagraphs(content)
			for _, para := range paragraphs {
				para = strings.TrimSpace(para)
				if para == "" {
					continue
				}

				paraTokens := c.CountTokens(para)
				if paraTokens > c.maxTokens {
					// Paragraph still too large, split by sentences
					sentences := splitIntoSentences(para)
					current := ""
					for _, sent := range sentences {
						test := current + " " + sent
						if c.CountTokens(test) > c.maxTokens && current != "" {
							paraChunk := Chunk{
								Level:       "paragraph",
								Title:       section.Heading.Text,
								Content:     strings.TrimSpace(current),
								TokenCount:  c.CountTokens(current),
								ParentIndex: &sectionIdx,
								Breadcrumbs: breadcrumbs,
							}
							chunks = append(chunks, paraChunk)
							current = sent
						} else {
							current = test
						}
					}
					if strings.TrimSpace(current) != "" {
						paraChunk := Chunk{
							Level:       "paragraph",
							Title:       section.Heading.Text,
							Content:     strings.TrimSpace(current),
							TokenCount:  c.CountTokens(current),
							ParentIndex: &sectionIdx,
							Breadcrumbs: breadcrumbs,
						}
						chunks = append(chunks, paraChunk)
					}
				} else {
					paraChunk := Chunk{
						Level:       "paragraph",
						Title:       section.Heading.Text,
						Content:     para,
						TokenCount:  paraTokens,
						ParentIndex: &sectionIdx,
						Breadcrumbs: breadcrumbs,
					}
					chunks = append(chunks, paraChunk)
				}
			}
		}
	}

	return chunks, nil
}

// CountTokens estimates the number of tokens in text.
// Uses a simple approximation of ~4 characters per token.
func (c *Chunker) CountTokens(text string) int {
	// Simple approximation: ~4 characters per token
	return (len(text) + 3) / 4
}

// splitIntoParagraphs splits text on double newlines.
func splitIntoParagraphs(text string) []string {
	// Split on double newlines or more
	parts := strings.Split(text, "\n\n")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// splitIntoSentences splits text on sentence boundaries.
func splitIntoSentences(text string) []string {
	// Simple sentence splitting on . ! ?
	var sentences []string
	current := ""

	for _, r := range text {
		current += string(r)
		if r == '.' || r == '!' || r == '?' {
			sentences = append(sentences, strings.TrimSpace(current))
			current = ""
		}
	}

	if strings.TrimSpace(current) != "" {
		sentences = append(sentences, strings.TrimSpace(current))
	}

	return sentences
}
