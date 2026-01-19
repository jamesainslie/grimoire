package git_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesainslie/grimoire/internal/source/git"
)

func TestFetcher_Fetch(t *testing.T) {
	// Use a small, stable public repo for testing
	// This test requires network access
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cacheDir := t.TempDir()
	fetcher := git.NewFetcher(cacheDir)

	ctx := context.Background()

	// Fetch a small public repo
	localPath, err := fetcher.Fetch(ctx, "https://github.com/golang/example")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Verify the repo was cloned
	if _, err := os.Stat(filepath.Join(localPath, ".git")); os.IsNotExist(err) {
		t.Error("Fetch() did not create .git directory")
	}

	// Verify we can find Go files (use ** for recursive search)
	files, err := fetcher.ListFiles(localPath, []string{"**/*.go"})
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	if len(files) == 0 {
		t.Error("ListFiles() found no .go files")
	}
}

func TestFetcher_FetchTwice(t *testing.T) {
	// Second fetch should be faster (pull instead of clone)
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cacheDir := t.TempDir()
	fetcher := git.NewFetcher(cacheDir)

	ctx := context.Background()

	// First fetch (clone)
	path1, err := fetcher.Fetch(ctx, "https://github.com/golang/example")
	if err != nil {
		t.Fatalf("First Fetch() error = %v", err)
	}

	// Second fetch (should use existing repo)
	path2, err := fetcher.Fetch(ctx, "https://github.com/golang/example")
	if err != nil {
		t.Fatalf("Second Fetch() error = %v", err)
	}

	if path1 != path2 {
		t.Errorf("Fetch() returned different paths: %s vs %s", path1, path2)
	}
}

func TestFetcher_ListFiles(t *testing.T) {
	t.Parallel()

	// Create a temp directory with test files
	dir := t.TempDir()

	// Create test file structure
	files := []string{
		"README.md",
		"docs/guide.md",
		"docs/api.md",
		"src/main.go",
		"src/util.go",
	}
	for _, f := range files {
		path := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	fetcher := git.NewFetcher(t.TempDir())

	tests := []struct {
		name     string
		patterns []string
		want     int
	}{
		{
			name:     "all markdown files",
			patterns: []string{"**/*.md"},
			want:     3,
		},
		{
			name:     "all go files",
			patterns: []string{"**/*.go"},
			want:     2,
		},
		{
			name:     "docs only",
			patterns: []string{"docs/*.md"},
			want:     2,
		},
		{
			name:     "multiple patterns",
			patterns: []string{"*.md", "docs/*.md"},
			want:     3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := fetcher.ListFiles(dir, tt.patterns)
			if err != nil {
				t.Fatalf("ListFiles() error = %v", err)
			}
			if len(files) != tt.want {
				t.Errorf("ListFiles() = %d files, want %d (got: %v)", len(files), tt.want, files)
			}
		})
	}
}

func TestFetcher_URLToPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		want string
	}{
		{
			url:  "https://github.com/uber-go/guide",
			want: "github.com/uber-go/guide",
		},
		{
			url:  "https://github.com/golang/go.git",
			want: "github.com/golang/go",
		},
		{
			url:  "git@github.com:user/repo.git",
			want: "github.com/user/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := git.URLToPath(tt.url)
			if got != tt.want {
				t.Errorf("URLToPath(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
