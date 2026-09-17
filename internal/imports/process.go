// Run noninteractive Git with bounded diagnostics and whole-process-group cancellation.

package imports

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Error identifies a failure without exposing Git diagnostics or credential-bearing transport URLs.
type Error struct {
	Code    string
	Problem string
	// Cause preserves cancellation identity without including it in the public diagnostic.
	Cause error
}

// Error returns only the safe, caller-facing diagnostic.
func (e *Error) Error() string { return e.Problem }

// Unwrap retains errors.Is and errors.As semantics for the underlying failure.
func (e *Error) Unwrap() error { return e.Cause }

// fail constructs an import failure with a stable category and safe explanation.
func fail(code, problem string, cause error) error {
	return &Error{Code: code, Problem: problem, Cause: cause}
}

// gitRunner owns one invocation's executable and environment; it never changes process-wide state.
type gitRunner struct {
	executable  string
	environment []string
}

// gitResult separates a normal nonzero Git exit from startup, cancellation, and output-limit failures.
type gitResult struct {
	output []byte
	status int
}

// gitEnvironment preserves credentials and routing while removing inherited repository and prompting state.
func gitEnvironment(base []string) []string {
	blocked := map[string]bool{"GIT_DIR": true, "GIT_COMMON_DIR": true, "GIT_WORK_TREE": true, "GIT_INDEX_FILE": true, "GIT_OBJECT_DIRECTORY": true, "GIT_ALTERNATE_OBJECT_DIRECTORIES": true, "GIT_SHALLOW_FILE": true, "GIT_NAMESPACE": true, "GIT_TERMINAL_PROMPT": true, "GCM_INTERACTIVE": true, "GIT_NO_REPLACE_OBJECTS": true}
	env := make([]string, 0, len(base)+3)
	for _, item := range base {
		name, _, _ := strings.Cut(item, "=")
		if !blocked[name] {
			env = append(env, item)
		}
	}
	return append(env, "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never", "GIT_NO_REPLACE_OBJECTS=1")
}

// outputBudget counts both streams under one ceiling while retaining stdout only.
type outputBudget struct {
	mu        sync.Mutex
	remaining int
	stdout    bytes.Buffer
	exceeded  bool
	stop      func() error
}

// budgetStream determines whether bounded bytes are retained or discarded as private diagnostics.
type budgetStream struct {
	budget *outputBudget
	retain bool
}

// Write drains diagnostics and stops the complete Git process group on combined-output overflow.
func (w budgetStream) Write(data []byte) (int, error) {
	b := w.budget
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(data) > b.remaining {
		b.exceeded = true
		_ = b.stop()
		return 0, io.ErrShortWrite
	}
	b.remaining -= len(data)
	if w.retain {
		return b.stdout.Write(data)
	}
	return len(data), nil
}

// run executes literal arguments without a shell and drains both streams before returning.
func (r gitRunner) run(ctx context.Context, cwd string, args []string, maxBytes int, input []byte) (gitResult, error) {
	if err := ctx.Err(); err != nil {
		return gitResult{}, contextFailure(err)
	}
	commandArgs := []string{"-c", "core.hooksPath=/dev/null", "-c", "protocol.allow=never", "-c", "protocol.https.allow=always", "-c", "protocol.ssh.allow=always", "-c", "protocol.ext.allow=never"}
	cmd := exec.CommandContext(ctx, r.executable, append(commandArgs, args...)...)
	cmd.Dir = cwd
	cmd.Env = r.environment
	cmd.Stdin = bytes.NewReader(input)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = time.Second
	budget := &outputBudget{remaining: maxBytes, stop: cmd.Cancel}
	cmd.Stdout = budgetStream{budget: budget, retain: true}
	cmd.Stderr = budgetStream{budget: budget}
	if err := cmd.Start(); err != nil {
		return gitResult{}, fail("git-unavailable", "Cannot start Git; install Git 2.30 or later and check PATH.", nil)
	}
	err := cmd.Wait()
	// Descendants must not outlive this invocation, even if Git exited before its helper.
	_ = cmd.Cancel()
	if ctx.Err() != nil {
		return gitResult{}, contextFailure(ctx.Err())
	}
	if budget.exceeded {
		return gitResult{}, fail("limit-exceeded", "Git output exceeds the import limit.", nil)
	}
	if errors.Is(err, exec.ErrWaitDelay) {
		return gitResult{}, fail("git-failed", "Git helpers did not close their streams.", nil)
	}
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		return gitResult{}, fail("git-failed", "Git could not complete its input or output.", nil)
	}
	return gitResult{output: budget.stdout.Bytes(), status: cmd.ProcessState.ExitCode()}, nil
}

// command requires a successful exit and never includes discarded Git diagnostics in its error.
func (r gitRunner) command(ctx context.Context, cwd string, args []string, maxBytes int) ([]byte, error) {
	result, err := r.run(ctx, cwd, args, maxBytes, nil)
	if err != nil {
		return nil, err
	}
	if result.status != 0 {
		return nil, fail("git-failed", "Git could not read the requested revision.", nil)
	}
	return result.output, nil
}

// contextFailure distinguishes an expired deadline from explicit cancellation while preserving its cause.
func contextFailure(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fail("timed-out", "Git operation exceeded its deadline.", err)
	}
	return fail("cancelled", "Git operation was cancelled.", err)
}
