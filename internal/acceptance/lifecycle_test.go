// Exercise native CLI composition against original library fixtures and real Git history.

package acceptance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNativeLifecycle proves authoring, sync, offline operation, and failure preservation through actual processes.
func TestNativeLifecycle(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "code-rules")
	command := exec.Command("go", "build", "-o", binary, "../../cmd/code-rules")
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatal(err, string(data))
	}
	for _, scenario := range []string{"lifecycle", "update", "changed-vendor", "failed-sync"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			report, err := Run(ctx, binary, scenario)
			if err != nil {
				t.Fatalf("%v\n%+v", err, report.Steps)
			}
			if len(report.Verified) < 6 || len(report.Files) == 0 {
				t.Fatal("pilot returned incomplete evidence", report)
			}
		})
	}
}

// TestPilotRejectsWritingChecks proves that extra files and empty directories cannot falsely pass read-only checks.
func TestPilotRejectsWritingChecks(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "code-rules")
	if data, err := exec.Command("go", "build", "-o", binary, "../../cmd/code-rules").CombinedOutput(); err != nil {
		t.Fatal(err, string(data))
	}
	for _, change := range []string{"printf unexpected > unrelated-user-file.txt", "/bin/mkdir unexpected-empty-directory"} {
		t.Run(change, func(t *testing.T) {
			wrapper := filepath.Join(t.TempDir(), "wrapper")
			script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = check ]; then\n IFS= read -r line < .code-rules/generated/RULES.md\n if [ \"$line\" = 'Stale output' ]; then %s; fi\nfi\nexec '%s' \"$@\"\n", change, binary)
			if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			if _, err := Run(ctx, wrapper, "lifecycle"); err == nil || !strings.Contains(err.Error(), "read-only check changed project") {
				t.Fatal("mutation not detected", err)
			}
		})
	}
}

// TestPilotRejectsSilentRefusal ensures a failed command cannot pass without a consumer-visible diagnostic.
func TestPilotRejectsSilentRefusal(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "code-rules")
	if data, err := exec.Command("go", "build", "-o", binary, "../../cmd/code-rules").CombinedOutput(); err != nil {
		t.Fatal(err, string(data))
	}
	wrapper := filepath.Join(t.TempDir(), "wrapper")
	script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = build ] && [ -f .code-rules/vendor/team/LICENSE.md ]; then\n IFS= read -r line < .code-rules/vendor/team/LICENSE.md\n if [ \"$line\" = 'Manual edit' ]; then exit 1; fi\nfi\nexec '%s' \"$@\"\n", binary)
	if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if _, err := Run(ctx, wrapper, "changed-vendor"); err == nil || !strings.Contains(err.Error(), "refusal returned no diagnostic") {
		t.Fatal("silent refusal passed", err)
	}
}
