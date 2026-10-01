// Share the Git runner's failure type, so every import failure carries one stable code, and explain Git's failures
// from its diagnostics without showing credentials.

package imports

import (
	"strings"

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
	view := r.view(diagnostics)
	if view.matches(connectionFailures) {
		return fail("connection-failed", "Could not connect to the library's repository"+view.reason(connectionFailures, ": ", "")+". Check the repository address and your network connection.", nil)
	}
	return fail("not-found-or-no-access", "Repository not found or no access; check its address and Git credentials.", nil)
}

// gitFailure explains a Git command that failed while reading the library: a host it couldn't reach, with code
// connection-failed; a server that refused to send files by object ID, with code object-fetch-refused; or
// otherwise problem, with code, followed by Git's last error line when it printed one.
func (r *repository) gitFailure(code, problem string, diagnostics []byte) error {
	view := r.view(diagnostics)
	if view.matches(connectionFailures) {
		return fail("connection-failed", "Could not connect to the library's repository"+view.reason(connectionFailures, ": ", "")+". Check the repository address and your network connection.", nil)
	}
	if view.matches(objectRefusals) {
		return fail("object-fetch-refused", "The library's server refused to send a file by its object ID"+view.reason(objectRefusals, " (", ")")+", which Code Rules needs to read one version of each rule without downloading the whole repository. GitHub.com and GitLab.com allow it; ask the administrator of a self-hosted server to enable Git protocol version 2 or uploadpack.allowAnySHA1InWant.", nil)
	}
	if view.withheld {
		return fail(code, strings.TrimSuffix(problem, ".")+". "+gitexec.Withheld+".", nil)
	}
	for i := len(view.shown) - 1; i >= 0; i-- {
		if reason := quote(view.shown[i]); reason != "" {
			return fail(code, strings.TrimSuffix(problem, ".")+": "+reason+".", nil)
		}
	}
	return fail(code, problem, nil)
}

// diagnosticView holds two views of Git's diagnostics. private, never shown, holds the lines without terminal
// sequences, control characters, or Git's labels, to decide which failure they describe. shown holds the lines
// gitexec's Show allows showing, without Git's labels, or none when it withheld them.
type diagnosticView struct {
	private, shown []string
	withheld       bool
}

// view returns the views of diagnostics, with the credentials of the runner's environment and of the library's
// address, as configured and as Git rewrites it, deciding what may be shown.
func (r *repository) view(diagnostics []byte) diagnosticView {
	view := diagnosticView{}
	for line := range strings.SplitSeq(gitexec.Printable(string(diagnostics)), "\n") {
		text, _ := gitexec.WithoutPrefix(line)
		view.private = append(view.private, text)
	}
	// The decision reads the lines without the labels Git adds, which would otherwise separate wrapped text.
	_, ok := r.runner.Credentials(r.url, r.effective).Show(strings.Join(view.private, "\n"))
	view.withheld = !ok
	if ok {
		for _, text := range view.private {
			view.shown = append(view.shown, gitexec.RedactFormats(text))
		}
	}
	return view
}

// matches reports whether a private line contains one of texts.
func (v diagnosticView) matches(texts []string) bool {
	_, found := reasonAfter(v.private, texts)
	return found
}

// reason returns the shown text from the first of texts to its line's end, between before and after; a note that
// Git's message was withheld, between the same; or "" when no shown line has one of texts.
func (v diagnosticView) reason(texts []string, before, after string) string {
	if v.withheld {
		return before + gitexec.Withheld + after
	}
	if reason, found := reasonAfter(v.shown, texts); found && reason != "" {
		return before + reason + after
	}
	return ""
}

// reasonAfter returns the part of the first of lines containing one of texts, from the earliest such text to the
// line's end, as quote returns it, and whether any line contains one.
func reasonAfter(lines, texts []string) (string, bool) {
	for _, line := range lines {
		start := -1
		for _, text := range texts {
			if at := strings.Index(line, text); at >= 0 && (start < 0 || at < start) {
				start = at
			}
		}
		if start >= 0 {
			return quote(line[start:]), true
		}
	}
	return "", false
}

// quote returns a shown line of Git's diagnostics without its trailing punctuation and at most maxReasonRunes long.
func quote(line string) string {
	return strings.TrimRight(gitexec.Shorten(strings.TrimRight(line, ".: "), maxReasonRunes), ".: ")
}
