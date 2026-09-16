// Run the catalog walkthrough against disposable on-disk fixtures, never arbitrary user directories.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

// loadLibraryFixture writes a bounded request into a new directory and removes it after reading.
// All paths and link targets stay within that fixture. Callers cannot choose a host directory.
func loadLibraryFixture(input libraryFixture) (result any, err error) {
	selection, err := rules.ParseGroupSelection(input.Groups, "groups")
	if err != nil {
		return nil, err
	}
	for path := range input.Files {
		if !fixturePath(path) {
			return nil, &rules.ValidationError{Location: "files", Problem: "fixture paths must be contained relative paths"}
		}
	}
	for path, target := range input.Links {
		if !fixturePath(path) || !fixturePath(target) {
			return nil, &rules.ValidationError{Location: "links", Problem: "fixture links must use contained relative paths"}
		}
	}
	directory, err := os.MkdirTemp("", "code-rules-lab-library-")
	if err != nil {
		return nil, fmt.Errorf("create library fixture: %v", err)
	}
	// Remove only the directory created by this invocation, including after a failed load.
	defer func() {
		if cleanupErr := os.RemoveAll(directory); cleanupErr != nil {
			result = nil
			err = fmt.Errorf("remove library fixture: %v (load result: %v)", cleanupErr, err)
		}
	}()
	for path, text := range input.Files {
		target := filepath.Join(directory, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return nil, fmt.Errorf("create fixture directory: %v", err)
		}
		if err := os.WriteFile(target, []byte(text), 0600); err != nil {
			return nil, fmt.Errorf("write fixture file: %v", err)
		}
	}
	// Links are created after regular files so writes cannot follow a fixture-supplied link.
	for path, target := range input.Links {
		link := filepath.Join(directory, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(link), 0700); err != nil {
			return nil, fmt.Errorf("create fixture link directory: %v", err)
		}
		if err := os.Symlink(filepath.Join(directory, filepath.FromSlash(target)), link); err != nil {
			return nil, fmt.Errorf("create fixture link: %v", err)
		}
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open library fixture: %v", err)
	}
	defer root.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	catalog, err := library.Load(ctx, root, input.Source, selection)
	if err != nil {
		return nil, err
	}
	// Expose readable fixture text instead of base64-encoding Go's byte slices in the UI.
	files := make(map[string]string, len(catalog.Files))
	for path, data := range catalog.Files {
		files[path] = string(data)
	}
	return struct {
		Groups    []library.Group            `json:"groups"`
		Licenses  []rules.LicenseDeclaration `json:"licenses"`
		FilesRead map[string]string          `json:"filesRead"`
	}{Groups: catalog.Groups, Licenses: catalog.Licenses, FilesRead: files}, nil
}

// fixturePath limits the adapter's writes to simple portable relative names.
func fixturePath(path string) bool {
	return path != "." && fs.ValidPath(path) && !strings.ContainsAny(path, "\\:\x00")
}
