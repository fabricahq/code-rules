// Exercise how trailguide library check reads the repositories a library borrows skills from, at the refs it declares.

package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/trailguide/internal/test/gitfixture"
)

// checkBorrowingLibrary runs library check in a library outside Git whose configuration is config, with an own
// skill, skills/review, and pocock, a repository that isn't a library, with the skills skills/ops/deploy and
// skills/team/review, tagged stable.
func checkBorrowingLibrary(t *testing.T, config string) (string, string, int) {
	t.Helper()
	ctx := context.Background()
	pocock, err := gitfixture.New(ctx, map[string][]byte{
		"skills/ops/deploy/SKILL.md":  []byte("---\nname: deploy\ndescription: Deploy a service.\n---\n# Deploy\n"),
		"skills/team/review/SKILL.md": []byte("---\nname: review\ndescription: Review a change.\n---\n# Review\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pocock.Close() })
	if _, err := pocock.Command(ctx, "tag", "stable", pocock.FirstCommit); err != nil {
		t.Fatal(err)
	}
	environment, err := pocock.Route(map[string]*gitfixture.Fixture{"pocock": pocock})
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	for name, text := range map[string]string{
		".trailguide/config.yaml":            config,
		".trailguide/skills/review/SKILL.md": "---\nname: review\ndescription: Review a pull request.\n---\n# Review\n",
	} {
		file := filepath.Join(directory, name)
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return runCLIWithEnvironment(t, buildCLI(t), directory, environment, "library", "check")
}

func TestLibraryCheck_AcceptsBorrowedSkillsTheirSourceHas(t *testing.T) {
	out, diagnostic, code := checkBorrowingLibrary(t, "schemaVersion: 2\nsources:\n  pocock:\n    repository: git@fixture.invalid:pocock\n    ref: stable\nlibrary:\n  skills:\n    required: [pocock:skills/ops/deploy]\n")
	if code != 0 || !strings.Contains(out, "Library is valid") {
		t.Fatalf("exit %d:\n%s%s", code, out, diagnostic)
	}
}

func TestLibraryCheck_RejectsABorrowedSkillItsSourceLacks(t *testing.T) {
	out, diagnostic, code := checkBorrowingLibrary(t, "schemaVersion: 2\nsources:\n  pocock:\n    repository: git@fixture.invalid:pocock\n    ref: stable\nlibrary:\n  skills:\n    offered: [pocock:skills/ops/grill]\n")
	want := "library.skills.offered names pocock:skills/ops/grill, which pocock doesn't have at stable; check the skill ID, or set sources.pocock.ref to a revision that has it"
	if code == 0 || !strings.Contains(out+diagnostic, want) {
		t.Fatalf("exit %d, want %q:\n%s%s", code, want, out, diagnostic)
	}
}

func TestLibraryCheck_RejectsABorrowedSkillWithTheNameOfItsOwn(t *testing.T) {
	out, diagnostic, code := checkBorrowingLibrary(t, "schemaVersion: 2\nsources:\n  pocock:\n    repository: git@fixture.invalid:pocock\n    ref: stable\nlibrary:\n  skills:\n    recommended: [pocock:skills/team/review]\n")
	want := "library.skills lists skills/review and pocock:skills/team/review, which both install as review; list one of them, or rename the folder of your own skill"
	if code == 0 || !strings.Contains(out+diagnostic, want) {
		t.Fatalf("exit %d, want %q:\n%s%s", code, want, out, diagnostic)
	}
}
