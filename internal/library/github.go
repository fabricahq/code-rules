// Find a library's GitHub.com repository and create its GitHub Release pages with the GitHub CLI, gh.

package library

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/fabricahq/code-rules/internal/gitexec"
	"github.com/fabricahq/code-rules/internal/rules"
)

// maxGitHubCLIOutput bounds what one gh invocation may print on each stream.
const maxGitHubCLIOutput = 1024 * 1024

// gitHubCLI runs the GitHub CLI without prompts.
type gitHubCLI struct {
	executable  string
	environment []string
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

// findGitHubCLI locates gh on environment's PATH and requires it to be signed in to GitHub.com.
func findGitHubCLI(ctx context.Context, environment []string) (gitHubCLI, error) {
	if environment == nil {
		environment = os.Environ()
	}
	executable := gitexec.LookPath("gh", environment)
	if executable == "" {
		return gitHubCLI{}, failure("github-cli-missing", "code-rules library release creates the GitHub Release page with the GitHub CLI, gh, which isn't installed. Install it from https://cli.github.com and sign in with gh auth login, or pass --no-github-release to publish the tag only.", nil)
	}
	cli := gitHubCLI{executable: executable, environment: append(slices.Clone(environment), "GH_PROMPT_DISABLED=1", "GH_NO_UPDATE_NOTIFIER=1")}
	if _, _, status, err := cli.run(ctx, "", "auth", "status", "--hostname", "github.com"); err != nil {
		return gitHubCLI{}, err
	} else if status != 0 {
		return gitHubCLI{}, failure("github-cli-signed-out", "the GitHub CLI, gh, isn't signed in to GitHub.com, so it can't create the GitHub Release page. Sign in with gh auth login, or pass --no-github-release to publish the tag only.", nil)
	}
	return cli, nil
}

// releasePage returns the URL of tag's GitHub Release page in repository, or false when gh can't show one,
// usually because it doesn't exist yet.
func (c gitHubCLI) releasePage(ctx context.Context, repository, tag string) (string, bool, error) {
	stdout, _, status, err := c.run(ctx, "", "release", "view", tag, "--repo", "github.com/"+repository, "--json", "url", "--jq", ".url")
	if err != nil || status != 0 {
		return "", false, err
	}
	return strings.TrimSpace(stdout), true, nil
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

// run executes gh with args and stdin, returning its output and exit status. Cancellation stops gh; output
// beyond maxGitHubCLIOutput on either stream fails.
func (c gitHubCLI) run(ctx context.Context, stdin string, args ...string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, c.executable, args...)
	cmd.Env = c.environment
	cmd.Stdin = strings.NewReader(stdin)
	cmd.WaitDelay = time.Second
	stdout, stderr := &boundedBuffer{limit: maxGitHubCLIOutput}, &boundedBuffer{limit: maxGitHubCLIOutput}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return "", "", 0, ctx.Err()
	}
	if stdout.exceeded || stderr.exceeded {
		return "", "", 0, failure("limit-exceeded", "the GitHub CLI's output exceeds its limit", nil)
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return stdout.String(), stderr.String(), exit.ExitCode(), nil
	}
	if err != nil {
		return "", "", 0, failure("github-cli-missing", "code-rules library release couldn't run the GitHub CLI, gh. Check that it's installed, or pass --no-github-release to publish the tag only.", nil)
	}
	return stdout.String(), stderr.String(), 0, nil
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

// boundedBuffer keeps at most limit bytes and records whether more arrived.
type boundedBuffer struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

// Write keeps data while it fits and fails once it doesn't, which stops copying from the process.
func (b *boundedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.buffer.Len()+len(data) > b.limit {
		b.exceeded = true
		return 0, io.ErrShortWrite
	}
	return b.buffer.Write(data)
}

// String returns what was kept.
func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}
