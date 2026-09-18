// Run the catalog walkthrough against disposable on-disk fixtures, never arbitrary user directories.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// libraryFixture describes fixture-owned text files and relative symlink targets.
type libraryFixture struct {
	Files  map[string]string `json:"files"`
	Links  map[string]string `json:"links,omitempty"`
	Groups json.RawMessage   `json:"groups"`
	Source string            `json:"source"`
}

// readLibraryFixture writes a bounded request into a new directory and removes it after reading.
// All paths and link targets stay within that fixture. Callers cannot choose a host directory.
func readLibraryFixture(input libraryFixture) (result library.Catalog, err error) {
	selection, err := rules.ParseGroupSelection(input.Groups, "groups")
	if err != nil {
		return library.Catalog{}, err
	}
	for path := range input.Files {
		if !fixturePath(path) {
			return library.Catalog{}, &rules.ValidationError{Location: "files", Problem: "fixture paths must be contained relative paths"}
		}
	}
	for path, target := range input.Links {
		if !fixturePath(path) || !fixturePath(target) {
			return library.Catalog{}, &rules.ValidationError{Location: "links", Problem: "fixture links must use contained relative paths"}
		}
	}
	directory, err := os.MkdirTemp("", "code-rules-lab-library-")
	if err != nil {
		return library.Catalog{}, fmt.Errorf("create library fixture: %v", err)
	}
	// Remove only the directory created by this invocation, including after a failed load.
	defer func() {
		if cleanupErr := os.RemoveAll(directory); cleanupErr != nil {
			result = library.Catalog{}
			err = fmt.Errorf("remove library fixture: %v (load result: %v)", cleanupErr, err)
		}
	}()
	for path, text := range input.Files {
		target := filepath.Join(directory, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return library.Catalog{}, fmt.Errorf("create fixture directory: %v", err)
		}
		if err := os.WriteFile(target, []byte(text), 0600); err != nil {
			return library.Catalog{}, fmt.Errorf("write fixture file: %v", err)
		}
	}
	// Links are created after regular files so writes cannot follow a fixture-supplied link.
	for path, target := range input.Links {
		link := filepath.Join(directory, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(link), 0700); err != nil {
			return library.Catalog{}, fmt.Errorf("create fixture link directory: %v", err)
		}
		if err := os.Symlink(filepath.Join(directory, filepath.FromSlash(target)), link); err != nil {
			return library.Catalog{}, fmt.Errorf("create fixture link: %v", err)
		}
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return library.Catalog{}, fmt.Errorf("open library fixture: %v", err)
	}
	defer root.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	catalog, err := library.Load(ctx, root, input.Source, selection)
	if err != nil {
		return library.Catalog{}, err
	}
	return catalog, nil
}

// loadLibraryFixture projects the native catalog into readable JSON for the walkthrough.
func loadLibraryFixture(input libraryFixture) (any, error) {
	catalog, err := readLibraryFixture(input)
	if err != nil {
		return nil, err
	}
	// Expose readable fixture text instead of base64-encoding Go's byte slices in the UI.
	files := make(map[string]string, len(catalog.SupportingFiles))
	for path, data := range catalog.SupportingFiles {
		files[path] = string(data)
	}
	return struct {
		Groups          []library.Group           `json:"groups"`
		License         *rules.LicenseDeclaration `json:"license"`
		SupportingFiles map[string]string         `json:"supportingFiles"`
		FilesRead       []string                  `json:"filesRead"`
	}{Groups: catalog.Groups, License: catalog.License, SupportingFiles: files, FilesRead: catalog.Paths()}, nil
}

// fixturePath limits the adapter's writes to simple portable relative names.
func fixturePath(path string) bool {
	return path != "." && fs.ValidPath(path) && !strings.ContainsAny(path, "\\:\x00")
}

// validateFixtureText rejects null map values that encoding/json would coerce into empty strings.
func validateFixtureText(input json.RawMessage) error {
	var fields struct {
		Files map[string]json.RawMessage
		Links map[string]json.RawMessage
	}
	if err := json.Unmarshal(input, &fields); err != nil {
		return err
	}
	for _, entries := range []map[string]json.RawMessage{fields.Files, fields.Links} {
		for _, key := range slices.Sorted(maps.Keys(entries)) {
			if bytes.Equal(bytes.TrimSpace(entries[key]), []byte("null")) {
				return fmt.Errorf("fixture file contents and link targets must be strings, not null")
			}
		}
	}
	return nil
}
