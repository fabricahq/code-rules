// Check where a library's files live in its repository, and that library paths and repository paths map back and
// forth.

package librarytree_test

import (
	"slices"
	"testing"

	"github.com/fabricahq/trailguide/internal/librarytree"
)

// TestRepositoryPath_PlacesLibraryFilesInTheTrailGuideDirectory covers rules, shared assets, change notes, the
// manifest, and files outside the Trail Guide directory, such as license files.
func TestRepositoryPath_PlacesLibraryFilesInTheTrailGuideDirectory(t *testing.T) {
	for file, want := range map[string]string{
		"config.yaml":                           ".trailguide/config.yaml",
		"techs/go/errors.md":                    ".trailguide/rules/techs/go/errors.md",
		"techs/go/_group.yaml":                  ".trailguide/rules/techs/go/_group.yaml",
		"techs/go/assets/errors/diagram.png":    ".trailguide/rules/techs/go/assets/errors/diagram.png",
		"practices/testing":                     ".trailguide/rules/practices/testing",
		"techs":                                 ".trailguide/rules/techs",
		"assets/retry-lifecycle.svg":            ".trailguide/assets/retry-lifecycle.svg",
		"assets":                                ".trailguide/assets",
		"changes/2026-10-01-errors-7f3a9c.yaml": ".trailguide/changes/2026-10-01-errors-7f3a9c.yaml",
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
// files in the Trail Guide directory, and repository-root paths that library paths of the library's own files hide.
func TestLibraryPath_LeavesOutFilesTheLibraryDoesNotPublish(t *testing.T) {
	for _, file := range []string{
		".trailguide",
		".trailguide/vendor/team/config.yaml",
		".trailguide/generated/RULES.md",
		".trailguide/README.md",
		".trailguide/rules/README.md",
		".trailguide/rules/assets/diagram.svg",
		".trailguide/skills/review/SKILL.md",
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

// TestUpgrade_ReadsATreeFromBeforeTheRename spells the Trail Guide directory of a tree that keeps it under the name Code
// Rules used as .trailguide/, and leaves a tree that already has .trailguide/ alone.
func TestUpgrade_ReadsATreeFromBeforeTheRename(t *testing.T) {
	for _, test := range []struct {
		paths  []string
		legacy bool
	}{
		{[]string{".code-rules/config.yaml", ".code-rules/rules/techs/go/errors.md", "LICENSE.md"}, true},
		{[]string{".trailguide/config.yaml", ".code-rules/config.yaml"}, false},
		{[]string{".trailguide/config.yaml"}, false},
		{[]string{"rule-library.yaml", "techs/go/errors.md"}, false},
		{[]string{".code-rules-lock/owner.json"}, false},
	} {
		if got := librarytree.UsesLegacyDirectory(slices.Values(test.paths)); got != test.legacy {
			t.Errorf("UsesLegacyDirectory(%q) = %v; want %v", test.paths, got, test.legacy)
		}
	}
	for file, want := range map[string]string{
		".code-rules/config.yaml":              ".trailguide/config.yaml",
		".code-rules/rules/techs/go/errors.md": ".trailguide/rules/techs/go/errors.md",
		".code-rules":                          ".code-rules",
		".code-rules-lock/owner.json":          ".code-rules-lock/owner.json",
		"LICENSE.md":                           "LICENSE.md",
	} {
		if got := librarytree.Upgrade(file); got != want {
			t.Errorf("Upgrade(%q) = %q; want %q", file, got, want)
		}
	}
}

// TestSnapshotPath_MirrorsTheTrailGuideDirectory stores rules under rules/, beside shared assets and the manifest, and
// keeps files outside the Trail Guide directory, such as license files, at their repository paths.
func TestSnapshotPath_MirrorsTheTrailGuideDirectory(t *testing.T) {
	for file, want := range map[string]string{
		"config.yaml":                        "config.yaml",
		"techs/go/errors.md":                 "rules/techs/go/errors.md",
		"practices/testing/_group.yaml":      "rules/practices/testing/_group.yaml",
		"techs/go/assets/errors/diagram.png": "rules/techs/go/assets/errors/diagram.png",
		"assets/retry-lifecycle.svg":         "assets/retry-lifecycle.svg",
		"LICENSE.md":                         "LICENSE.md",
		"docs/NOTICE.md":                     "docs/NOTICE.md",
	} {
		if got := librarytree.SnapshotPath(file); got != want {
			t.Errorf("SnapshotPath(%q) = %q; want %q", file, got, want)
		}
	}
}
