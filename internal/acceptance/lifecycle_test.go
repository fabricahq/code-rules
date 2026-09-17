// Exercise native CLI composition against original library fixtures and real Git history.

package acceptance

import (
	"context"
	"os/exec"
	"path/filepath"
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
