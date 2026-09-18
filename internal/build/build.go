// Package build generates complete rule guidance from validated inputs without filesystem or network access.
package build

import (
	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// Library supplies a validated selected catalog and its caller-verified commit.
// Snapshot freshness and Git authenticity belong to the import/snapshot boundary.
type Library struct {
	Catalog library.Catalog
	Commit  string
	Tag     string
}

// Output owns generated-root-relative file contents, including byte-exact license and notice copies.
type Output struct {
	Files map[string][]byte `json:"files"`
}

// Options provides the declared tool version and Markdown line limit and inline byte limit.
type Options struct {
	ToolVersion string
	// IndexMaxLines defaults to 750 when zero; negative limits are invalid.
	IndexMaxLines int
	// GroupInlineMaxBytes defaults to 8 KiB when nil; zero forces summary-only group pages.
	GroupInlineMaxBytes *int
}

// Generate applies adoption policies and renders the complete generated file tree.
// Configuration and catalogs must be validated; callers verify each library's source identity.
// It leaves inputs unchanged, preserves retained bytes, and returns no partial output on failure.
func Generate(config rules.Configuration, libraries map[string]Library, localFiles map[string][]byte, options Options) (Output, error) {
	resolved, err := resolve(config, libraries, localFiles)
	if err != nil {
		return Output{}, err
	}
	if options.IndexMaxLines == 0 {
		options.IndexMaxLines = defaultIndexMaxLines
	}
	return prepare(resolved, options)
}
