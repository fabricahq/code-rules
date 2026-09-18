// Verify that dependency discovery observes cancellation even when no file read remains.

package library

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
	"testing"
)

// TestSupportingLinksCanceledWithoutReads covers link-only work after catalog files are already in memory.
func TestSupportingLinksCanceledWithoutReads(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, document := range []string{"plain text", "[external](https://example.com)", "[self]()"} {
		r := reader{ctx: ctx, files: map[string][]byte{"techs/go/r.md": []byte(document)}}
		if err := r.supportingLinks(nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("%q: %v", document, err)
		}
	}
}

// TestAssetLookupOperationalErrors preserves filesystem error identity instead of reporting invalid content.
func TestAssetLookupOperationalErrors(t *testing.T) {
	directory := t.TempDir()
	assetDirectory := "techs/go/assets"
	if err := os.MkdirAll(filepath.Join(directory, assetDirectory, "errors"), 0700); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(directory, assetDirectory))
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	r := reader{ctx: context.Background(), input: rootFiles{ctx: context.Background(), root: root}, directories: map[string][]fs.DirEntry{assetDirectory: entries}}
	for name, err := range map[string]error{"link": r.linkExists("assets/guide.md"), "owner": r.ownedAssets(assetDirectory, nil)} {
		var validation *rules.ValidationError
		if !errors.Is(err, os.ErrClosed) || errors.As(err, &validation) {
			t.Fatalf("%s: lost operational error: %v", name, err)
		}
		if !strings.Contains(err.Error(), "assets") && !strings.Contains(err.Error(), "techs/go/errors.md") {
			t.Fatalf("%s: lost path: %v", name, err)
		}
	}
}
