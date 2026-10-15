// Check where a library's files live in its repository, and that library paths and repository paths map back and
// forth.

package librarytree_test

import (
	"slices"
	"testing"

	"github.com/fabricahq/code-rules/internal/librarytree"
)

// TestRepositoryPath_PlacesLibraryFilesInTheSyllabusDirectory covers rules, shared assets, change notes, the
// manifest, and files outside the Syllabus directory, such as license files.
func TestRepositoryPath_PlacesLibraryFilesInTheSyllabusDirectory(t *testing.T) {
	for file, want := range map[string]string{
		"config.yaml":                           ".syllabus/config.yaml",
		"techs/go/errors.md":                    ".syllabus/rules/techs/go/errors.md",
		"techs/go/_group.yaml":                  ".syllabus/rules/techs/go/_group.yaml",
		"techs/go/assets/errors/diagram.png":    ".syllabus/rules/techs/go/assets/errors/diagram.png",
		"practices/testing":                     ".syllabus/rules/practices/testing",
		"techs":                                 ".syllabus/rules/techs",
		"assets/retry-lifecycle.svg":            ".syllabus/assets/retry-lifecycle.svg",
		"assets":                                ".syllabus/assets",
		"changes/2026-10-01-errors-7f3a9c.yaml": ".syllabus/changes/2026-10-01-errors-7f3a9c.yaml",
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
// files in the Syllabus directory, and repository-root paths that library paths of the library's own files hide.
func TestLibraryPath_LeavesOutFilesTheLibraryDoesNotPublish(t *testing.T) {
	for _, file := range []string{
		".syllabus",
		".syllabus/vendor/team/config.yaml",
		".syllabus/generated/RULES.md",
		".syllabus/README.md",
		".syllabus/rules/README.md",
		".syllabus/rules/assets/diagram.svg",
		".syllabus/skills/review/SKILL.md",
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

// TestUpgrade_ReadsATreeFromBeforeTheRename spells the Syllabus directory of a tree that keeps it under the name Code
// Rules used as .syllabus/, and leaves a tree that already has .syllabus/ alone.
func TestUpgrade_ReadsATreeFromBeforeTheRename(t *testing.T) {
	for _, test := range []struct {
		paths  []string
		legacy bool
	}{
		{[]string{".code-rules/config.yaml", ".code-rules/rules/techs/go/errors.md", "LICENSE.md"}, true},
		{[]string{".syllabus/config.yaml", ".code-rules/config.yaml"}, false},
		{[]string{".syllabus/config.yaml"}, false},
		{[]string{"rule-library.yaml", "techs/go/errors.md"}, false},
		{[]string{".code-rules-lock/owner.json"}, false},
	} {
		if got := librarytree.UsesLegacyDirectory(slices.Values(test.paths)); got != test.legacy {
			t.Errorf("UsesLegacyDirectory(%q) = %v; want %v", test.paths, got, test.legacy)
		}
	}
	for file, want := range map[string]string{
		".code-rules/config.yaml":              ".syllabus/config.yaml",
		".code-rules/rules/techs/go/errors.md": ".syllabus/rules/techs/go/errors.md",
		".code-rules":                          ".code-rules",
		".code-rules-lock/owner.json":          ".code-rules-lock/owner.json",
		"LICENSE.md":                           "LICENSE.md",
	} {
		if got := librarytree.Upgrade(file); got != want {
			t.Errorf("Upgrade(%q) = %q; want %q", file, got, want)
		}
	}
}
