// Package git provides functionality for fetching and managing git repositories.
package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Fetcher clones and updates git repositories.
type Fetcher struct {
	cacheDir string
}

// NewFetcher creates a new git fetcher with the given cache directory.
func NewFetcher(cacheDir string) *Fetcher {
	return &Fetcher{cacheDir: cacheDir}
}

// Fetch clones or updates a repository and returns the local path.
func (f *Fetcher) Fetch(ctx context.Context, url string) (string, error) {
	localPath := filepath.Join(f.cacheDir, URLToPath(url))

	// Check if already cloned
	if _, err := os.Stat(filepath.Join(localPath, ".git")); err == nil {
		// Repository exists, try to pull
		repo, err := git.PlainOpen(localPath)
		if err != nil {
			return "", fmt.Errorf("open existing repo: %w", err)
		}

		wt, err := repo.Worktree()
		if err != nil {
			return "", fmt.Errorf("get worktree: %w", err)
		}

		err = wt.PullContext(ctx, &git.PullOptions{
			Auth: &http.BasicAuth{}, // Anonymous access
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			// Pull failed, but we still have the repo - continue with what we have
			// This handles cases like detached HEAD, etc.
		}

		return localPath, nil
	}

	// Clone the repository
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	_, err := git.PlainCloneContext(ctx, localPath, false, &git.CloneOptions{
		URL:      url,
		Depth:    1, // Shallow clone for speed
		Progress: nil,
	})
	if err != nil {
		return "", fmt.Errorf("clone repository: %w", err)
	}

	return localPath, nil
}

// ListFiles returns files matching the given glob patterns.
func (f *Fetcher) ListFiles(rootPath string, patterns []string) ([]string, error) {
	var matches []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		// Handle ** patterns by walking the directory
		if strings.Contains(pattern, "**") {
			err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}

				relPath, err := filepath.Rel(rootPath, path)
				if err != nil {
					return err
				}

				// Convert ** pattern to work with filepath.Match
				// **/*.md should match any .md file at any depth
				ext := filepath.Ext(pattern)
				if ext != "" && filepath.Ext(relPath) == ext {
					if !seen[relPath] {
						seen[relPath] = true
						matches = append(matches, relPath)
					}
				}

				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("walk directory: %w", err)
			}
		} else {
			// Simple glob pattern
			globPath := filepath.Join(rootPath, pattern)
			globMatches, err := filepath.Glob(globPath)
			if err != nil {
				return nil, fmt.Errorf("glob pattern %q: %w", pattern, err)
			}

			for _, match := range globMatches {
				relPath, err := filepath.Rel(rootPath, match)
				if err != nil {
					continue
				}
				if !seen[relPath] {
					seen[relPath] = true
					matches = append(matches, relPath)
				}
			}
		}
	}

	return matches, nil
}

// URLToPath converts a git URL to a filesystem-safe path.
func URLToPath(url string) string {
	// Remove protocol
	path := url
	path = strings.TrimPrefix(path, "https://")
	path = strings.TrimPrefix(path, "http://")
	path = strings.TrimPrefix(path, "git@")

	// Handle SSH URLs (git@github.com:user/repo.git)
	path = strings.Replace(path, ":", "/", 1)

	// Remove .git suffix
	path = strings.TrimSuffix(path, ".git")

	return path
}
