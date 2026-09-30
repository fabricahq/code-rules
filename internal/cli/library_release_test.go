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
}

// withheldServerMessage replaces a refused push's remote: lines that contained a known credential.
const withheldServerMessage = "  The server's messages were withheld, because they contained a credential.\n"

// TestLibraryRelease_ShowsTheServersReasonWithoutCredentials pushes to a remote whose URL carries credentials, with
// a known token in GH_TOKEN, to a pre-receive hook that prints text and then refuses the tag. Unknown credential
// formats are redacted from the lines shown. When the text holds a known credential, whether on one line, split by
// a control character, wrapped across lines, or straddling the line and length limits, the server's lines are
// withheld and only Git's rejection line is shown. Human and JSON output agree, and no credential or fragment of
// one appears in either.
func TestLibraryRelease_ShowsTheServersReasonWithoutCredentials(t *testing.T) {
	binary := buildCLI(t)
	filler := strings.Repeat("echo 'Policy line.' >&2\n", 19)
	for _, test := range []struct {
		name, hook string
		shown      []string
		withheld   bool
	}{
		{"unknown credential formats", "echo 'Release tags need approval from the maintainers.' >&2\n" +
			"echo 'Clone https://someone:hunter22@mirror.invalid/rules' >&2\n" +
			"echo 'Tokens ghp_classic1234 github_pat_11AB_cd34 glpat-gitlab_56-ef' >&2\n" +
			"echo 'Authorization: Basic c2VjcmV0' >&2\n" +
			"echo 'Sent Bearer bearer-token-5678' >&2\n", []string{
			"  remote: Release tags need approval from the maintainers.\n",
			"  remote: Clone https://[redacted]@mirror.invalid/rules\n",
			"  remote: Tokens [redacted] [redacted] [redacted]\n",
			"  remote: Authorization: [redacted]\n",
			"  remote: Sent Bearer [redacted]\n",
		}, false},
		{"known credentials on their own lines", "echo 'Clone https://fixture-user:fixture-secret%2Fpass@fixture.invalid/rules' >&2\n" +
			"echo 'Password fixture-secret/pass, token opaque-secret-value' >&2\n", nil, true},
		{"token split by a tab", "printf 'token opaque-\\tsecret-value\\n' >&2\n", nil, true},
		{"token wrapped across two lines", "echo 'token opaque-' >&2\necho 'secret-value' >&2\n", nil, true},
		{"token straddling the line limit", filler + "echo 'opaque-' >&2\necho 'secret-value' >&2\n", nil, true},
		{"token straddling the length limit", "echo '" + strings.Repeat("x", 195) + "opaque-secret-value' >&2\n", nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, dir := releasedLibrary(t)
			if _, err := fixture.CommandIn(context.Background(), dir, "remote", "set-url", "origin", "ssh://fixture-user:fixture-secret%2Fpass@fixture.invalid/rules"); err != nil {
				t.Fatal(err)
			}
			pendingPatch(t, fixture, dir)
			hooks := filepath.Join(fixture.Worktree(), ".git", "hooks")
			if err := os.MkdirAll(hooks, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(hooks, "pre-receive"), []byte("#!/bin/sh\n"+test.hook+"exit 1\n"), 0700); err != nil {
				t.Fatal(err)
			}
			environment, _ := releaseEnvironment(t, fixture, nil)
			environment = append(environment, "GH_TOKEN=opaque-secret-value")
			out, human, code := runCLIWithEnvironment(t, binary, dir, environment, "library", "release")
			if code != 1 || out != "" {
				t.Fatalf("exit %d, stdout %q, stderr:\n%s", code, out, human)
			}
			structured, diagnostic, code := runCLIWithEnvironment(t, binary, dir, environment, "library", "release", "--json")
			var response struct {
				OK    bool
				Error responseError
			}
			if err := json.Unmarshal([]byte(structured), &response); err != nil || code != 1 || diagnostic != "" || response.OK || response.Error.Code != "push-failed" {
				t.Fatalf("exit %d, %v, stderr %q, stdout:\n%s", code, err, diagnostic, structured)
			}
			shown := append([]string{"Git couldn't push release/2 to origin, which refused it:\n", "  ! [remote rejected] release/2 -> release/2 (pre-receive hook declined)\n"}, test.shown...)
			if test.withheld {
				shown = append(shown, withheldServerMessage)
			}
			for name, text := range map[string]string{"stderr": human, "JSON message": response.Error.Message} {
				for _, want := range shown {
					if !strings.Contains(text, want) {
						t.Errorf("%s lacks %q:\n%s", name, want, text)
					}
				}
				if test.withheld && strings.Contains(text, "remote:") {
					t.Errorf("%s shows the server's lines although they held a credential:\n%s", name, text)
				}
				for _, secret := range []string{"fixture-user", "fixture-secret", "opaque", "secret-value", "hunter22", "ghp_", "github_pat_", "glpat-", "c2VjcmV0", "bearer-token", "To ssh:"} {
					if strings.Contains(text, secret) {
						t.Errorf("%s shows %q:\n%s", name, secret, text)
					}
				}
			}
		})
	}
}
