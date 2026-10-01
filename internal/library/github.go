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

// gitHubRepository returns OWNER/REPO when a remote URL names a repository on GitHub.com, including one with
// credentials in it, and "" for other hosts and for URLs Code Rules doesn't recognize, such as SSH host aliases.
func gitHubRepository(remoteURL string) string {
	repository, ok := parseRemote(remoteURL)
	if !ok || repository.Web == nil {
		return ""
	}
	path, _ := strings.CutPrefix(repository.Web.Root, "https://github.com/")
	if path == repository.Web.Root {
		return ""
	}
	return path
}

// hiddenRemote stands in for a remote URL that can't be parsed, whose credentials, if any, can't be found and
// removed.
const hiddenRemote = "(a URL that can't be parsed, hidden in case it contains credentials)"

// displayRepository returns a remote URL to show people, without credentials, a query, or a fragment, or
// hiddenRemote for a URL that can't be parsed.
func displayRepository(remoteURL string) string {
	if repository, ok := parseRemote(remoteURL); ok {
		if repository.Web != nil {
			return repository.Web.Root
		}
		return repository.Identity
	}
	if address, ok := withoutCredentials(remoteURL); ok {
		return address
	}
	return hiddenRemote
}

// parseRemote identifies the repository a remote URL names, ignoring any credentials, query, or fragment in it.
// It reports false for URLs that project configuration wouldn't accept, such as local paths.
func parseRemote(remoteURL string) (rules.Repository, bool) {
	address, ok := withoutCredentials(remoteURL)
	if !ok {
		return rules.Repository{}, false
	}
	encoded, err := json.Marshal(address)
	if err != nil {
		return rules.Repository{}, false
	}
	repository, err := rules.ParseRepository(encoded, "remote")
	return repository, err == nil
}

// withoutCredentials removes the parts of a remote URL that can carry credentials: a password, an HTTPS user
// name such as a token, a query, and a fragment. An SSH user name, which names the account, stays. It reports
// false for an address with a scheme, such as https://, that isn't a URL it can parse, because it can't tell
// where credentials in it are.
func withoutCredentials(remoteURL string) (string, bool) {
	parsed, err := url.Parse(remoteURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		if strings.Contains(remoteURL, "://") {
			return "", false
		}
		// A user@host:path address or a local path isn't a URL; Git reads neither a query nor a fragment in it.
		if end := strings.IndexAny(remoteURL, "?#"); end >= 0 {
			return remoteURL[:end], true
		}
		return remoteURL, true
	}
	if parsed.User != nil && parsed.Scheme == "ssh" {
		parsed.User = url.User(parsed.User.Username())
	} else {
		parsed.User = nil
	}
	parsed.RawQuery, parsed.ForceQuery, parsed.Fragment, parsed.RawFragment = "", false, "", ""
	return parsed.String(), true
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
		page, err := c.pageURL(stdout, tag)
		return page, err == nil, err
	case strings.Contains(stderr, releaseNotFound):
		return "", false, nil
	}
	problem := "the GitHub CLI couldn't look up the GitHub Release page for " + tag
	if reason := c.reason(stderr); reason != "" {
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
		if reason := c.reason(stderr); reason != "" {
			problem += " (" + reason + ")"
		}
		return "", failure("github-release-failed", problem+". The tag is published; run code-rules library release again to create the page.", nil)
	}
	return c.pageURL(stdout, tag)
}

// pageURL returns the GitHub Release page URL gh printed for tag: an https://github.com/ URL without user
// information, which gitexec's Show allows showing. Anything else fails with github-release-failed.
func (c gitHubCLI) pageURL(stdout, tag string) (string, error) {
	text := strings.TrimSpace(stdout)
	page, err := url.Parse(text)
	if _, shown := c.runner.Credentials().Show(text); err != nil || !shown || page.Scheme != "https" || page.Host != "github.com" || page.User != nil || strings.ContainsAny(text, " \t\r\n") {
		return "", failure("github-release-failed", "the GitHub CLI didn't print a GitHub Release page URL for "+tag+". The tag is published; check the page on GitHub.com, and run code-rules library release again to finish.", nil)
	}
	return text, nil
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

// reason returns gh's last nonblank line of diagnostics, without control characters and at most 200 characters
// long, to explain a failure, when gitexec's Show allows showing the diagnostics, and otherwise a note that they
// were withheld.
func (c gitHubCLI) reason(stderr string) string {
	shown, ok := c.runner.Credentials().Show(stderr)
	if !ok {
		return "the GitHub CLI's message was withheld because it contained a credential"
	}
	lines := strings.Split(strings.TrimSpace(gitexec.Printable(shown)), "\n")
	return gitexec.Shorten(strings.TrimSpace(lines[len(lines)-1]), 200)
}
