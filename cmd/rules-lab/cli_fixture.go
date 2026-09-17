// Execute the compiled native CLI against confined fixture projects and capture process-level evidence.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
)

// cliFixture adds reviewed command arguments to the existing isolated sync fixture.
type cliFixture struct {
	syncFixture
	Arguments []string `json:"arguments"`
}

// cliObservation records actual standard streams, exit status, and before/after bytes independently of adapter success.
type cliObservation struct {
	Arguments []string          `json:"arguments"`
	Stdout    string            `json:"stdout"`
	Stderr    string            `json:"stderr"`
	ExitCode  int               `json:"exitCode"`
	Before    map[string]string `json:"before"`
	After     map[string]string `json:"after"`
}

// cliResponse reports setup failures separately from nonzero exit statuses returned by the reviewed executable.
func cliResponse(input json.RawMessage) (response, error) {
	var fixture cliFixture
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return response{Error: &failure{Name: "ValidationError", Message: "expected a CLI fixture"}}, nil
	}
	if err := validateCLIArguments(fixture.Arguments); err != nil {
		return response{Error: describeProjectError(err)}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	binary, err := fixtureCLIPath()
	if err != nil {
		return response{}, err
	}
	version, err := exec.CommandContext(ctx, binary, "--version").Output()
	if err != nil {
		return response{}, err
	}
	fixture.toolVersion = strings.TrimSpace(string(version))
	observed, err := withImportFixture(ctx, fixture.importFixture, func(git imports.Options) (any, error) {
		return withSyncProject(ctx, cancel, fixture.syncFixture, git, func(root *os.Root, _ project.Options, before *project.Tree) (any, error) {
			return executeFixtureCLI(ctx, root, git, fixture.Arguments, before)
		})
	})
	if err != nil {
		failure := describeProjectError(err)
		if failure == nil {
			failure = gitFailure(err)
		}
		if failure == nil {
			return response{}, err
		}
		return response{Error: failure}, nil
	}
	return response{OK: true, Value: observed}, nil
}

// validateCLIArguments allows current commands while preventing config paths outside the disposable root.
func validateCLIArguments(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "sync", "build", "check", "--help", "-h", "--version", "-v", "unknown":
		default:
			return &rules.ValidationError{Location: "arguments", Problem: "use sync, build, check, help, or version in this walkthrough"}
		}
	}
	hasConfig := false
	for i, arg := range args {
		if arg == "--config" {
			hasConfig = i+1 < len(args) && args[i+1] == "config.json"
			if i+1 < len(args) && args[i+1] != "config.json" {
				return &rules.ValidationError{Location: "arguments", Problem: "the walkthrough config path must be config.json"}
			}
		} else if len(arg) >= 9 && arg[:9] == "--config=" && arg != "--config=config.json" {
			return &rules.ValidationError{Location: "arguments", Problem: "the walkthrough config path must be config.json"}
		}
		if arg == "--config=config.json" {
			hasConfig = true
		}
	}
	if len(args) > 0 && (args[0] == "sync" || args[0] == "build" || args[0] == "check") && !hasConfig {
		return &rules.ValidationError{Location: "arguments", Problem: "operational walkthrough commands require --config config.json so only the reviewed fixture configuration is used"}
	}
	return nil
}

// executeFixtureCLI invokes the sibling code-rules executable with fixture-only Git settings and no shell.
func executeFixtureCLI(ctx context.Context, root *os.Root, git imports.Options, args []string, before *project.Tree) (any, error) {
	binary, err := fixtureCLIPath()
	if err != nil {
		return nil, err
	}
	command := exec.CommandContext(ctx, binary, args...)
	command.Dir = root.Name()
	command.Env = git.Environment
	command.Cancel = func() error { return command.Process.Signal(os.Interrupt) }
	command.WaitDelay = 5 * time.Second
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	code := 0
	if err := command.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			return nil, err
		}
		code = exit.ExitCode()
	}
	after, err := project.ReadTree(context.Background(), root, ".")
	if err != nil {
		return nil, err
	}
	return &cliObservation{Arguments: args, Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code, Before: displayProjectTree(before), After: displayProjectTree(after)}, nil
}

// fixtureCLIPath locates the trusted compiled command beside the running lab executable.
func fixtureCLIPath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(executable), "code-rules"), nil
}
