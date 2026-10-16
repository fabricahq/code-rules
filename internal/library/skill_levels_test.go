// Test that library check accepts skill levels only for skills the library has.

package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fabricahq/syllabus/internal/librarytree"
)

func TestLibraryCheck_RequiresEveryLeveledSkillToExist(t *testing.T) {
	options := Options{Directory: t.TempDir()}
	if _, err := Initialize(context.Background(), options, nil, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		file := filepath.Join(options.Directory, librarytree.Directory, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("skills/ship/SKILL.md", "---\nname: ship\ndescription: Ship a service.\n---\n# Ship\n")
	write("config.yaml", "schemaVersion: 2\nsources: {}\nlibrary:\n  skills:\n    required: [skills/ship]\n    recommended: [skills/review]\n")
	_, err := Check(context.Background(), options)
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Location != ".syllabus/config.yaml: library.skills.recommended" || validation.Problem != "skills/review names no skill of the library, because .syllabus/skills/review/SKILL.md doesn't exist; add the skill, or remove it from the list" {
		t.Fatalf("got %v; want the missing recommended skill reported", err)
	}
	write("skills/review/SKILL.md", "---\nname: review\ndescription: Review a change.\n---\n# Review\n")
	if _, err := Check(context.Background(), options); err != nil {
		t.Fatalf("check with both skills: %v", err)
	}
}
