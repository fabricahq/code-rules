// Exercise preview warnings through packaged executables, including machine-readable output.

package distribution

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPackagedPreviewWarning(t *testing.T) {
	source, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	revision := strings.Repeat("a", 40)
	for _, candidate := range []bool{true, false} {
		name := "release"
		if candidate {
			name = "preview"
		}
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			artifact, err := buildTarget(context.Background(), source, directory, "0.1.0", revision, runtime.GOOS+"/"+runtime.GOARCH, []byte("Fixture terms"), candidate)
			if err != nil {
				t.Fatal(err)
			}
			archive, err := os.ReadFile(filepath.Join(directory, artifact.File))
			if err != nil {
				t.Fatal(err)
			}
			entries, err := readArchive(archive, artifact.BinaryBytes)
			if err != nil {
				t.Fatal(err)
			}
			installed := filepath.Join(directory, "installed")
			if err := installEntries(installed, entries, writeEntry); err != nil {
				t.Fatal(err)
			}
			for _, test := range []struct {
				args []string
				exit int
			}{
				{nil, 0},
				{[]string{"--help"}, 0},
				{[]string{"project", "build", "--help"}, 0},
				{[]string{"--version"}, 0},
				{[]string{"unknown-command"}, 2},
				{[]string{"project", "build"}, 1},
				{[]string{"project", "check", "--json"}, 1},
				{[]string{"project", "init", "--json"}, 0},
			} {
				args := test.args
				command := exec.Command(filepath.Join(installed, "code-rules"), args...)
				command.Dir = t.TempDir()
				var stdout, stderr bytes.Buffer
				command.Stdout, command.Stderr = &stdout, &stderr
				err := command.Run()
				if err != nil {
					if _, ok := err.(*exec.ExitError); !ok {
						t.Fatal(err)
					}
				}
				if got := command.ProcessState.ExitCode(); got != test.exit {
					t.Fatalf("%v: exit %d, want %d; stdout=%q stderr=%q", args, got, test.exit, stdout.String(), stderr.String())
				}
				warning := "WARNING: Unreleased preview from commit " + revision + ". For testing only; not for production use.\n"
				if candidate && (!strings.HasPrefix(stderr.String(), warning) || strings.Count(stderr.String(), warning) != 1) {
					t.Fatalf("%v: missing leading preview warning: %q", args, stderr.String())
				}
				if !candidate && strings.Contains(stderr.String(), "WARNING:") || strings.Contains(stdout.String(), "WARNING:") {
					t.Fatalf("%v: unexpected warning: stdout=%q stderr=%q", args, stdout.String(), stderr.String())
				}
				if len(args) > 0 && args[len(args)-1] == "--json" && !json.Valid(stdout.Bytes()) {
					t.Fatalf("%v: invalid JSON: %q", args, stdout.String())
				}
				if len(args) == 1 && args[0] == "--version" && stdout.String() != "0.1.0\n" {
					t.Fatalf("version changed: %q", stdout.String())
				}
			}
		})
	}
}
