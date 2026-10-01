// Exercise the decision to show or withhold text that could reveal a known credential, and the redaction of common
// credential formats in text that is shown.

package gitexec

import (
	"strings"
	"testing"
)

// TestShow_WithholdsTextRevealingAKnownCredentialInAnySpelling covers each spelling, escaping, and wrapping of a
// known token, a remote's password and HTTPS user name, and fragments of eight or more characters, and shows text
// that reveals none. Removing the decision fails every withheld case.
func TestShow_WithholdsTextRevealingAKnownCredentialInAnySpelling(t *testing.T) {
	for _, test := range []struct {
		name, token, remote, text string
		shown                     bool
	}{
		{"no credential", "opaque-secret-value", "", "remote: rejected by policy", true},
		{"token as written", "opaque-secret-value", "", "token opaque-secret-value", false},
		{"percent-encoded once", "opaque-secret-value", "", "opaque%2dsecret%2Dvalue", false},
		{"percent-encoded twice", "opaque-secret-value", "", "opaque%252dsecret-value", false},
		{"every character percent-encoded", "abc", "", "leaked %61%62%63 here", false},
		{"erase-line sequence inside", "opaque-secret-value", "", "opaque-\x1b[Ksecret-value", false},
		{"color sequences inside", "opaque-secret-value", "", "opaque-\x1b[31msecret-value\x1b[0m", false},
		{"escaped color sequence inside", "opaque-secret-value", "", `opaque-\x1b[31msecret-value`, false},
		{"zero-width space inside", "opaque-secret-value", "", "opaque-​secret-value", false},
		{"unicode escape", "opaque-secret-value", "", `opaque-secret-value`, false},
		{"decomposed accent", "café-secret-value", "", "café-secret-value", false},
		{"compatibility characters", "opaque-secret-value", "", "ｏｐａｑｕｅ-secret-value", false},
		{"uppercase", "opaque-secret-value", "", "OPAQUE-SECRET-VALUE", false},
		{"wrapped differently at each boundary", "alpha beta-gamma", "", "alpha\nbeta-\ngamma", false},
		{"prefix inside the token", "opaque-secret-value", "", "remote: opaque-\nremote: remote: secret-value", false},
		{"fragment of eight characters", "opaque-secret-value", "", "token ...ecretval...", false},
		{"fragment of seven characters", "opaque-secret-value", "", "token ...cretval...", true},
		{"short token whole", "on", "", "Connection refused", false},
		{"short token absent", "xy1", "", "Connection refused", true},
		{"remote password", "", "ssh://reader:pa%24%24word@example.com/rules", "password pa$$word rejected", false},
		{"HTTPS user name token", "", "https://opaque-user-token@example.com/rules", "user-token rejected", false},
		{"SSH user name", "", "ssh://deploy-account@example.com/rules", "user deploy-account", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			credentials := Runner{environment: []string{"GH_TOKEN=" + test.token}}.Credentials(test.remote)
			if _, shown := credentials.Show(test.text); shown != test.shown {
				t.Fatalf("Show(%q) shown %v, want %v", test.text, shown, test.shown)
			}
		})
	}
}

// TestShow_WithholdsTextTooLargeToCheckAndAWrappedCredentialURL without checking it, so the decision is bounded,
// and a URL whose user information continues on the next line.
func TestShow_WithholdsTextTooLargeToCheckAndAWrappedCredentialURL(t *testing.T) {
	credentials := Runner{environment: []string{"GH_TOKEN=opaque-secret-value"}}.Credentials()
	if _, shown := credentials.Show(strings.Repeat("token opaque-secret-value\n", 1<<20/26)); shown {
		t.Fatal("showed a megabyte of text")
	}
	if _, shown := credentials.Show(strings.Repeat("x", maxCheckedBytes+1)); shown {
		t.Fatal("showed text over the limit")
	}
	if _, shown := credentials.Show(strings.Repeat("x", maxCheckedBytes)); !shown {
		t.Fatal("withheld text at the limit")
	}
	if _, shown := credentials.Show("Clone https://someone:hunt\nremote: er22@mirror.invalid/rules"); shown {
		t.Fatal("showed a credential URL wrapped across lines")
	}
}

// TestShow_RedactsCommonCredentialFormatsInTextItShows, up to the last @ of a URL's authority.
func TestShow_RedactsCommonCredentialFormatsInTextItShows(t *testing.T) {
	for _, test := range []struct{ text, want string }{
		{"fetch https://someone:hunter22@other.example/x", "fetch https://[redacted]@other.example/x"},
		{"clone https://someone:p@ssword@mirror.invalid/rules", "clone https://[redacted]@mirror.invalid/rules"},
		{"https://example.com/acme/rules.git", "https://example.com/acme/rules.git"},
		{"ghp_abcdefghijklmnop1234 leaked", "[redacted] leaked"},
		{"gho_abc1 ghs_abc2 ghu_abc3 ghr_abc4", "[redacted] [redacted] [redacted] [redacted]"},
		{"key=github_pat_11ABC_def123", "key=[redacted]"},
		{"glpat-AbC_12-xyz end", "[redacted] end"},
		{"Authorization: Basic dXNlcjpwYXNz\nnext line", "Authorization: [redacted]\nnext line"},
		{"sent Bearer abc.def.ghi to the server", "sent Bearer [redacted] to the server"},
	} {
		if got, shown := (Credentials{}).Show(test.text); !shown || got != test.want {
			t.Errorf("Show(%q) = %q, %v; want %q", test.text, got, shown, test.want)
		}
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
