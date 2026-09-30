// Check the fake GitHub CLI through real processes with a PATH that holds only the fake.

package ghfixture

import (
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

// TestInstall_AnswersExactCallsAndRecordsThem lets release tests assert the GitHub Release page they create.
func TestInstall_AnswersExactCallsAndRecordsThem(t *testing.T) {
	bin := t.TempDir()
	fake, err := Install(bin, []Response{
		{Args: []string{"auth", "status"}},
		{Args: []string{"release", "view", "release/1"}, Stdout: "{}\n", Stderr: "release not found\n", ExitCode: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	// run returns stdout followed by stderr, and the exit status.
	run := func(stdin string, args ...string) (string, int) {
		cmd := exec.Command(bin+"/gh", args...)
		cmd.Env = []string{"PATH=" + bin}
		cmd.Stdin = strings.NewReader(stdin)
		var diagnostics strings.Builder
		cmd.Stderr = &diagnostics
		out, err := cmd.Output()
		out = append(out, diagnostics.String()...)
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return string(out), exit.ExitCode()
		}
		if err != nil {
			t.Fatal(err)
		}
		return string(out), 0
	}
	if out, code := run("", "auth", "status"); out != "" || code != 0 {
		t.Fatalf("auth status: %q, %d", out, code)
	}
	if out, code := run("", "release", "view", "release/1"); out != "{}\nrelease not found\n" || code != 1 {
		t.Fatalf("release view: %q, %d", out, code)
	}
	notes := "## Library release 1\n\nEvery rule starts at 1.0.0.\n"
	if _, code := run(notes, "release", "create", "release/1", "--title", "Library release 1", "--notes-file", "-"); code != 1 {
		t.Fatalf("unexpected call exited %d", code)
	}
	calls, err := fake.Calls()
	if err != nil {
		t.Fatal(err)
	}
	want := []Call{
		{Args: []string{"auth", "status"}},
		{Args: []string{"release", "view", "release/1"}},
		{Args: []string{"release", "create", "release/1", "--title", "Library release 1", "--notes-file", "-"}, Stdin: notes},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls %+v, want %+v", calls, want)
	}
}
