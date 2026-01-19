package ingest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesainslie/grimoire/internal/ingest"
)

func TestLoadLanguagePack(t *testing.T) {
	t.Parallel()

	// Create a test manifest
	manifest := `language: go
display_name: Go
sources:
  - name: uber-style-guide
    type: git
    url: https://github.com/uber-go/guide
    paths:
      - style.md
    tier: 2
  - name: go-wiki
    type: git
    url: https://github.com/golang/go
    paths:
      - wiki/*.md
    tier: 1
  - name: dave-cheney-blog
    type: web
    url: https://dave.cheney.net
    patterns:
      - /practical-go/*
    tier: 4
`

	dir := t.TempDir()
	path := filepath.Join(dir, "sources.yaml")
	if err := os.WriteFile(path, []byte(manifest), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	pack, err := ingest.LoadLanguagePack(path)
	if err != nil {
		t.Fatalf("LoadLanguagePack() error = %v", err)
	}

	if pack.Language != "go" {
		t.Errorf("Language = %q, want %q", pack.Language, "go")
	}

	if pack.DisplayName != "Go" {
		t.Errorf("DisplayName = %q, want %q", pack.DisplayName, "Go")
	}

	if len(pack.Sources) != 3 {
		t.Errorf("Sources = %d, want 3", len(pack.Sources))
	}

	// Check first source
	if pack.Sources[0].Name != "uber-style-guide" {
		t.Errorf("Sources[0].Name = %q, want %q", pack.Sources[0].Name, "uber-style-guide")
	}
	if pack.Sources[0].Type != "git" {
		t.Errorf("Sources[0].Type = %q, want %q", pack.Sources[0].Type, "git")
	}
	if pack.Sources[0].Tier != 2 {
		t.Errorf("Sources[0].Tier = %d, want 2", pack.Sources[0].Tier)
	}
	if len(pack.Sources[0].Paths) != 1 {
		t.Errorf("Sources[0].Paths = %d, want 1", len(pack.Sources[0].Paths))
	}

	// Check web source
	if pack.Sources[2].Type != "web" {
		t.Errorf("Sources[2].Type = %q, want %q", pack.Sources[2].Type, "web")
	}
	if len(pack.Sources[2].Patterns) != 1 {
		t.Errorf("Sources[2].Patterns = %d, want 1", len(pack.Sources[2].Patterns))
	}
}

func TestLoadLanguagePack_InvalidYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(path, []byte("this is not: valid: yaml:"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := ingest.LoadLanguagePack(path)
	if err == nil {
		t.Error("LoadLanguagePack() expected error for invalid YAML")
	}
}

func TestLoadLanguagePack_MissingFile(t *testing.T) {
	t.Parallel()

	_, err := ingest.LoadLanguagePack("/nonexistent/path/sources.yaml")
	if err == nil {
		t.Error("LoadLanguagePack() expected error for missing file")
	}
}

func TestLanguagePack_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pack    ingest.LanguagePack
		wantErr bool
	}{
		{
			name: "valid pack",
			pack: ingest.LanguagePack{
				Language:    "go",
				DisplayName: "Go",
				Sources: []ingest.SourceDef{
					{Name: "test", Type: "git", URL: "https://example.com"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing language",
			pack: ingest.LanguagePack{
				DisplayName: "Go",
				Sources:     []ingest.SourceDef{{Name: "test", Type: "git", URL: "https://example.com"}},
			},
			wantErr: true,
		},
		{
			name: "missing display name",
			pack: ingest.LanguagePack{
				Language: "go",
				Sources:  []ingest.SourceDef{{Name: "test", Type: "git", URL: "https://example.com"}},
			},
			wantErr: true,
		},
		{
			name: "no sources",
			pack: ingest.LanguagePack{
				Language:    "go",
				DisplayName: "Go",
				Sources:     []ingest.SourceDef{},
			},
			wantErr: true,
		},
		{
			name: "invalid source type",
			pack: ingest.LanguagePack{
				Language:    "go",
				DisplayName: "Go",
				Sources:     []ingest.SourceDef{{Name: "test", Type: "invalid", URL: "https://example.com"}},
			},
			wantErr: true,
		},
		{
			name: "source missing URL",
			pack: ingest.LanguagePack{
				Language:    "go",
				DisplayName: "Go",
				Sources:     []ingest.SourceDef{{Name: "test", Type: "git"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pack.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
