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
