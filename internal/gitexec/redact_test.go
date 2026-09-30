// Exercise credential redaction by exact value and by pattern, in every position a secret can take in a line.

package gitexec

import (
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
			if got := runner.Redact(test.text, remote); got != test.want {
				t.Fatalf("Redact(%q) = %q, want %q", test.text, got, test.want)
			}
		})
	}
}

// TestRedact_LeavesShortValuesToThePatterns doesn't shred text by replacing a short user name, such as git, but
// still removes it from a URL.
func TestRedact_LeavesShortValuesToThePatterns(t *testing.T) {
	runner := Runner{}
	got := runner.Redact("the git server at https://git:pw@example.com refused", "https://git:pw@example.com/rules.git")
	if want := "the git server at https://[redacted]@example.com refused"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if strings.Contains(runner.Redact("git", "https://git@example.com/rules.git"), Redacted) {
		t.Fatal("redacted a three-byte user name by value")
	}
}
