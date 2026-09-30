// Make Git's diagnostics safe to show: remove control characters, find known credentials however they're encoded or
// split across lines, and redact them and common credential formats.

package gitexec

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Redacted replaces each credential that redaction removes.
const Redacted = "[redacted]"

// tokenVariables name the environment variables whose values the GitHub CLI and Git credential helpers commonly
// read as access tokens.
var tokenVariables = []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"}

// minNameBytes is the shortest user name redacted by exact value. A user name identifies an account rather than
// granting access, and replacing every occurrence of a short one, such as git, would garble the text; the URL
// pattern still removes it from URLs. Passwords and tokens are redacted whatever their length.
const minNameBytes = 4

// secretPatterns match common credential formats, as a second layer after known values: user information in any
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

// Credentials are the credentials a runner's diagnostics may contain: passwords and tokens, which grant access, and
// user names. Each matches its value in any percent-encoding, such as p@ss, p%40ss, or %70%40ss.
type Credentials struct {
	secrets []*regexp.Regexp
	names   []*regexp.Regexp
}

// Credentials returns the known credentials: the values of GH_TOKEN, GITHUB_TOKEN, GH_ENTERPRISE_TOKEN, and
// GITHUB_ENTERPRISE_TOKEN in the runner's environment, and the passwords and user names in remotes' URLs.
func (r Runner) Credentials(remotes ...string) Credentials {
	secrets, names := []string{}, []string{}
	for _, item := range r.environment {
		name, value, _ := strings.Cut(item, "=")
		if slices.Contains(tokenVariables, name) {
			secrets = append(secrets, value)
		}
	}
	for _, remote := range remotes {
		user, password := userInformation(remote)
		secrets = append(secrets, password...)
		for _, name := range user {
			if len(name) >= minNameBytes {
				names = append(names, name)
			}
		}
	}
	return Credentials{secrets: valuePatterns(secrets), names: valuePatterns(names)}
}

// Redact returns text with each known credential, in any percent-encoding, and each common credential format
// replaced by [redacted]. It doesn't remove control characters; Lines does that first.
func (c Credentials) Redact(text string) string {
	for _, pattern := range append(slices.Clone(c.secrets), c.names...) {
		text = pattern.ReplaceAllString(text, Redacted)
	}
	for _, secret := range secretPatterns {
		text = secret.pattern.ReplaceAllString(text, secret.replacement)
	}
	return text
}

// Lines makes lines of Git's diagnostics safe to show, in order: it removes control characters, which could
// otherwise split a credential that removing them later would join, and surrounding space from each line; checks
// the result for known passwords and tokens, within lines and across line boundaries, as when a server wraps text;
// then redacts each line. It returns the redacted lines, in the same order and number; known when a password or
// token appeared anywhere; and split when one appeared only across a line boundary, which redacting each line can't
// remove, so the caller must not show the lines. Callers shorten lines only after this, so truncation never cuts a
// credential.
func (c Credentials) Lines(lines []string) (safe []string, known, split bool) {
	normalized := make([]string, len(lines))
	for i, line := range lines {
		normalized[i] = strings.TrimSpace(strings.Map(func(r rune) rune {
			if unicode.IsControl(r) {
				return -1
			}
			return r
		}, line))
	}
	for _, pattern := range c.secrets {
		for _, line := range normalized {
			known = known || pattern.MatchString(line)
		}
		// A server can wrap text with or without a space where the line broke.
		for _, separator := range []string{"", " "} {
			if crossesBoundary(pattern, normalized, separator) {
				known, split = true, true
			}
		}
	}
	safe = make([]string, len(normalized))
	for i, line := range normalized {
		safe[i] = c.Redact(line)
	}
	return safe, known, split
}

// crossesBoundary reports whether pattern matches the lines joined with separator somewhere that spans the end of
// one line.
func crossesBoundary(pattern *regexp.Regexp, lines []string, separator string) bool {
	joined := strings.Join(lines, separator)
	ends := []int{}
	offset := 0
	for _, line := range lines {
		offset += len(line)
		ends = append(ends, offset)
		offset += len(separator)
	}
	for _, match := range pattern.FindAllStringIndex(joined, -1) {
		for _, end := range ends {
			if match[0] < end && end < match[1] {
				return true
			}
		}
	}
	return false
}

// gitPrefixes are the labels Git starts its diagnostic lines with.
var gitPrefixes = []string{"fatal:", "error:", "warning:", "hint:", "remote:"}

// WithoutPrefix returns a diagnostic line without the label Git starts it with, such as fatal:, and the label, or
// "" when it has none. Checking text that wraps across lines needs the lines without their labels.
func WithoutPrefix(line string) (text, prefix string) {
	line = strings.TrimSpace(line)
	for _, prefix := range gitPrefixes {
		if text, ok := strings.CutPrefix(line, prefix); ok {
			return strings.TrimSpace(text), prefix
		}
	}
	return line, ""
}

// Shorten returns text cut to at most maxRunes runes, with ... marking a cut. Cut only text Lines made safe.
func Shorten(text string, maxRunes int) string {
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	return string([]rune(text)[:maxRunes]) + "..."
}

// userInformation returns the user name and password in a URL's user information, each as written and decoded, or
// nothing when the URL has none.
func userInformation(remote string) (user, password []string) {
	scheme, rest, ok := strings.Cut(remote, "://")
	if !ok || scheme == "" {
		return nil, nil
	}
	authority, _, _ := strings.Cut(rest, "/")
	at := strings.LastIndex(authority, "@")
	if at < 0 {
		return nil, nil
	}
	name, secret, hasSecret := strings.Cut(authority[:at], ":")
	forms := func(value string) []string {
		result := []string{value}
		if decoded, err := url.PathUnescape(value); err == nil {
			result = append(result, decoded)
		}
		return result
	}
	user = forms(name)
	if hasSecret {
		password = forms(secret)
	}
	return user, password
}

// valuePatterns returns, longest value first so a value containing another is replaced whole, a pattern for each
// distinct nonempty value that matches it with each character written plainly or percent-encoded in either case,
// and a space also as +.
func valuePatterns(values []string) []*regexp.Regexp {
	distinct := []string{}
	for _, value := range values {
		if value != "" && !slices.Contains(distinct, value) {
			distinct = append(distinct, value)
		}
	}
	slices.SortStableFunc(distinct, func(a, b string) int { return len(b) - len(a) })
	patterns := []*regexp.Regexp{}
	for _, value := range distinct {
		var expression strings.Builder
		for _, r := range value {
			alternatives := []string{regexp.QuoteMeta(string(r))}
			if r == ' ' {
				alternatives = append(alternatives, `\+`)
			}
			var encoded strings.Builder
			for _, b := range []byte(string(r)) {
				encoded.WriteString("%" + caseless(fmt.Sprintf("%02x", b)))
			}
			alternatives = append(alternatives, encoded.String())
			expression.WriteString("(?:" + strings.Join(alternatives, "|") + ")")
		}
		patterns = append(patterns, regexp.MustCompile(expression.String()))
	}
	return patterns
}

// caseless returns a pattern matching hex digits in either case, such as 2[fF] for 2f.
func caseless(hex string) string {
	var out strings.Builder
	for _, digit := range hex {
		if digit >= 'a' && digit <= 'f' {
			out.WriteString("[" + string(digit) + string(unicode.ToUpper(digit)) + "]")
		} else {
			out.WriteRune(digit)
		}
	}
	return out.String()
}
