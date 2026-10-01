// Decide whether text from Git, a Git server, or the GitHub CLI may be shown: withhold the whole text when it could
// reveal a known credential in any spelling, and otherwise redact common credential formats.

package gitexec

import (
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Redacted replaces each credential that RedactFormats removes.
const Redacted = "[redacted]"

// Withheld stands in for Git's text that Show withheld.
const Withheld = "Git's message was withheld because it contained a credential"

// tokenVariables name the environment variables whose values the GitHub CLI and Git credential helpers commonly
// read as access tokens.
var tokenVariables = []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"}

// maxCheckedBytes bounds the text Show checks; longer text is withheld unchecked, so checking stays linear.
const maxCheckedBytes = 64 << 10

// windowRunes is the length of the pieces of a known credential that Show looks for, so a fragment of a credential
// wrapped or split anywhere is found once it is this long. A credential shorter than this is looked for whole.
const windowRunes = 8

// decodingRounds bounds how many layers of percent-encoding, backslash escapes, and terminal sequences Show undoes.
const decodingRounds = 3

// secretPatterns match common credential formats that aren't known values: user information in any URL, up to the
// last @ of its authority, such as https://user:token@host; GitHub and GitLab token formats; Authorization header
// values; and Bearer tokens.
var secretPatterns = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.-]*://)[^/\s]*@`), "${1}" + Redacted + "@"},
	{regexp.MustCompile(`(?i)(authorization:[ \t]*)[^\r\n]+`), "${1}" + Redacted},
	{regexp.MustCompile(`(?i)(\bbearer[ \t]+)[^\s"']+`), "${1}" + Redacted},
	{regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+|glpat-[A-Za-z0-9_-]+)`), Redacted},
}

// brokenCredentialURL matches a URL whose authority runs to the end of a line and whose next line continues it up
// to an @, as a credential-bearing URL wrapped across lines looks.
var brokenCredentialURL = regexp.MustCompile(`[A-Za-z][A-Za-z0-9+.-]*://[^/\s@]*[\r\n]+(?:[ \t]*remote:)?[ \t]*[^/\s]*@`)

// Credentials are the credentials a runner's text may reveal, as matching skeletons.
type Credentials struct {
	skeletons []string
}

// Credentials returns the known credentials: the values of GH_TOKEN, GITHUB_TOKEN, GH_ENTERPRISE_TOKEN, and
// GITHUB_ENTERPRISE_TOKEN in the runner's environment, and the password in each of remotes' URLs, with the user name
// too for HTTP and HTTPS, where a user name can be a token. An SSH user name names an account and isn't secret.
func (r Runner) Credentials(remotes ...string) Credentials {
	values := []string{}
	for _, item := range r.environment {
		name, value, _ := strings.Cut(item, "=")
		if slices.Contains(tokenVariables, name) {
			values = append(values, value)
		}
	}
	for _, remote := range remotes {
		values = append(values, userInformation(remote)...)
	}
	credentials := Credentials{}
	for _, value := range values {
		if skeleton := Skeleton(value); skeleton != "" && !slices.Contains(credentials.skeletons, skeleton) {
			credentials.skeletons = append(credentials.skeletons, skeleton)
		}
	}
	return credentials
}

// Show returns text to show, and true, or "" and false when it must be withheld whole. It decides once, on the whole
// text, before any line is selected or cut: text over 64 KiB is withheld unchecked; text whose skeleton contains a
// known credential's skeleton, or for a credential of eight or more letters and digits any eight of them in a row,
// is withheld, so no spelling, escaping, or wrapping of a credential and no fragment of eight or more characters
// can show; and text with a credential-bearing URL wrapped across lines is withheld. Shown text has common
// credential formats redacted, as RedactFormats does. Callers then escape control characters for terminals, and
// select and shorten lines.
func (c Credentials) Show(text string) (string, bool) {
	if len(text) > maxCheckedBytes || brokenCredentialURL.MatchString(text) {
		return "", false
	}
	skeleton := Skeleton(text)
	for _, secret := range c.skeletons {
		runes := []rune(secret)
		if len(runes) < windowRunes {
			if strings.Contains(skeleton, secret) {
				return "", false
			}
			continue
		}
		for start := 0; start+windowRunes <= len(runes); start++ {
			if strings.Contains(skeleton, string(runes[start:start+windowRunes])) {
				return "", false
			}
		}
	}
	return RedactFormats(text), true
}

