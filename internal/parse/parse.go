// Package parse provides Markdown parsing with heading extraction.
package parse

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Document represents a parsed Markdown document.
type Document struct {
	Title       string
	Headings    []Heading
	Sections    []Section
	CodeBlocks  []CodeBlock
	FrontMatter map[string]string
	RawContent  []byte
}

// Heading represents a heading in the document.
type Heading struct {
	Level  int
	Text   string
	Anchor string
}

// Section represents a section of content under a heading.
type Section struct {
	Heading  *Heading
	Content  string
	Children []Section
}

// CodeBlock represents a fenced code block.
type CodeBlock struct {
	Language string
	Content  string
}

// Parse parses Markdown content and extracts structure.
func Parse(content []byte) (*Document, error) {
	doc := &Document{
		RawContent:  content,
		FrontMatter: make(map[string]string),
	}

	// Extract front matter if present
	content = extractFrontMatter(content, doc)

	// Parse with goldmark
	md := goldmark.New()
	reader := text.NewReader(content)
	root := md.Parser().Parse(reader)

	// Walk the AST to extract structure
	// Sections are stored flat in doc.Sections (one per heading)
	var currentSection *Section

	ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n := node.(type) {
		case *ast.Heading:
			heading := Heading{
				Level:  n.Level,
				Text:   extractText(n, content),
				Anchor: generateAnchor(extractText(n, content)),
			}
			doc.Headings = append(doc.Headings, heading)

			// Set title from first h1 if not set from frontmatter
			if doc.Title == "" && n.Level == 1 {
				doc.Title = heading.Text
			}

			// Create a new section for this heading (flat list)
			doc.Sections = append(doc.Sections, Section{
				Heading: &doc.Headings[len(doc.Headings)-1],
			})
			currentSection = &doc.Sections[len(doc.Sections)-1]

		case *ast.FencedCodeBlock:
			lang := string(n.Language(content))
			codeContent := extractCodeContent(n, content)
			doc.CodeBlocks = append(doc.CodeBlocks, CodeBlock{
				Language: lang,
				Content:  codeContent,
			})

			// Add to current section's content
			if currentSection != nil {
				currentSection.Content += "\n```" + lang + "\n" + codeContent + "```\n"
			}

		case *ast.Paragraph:
			// Add paragraph content to current section
			if currentSection != nil {
				currentSection.Content += extractText(n, content) + "\n\n"
			}
		}

		return ast.WalkContinue, nil
	})

	// If no sections were created but there's content, create a root section
	if len(doc.Sections) == 0 && len(content) > 0 {
		doc.Sections = append(doc.Sections, Section{
			Content: string(content),
		})
	}

	return doc, nil
}

// extractFrontMatter extracts YAML front matter from content.
func extractFrontMatter(content []byte, doc *Document) []byte {
	if !bytes.HasPrefix(content, []byte("---\n")) {
		return content
	}

	// Find closing ---
	end := bytes.Index(content[4:], []byte("\n---"))
	if end == -1 {
		return content
	}

	frontMatter := content[4 : 4+end]
	remaining := content[4+end+4:]

	// Parse simple key: value pairs
	lines := bytes.Split(frontMatter, []byte("\n"))
	for _, line := range lines {
		if idx := bytes.Index(line, []byte(":")); idx > 0 {
			key := strings.TrimSpace(string(line[:idx]))
			value := strings.TrimSpace(string(line[idx+1:]))
			doc.FrontMatter[key] = value

			// Use title from frontmatter
			if key == "title" {
				doc.Title = value
			}
		}
	}

	return remaining
}

// extractText extracts text content from a node.
func extractText(node ast.Node, source []byte) string {
	var buf bytes.Buffer
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*ast.Text); ok {
			buf.Write(textNode.Segment.Value(source))
		} else {
			buf.WriteString(extractText(child, source))
		}
	}
	return buf.String()
}

// extractCodeContent extracts content from a fenced code block.
func extractCodeContent(node *ast.FencedCodeBlock, source []byte) string {
	var buf bytes.Buffer
	lines := node.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		buf.Write(line.Value(source))
	}
	return buf.String()
}

// generateAnchor generates a URL-safe anchor from heading text.
func generateAnchor(text string) string {
	anchor := strings.ToLower(text)
	anchor = strings.ReplaceAll(anchor, " ", "-")
	// Remove non-alphanumeric except dashes
	var result strings.Builder
	for _, r := range anchor {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}
