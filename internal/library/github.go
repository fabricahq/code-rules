// Find a library's GitHub.com repository and create its GitHub Release pages with the GitHub CLI, gh.

package library

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/fabricahq/code-rules/internal/gitexec"
	"github.com/fabricahq/code-rules/internal/rules"
)

// maxGitHubCLIOutput bounds what one gh invocation may print on both streams together.
const maxGitHubCLIOutput = 1024 * 1024

// gitHubCLI runs the GitHub CLI without prompts, with the same output budget and process-group cancellation
// as Git.
type gitHubCLI struct {
	runner gitexec.Runner
	// dir is where gh runs; every call names its repository, so gh never reads the directory's Git state.
	dir string
}

// gitHubRepository returns OWNER/REPO when a remote URL names a repository on GitHub.com, and "" for other
// hosts and for URLs Code Rules doesn't recognize, such as SSH host aliases.
func gitHubRepository(remoteURL string) string {
	encoded, err := json.Marshal(remoteURL)
	if err != nil {
		return ""
	}
	repository, err := rules.ParseRepository(encoded, "remote")
	if err != nil || repository.Web == nil {
		return ""
	}
	path, _ := strings.CutPrefix(repository.Web.Root, "https://github.com/")
	if path == repository.Web.Root {
		return ""
	}
	return path
}

// displayRepository returns a remote URL to show people, without any credentials embedded in it.
func displayRepository(remoteURL string) string {
	if encoded, err := json.Marshal(remoteURL); err == nil {
		if repository, err := rules.ParseRepository(encoded, "remote"); err == nil {
			if repository.Web != nil {
				return repository.Web.Root
			}
			return repository.Identity
		}
	}
	if parsed, err := url.Parse(remoteURL); err == nil && parsed.User != nil {
		parsed.User = nil
		return parsed.String()
	}
	return remoteURL
}

// findGitHubCLI locates gh on environment's PATH, or the process's when it's nil, and requires it to be signed
// in to GitHub.com. gh runs in dir.
func findGitHubCLI(ctx context.Context, environment []string, dir string) (gitHubCLI, error) {
	if environment == nil {
		environment = os.Environ()
	}
	runner, err := gitexec.Command(gitexec.Options{Environment: append(slices.Clone(environment), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1")}, "gh")
	if errors.Is(err, gitexec.ErrNotFound) {
		return gitHubCLI{}, failure("github-cli-missing", "code-rules library release creates the GitHub Release page with the GitHub CLI, gh, which isn't installed. Install it from https://cli.github.com and sign in with gh auth login, or pass --no-github-release to publish the tag only.", nil)
	}
	if err != nil {
		return gitHubCLI{}, err
	}
	cli := gitHubCLI{runner: runner, dir: dir}
	if _, _, status, err := cli.run(ctx, "", "auth", "status", "--hostname", "github.com"); err != nil {
		return gitHubCLI{}, err
	} else if status != 0 {
		return gitHubCLI{}, failure("github-cli-signed-out", "the GitHub CLI, gh, isn't signed in to GitHub.com, so it can't create the GitHub Release page. Sign in with gh auth login, or pass --no-github-release to publish the tag only.", nil)
	}
	return cli, nil
}

// releaseNotFound is how gh reports that a repository has no GitHub Release page for a tag.
const releaseNotFound = "release not found"

// releasePage returns the URL of tag's GitHub Release page in repository, or false when gh reports that it
// doesn't exist. Any other failure, such as an API error, fails with github-release-failed, so a page that
// can't be read is never created again.
func (c gitHubCLI) releasePage(ctx context.Context, repository, tag string) (string, bool, error) {
	stdout, stderr, status, err := c.run(ctx, "", "release", "view", tag, "--repo", "github.com/"+repository, "--json", "url", "--jq", ".url")
	switch {
	case err != nil:
		return "", false, err
	case status == 0:
		return strings.TrimSpace(stdout), true, nil
	case strings.Contains(stderr, releaseNotFound):
		return "", false, nil
	}
	problem := "the GitHub CLI couldn't look up the GitHub Release page for " + tag
	if reason := diagnosticLine(stderr); reason != "" {
		problem += " (" + reason + ")"
	}
	return "", false, failure("github-release-failed", problem+". The tag is published; run code-rules library release again to finish.", nil)
}

// createReleasePage creates tag's GitHub Release page in repository, titled with the tag and with notes as its
// body, and returns its URL. gh refuses when the tag isn't on GitHub.
func (c gitHubCLI) createReleasePage(ctx context.Context, repository, tag, notes string) (string, error) {
	stdout, stderr, status, err := c.run(ctx, notes+"\n", "release", "create", tag, "--repo", "github.com/"+repository, "--verify-tag", "--title", tag, "--notes-file", "-")
	if err != nil {
		return "", err
	}
	if status != 0 {
		problem := "the GitHub CLI couldn't create the GitHub Release page for " + tag
		if reason := diagnosticLine(stderr); reason != "" {
			problem += " (" + reason + ")"
		}
		return "", failure("github-release-failed", problem+". The tag is published; run code-rules library release again to create the page.", nil)
	}
	return strings.TrimSpace(stdout), nil
}

// run executes gh with args and stdin, returning its stdout, its stderr, and its exit status. Output beyond
// maxGitHubCLIOutput and cancellation stop gh and everything it started.
func (c gitHubCLI) run(ctx context.Context, stdin string, args ...string) (string, string, int, error) {
	result, err := c.runner.Run(ctx, c.dir, args, maxGitHubCLIOutput, []byte(stdin))
	var failed *gitexec.Error
	switch {
	case err == nil:
		return string(result.Output), string(result.Diagnostics), result.Status, nil
	case errors.As(err, &failed) && failed.Code == "limit-exceeded":
		return "", "", 0, failure("limit-exceeded", "the GitHub CLI's output exceeds its limit", nil)
	case errors.As(err, &failed) && (failed.Code == "cancelled" || failed.Code == "timed-out"):
		return "", "", 0, err
	}
	return "", "", 0, failure("github-cli-failed", "code-rules library release couldn't run the GitHub CLI, gh. Check that it's installed and works, or pass --no-github-release to publish the tag only.", nil)
}

// diagnosticLine returns gh's last nonblank line of diagnostics, without control characters and at most
// 200 characters long, to explain a failure.
func diagnosticLine(stderr string) string {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	line := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(lines[len(lines)-1]))
	if runes := []rune(line); len(runes) > 200 {
		line = string(runes[:200]) + "..."
	}
	return line
}
