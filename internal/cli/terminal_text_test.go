// Check that human output escapes terminal control characters from libraries and other untrusted text.

package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// TestTerminalText_EscapesControlCharactersAndKeepsNewlines covers each kind of character a terminal could act on.
func TestTerminalText_EscapesControlCharactersAndKeepsNewlines(t *testing.T) {
	for input, want := range map[string]string{
		"":                               "",
		"Plain text, é, and ✓.\nNext.\n": "Plain text, é, and ✓.\nNext.\n",
		"\x1b[2J\x1b[HNo risky changes.": `\x1b[2J\x1b[HNo risky changes.`,
		"a\rb\tc\x00d\x7fe":              `a\x0db\x09c\x00d\x7fe`,
		"\u009b2J\u0085":                 `\u009b2J\u0085`,
		"bad \xff\x9b byte":              `bad \xff\x9b byte`,
	} {
		if got := terminalText(input); got != want {
			t.Errorf("terminalText(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestUpdate_EscapesControlCharactersFromTheLibrary shows a library release's summary with an escape sequence, and
// a diagnostic naming an argument with one, as visible text through the real binary, so a library can't clear or
// rewrite the preview people approve.
func TestUpdate_EscapesControlCharactersFromTheLibrary(t *testing.T) {
	ctx := context.Background()
	f, err := gitfixture.New(ctx, map[string][]byte{
		"rule-library.yaml":    []byte(`{"formatVersion":1}`),
		"techs/go/_group.yaml": []byte(`{"name":"Go","description":"Go guidance.","whenToRead":"When writing Go."}`),
		"techs/go/errors.md":   updateRule("errors 1.0.0"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if err := f.Release(ctx, 1, "release: 1\nrules:\n  techs/go/errors: 1.0.0\nchanges:\n  techs/go/errors: {change: new, summary: Add the rule.}\n"); err != nil {
		t.Fatal(err)
	}
	u := updateFixture{binary: buildCLI(t), directory: t.TempDir(), fixture: f}
	if out, diagnostic, code := runCLI(t, u.binary, u.directory, "project", "init"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	u.write(t, "config.yaml", "schemaVersion: 1\nsources:\n  team:\n    repository: "+f.Repository+"\n    groups:\n      - techs/go\n")
	if out, diagnostic, code := u.run(t, "project", "sync"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	if _, err := f.Commit(ctx, f.Worktree(), "Second release", map[string][]byte{"techs/go/errors.md": updateRule("errors 1.1.0")}); err != nil {
		t.Fatal(err)
	}
	if err := f.Release(ctx, 2, "release: 2\nrules:\n  techs/go/errors: 1.1.0\nchanges:\n  techs/go/errors: {change: minor, from: 1.0.0, summary: \"\\e[2J\\e[HNo risky changes.\\x9b\"}\n"); err != nil {
		t.Fatal(err)
	}
	out, diagnostic, code := u.run(t, "project", "update")
	if code != 0 || strings.ContainsAny(out+diagnostic, "\x1b\u009b") || !strings.Contains(out, `            \x1b[2J\x1b[HNo risky changes.\u009b`+"\n") {
		t.Fatalf("exit %d, stderr %q, stdout %q", code, diagnostic, out)
	}
	out, diagnostic, code = u.run(t, "project", "update", "team:techs/go/\x1b[2Jerrors")
	if code != 2 || out != "" || strings.Contains(diagnostic, "\x1b") || !strings.Contains(diagnostic, `team:techs/go/\x1b[2Jerrors`) {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, diagnostic)
	}
}
