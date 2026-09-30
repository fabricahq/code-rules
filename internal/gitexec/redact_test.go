// Exercise credential redaction by exact value and by pattern, in every position a secret can take in a line.

package gitexec

import (
	"slices"
	"strings"
	"testing"
)

// TestRedact_RemovesEveryFormOfEachSecret covers each exact value, as written, decoded, and URL-encoded, in and
// around other text, and each pattern. Removing either layer fails the cases that rely on it.
func TestRedact_RemovesEveryFormOfEachSecret(t *testing.T) {
	runner := Runner{environment: []string{"PATH=/bin", "GH_TOKEN=env-token-value", "GITHUB_TOKEN=plain-github-token", "GH_ENTERPRISE_TOKEN=enterprise/token", "GITHUB_ENTERPRISE_TOKEN=second-enterprise", "OTHER_TOKEN=keep-this-value"}}
	remote := "https://deploy-user:p%40ss%2Fword@example.com/acme/rules.git"
	for _, test := range []struct {
		name, text, want string
	}{
		{"URL user name alone", "user deploy-user", "user [redacted]"},
		{"URL password as written", "token p%40ss%2Fword here", "token [redacted] here"},
		{"URL password decoded", "token p@ss/word here", "token [redacted] here"},
		{"environment token at line start", "env-token-value was rejected", "[redacted] was rejected"},
		{"environment token at line end", "rejected env-token-value", "rejected [redacted]"},
		{"environment token inside other text", "xenv-token-valuex", "x[redacted]x"},
		{"several tokens in one line", "plain-github-token and second-enterprise and env-token-value", "[redacted] and [redacted] and [redacted]"},
		{"environment token decoded and encoded", "enterprise/token enterprise%2Ftoken", "[redacted] [redacted]"},
		{"unrelated variable stays", "keep-this-value", "keep-this-value"},
		{"credential URL of another host", "fetch https://someone:hunter22@other.example/x", "fetch https://[redacted]@other.example/x"},
		{"token as a URL user name", "ssh://ghx-token-like@host/repo", "ssh://[redacted]@host/repo"},
		{"URL without credentials stays", "https://example.com/acme/rules.git", "https://example.com/acme/rules.git"},
		{"GitHub classic token", "ghp_abcdefghijklmnop1234 leaked", "[redacted] leaked"},
		{"GitHub OAuth, server, user, and refresh tokens", "gho_abc1 ghs_abc2 ghu_abc3 ghr_abc4", "[redacted] [redacted] [redacted] [redacted]"},
		{"GitHub fine-grained token", "key=github_pat_11ABC_def123", "key=[redacted]"},
		{"GitLab token", "glpat-AbC_12-xyz end", "[redacted] end"},
		{"Authorization header", "Authorization: Basic dXNlcjpwYXNz\nnext line", "Authorization: [redacted]\nnext line"},
		{"lowercase authorization header", "authorization:token-value", "authorization:[redacted]"},
		{"Bearer token", "sent Bearer abc.def.ghi to the server", "sent Bearer [redacted] to the server"},
		{"text without secrets", "remote: rejected by policy", "remote: rejected by policy"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := runner.Credentials(remote).Redact(test.text); got != test.want {
				t.Fatalf("Redact(%q) = %q, want %q", test.text, got, test.want)
			}
		})
	}
}

// TestRedact_RemovesShortAndDifferentlyEncodedSecretsStandingAlone redacts every nonempty password and token by
// value, whatever its length, in every percent-encoding of it, outside any URL.
func TestRedact_RemovesShortAndDifferentlyEncodedSecretsStandingAlone(t *testing.T) {
	for _, test := range []struct {
		name, environment, remote, text, want string
	}{
		{"short password", "", "https://deploy:abc@example.com/rules.git", "remote: password abc rejected", "remote: password [redacted] rejected"},
		{"lowercase percent-encoding", "", "https://deploy-user:p%40ss%2Fword@example.com/rules.git", "remote: password p%40ss%2fword rejected", "remote: password [redacted] rejected"},
		{"mixed encoded and plain characters", "", "https://deploy-user:p%40ss%2Fword@example.com/rules.git", "token p@ss%2fword and p%40ss/word", "token [redacted] and [redacted]"},
		{"every character encoded", "", "https://deploy-user:abc@example.com/rules.git", "leaked %61%62%63 here", "leaked [redacted] here"},
		{"one-character token", "GH_TOKEN=q", "", "status q", "status [redacted]"},
		{"short token", "GITHUB_TOKEN=xy1", "", "xy1 expired", "[redacted] expired"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := Runner{environment: []string{test.environment}}
			remotes := []string{}
			if test.remote != "" {
				remotes = append(remotes, test.remote)
			}
			if got := runner.Credentials(remotes...).Redact(test.text); got != test.want {
				t.Fatalf("Redact(%q) = %q, want %q", test.text, got, test.want)
			}
		})
	}
}

// TestRedact_LeavesShortUserNamesToThePatterns doesn't shred text by replacing a short user name, such as git, but
// still removes it from a URL.
func TestRedact_LeavesShortUserNamesToThePatterns(t *testing.T) {
	runner := Runner{}
	got := runner.Credentials("https://git:pw@example.com/rules.git").Redact("the git server at https://git:pw@example.com refused")
	if want := "the git server at https://[redacted]@example.com refused"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if strings.Contains(runner.Credentials("https://git@example.com/rules.git").Redact("git"), Redacted) {
		t.Fatal("redacted a three-byte user name by value")
	}
}

// TestLines_FindsKnownSecretsHoweverTheyAreSplit reports a token split by control characters, which it removes
// before redacting, and one wrapped across lines, which redacting each line can't remove.
func TestLines_FindsKnownSecretsHoweverTheyAreSplit(t *testing.T) {
	credentials := Runner{environment: []string{"GH_TOKEN=opaque-secret-value"}}.Credentials()
	for _, test := range []struct {
		name         string
		lines, safe  []string
		known, split bool
	}{
		{"no secret", []string{"rejected by policy"}, []string{"rejected by policy"}, false, false},
		{"secret on one line", []string{" token opaque-secret-value "}, []string{"token [redacted]"}, true, false},
		{"secret split by a tab", []string{"token opaque-\tsecret-value"}, []string{"token [redacted]"}, true, false},
		{"secret split by an escape sequence", []string{"opaque-\x1b[Ksecret-value"}, []string{"opaque-[Ksecret-value"}, false, false},
		{"secret wrapped without a space", []string{"token opaque-", "secret-value"}, []string{"token opaque-", "secret-value"}, true, true},
		{"secret wrapped at a space", []string{"token opaque-secret", "-value"}, []string{"token opaque-secret", "-value"}, true, true},
		{"percent-encoded secret wrapped", []string{"opaque%2d", "secret-value"}, []string{"opaque%2d", "secret-value"}, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			safe, known, split := credentials.Lines(test.lines)
			if !slices.Equal(safe, test.safe) || known != test.known || split != test.split {
				t.Fatalf("got %q, known %v, split %v; want %q, %v, %v", safe, known, split, test.safe, test.known, test.split)
			}
		})
	}
}

// TestShorten_CutsLongTextAtARuneBoundary and leaves text within the limit unchanged.
func TestShorten_CutsLongTextAtARuneBoundary(t *testing.T) {
	if got := Shorten("ééé", 2); got != "éé..." {
		t.Fatalf("got %q", got)
	}
	if got := Shorten("abc", 3); got != "abc" {
		t.Fatalf("got %q", got)
	}
}
