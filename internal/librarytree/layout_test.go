// Check where a library's files live in its repository, and that library paths and repository paths map back and
// forth.

package librarytree_test

import (
	"testing"

	"github.com/fabricahq/code-rules/internal/librarytree"
)

// TestRepositoryPath_PlacesLibraryFilesInTheCodeRulesDirectory covers rules, shared assets, change notes, the
// manifest, and files outside the Code Rules directory, such as license files.
func TestRepositoryPath_PlacesLibraryFilesInTheCodeRulesDirectory(t *testing.T) {
	for file, want := range map[string]string{
		"config.yaml":                           ".code-rules/config.yaml",
		"techs/go/errors.md":                    ".code-rules/rules/techs/go/errors.md",
		"techs/go/_group.yaml":                  ".code-rules/rules/techs/go/_group.yaml",
		"techs/go/assets/errors/diagram.png":    ".code-rules/rules/techs/go/assets/errors/diagram.png",
		"practices/testing":                     ".code-rules/rules/practices/testing",
		"techs":                                 ".code-rules/rules/techs",
		"assets/retry-lifecycle.svg":            ".code-rules/assets/retry-lifecycle.svg",
		"assets":                                ".code-rules/assets",
		"changes/2026-10-01-errors-7f3a9c.yaml": ".code-rules/changes/2026-10-01-errors-7f3a9c.yaml",
		"LICENSE.md":                            "LICENSE.md",
		"docs/NOTICE.md":                        "docs/NOTICE.md",
	} {
		if got := librarytree.RepositoryPath(file); got != want {
			t.Errorf("RepositoryPath(%q) = %q; want %q", file, got, want)
		}
		if back, ok := librarytree.LibraryPath(want); !ok || back != file {
			t.Errorf("LibraryPath(%q) = %q, %v; want %q", want, back, ok, file)
		}
	}
}

// TestLibraryPath_LeavesOutFilesTheLibraryDoesNotPublish covers the project's vendor/ and generated/ folders, other
// files in the Code Rules directory, and repository-root paths that library paths of the library's own files hide.
func TestLibraryPath_LeavesOutFilesTheLibraryDoesNotPublish(t *testing.T) {
	for _, file := range []string{
		".code-rules",
		".code-rules/vendor/team/config.yaml",
		".code-rules/generated/RULES.md",
		".code-rules/README.md",
		".code-rules/rules/README.md",
		".code-rules/rules/assets/diagram.svg",
		".code-rules/skills/review/SKILL.md",
		"config.yaml",
		"techs/go/errors.md",
		"practices/testing/_group.yaml",
		"assets/retry-lifecycle.svg",
		"changes/2026-10-01-errors-7f3a9c.yaml",
	} {
		if got, ok := librarytree.LibraryPath(file); ok {
			t.Errorf("LibraryPath(%q) = %q; want no library path", file, got)
		}
	}
}
