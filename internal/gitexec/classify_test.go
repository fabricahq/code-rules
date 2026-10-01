// Check how text from Git is classified without being shown.

package gitexec

import "testing"

// TestMentions_IgnoresCaseTerminalSequencesAndControlCharacters inside a phrase, and finds nothing that isn't there.
func TestMentions_IgnoresCaseTerminalSequencesAndControlCharacters(t *testing.T) {
	for _, test := range []struct {
		text  string
		found bool
	}{
		{"fatal: Connection refused", true},
		{"fatal: CONNECTION REFUSED", true},
		{"fatal: Connection \x1b[31mrefused\x1b[0m", true},
		{"fatal: Connection\x1b]0;title\x07 refused", true},
		{"fatal: Connec\ttion refused", true},
		{"fatal: Connection accepted", false},
		{"", false},
	} {
		if found := Mentions([]byte(test.text), "connection refused"); found != test.found {
			t.Errorf("Mentions(%q) = %v, want %v", test.text, found, test.found)
		}
	}
}
