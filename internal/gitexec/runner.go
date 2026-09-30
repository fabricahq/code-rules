// Package gitexec runs noninteractive Git, and other trusted programs such as the GitHub CLI, with bounded
// output and whole-process-group cancellation.
// It never includes Git's own diagnostics in errors, because they can contain credential-bearing URLs.
package gitexec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Error identifies a failure by a stable Code without exposing Git diagnostics or credential-bearing transport URLs.
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

// Fail constructs an *Error with a stable code and a safe, caller-facing problem.
func Fail(code, problem string, cause error) error {
	return &Error{Code: code, Problem: problem, Cause: cause}
}

// Options selects trusted process settings. They are application settings, never repository or configuration fields.
type Options struct {
	// GitPath is the Git executable, resolved against the environment's PATH when it has no slash; empty means "git".
	GitPath string
	// Environment, when nonnil, replaces the inherited environment before isolation overrides are applied.
	Environment []string
}

// Runner runs one Git executable with a fixed environment; it never changes process-wide state.
type Runner struct {
	executable  string
	environment []string
	// config is prepended to every invocation as -c options.
	config []string
}

// Result separates a normal nonzero Git exit from startup, cancellation, and output-limit failures.
type Result struct {
	Output []byte
	// Diagnostics is the program's stderr. Git's can contain credential-bearing URLs, so never show it without
	// Runner.Redact, and only the lines a caller needs.
	Diagnostics []byte
	Status      int
}

// isolatedConfig disables hooks and restricts transports for repositories the user doesn't own.
var isolatedConfig = []string{"-c", "core.hooksPath=/dev/null", "-c", "protocol.allow=never", "-c", "protocol.https.allow=always", "-c", "protocol.ssh.allow=always", "-c", "protocol.ext.allow=never"}

// Isolated returns a Runner for fetching and reading repositories the user doesn't own.
// It disables hooks and allows only HTTPS and SSH transports, whatever the user's Git configuration says.
func Isolated(options Options) (Runner, error) {
	return newRunner(options, isolatedConfig)
}

// Owned returns a Runner for the user's own repository. It honors the user's Git configuration,
// credentials, and hooks, so a push runs the same pre-push hooks as git push would.
func Owned(options Options) (Runner, error) {
	return newRunner(options, nil)
}

// ErrNotFound reports that Command found no executable with the requested name on the environment's PATH.
var ErrNotFound = errors.New("executable not found on PATH")

// Command returns a Runner for another trusted program, such as the GitHub CLI, found by name on the PATH of
// options.Environment, or of the process when it's nil. The program gets the same output budget, process-group
// cancellation, and prompt-free environment as Git, without Git's configuration arguments. Failure codes and
// messages still name Git, so callers translate them. It returns ErrNotFound when there's no such program.
func Command(options Options, name string) (Runner, error) {
	env := options.Environment
	if env == nil {
		env = os.Environ()
	}
	executable := lookPath(name, env)
	if executable == "" {
		return Runner{}, ErrNotFound
	}
	return Runner{executable: executable, environment: gitEnvironment(env)}, nil
}

// newRunner resolves the executable against the child's environment; prompts are always disabled.
func newRunner(options Options, config []string) (Runner, error) {
	if options.GitPath == "" {
		options.GitPath = "git"
	}
	env := options.Environment
	if env == nil {
		env = os.Environ()
	}
	executable, err := gitExecutable(options.GitPath, env)
	if err != nil {
		return Runner{}, err
	}
	return Runner{executable: executable, environment: gitEnvironment(env), config: config}, nil
}

var gitVersion = regexp.MustCompile(`^git version ([0-9]+)\.([0-9]+)`)

