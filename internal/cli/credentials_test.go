// Exercise how messages that quote Git, a Git server, or the GitHub CLI show or withhold their text, through the
// compiled CLI, in human and JSON output.

package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/ghfixture"
	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// withheldGit is how a message says it left out Git's text because the text held a credential.
const withheldGit = "Git's message was withheld because it contained a credential"

// failureOutput runs args in dir with environment, in human and JSON output, and returns the human stderr and the
// JSON error, requiring exit code 1 and the JSON error code.
func failureOutput(t *testing.T, binary, dir string, environment []string, code string, args ...string) (string, responseError) {
	t.Helper()
	out, human, exit := runCLIWithEnvironment(t, binary, dir, environment, args...)
	if exit != 1 || out != "" {
		t.Fatalf("exit %d, stdout %q, stderr:\n%s", exit, out, human)
	}
	structured, diagnostic, exit := runCLIWithEnvironment(t, binary, dir, environment, append(args, "--json")...)
	var response struct {
		OK    bool
		Error responseError
	}
	if err := json.Unmarshal([]byte(structured), &response); err != nil || exit != 1 || diagnostic != "" || response.OK || response.Error.Code != code {
		t.Fatalf("exit %d, %v, stderr %q, stdout:\n%s; want code %s", exit, err, diagnostic, structured, code)
	}
	return human, response.Error
}

// requireShown checks that the human stderr and the JSON message contain each of shown and none of hidden.
func requireShown(t *testing.T, human string, structured responseError, shown, hidden []string) {
	t.Helper()
	for name, text := range map[string]string{"stderr": human, "JSON message": structured.Message} {
		for _, want := range shown {
			if !strings.Contains(text, want) {
				t.Errorf("%s lacks %q:\n%s", name, want, text)
			}
		}
		for _, secret := range hidden {
			if strings.Contains(text, secret) {
				t.Errorf("%s shows %q:\n%s", name, secret, text)
			}
		}
	}
}

