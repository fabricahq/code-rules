// Exercise complete asset directories, recursive dependencies, and unsafe targets on real filesystems.

package library_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// TestLoadAssets retains complete owned/shared trees without adopting referenced rules.
func TestLoadAssets(t *testing.T) {
	files := validFiles()
	files["techs/go/errors.md"] += "\n![image](assets/errors/image.bin) [shared](/assets/guide.md) [other](../rust/other.md)\n"
	files["techs/go/assets/errors/image.bin"] = "\x00\xff\r\n"
	files["techs/go/assets/errors/unused.bin"] = "unreferenced owned"
	files["assets/guide.md"] = "[cycle](guide.md) [next](next.md)"
	files["assets/next.md"] = "[cycle](guide.md)"
	files["assets/unreferenced.bin"] = "shared companion"
	files["techs/rust/other.md"] = "not adopted or parsed"
	_, root := fixture(t, files)
	got, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Groups: []string{"techs/go"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Groups) != 1 || len(got.Groups[0].Rules) != 1 {
		t.Fatalf("adopted another rule: %+v", got.Groups)
	}
	for _, file := range []string{"techs/go/assets/errors/image.bin", "techs/go/assets/errors/unused.bin", "assets/guide.md", "assets/next.md", "assets/unreferenced.bin"} {
		if string(got.SupportingFiles[file]) != files[file] {
			t.Errorf("lost bytes: %s", file)
		}
	}
	if _, ok := got.SupportingFiles["techs/rust/other.md"]; ok {
		t.Fatal("retained unselected rule")
	}
}

// TestLoadRejectsAssetFailures returns no partial catalog for invalid dependency closures.
func TestLoadRejectsAssetFailures(t *testing.T) {
	for _, test := range []struct{ name, link, file, content string }{
		{"missing", "assets/errors/missing.png", "", ""},
		{"escape", "../../../outside", "", ""},
		{"other owner", "assets/another/file.png", "techs/go/assets/another/file.png", "x"},
		{"arbitrary supporting", "data.txt", "techs/go/assets/errors/a.bin", "x"},
		{"invalid Markdown", "/assets/guide.md", "assets/guide.md", "\xff"},
		{"lfs", "assets/errors/image.png", "techs/go/assets/errors/image.png", "version https://git-lfs.github.com/spec/v1\noid sha256:123"},
		{"recursive escape", "/assets/guide.md", "assets/guide.md", "[bad](../../outside)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := validFiles()
			files["techs/go/errors.md"] += "\n[x](" + test.link + ")\n"
			if test.file != "" {
				files[test.file] = test.content
			}
			_, root := fixture(t, files)
			got, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Groups: []string{"techs/go"}})
			if err == nil || got.Groups != nil {
				t.Fatalf("accepted unsafe assets: %+v, %v", got, err)
			}
		})
	}
}

// TestLoadSkipsUnusedSharedAndRejectsLinkedAssets keeps unrelated files out while refusing selected symlinks.
func TestLoadSkipsUnusedSharedAndRejectsLinkedAssets(t *testing.T) {
	files := validFiles()
	files["assets/bad.md"] = "[escape](../../outside)"
	directory, root := fixture(t, files)
	if _, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Pattern: "*"}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(directory, "techs/go/assets/errors"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(directory, "assets/bad.md"), filepath.Join(directory, "techs/go/assets/errors/link.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := library.Load(context.Background(), root, "team", rules.GroupSelection{Pattern: "*"}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink: %v", err)
	}
}
