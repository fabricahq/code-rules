// Share the Git runner's failure type, so every import failure carries one stable code, and explain Git's failures
// from its diagnostics without showing credentials.

package imports

import (
	"strings"
	"unicode"

	"github.com/fabricahq/code-rules/internal/gitexec"
)

// Error is the type of every import failure, including Git failures. Its Code is stable for callers.
type Error = gitexec.Error

// fail constructs an import failure with a stable category and safe explanation.
func fail(code, problem string, cause error) error {
	return gitexec.Fail(code, problem, cause)
}

// connectionFailures are the texts Git and its transports print when they can't reach a host at all, such as when
// the host name doesn't resolve, nothing listens, or TLS fails, as opposed to a server that answers and refuses.
var connectionFailures = []string{
	"Could not resolve host", "Couldn't resolve host", "Could not resolve hostname", "Temporary failure in name resolution",
	"nodename nor servname provided", "Name or service not known", "Failed to connect", "Could not connect to server",
	"Connection refused", "Connection timed out", "Operation timed out", "Connection reset", "Network is unreachable",
	"No route to host", "SSL certificate problem", "SSL connect error", "Host key verification failed",
}

// objectRefusals are the texts Git prints when a server refuses to send an object by its ID, which only servers
// that speak Git protocol version 0 or 1 without uploadpack.allowAnySHA1InWant do.
var objectRefusals = []string{"Server does not allow request for unadvertised object", "not our ref"}

// maxReasonRunes bounds the part of Git's diagnostics a failure quotes.
const maxReasonRunes = 200

// remoteFailure explains a Git command that couldn't read the library's repository: a host it couldn't reach, with
// code connection-failed, or otherwise a repository that doesn't exist or that the credentials can't read, with
// code not-found-or-no-access, since servers report both the same way.
func (r *repository) remoteFailure(diagnostics []byte) error {
	if reason := r.gitReason(diagnostics, connectionFailures); reason != "" {
		return fail("connection-failed", "Could not connect to the library's repository: "+reason+". Check the repository address and your network connection.", nil)
	}
	return fail("not-found-or-no-access", "Repository not found or no access; check its address and Git credentials.", nil)
}

// gitFailure explains a Git command that failed while reading the library: a host it couldn't reach, with code
// connection-failed; a server that refused to send files by object ID, with code object-fetch-refused; or
// otherwise problem, with code, followed by Git's last error line when it printed one.
func (r *repository) gitFailure(code, problem string, diagnostics []byte) error {
	if reason := r.gitReason(diagnostics, connectionFailures); reason != "" {
		return fail("connection-failed", "Could not connect to the library's repository: "+reason+". Check the repository address and your network connection.", nil)
	}
	if reason := r.gitReason(diagnostics, objectRefusals); reason != "" {
		return fail("object-fetch-refused", "The library's server refused to send a file by its object ID ("+reason+"), which Code Rules needs to read one version of each rule without downloading the whole repository. GitHub.com and GitLab.com allow it; ask the administrator of a self-hosted server to enable Git protocol version 2 or uploadpack.allowAnySHA1InWant.", nil)
	}
	lines := strings.Split(strings.TrimSpace(string(diagnostics)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if reason := r.quote(lines[i]); reason != "" {
			return fail(code, strings.TrimSuffix(problem, ".")+": "+reason+".", nil)
		}
	}
	return fail(code, problem, nil)
}

// gitReason returns the part of the first diagnostics line containing one of texts, from the earliest such text
// to the line's end, as quote returns it, or "" when no line contains one.
func (r *repository) gitReason(diagnostics []byte, texts []string) string {
	for line := range strings.SplitSeq(string(diagnostics), "\n") {
		start := -1
		for _, text := range texts {
			if at := strings.Index(line, text); at >= 0 && (start < 0 || at < start) {
				start = at
			}
		}
		if start >= 0 {
			return r.quote(line[start:])
		}
	}
	return ""
}

// quote returns a line of Git's diagnostics to show: without a fatal:, error:, or remote: prefix, credentials, or
// control characters, without trailing punctuation, and at most maxReasonRunes long.
func (r *repository) quote(line string) string {
	line = strings.TrimSpace(line)
	for _, prefix := range []string{"fatal:", "error:", "remote:"} {
		line = strings.TrimSpace(strings.TrimPrefix(line, prefix))
	}
	line = strings.Map(func(c rune) rune {
		if unicode.IsControl(c) {
			return -1
		}
		return c
	}, r.runner.Redact(line, r.url))
	line = strings.TrimRight(line, ".: ")
	if runes := []rune(line); len(runes) > maxReasonRunes {
		line = string(runes[:maxReasonRunes]) + "..."
	}
	return line
}