// RedactFormats returns text with common credential formats replaced by [redacted]: user information in URLs,
// GitHub and GitLab tokens, Authorization header values, and Bearer tokens.
func RedactFormats(text string) string {
	for _, secret := range secretPatterns {
		text = secret.pattern.ReplaceAllString(text, secret.replacement)
	}
	return text
}

// Skeleton returns the form of text that Show compares: up to three rounds of decoding percent-encoding and
// backslash escapes such as s and \x73 and removing terminal escape sequences, then Unicode NFKC normalization
// and lowercasing, keeping only letters and digits. Equivalent spellings of a value share a skeleton.
func Skeleton(text string) string {
	for range decodingRounds {
		decoded := terminalSequences.ReplaceAllString(decodeBackslashes(decodePercent(text)), "")
		if decoded == text {
			break
		}
		text = decoded
	}
	text = terminalSequences.ReplaceAllString(text, "")
	text = strings.ToLower(norm.NFKC.String(text))
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, text)
}

// terminalSequences match complete terminal escape sequences: CSI sequences such as ESC [ 31 m and the 8-bit CSI,
// OSC strings ended by BEL or ESC \, and other two-character escapes.
var terminalSequences = regexp.MustCompile("(?:\x1b\\[|\u009b)[0-?]*[ -/]*[@-~]|\x1b\\][^\x07\x1b]*(?:\x07|\x1b\\\\)?|\x1b[ -~]")

// percentEscape matches one percent-encoded byte.
var percentEscape = regexp.MustCompile(`%[0-9A-Fa-f]{2}`)

// decodePercent decodes each valid percent-encoded byte, leaving other percent signs as they are.
func decodePercent(text string) string {
	return percentEscape.ReplaceAllStringFunc(text, func(escape string) string {
		value, _ := strconv.ParseUint(escape[1:], 16, 8)
		return string([]byte{byte(value)})
	})
}

// backslashEscape matches \uXXXX, \UXXXXXXXX, \u{X...}, and \xXX escapes.
var backslashEscape = regexp.MustCompile(`\\(?:u\{[0-9A-Fa-f]{1,6}\}|u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8}|x[0-9A-Fa-f]{2})`)

// decodeBackslashes decodes each backslash escape of a character code; \xXX decodes to that byte.
func decodeBackslashes(text string) string {
	return backslashEscape.ReplaceAllStringFunc(text, func(escape string) string {
		digits := strings.Trim(escape[2:], "{}")
		value, err := strconv.ParseUint(digits, 16, 32)
		if err != nil {
			return escape
		}
		if escape[1] == 'x' {
			return string([]byte{byte(value)})
		}
		return string(rune(value))
	})
}

// userInformation returns the password in a URL's user information, as written and decoded, and, for an HTTP or
// HTTPS URL, the user name the same way. It returns nothing for an address without user information.
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
	name, secret, hasSecret := strings.Cut(authority[:at], ":")
	forms := func(value string) []string {
		result := []string{value}
		if decoded, err := url.PathUnescape(value); err == nil {
			result = append(result, decoded)
		}
		return result
	}
	values := []string{}
	if hasSecret {
		values = append(values, forms(secret)...)
	}
	if strings.EqualFold(scheme, "http") || strings.EqualFold(scheme, "https") {
		values = append(values, forms(name)...)
	}
	return values
}

// Shorten returns text cut to at most maxRunes runes, with ... marking a cut. Cut only text Show returned.
func Shorten(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "..."
}

// Printable returns text without terminal escape sequences or other control characters, for deciding what a line
// says; it doesn't make text safe to show, which only Show decides.
func Printable(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, terminalSequences.ReplaceAllString(text, ""))
}

// WithoutPrefix returns a diagnostic line without the label Git starts it with, such as fatal:, and the label, or
// "" when it has none.
func WithoutPrefix(line string) (text, prefix string) {
	line = strings.TrimSpace(line)
	for _, prefix := range []string{"fatal:", "error:", "warning:", "hint:", "remote:"} {
		if text, ok := strings.CutPrefix(line, prefix); ok {
			return strings.TrimSpace(text), prefix
		}
	}
	return line, ""
}
