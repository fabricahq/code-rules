// Check subprocess ownership, helper cleanup, and exit observation with actual child processes.

package gitexec

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// TestExitObservationDoesNotReap leaves exit status available after observing an already-exited child.
func TestExitObservationDoesNotReap(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "exit 7")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Wait()
	// Observe twice: the second observation must also work after the child has exited.
	for range 2 {
		if err := waitForExit(cmd.Process.Pid); err != nil {
			t.Fatal(err)
		}
	}
	err := cmd.Wait()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 7 {
		t.Fatalf("exit status was lost: %v", err)
	}
}

// TestReleasedGroupCannotSignal prevents a late callback from targeting a reused process identity.
func TestReleasedGroupCannotSignal(t *testing.T) {
	cmd := exec.Command("/bin/sleep", "0.1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Wait()
	defer cmd.Process.Kill()
	// Represent an ID reused by this live child after the original group was released.
	group := processGroup{command: cmd, released: true}
	if err := group.stop(false); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("late signal returned %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("late callback killed the unrelated child: %v", err)
	}
}

// TestCompletedGitStopsHelpers prevents helpers from continuing after their Git leader exits successfully.
func TestCompletedGitStopsHelpers(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "escaped")
	script := filepath.Join(dir, "git")
	body := "#!/bin/sh\n(sleep 0.3; printf escaped > " + gitfixture.Quote(marker) + ") &\nprintf done\n"
	if err := os.WriteFile(script, []byte(body), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := (Runner{executable: script, environment: gitEnvironment(os.Environ())}).Run(ctx, dir, nil, 100, nil)
	if err != nil || result.Status != 0 || string(result.Output) != "done" {
		t.Fatalf("completed Git result: %+v, %v", result, err)
	}
	// Give an incorrectly retained helper time to expose its observable side effect.
	time.Sleep(500 * time.Millisecond)
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("helper outlived its command", err)
	}
}

// TestStoppedGitWaitsForCancellation does not mistake a stopped process for an exited one.
func TestStoppedGitWaitsForCancellation(t *testing.T) {
	script := filepath.Join(t.TempDir(), "git")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nkill -STOP $$\nprintf continued\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := (Runner{executable: script, environment: gitEnvironment(os.Environ())}).Run(ctx, t.TempDir(), nil, 100, nil)
	requireCode(t, err, "timed-out")
}

// requireCode checks the stable failure category without matching Git's private diagnostics.
func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	var failure *Error
	if !errors.As(err, &failure) || failure.Code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
}

// TestProcessLimitsAndDeadlines verifies combined output limits, deadlines, and secret-free diagnostics.
func TestProcessLimitsAndDeadlines(t *testing.T) {
	for _, stream := range []string{"stdout", "stderr"} {
		t.Run(stream, func(t *testing.T) {
			script := filepath.Join(t.TempDir(), "git")
			redirect := ""
			if stream == "stderr" {
				redirect = " >&2"
			}
			if err := os.WriteFile(script, []byte("#!/bin/sh\nwhile :; do printf secret-token"+redirect+"; done\n"), 0700); err != nil {
				t.Fatal(err)
			}
			_, err := (Runner{executable: script, environment: gitEnvironment(os.Environ())}).Run(context.Background(), t.TempDir(), nil, 100, nil)
			requireCode(t, err, "limit-exceeded")
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("stderr leaked")
			}
		})
	}
	script := filepath.Join(t.TempDir(), "git")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30 &\nwait\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := (Runner{executable: script, environment: gitEnvironment(os.Environ())}).Run(ctx, t.TempDir(), nil, 100, nil)
	requireCode(t, err, "timed-out")
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 3*time.Second {
		t.Fatal("deadline or child-pipe cleanup failed", err)
	}
}

