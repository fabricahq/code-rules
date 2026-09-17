// Verify complete inventory bounds include unused assets and empty directories.

package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInventoryLimits checks exact bounds, aggregate bytes, entry counts, and cancellation.
func TestInventoryLimits(t *testing.T) {
	chunk := make([]byte, maxFileBytes)
	files := map[string][]byte{}
	for i := range 8 {
		files[string(rune('a'+i))] = chunk
	}
	if err := ValidateInventoryLimits(context.Background(), files, nil); err != nil {
		t.Fatal(err)
	}
	files["extra"] = []byte{1}
	if err := ValidateInventoryLimits(context.Background(), files, nil); err == nil {
		t.Fatal("aggregate limit ignored")
	}
	if err := ValidateInventoryLimits(context.Background(), map[string][]byte{"large": make([]byte, maxFileBytes+1)}, nil); err == nil {
		t.Fatal("file limit ignored")
	}
	if err := ValidateInventoryLimits(context.Background(), nil, make([]string, maxFiles)); err != nil {
		t.Fatal(err)
	}
	if err := ValidateInventoryLimits(context.Background(), map[string][]byte{"file": nil}, make([]string, maxFiles)); err == nil {
		t.Fatal("entry limit ignored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ValidateInventoryLimits(ctx, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

// countedInventorySource records actual file requests without changing bounded filesystem behavior.
type countedInventorySource struct {
	rootFiles
	reads []string
}

// ReadFile records each request before delegating to the bounded local reader.
func (s *countedInventorySource) ReadFile(name string) ([]byte, error) {
	s.reads = append(s.reads, name)
	return s.rootFiles.ReadFile(name)
}

// TestInventoryStopsAtOversizedAsset proves capture rejects early rather than loading later owned trees.
func TestInventoryStopsAtOversizedAsset(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"assets", "techs"} {
		if err := os.Mkdir(filepath.Join(directory, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, "rule-library.json"), []byte(`{"formatVersion":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(directory, "assets/large.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(256 * 1024 * 1024); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "techs/later"), []byte("must not be read"), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	source := &countedInventorySource{rootFiles: rootFiles{ctx: context.Background(), root: root}}
	inventory, err := readInventory(context.Background(), source)
	if err == nil || !strings.Contains(err.Error(), "limits") || inventory.Files != nil {
		t.Fatal(inventory, err)
	}
	if len(source.reads) != 2 || source.reads[1] != "assets/large.bin" {
		t.Fatal("capture continued past limit", source.reads)
	}
}
