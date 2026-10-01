// Classify text from Git, a Git server, or the GitHub CLI privately, so callers explain a failure with their own
// static message and never show the text.

package gitexec

import (
	"regexp"
	"strings"
	"unicode"
)

// ConnectionFailures are the texts Git and its transports print when they can't reach a host at all, such as when
// the host name doesn't resolve, nothing listens, or TLS fails, as opposed to a server that answers and refuses.
var ConnectionFailures = []string{
	"could not resolve host", "couldn't resolve host", "could not resolve hostname", "temporary failure in name resolution",
	"nodename nor servname provided", "name or service not known", "failed to connect", "could not connect to server",
	"connection refused", "connection timed out", "operation timed out", "connection reset", "network is unreachable",
	"no route to host", "ssl certificate problem", "ssl connect error", "host key verification failed",
}

// AccessFailures are the texts Git and Git servers print when a server refuses the credentials or their access,
// such as GitHub's "Permission to OWNER/REPO denied to USER" and SSH's "Permission denied (publickey)".
var AccessFailures = []string{
	"permission denied", "permission to", "authentication failed", "access denied", "returned error: 401",
	"returned error: 403", "could not read username", "could not read password", "not allowed to push",
	"insufficient permission",
}

// Mentions reports whether text contains any of phrases, ignoring case, terminal escape sequences, and other
// control characters, so text that splits a phrase with them still matches. Phrases are lowercase. Use the answer
// only to choose a static message: the text itself must never reach output, logs, or JSON.
func Mentions(text []byte, phrases ...string) bool {
	plain := strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, terminalSequences.ReplaceAllString(string(text), "")))
	for _, phrase := range phrases {
		if strings.Contains(plain, phrase) {
			return true
		}
	}
	return false
}

// terminalSequences match complete terminal escape sequences: CSI sequences such as ESC [ 31 m and the 8-bit CSI,
// OSC strings ended by BEL or ESC \, and other two-character escapes.
var terminalSequences = regexp.MustCompile("(?:\x1b\\[|\u009b)[0-?]*[ -/]*[@-~]|\x1b\\][^\x07\x1b]*(?:\x07|\x1b\\\\)?|\x1b[ -~]")