// TestOwned_RunsTheAuthorsHooks keeps an author's pre-push checks in force for pushes in their own repository,
// while Isolated never runs repository hooks.
func TestOwned_RunsTheAuthorsHooks(t *testing.T) {
	ctx := context.Background()
	f, err := gitfixture.New(ctx, map[string][]byte{"rule-library.yaml": []byte("formatVersion: 1\n")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	checkout, err := f.Clone(ctx)
	if err != nil {
		t.Fatal(err)
	}
	pushed, err := f.Commit(ctx, checkout, "Change the library", map[string][]byte{"changes/one.yaml": []byte("summary: One.\n")})
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "hook-ran")
	hooks := filepath.Join(t.TempDir(), "hooks")
	if err := os.Mkdir(hooks, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "pre-push"), []byte("#!/bin/sh\n: > "+gitfixture.Quote(marker)+"\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := f.CommandIn(ctx, checkout, "config", "core.hooksPath", hooks); err != nil {
		t.Fatal(err)
	}
	options := Options{GitPath: f.GitPath, Environment: f.Environment}
	remoteMain := func() string {
		head, err := f.Command(ctx, "rev-parse", "main")
		if err != nil {
			t.Fatal(err)
		}
		return head
	}

	owned, err := Owned(options)
	if err != nil {
		t.Fatal(err)
	}
	result, err := owned.Run(ctx, checkout, []string{"push", "--quiet", "origin", "main"}, 1<<20, nil)
	if err != nil || result.Status == 0 {
		t.Fatalf("push through Owned succeeded despite a rejecting pre-push hook: %+v, %v", result, err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("Owned didn't run the pre-push hook", err)
	}
	if head := remoteMain(); head != f.LatestCommit {
		t.Fatalf("rejected push changed the remote to %s", head)
	}

	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	isolated, err := Isolated(options)
	if err != nil {
		t.Fatal(err)
	}
	result, err = isolated.Run(ctx, checkout, []string{"push", "--quiet", "origin", "main"}, 1<<20, nil)
	if err != nil || result.Status != 0 {
		t.Fatalf("push through Isolated failed: %+v, %v", result, err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("Isolated ran a repository hook", err)
	}
	if head := remoteMain(); head != pushed {
		t.Fatalf("remote main is %s, want %s", head, pushed)
	}
}

// program installs an executable shell script named tool as the only program on a new PATH, returning an
// environment that finds it.
func program(t *testing.T, body string) []string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tool"), []byte("#!/bin/sh\n"+body), 0700); err != nil {
		t.Fatal(err)
	}
	return []string{"PATH=" + dir}
}

// TestCommand_RunsAProgramWithGitsLimits finds the program on the given PATH, keeps its stderr apart, stops a
// still-running program as soon as its output exceeds the budget, and cancels its descendants.
func TestCommand_RunsAProgramWithGitsLimits(t *testing.T) {
	if _, err := Command(Options{Environment: []string{"PATH=" + t.TempDir()}}, "tool"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing program: %v", err)
	}
	runner, err := Command(Options{Environment: program(t, "printf out\nprintf err >&2\nexit 3\n")}, "tool")
	if err != nil {
		t.Fatal(err)
	}
	result, err := runner.Run(context.Background(), t.TempDir(), nil, 100, nil)
	if err != nil || string(result.Output) != "out" || string(result.Diagnostics) != "err" || result.Status != 3 {
		t.Fatalf("result %+v: %v", result, err)
	}
	// The helper keeps the stream open and the program keeps running, so only stopping the group ends it.
	runner, err = Command(Options{Environment: program(t, "/bin/sleep 30 &\nwhile :; do printf x; done\n")}, "tool")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = runner.Run(context.Background(), t.TempDir(), nil, 100, nil)
	requireCode(t, err, "limit-exceeded")
	if time.Since(start) > 3*time.Second {
		t.Fatal("overflow didn't stop the program promptly")
	}
	runner, err = Command(Options{Environment: program(t, "/bin/sleep 30 &\nwait\n")}, "tool")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start = time.Now()
	_, err = runner.Run(ctx, t.TempDir(), nil, 100, nil)
	requireCode(t, err, "timed-out")
	if time.Since(start) > 3*time.Second {
		t.Fatal("cancellation left a descendant holding the program's output")
	}
}
