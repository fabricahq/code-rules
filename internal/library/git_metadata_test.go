// Check that values library release reads from Git's output reach messages only after their form is validated.

package library

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// metadataMarker stands for anything a Git executable could print where Code Rules expects machine metadata.
const metadataMarker = "EXTERNAL-TEXT-MARKER"

// wrapGit makes options run a Git wrapper that prints output, with printf escapes, for a command whose arguments
// contain match, and runs the fixture's Git otherwise.
func wrapGit(t *testing.T, fixture *gitfixture.Fixture, options *Options, match, output string) {
	t.Helper()
	wrapper := filepath.Join(t.TempDir(), "git")
	script := "#!/bin/sh\ncase \" $* \" in\n*" + gitfixture.Quote(match) + "*) printf " + gitfixture.Quote(output) + "; exit 0 ;;\nesac\nexec " + gitfixture.Quote(fixture.GitPath) + " \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	options.Git.GitPath = wrapper
}

// TestRelease_ShowsOnlyValidatedMachineMetadataFromGit: a commit count, an object ID, or an object type in an
// unexpected form fails with a static message rather than reach the error. An unpushed commit makes release
// compare the branches.
func TestRelease_ShowsOnlyValidatedMachineMetadataFromGit(t *testing.T) {
	sha := strings.Repeat("a", 40)
	for _, test := range []struct {
		name, match, output string
		unpushed            bool
	}{
		{"commit count", "--left-right --count", metadataMarker + "\\t0\\n", true},
		{"signed commit count", "--left-right --count", "+5\\t0\\n", true},
		{"object type", "--format=%(refname)", "refs/tags/release/1\\000" + metadataMarker + "\\000" + sha + "\\000100\\000commit\\000" + sha + "\\n", false},
		{"tagged object type", "--format=%(refname)", "refs/tags/release/1\\000tag\\000" + sha + "\\000100\\000" + metadataMarker + "\\000" + sha + "\\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, options := authorClone(t, libraryFiles())
			if test.unpushed {
				if _, err := fixture.Commit(context.Background(), options.Directory, "Unpushed", map[string][]byte{"README.md": []byte("Unpushed.\n")}); err != nil {
					t.Fatal(err)
				}
			}
			wrapGit(t, fixture, &options, test.match, test.output)
			_, err := Release(context.Background(), ReleaseRequest{Options: options, NoGitHubRelease: true})
			if err == nil || strings.Contains(err.Error(), metadataMarker) || strings.Contains(err.Error(), "+5") || !strings.Contains(err.Error(), "unexpected format") {
				t.Fatalf("got %v, want a static unexpected-format failure", err)
			}
		})
	}
}

// TestCreateTag_RefusesAnObjectIDInAnUnexpectedForm before passing it to Git again.
func TestCreateTag_RefusesAnObjectIDInAnUnexpectedForm(t *testing.T) {
	fixture, options := authorClone(t, libraryFiles())
	wrapGit(t, fixture, &options, "--verify refs/tags/release/1", metadataMarker+"\\n")
	git, err := openLibraryGit(context.Background(), options.Directory, options.Git)
	if err != nil {
		t.Fatal(err)
	}
	_, err = git.createTag(context.Background(), "release/1", fixture.LatestCommit, []byte("Notes.\n"))
	if errorCode(err) != "git-failed" || strings.Contains(err.Error(), metadataMarker) {
		t.Fatalf("got %v", err)
	}
}
