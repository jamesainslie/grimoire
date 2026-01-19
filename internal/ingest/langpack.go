// Package ingest provides functionality for loading and processing documentation sources.
package ingest

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LanguagePack defines the sources for a programming language.
type LanguagePack struct {
	Language    string      `yaml:"language"`
	DisplayName string      `yaml:"display_name"`
	Sources     []SourceDef `yaml:"sources"`
}

// SourceDef defines a documentation source.
type SourceDef struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"` // "git" or "web"
	URL      string   `yaml:"url"`
	Paths    []string `yaml:"paths,omitempty"`    // For git: paths within repo
	Patterns []string `yaml:"patterns,omitempty"` // For web: URL patterns
	Tier     int      `yaml:"tier,omitempty"`     // Priority tier (1=official, 2=industry, etc.)
}

// LoadLanguagePack loads a language pack from a YAML file.
func LoadLanguagePack(path string) (*LanguagePack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var pack LanguagePack
	if err := yaml.Unmarshal(data, &pack); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	return &pack, nil
}

// Validate checks that the language pack is valid.
func (p *LanguagePack) Validate() error {
	if p.Language == "" {
		return errors.New("language is required")
	}
	if p.DisplayName == "" {
		return errors.New("display_name is required")
	}
	if len(p.Sources) == 0 {
		return errors.New("at least one source is required")
	}

	for i, src := range p.Sources {
		if err := src.Validate(); err != nil {
			return fmt.Errorf("source[%d] (%s): %w", i, src.Name, err)
		}
	}

	return nil
}

// Validate checks that the source definition is valid.
func (s *SourceDef) Validate() error {
	if s.Name == "" {
		return errors.New("name is required")
	}
	if s.Type != "git" && s.Type != "web" {
		return fmt.Errorf("invalid type %q (must be 'git' or 'web')", s.Type)
	}
	if s.URL == "" {
		return errors.New("url is required")
	}
	return nil
}
