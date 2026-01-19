package langpack_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jamesainslie/grimoire/internal/ingest"
)

func TestGoLanguagePack(t *testing.T) {
	t.Parallel()

	// Get path to this test file's directory
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	path := filepath.Join(dir, "sources.yaml")

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

	if err := pack.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	// Should have at least 5 sources (Tier 1-3 git repos)
	if len(pack.Sources) < 5 {
		t.Errorf("Sources = %d, want at least 5", len(pack.Sources))
	}

	// Check for required sources
	requiredSources := []string{"go-wiki", "uber-style-guide", "learn-go-with-tests"}
	for _, name := range requiredSources {
		found := false
		for _, src := range pack.Sources {
			if src.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing required source: %s", name)
		}
	}
}
