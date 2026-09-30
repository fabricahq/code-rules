// Redact credentials from Git's diagnostics before any of their text is shown.

package gitexec

import (
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// Redacted replaces each credential Redact removes.
const Redacted = "[redacted]"

// tokenVariables name the environment variables whose values the GitHub CLI and Git credential helpers commonly
// read as access tokens.
var tokenVariables = []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"}

// minSecretBytes is the shortest value Redact replaces by exact value. Replacing every occurrence of a shorter
// user name, such as git, would garble the text, and the patterns still remove it from URLs.
const minSecretBytes = 4

// secretPatterns match common credential formats, as a second layer after exact values: user information in any
// URL, such as https://user:token@host; GitHub and GitLab token formats; Authorization header values; and Bearer
// tokens.
var secretPatterns = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.-]*://)[^/\s@]+@`), "${1}" + Redacted + "@"},
	{regexp.MustCompile(`(?i)(authorization:[ \t]*)[^\r\n]+`), "${1}" + Redacted},
	{regexp.MustCompile(`(?i)(\bbearer[ \t]+)[^\s"']+`), "${1}" + Redacted},
	{regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+|glpat-[A-Za-z0-9_-]+)`), Redacted},
}

// Redact returns text with credentials replaced by [redacted]. It first replaces, by exact value, the user name
// and password in each of remotes' URLs, as written, decoded, and URL-encoded, and the values of GH_TOKEN,
// GITHUB_TOKEN, GH_ENTERPRISE_TOKEN, and GITHUB_ENTERPRISE_TOKEN in the runner's environment; values shorter than
// four bytes are left to the patterns. It then replaces common credential formats by pattern: user information in
// any URL, GitHub token prefixes such as ghp_ and github_pat_, GitLab's glpat-, Authorization header values, and
// Bearer tokens.
func (r Runner) Redact(text string, remotes ...string) string {
	return redact(text, secrets(r.environment, remotes))
}

// secrets returns the exact values Redact removes, longest first so a value containing another is removed whole.
func secrets(environment, remotes []string) []string {
	values := []string{}
	for _, item := range environment {
		name, value, _ := strings.Cut(item, "=")
		if slices.Contains(tokenVariables, name) {
			values = append(values, value)
		}
	}
	for _, remote := range remotes {
		values = append(values, userInformation(remote)...)
	}
	result := []string{}
	for _, value := range values {
		for _, form := range []string{value, url.PathEscape(value), url.QueryEscape(value)} {
			if len(form) >= minSecretBytes && !slices.Contains(result, form) {
				result = append(result, form)
			}
		}
	}
	slices.SortStableFunc(result, func(a, b string) int { return len(b) - len(a) })
	return result
}

// userInformation returns the user name and password in a URL's user information, each as written and decoded, or
// nothing when the URL has none.
func userInformation(remote string) []string {
	scheme, rest, ok := strings.Cut(remote, "://")
	if !ok || scheme == "" {
		return nil
	}
	authority, _, _ := strings.Cut(rest, "/")
	at := strings.LastIndex(authority, "@")
	if at < 0 {
		return nil
	}
	values := []string{}
	for _, part := range strings.SplitN(authority[:at], ":", 2) {
		values = append(values, part)
		if decoded, err := url.PathUnescape(part); err == nil {
			values = append(values, decoded)
		}
	}
	return values
}

// redact replaces each of secrets in text, then each secret pattern.
func redact(text string, secrets []string) string {
	for _, secret := range secrets {
		text = strings.ReplaceAll(text, secret, Redacted)
	}
	for _, secret := range secretPatterns {
		text = secret.pattern.ReplaceAllString(text, secret.replacement)
	}
	return text
}
