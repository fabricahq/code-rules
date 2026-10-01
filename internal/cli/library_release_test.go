// Exercise code-rules library release through the compiled CLI, a Git fixture that accepts pushes, and a fake gh.

package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/ghfixture"
	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// releaseEnvironment returns the fixture's environment with a PATH holding only Git and a fake gh that
// answers responses.
func releaseEnvironment(t *testing.T, fixture *gitfixture.Fixture, responses []ghfixture.Response) ([]string, *ghfixture.Fake) {
	t.Helper()
	bin := t.TempDir()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(git, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	fake, err := ghfixture.Install(bin, responses)
	if err != nil {
		t.Fatal(err)
	}
	environment := []string{}
	for _, item := range fixture.Environment {
		if !strings.HasPrefix(item, "PATH=") {
			environment = append(environment, item)
		}
	}
	return append(environment, "PATH="+bin), fake
}

// pendingPatch commits and pushes a patch to rule a with its change note, returning the new commit.
func pendingPatch(t *testing.T, fixture *gitfixture.Fixture, dir string) string {
	t.Helper()
	ctx := context.Background()
	commit, err := fixture.Commit(ctx, dir, "Clarify a", map[string][]byte{
		"practices/testing/a.md": []byte(libraryRule("Test the retry limit, clearly.")),
		"changes/a.yaml":         []byte("summary: Clarify the retry-limit rule.\nrules:\n  practices/testing/a: patch\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.CommandIn(ctx, dir, "push", "--quiet", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	return commit
}

// releaseTwoNotes are the notes of the library release pendingPatch prepares.
const releaseTwoNotes = "Library release 2 changes 1 rule: 1 patch.\n\n## Patch changes\n\n- **practices/testing/a** `1.0.0` → `1.0.1`\n  - Clarify the retry-limit rule.\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n| practices/testing/a | 1.0.1 |\n| practices/testing/b | 1.0.0 |\n\n</details>"

// TestLibraryRelease_DryRunShowsTheLibraryReleaseAndChangesNothing prints the repository, branch, commit,
// release number, each rule's change, and the complete release notes, in human and JSON output.
func TestLibraryRelease_DryRunShowsTheLibraryReleaseAndChangesNothing(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	commit := pendingPatch(t, fixture, dir)
	environment, fake := releaseEnvironment(t, fixture, nil)
	out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, environment, "library", "release", "--dry-run")
	want := "Dry run: library release 2, not published.\n  Repository:          git@fixture.invalid:rules\n  Branch:              main\n  Commit:              " + commit + "\n  Tag:                 release/2 (not created yet)\n  GitHub Release page: none, because origin isn't on GitHub.com\nWarning: License is undeclared. Decide terms before sharing this library.\n\nRules:\n  practices/testing/a  patch  1.0.0 -> 1.0.1\n\nRelease notes:\n\n" + releaseTwoNotes + "\n\nTo publish it, run:\n  code-rules library release\n"
	if code != 0 || diagnostic != "" || out != want {
		t.Fatalf("exit %d, stderr %q, stdout:\n%s\nwant:\n%s", code, diagnostic, out, want)
	}
	out, diagnostic, code = runCLIWithEnvironment(t, binary, dir, environment, "library", "release", "--dry-run", "--json")
	var response struct {
		OK    bool
		Value map[string]json.RawMessage
	}
	if err := json.Unmarshal([]byte(out), &response); err != nil || code != 0 || diagnostic != "" || !response.OK {
		t.Fatal(err, code, out, diagnostic)
	}
	for field, want := range map[string]string{
		"repository": `"git@fixture.invalid:rules"`, "remote": `"origin"`, "branch": `"main"`, "commit": `"` + commit + `"`,
		"dryRun": "true", "release": "2", "tag": `"release/2"`, "published": "false", "tagCreated": "false", "libraryFiles": "[]",
		"rules": `[{"id":"practices/testing/a","change":"patch","from":"1.0.0","to":"1.0.1","summaries":["Clarify the retry-limit rule."]}]`,
	} {
		if got := compactJSON(t, response.Value[field]); got != want {
			t.Errorf("%s: %s, want %s", field, got, want)
		}
	}
	var notes string
	if err := json.Unmarshal(response.Value["notes"], &notes); err != nil || notes != releaseTwoNotes {
		t.Errorf("notes %q, %v", notes, err)
	}
	if _, ok := response.Value["githubRelease"]; ok {
		t.Error("a dry run reported a GitHub Release page")
	}
	out, diagnostic, code = runCLIWithEnvironment(t, binary, dir, environment, "library", "release", "--dry-run", "--no-github-release")
	if code != 0 || diagnostic != "" || !strings.HasSuffix(out, "\n\nTo publish it, run:\n  code-rules library release --no-github-release\n") {
		t.Errorf("the dry run's next step drops --no-github-release: exit %d, stderr %q, stdout:\n%s", code, diagnostic, out)
	}
	if tags, err := fixture.Command(context.Background(), "tag", "--list", "release/*"); err != nil || tags != "release/1" {
		t.Fatalf("remote release tags %q: %v", tags, err)
	}
	if calls, err := fake.Calls(); err != nil || len(calls) != 0 {
		t.Fatal(calls, err)
	}
}

// TestLibraryRelease_PublishesTheTagAndGitHubReleasePage reports what it created, and a rerun reports that
// both already exist.
func TestLibraryRelease_PublishesTheTagAndGitHubReleasePage(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	commit := pendingPatch(t, fixture, dir)
	if _, err := fixture.CommandIn(context.Background(), dir, "remote", "set-url", "origin", "git@github.com:acme/rules.git"); err != nil {
		t.Fatal(err)
	}
	auth := ghfixture.Response{Args: []string{"auth", "status", "--hostname", "github.com"}}
	view := []string{"release", "view", "release/2", "--repo", "github.com/acme/rules", "--json", "url", "--jq", ".url"}
	page := "https://github.com/acme/rules/releases/tag/release/2"
	environment, fake := releaseEnvironment(t, fixture, []ghfixture.Response{auth, {Args: view, Stderr: "release not found\n", ExitCode: 1},
		{Args: []string{"release", "create", "release/2", "--repo", "github.com/acme/rules", "--verify-tag", "--title", "release/2", "--notes-file", "-"}, Stdout: page + "\n"}})
	out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, environment, "library", "release")
	want := "Published library release 2.\n  Repository:          https://github.com/acme/rules\n  Branch:              main\n  Commit:              " + commit + "\n  Tag:                 release/2 (created and pushed to origin)\n  GitHub Release page: " + page + " (created)\nWarning: License is undeclared. Decide terms before sharing this library.\n\nRules:\n  practices/testing/a  patch  1.0.0 -> 1.0.1\n"
	if code != 0 || diagnostic != "" || out != want {
		t.Fatalf("exit %d, stderr %q, stdout:\n%s\nwant:\n%s", code, diagnostic, out, want)
	}
	calls, err := fake.Calls()
	if err != nil || len(calls) != 3 || calls[2].Stdin != releaseTwoNotes+"\n" {
		t.Fatal(calls, err)
	}
	environment, _ = releaseEnvironment(t, fixture, []ghfixture.Response{auth, {Args: view, Stdout: page + "\n"}})
	out, diagnostic, code = runCLIWithEnvironment(t, binary, dir, environment, "library", "release", "--json")
	var response struct {
		OK    bool
		Value struct {
			Published     bool
			TagCreated    bool
			GitHubRelease json.RawMessage
		}
	}
	if err := json.Unmarshal([]byte(out), &response); err != nil || code != 0 || diagnostic != "" || !response.OK || !response.Value.Published || response.Value.TagCreated {
		t.Fatal(err, code, out, diagnostic)
	}
	if got := compactJSON(t, response.Value.GitHubRelease); got != `{"created":false,"url":"`+page+`"}` {
		t.Fatalf("githubRelease %s", got)
	}
	// A dry run doesn't ask gh whether the page exists, so it claims no more than a rerun would do.
	out, diagnostic, code = runCLIWithEnvironment(t, binary, dir, environment, "library", "release", "--dry-run")
	if code != 0 || diagnostic != "" || !strings.Contains(out, "\n  GitHub Release page: created with gh for acme/rules if it's missing\n") {
		t.Fatalf("exit %d, stderr %q, stdout:\n%s", code, diagnostic, out)
	}
}

// TestLibraryRelease_ReportsRefusalsWithCodes explains the repair in human output and gives the code in JSON.
func TestLibraryRelease_ReportsRefusalsWithCodes(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	if _, err := fixture.Commit(context.Background(), dir, "Unpushed", map[string][]byte{"README.md": []byte("Unpushed.\n")}); err != nil {
		t.Fatal(err)
	}
	environment, _ := releaseEnvironment(t, fixture, nil)
	out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, environment, "library", "release")
	if code != 1 || out != "" || !strings.Contains(diagnostic, "main has 1 commit that origin/main doesn't. A library release publishes only pushed commits: push them, then run code-rules library release again.") {
		t.Fatal(code, out, diagnostic)
	}
	out, diagnostic, code = runCLIWithEnvironment(t, binary, dir, environment, "library", "release", "--json")
	var response struct {
		OK    bool
		Error responseError
	}
	if err := json.Unmarshal([]byte(out), &response); err != nil || code != 1 || diagnostic != "" || response.OK || response.Error.Code != "branch-differs" {
		t.Fatal(err, code, out, diagnostic)
	}
}

// TestLibraryRelease_ReportsNothingToPublish succeeds without creating a tag.
func TestLibraryRelease_ReportsNothingToPublish(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	environment, _ := releaseEnvironment(t, fixture, nil)
	if _, err := fixture.Commit(context.Background(), dir, "Readme", map[string][]byte{"README.md": []byte("About.\n")}); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.CommandIn(context.Background(), dir, "push", "--quiet", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, environment, "library", "release")
	if code != 0 || diagnostic != "" || !strings.HasPrefix(out, "Nothing to publish: no pending change notes and no library-wide changes since the latest library release.\n  Repository:          git@fixture.invalid:rules\n") {
		t.Fatal(code, out, diagnostic)
	}
	// Values that don't apply when there's nothing to publish are left out, and lists are empty.
	out, diagnostic, code = runCLIWithEnvironment(t, binary, dir, environment, "library", "release", "--json")
	var response struct{ Value map[string]json.RawMessage }
	if err := json.Unmarshal([]byte(out), &response); err != nil || code != 0 || diagnostic != "" {
		t.Fatal(err, code, out, diagnostic)
	}
	for _, field := range []string{"notes", "tag", "githubRelease"} {
		if value, present := response.Value[field]; present {
			t.Errorf("value.%s is %s, want it left out", field, value)
		}
	}
	if string(response.Value["release"]) != "0" || string(response.Value["rules"]) != "[]" || string(response.Value["libraryFiles"]) != "[]" {
		t.Errorf("value:\n%s", out)
	}
}

// TestLibraryRelease_ReportsThePublishedTagWhenTheGitHubReleasePageFails in value, beside the error, so scripts can
// tell the tag is published and the page is missing; human output reports only the error.
func TestLibraryRelease_ReportsThePublishedTagWhenTheGitHubReleasePageFails(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := gitHubLibrary(t)
	environment, _ := releaseEnvironment(t, fixture, []ghfixture.Response{{Args: ghAuth}, {Args: ghView, Stderr: "release not found\n", ExitCode: 1}, {Args: ghCreate, Stderr: "HTTP 502: Bad Gateway\n", ExitCode: 1}})
	out, diagnostic, code := runCLIWithEnvironment(t, binary, dir, environment, "library", "release", "--json")
	var response struct {
		OK    bool
		Value *struct {
			Release       int
			Tag           string
			TagCreated    bool
			GitHubRelease json.RawMessage
		}
		Error responseError
	}
	if err := json.Unmarshal([]byte(out), &response); err != nil || code != 1 || diagnostic != "" || response.OK || response.Error.Code != "github-release-failed" {
		t.Fatalf("exit %d, %v, stderr %q:\n%s", code, err, diagnostic, out)
	}
	if value := response.Value; value == nil || value.Release != 2 || value.Tag != "release/2" || !value.TagCreated || value.GitHubRelease != nil {
		t.Fatalf("value %+v:\n%s", response.Value, out)
	}
}