// refusingHook installs a pre-receive hook in the fixture's repository that runs script and then refuses the push.
func refusingHook(t *testing.T, fixture *gitfixture.Fixture, script string) {
	t.Helper()
	hooks := filepath.Join(fixture.Worktree(), ".git", "hooks")
	if err := os.MkdirAll(hooks, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "pre-receive"), []byte("#!/bin/sh\n"+script+"exit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
}

// TestLibraryRelease_ShowsOrWithholdsTheServersReason pushes to a remote whose URL carries a password, with a
// token in GH_TOKEN, to a pre-receive hook that prints text and refuses the tag. Text without a known credential is
// shown, with unknown credential formats redacted. Text that holds a known credential in any spelling or wrapping
// is withheld whole, saying so, and no part of the credential appears.
func TestLibraryRelease_ShowsOrWithholdsTheServersReason(t *testing.T) {
	binary := buildCLI(t)
	for _, test := range []struct {
		name, token, hook string
		shown, hidden     []string
	}{
		{"unknown credential formats", "opaque-secret-value", "echo 'Release tags need approval from the maintainers.' >&2\n" +
			"echo 'Clone https://someone:p@ssword@mirror.invalid/rules or https://other:hunter22@mirror.invalid/x' >&2\n" +
			"echo 'Tokens ghp_classic1234 github_pat_11AB_cd34 glpat-gitlab_56-ef' >&2\n" +
			"echo 'Authorization: Basic c2VjcmV0' >&2\n" +
			"echo 'Sent Bearer bearer-token-5678' >&2\n", []string{
			"  remote: Release tags need approval from the maintainers.\n",
			"  remote: Clone https://[redacted]@mirror.invalid/rules or https://[redacted]@mirror.invalid/x\n",
			"  remote: Tokens [redacted] [redacted] [redacted]\n",
			"  remote: Authorization: [redacted]\n",
			"  remote: Sent Bearer [redacted]\n",
			"  ! [remote rejected] release/2 -> release/2 (pre-receive hook declined)\n",
		}, []string{"ssword", "hunter22", "ghp_", "github_pat_", "glpat-", "c2VjcmV0", "bearer-token", withheldGit}},
		{"credential URL wrapped across lines", "opaque-secret-value", "echo 'Clone https://someone:hunt' >&2\necho 'er22@mirror.invalid/rules' >&2\n", []string{withheldGit}, []string{"hunt", "er22", "remote:"}},
		{"remote password", "opaque-secret-value", "echo 'Password fixture-secret/pass' >&2\n", []string{withheldGit}, []string{"fixture-secret", "remote:"}},
		{"double percent-encoding", "opaque-secret-value", "echo 'token opaque%252dsecret-value' >&2\n", []string{withheldGit}, []string{"opaque", "secret", "remote:"}},
		{"erase-line sequence inside the token", "opaque-secret-value", "printf 'token opaque-\\033[Ksecret-value\\n' >&2\n", []string{withheldGit}, []string{"opaque", "secret", "remote:"}},
		{"color sequences inside the token", "opaque-secret-value", "printf 'token opaque-\\033[31msecret-value\\033[0m\\n' >&2\n", []string{withheldGit}, []string{"opaque", "secret", "remote:"}},
		{"zero-width separator", "opaque-secret-value", "printf 'token opaque-\\342\\200\\213secret-value\\n' >&2\n", []string{withheldGit}, []string{"opaque", "secret", "remote:"}},
		{"backslash escape", "opaque-secret-value", "printf '%s\\n' 'token opaque-\\u0073ecret-value' >&2\n", []string{withheldGit}, []string{"opaque", "ecret", "remote:"}},
		{"decomposed Unicode", "café-secret-value", "printf 'token cafe\\314\\201-secret-value\\n' >&2\n", []string{withheldGit}, []string{"secret", "remote:"}},
		{"wrapped differently at each boundary", "alpha beta-gamma", "echo 'alpha' >&2\necho 'beta-' >&2\necho 'gamma' >&2\n", []string{withheldGit}, []string{"alpha", "beta", "gamma", "remote:"}},
		{"hook's own remote: prefix inside the token", "opaque-secret-value", "printf 'opaque-\\nremote: secret-value\\n' >&2\n", []string{withheldGit}, []string{"opaque", "secret", "remote:"}},
		{"full token and wrapped fragments", "opaque-secret-value", "echo 'opaque-secret-value and opaque-' >&2\necho 'secret-value' >&2\n", []string{withheldGit}, []string{"opaque", "secret", "remote:"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, dir := releasedLibrary(t)
			if _, err := fixture.CommandIn(context.Background(), dir, "remote", "set-url", "origin", "ssh://fixture-user:fixture-secret%2Fpass@fixture.invalid/rules"); err != nil {
				t.Fatal(err)
			}
			pendingPatch(t, fixture, dir)
			refusingHook(t, fixture, test.hook)
			environment, _ := releaseEnvironment(t, fixture, nil)
			human, structured := failureOutput(t, binary, dir, append(environment, "GH_TOKEN="+test.token), "push-failed", "library", "release")
			requireShown(t, human, structured, append([]string{"Git couldn't push release/2 to origin"}, test.shown...), append([]string{"fixture-secret", "To ssh:"}, test.hidden...))
		})
	}
}

// TestLibraryRelease_WithholdsATokenTheRemoteAdvertisesAsItsDefaultBranch, which reaches the refusal through Git's
// standard output rather than its diagnostics.
func TestLibraryRelease_WithholdsATokenTheRemoteAdvertisesAsItsDefaultBranch(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	ctx := context.Background()
	if _, err := fixture.Command(ctx, "branch", "opaque-secret-value"); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.Command(ctx, "symbolic-ref", "HEAD", "refs/heads/opaque-secret-value"); err != nil {
		t.Fatal(err)
	}
	environment, _ := releaseEnvironment(t, fixture, nil)
	human, structured := failureOutput(t, binary, dir, append(environment, "GH_TOKEN=opaque-secret-value"), "not-default-branch", "library", "release")
	requireShown(t, human, structured, []string{"origin's default branch"}, []string{"opaque", "secret"})
}

// TestLibraryRelease_WithholdsAGitHubCLIMessageThatEchoesTheToken when creating the GitHub Release page fails.
func TestLibraryRelease_WithholdsAGitHubCLIMessageThatEchoesTheToken(t *testing.T) {
	binary := buildCLI(t)
	fixture, dir := releasedLibrary(t)
	pendingPatch(t, fixture, dir)
	if _, err := fixture.CommandIn(context.Background(), dir, "remote", "set-url", "origin", "git@github.com:acme/rules.git"); err != nil {
		t.Fatal(err)
	}
	auth := ghfixture.Response{Args: []string{"auth", "status", "--hostname", "github.com"}}
	view := ghfixture.Response{Args: []string{"release", "view", "release/2", "--repo", "github.com/acme/rules", "--json", "url", "--jq", ".url"}, Stderr: "release not found\n", ExitCode: 1}
	create := ghfixture.Response{Args: []string{"release", "create", "release/2", "--repo", "github.com/acme/rules", "--verify-tag", "--title", "release/2", "--notes-file", "-"}, Stderr: "HTTP 400: token opaque-secret-value rejected\n", ExitCode: 1}
	environment, _ := releaseEnvironment(t, fixture, []ghfixture.Response{auth, view, create})
	out, human, code := runCLIWithEnvironment(t, binary, dir, append(environment, "GH_TOKEN=opaque-secret-value"), "library", "release")
	if code != 1 || out != "" || !strings.Contains(human, "the GitHub CLI's message was withheld because it contained a credential") || strings.Contains(human, "opaque") {
		t.Fatalf("exit %d, stdout %q, stderr:\n%s", code, out, human)
	}
}

// syncFailure initializes a project in a new directory whose only source imports repository, syncs it with
// environment, and returns the failure's human stderr and JSON error, requiring code.
func syncFailure(t *testing.T, binary, repository string, environment []string, code string) (string, responseError) {
	t.Helper()
	dir := t.TempDir()
	if out, diagnostic, exit := runCLIWithEnvironment(t, binary, dir, environment, "project", "init"); exit != 0 {
		t.Fatal(exit, out, diagnostic)
	}
	config := "schemaVersion: 1\nsources:\n  team:\n    repository: " + repository + "\n    groups:\n      - techs/go\n"
	if err := os.WriteFile(filepath.Join(dir, ".code-rules", "config.yaml"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	return failureOutput(t, binary, dir, environment, code, "project", "sync")
}

// TestSync_ClassifiesAConnectionFailureEvenWhenAShortTokenWithholdsItsText: the token on appears in Connection
// refused, so the text is withheld, but the failure is still a connection failure.
func TestSync_ClassifiesAConnectionFailureEvenWhenAShortTokenWithholdsItsText(t *testing.T) {
	binary := buildCLI(t)
	environment := append(isolatedEnvironment(t), "GH_TOKEN=on")
	human, structured := syncFailure(t, binary, "https://127.0.0.1:1/acme/rules.git", environment, "connection-failed")
	requireShown(t, human, structured, []string{"Could not connect to the library's repository", withheldGit}, []string{"[redacted]"})
}

// TestSync_WithholdsTheCredentialOfARewrittenRemote that Git's url.*.insteadOf adds, which the configured address
// doesn't show.
func TestSync_WithholdsTheCredentialOfARewrittenRemote(t *testing.T) {
	binary := buildCLI(t)
	home := t.TempDir()
	global := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(global, []byte("[url \"ssh://reader:opaque-url-secret@fixture.invalid/\"]\n\tinsteadOf = https://rewritten.invalid/\n"), 0600); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(home, "ssh")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\necho \"ssh: Could not resolve hostname fixture.invalid: Connection refused by $*\" >&2\nexit 255\n"), 0700); err != nil {
		t.Fatal(err)
	}
	environment := append(isolatedEnvironment(t), "GIT_CONFIG_GLOBAL="+global, "GIT_SSH_COMMAND="+shim, "GIT_SSH_VARIANT=ssh")
	human, structured := syncFailure(t, binary, "https://rewritten.invalid/acme/rules.git", environment, "connection-failed")
	requireShown(t, human, structured, []string{withheldGit}, []string{"opaque-url-secret", "url-secret"})
}

// isolatedEnvironment returns the process environment without Git's or the user's configuration.
func isolatedEnvironment(t *testing.T) []string {
	t.Helper()
	environment := []string{}
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if !strings.HasPrefix(key, "GIT_") && key != "HOME" && key != "XDG_CONFIG_HOME" && key != "GH_TOKEN" && key != "GITHUB_TOKEN" {
			environment = append(environment, item)
		}
	}
	home := t.TempDir()
	return append(environment, "HOME="+home, "XDG_CONFIG_HOME="+home, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0")
}
