// Exercise the workflow's planning script with a real CLI and multi-commit approval history.

package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestWorkflowRetryPreservesApprovedRange(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Steps []struct{ ID, Run string }
		}
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	var script string
	for _, step := range workflow.Jobs["plan"].Steps {
		if step.ID == "plan" {
			script = step.Run
		}
	}
	if !strings.Contains(script, "go run ./cmd/release-plan") {
		t.Fatal("missing production planning command")
	}
	binary := filepath.Join(t.TempDir(), "release-plan")
	build := exec.Command("go", "build", "-o", binary, "./cmd/release-plan")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build planner: %v\n%s", err, output)
	}
	// Only substitute the executable location: the workflow shell and CLI remain real.
	script = strings.Replace(script, "go run ./cmd/release-plan", `"$PLANNER"`, 1)
	f, source := fixture(t)
	base := command(t, f, "rev-parse", "HEAD")
	for _, change := range []struct{ path, body string }{
		{"releases/v0.1.0.md", "Initial draft"},
		{"releases/v0.1.0.md", "Final approved notes\n"},
		{"README.md", "Final approved source"},
	} {
		write(t, source, change.path, change.body)
		command(t, f, "add", ".")
		command(t, f, "-c", "commit.gpgsign=false", "commit", "-m", "Release PR change")
	}
	head := command(t, f, "rev-parse", "HEAD")
	write(t, source, "README.md", "Later main changes")
	command(t, f, "add", ".")
	command(t, f, "-c", "commit.gpgsign=false", "commit", "-m", "Advance main")
	command(t, f, "update-ref", "refs/remotes/origin/main", "HEAD")
	for _, tc := range []struct {
		name, retryBase string
		wantError       bool
	}{
		{"full approved range", base, false},
		{"missing base", "", true},
		{"range without release request", command(t, f, "rev-parse", head+"^"), true},
		{"base after approved head", command(t, f, "rev-parse", "HEAD"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", script)
			cmd.Dir = source
			cmd.Env = append(os.Environ(), "EVENT=workflow_dispatch", "BASE=", "HEAD=", "RETRY="+head, "RETRY_BASE="+tc.retryBase,
				"PLANNER="+binary, "RUNNER_TEMP="+directory, "GITHUB_OUTPUT="+filepath.Join(directory, "outputs"), "GITHUB_STEP_SUMMARY="+filepath.Join(directory, "summary"))
			output, err := cmd.CombinedOutput()
			if tc.wantError {
				if err == nil {
					t.Fatal("accepted invalid retry range")
				}
				return
			}
			if err != nil {
				t.Fatalf("retry: %v\n%s", err, output)
			}
			var plan Plan
			if err := readJSON(filepath.Join(directory, "release-plan.json"), &plan); err != nil {
				t.Fatal(err)
			}
			if plan.Tag != "v0.1.0" || plan.Commit != head || plan.Notes != "Final approved notes\n" {
				t.Fatalf("lost approved source or notes: %+v", plan)
			}
		})
	}
}