// RequireVersion fails with git-unavailable unless the Git executable is version 2.30 or later.
func (r Runner) RequireVersion(ctx context.Context, dir string) error {
	version, err := r.Output(ctx, dir, []string{"--version"}, 4096)
	if err != nil {
		return err
	}
	parts := gitVersion.FindStringSubmatch(string(version))
	major, minor := 0, 0
	if parts != nil {
		major, _ = strconv.Atoi(parts[1])
		minor, _ = strconv.Atoi(parts[2])
	}
	if major < 2 || (major == 2 && minor < 30) {
		return Fail("git-unavailable", "Git 2.30 or later is required.", nil)
	}
	return nil
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

// gitExecutable resolves a trusted command against the child's PATH, without implicit current-directory lookup.
func gitExecutable(name string, environment []string) (string, error) {
	if strings.ContainsRune(name, '/') {
		return filepath.Abs(name)
	}
	if executable := lookPath(name, environment); executable != "" {
		return executable, nil
	}
	return "", Fail("git-unavailable", "Cannot find Git; install Git 2.30 or later and check PATH.", nil)
}

// lookPath returns the first executable file named name in the absolute directories of environment's PATH,
// or "" when there is none. Unlike exec.LookPath, it reads a child's environment rather than the process's,
// and never searches the current directory.
func lookPath(name string, environment []string) string {
	var search string
	for _, item := range environment {
		if value, ok := strings.CutPrefix(item, "PATH="); ok {
			search = value
		}
	}
	for _, directory := range filepath.SplitList(search) {
		candidate := filepath.Join(directory, name)
		if !filepath.IsAbs(candidate) {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode()&0111 == 0 {
			continue
		}
		return candidate
	}
	return ""
}

// outputBudget counts both streams under one ceiling, retaining each separately.
type outputBudget struct {
	mu             sync.Mutex
	remaining      int
	stdout, stderr bytes.Buffer
	exceeded       bool
	stop           func() error
}

// budgetStream writes one stream into its budget: stdout, or stderr when diagnostics is set.
type budgetStream struct {
	budget      *outputBudget
	diagnostics bool
}

// Write retains bounded output and stops the complete process group on combined-output overflow.
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
	if w.diagnostics {
		return b.stderr.Write(data)
	}
	return b.stdout.Write(data)
}

// processGroup serializes signalling with release of the owned, unreaped child.
type processGroup struct {
	mu       sync.Mutex
	command  *exec.Cmd
	released bool
}

// stop kills helpers while the leader's ID is owned; release forbids further signals before reaping.
func (g *processGroup) stop(release bool) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.released {
		return os.ErrProcessDone
	}
	g.released = release
	err := syscall.Kill(-g.command.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}

// Run executes literal arguments in dir without a shell and drains both streams before returning.
// maxBytes bounds stdout and stderr together, and a nonzero exit is not an error.
func (r Runner) Run(ctx context.Context, dir string, args []string, maxBytes int, input []byte) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, ContextFailure(err)
	}
	commandArgs := append(append([]string{}, r.config...), args...)
	cmd := exec.CommandContext(ctx, r.executable, commandArgs...)
	cmd.Dir = dir
	cmd.Env = r.environment
	cmd.Stdin = bytes.NewReader(input)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	group := &processGroup{command: cmd}
	cmd.Cancel = func() error { return group.stop(false) }
	cmd.WaitDelay = time.Second
	budget := &outputBudget{remaining: maxBytes, stop: cmd.Cancel}
	cmd.Stdout = budgetStream{budget: budget}
	cmd.Stderr = budgetStream{budget: budget, diagnostics: true}
	if err := cmd.Start(); err != nil {
		return Result{}, Fail("git-unavailable", "Cannot start Git; install Git 2.30 or later and check PATH.", nil)
	}
	// Keep the leader unreaped until every group signal is finished. Its ID cannot be reused yet.
	exitErr := waitForExit(cmd.Process.Pid)
	_ = group.stop(true)
	err := cmd.Wait()
	if ctx.Err() != nil {
		return Result{}, ContextFailure(ctx.Err())
	}
	if budget.exceeded {
		return Result{}, Fail("limit-exceeded", "Git output exceeds its limit.", nil)
	}
	if exitErr != nil {
		return Result{}, Fail("git-failed", "Cannot observe Git process completion.", nil)
	}
	if errors.Is(err, exec.ErrWaitDelay) {
		return Result{}, Fail("git-failed", "Git helpers did not close their streams.", nil)
	}
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		return Result{}, Fail("git-failed", "Git could not complete its input or output.", nil)
	}
	return Result{Output: budget.stdout.Bytes(), Diagnostics: budget.stderr.Bytes(), Status: cmd.ProcessState.ExitCode()}, nil
}

// Output returns stdout of a command that must exit successfully; a nonzero exit is a git-failed error.
func (r Runner) Output(ctx context.Context, dir string, args []string, maxBytes int) ([]byte, error) {
	result, err := r.Run(ctx, dir, args, maxBytes, nil)
	if err != nil {
		return nil, err
	}
	if result.Status != 0 {
		return nil, Fail("git-failed", "Git could not complete the requested operation.", nil)
	}
	return result.Output, nil
}

// ContextFailure distinguishes an expired deadline from explicit cancellation while preserving its cause.
func ContextFailure(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return Fail("timed-out", "Git operation exceeded its deadline.", err)
	}
	return Fail("cancelled", "Git operation was cancelled.", err)
}
