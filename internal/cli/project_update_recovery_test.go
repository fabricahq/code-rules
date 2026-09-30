// Interrupt a real code-rules project update partway through installing, then check that the next update or
// sync recovers the project and completes.

package cli

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// interruptedUpdate returns an update fixture whose `code-rules project update --yes` was killed after writing
// its recovery journal and before committing, so the next writer must roll its transaction back. The kill races
// the update's last steps, so it retries with a fresh project when the update committed first.
func interruptedUpdate(t *testing.T) updateFixture {
	t.Helper()
	binary := buildCLI(t)
	for attempt := 0; attempt < 10; attempt++ {
		u := newUpdateFixtureWith(t, binary)
		journal := filepath.Join(u.directory, ".code-rules", ".code-rules-transaction", "journal.json")
		command := exec.Command(u.binary, "project", "update", "--yes", "--keep", "team:techs/go/errors", "--reason", "Not yet.")
		command.Dir = u.directory
		command.Env = u.fixture.Environment
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		exited := make(chan error, 1)
		go func() { exited <- command.Wait() }()
		var waitErr error
	watch:
		for {
			select {
			case waitErr = <-exited:
				break watch
			default:
			}
			// Kill only once the journal is complete: from then on, recovery must restore or finish every target.
			if data, err := os.ReadFile(journal); err == nil && json.Valid(data) {
				_ = command.Process.Signal(syscall.SIGKILL)
				waitErr = <-exited
				break
			}
			time.Sleep(50 * time.Microsecond)
		}
		var exit *exec.ExitError
		_, journalErr := os.Stat(journal)
		_, committedErr := os.Stat(filepath.Join(filepath.Dir(journal), "committed.json"))
		if journalErr == nil && errors.Is(committedErr, os.ErrNotExist) && errors.As(waitErr, &exit) {
			return u
		}
	}
	t.Fatal("the update always finished before it could be interrupted")
	return updateFixture{}
}

// TestUpdate_RecoversAnInterruptedUpdateAndCompletes reruns the update after a killed one.
func TestUpdate_RecoversAnInterruptedUpdateAndCompletes(t *testing.T) {
	u := interruptedUpdate(t)
	if out, diagnostic, code := u.run(t, "project", "update", "--yes", "--keep", "team:techs/go/errors", "--reason", "Not yet."); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out, diagnostic)
	}
	requireRecoveredUpdate(t, u)
	if want := map[string]string{"backoff": "1.0.0", "errors": "1.0.0", "format": "1.0.1", "loaders": "1.1.0", "naming": "1.1.0", "verify": "1.0.0"}; !maps.Equal(u.versions(t), want) {
		t.Fatalf("versions %v", u.versions(t))
	}
}

// TestSync_RecoversAnInterruptedUpdate leaves a consistent project that checks offline.
func TestSync_RecoversAnInterruptedUpdate(t *testing.T) {
	u := interruptedUpdate(t)
	if out, diagnostic, code := u.run(t, "project", "sync"); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out, diagnostic)
	}
	requireRecoveredUpdate(t, u)
	// Rolling back restored the configuration with the output, so sync restores the versions from before the update.
	if want := map[string]string{"backoff": "1.0.0", "errors": "1.0.0", "format": "1.0.0", "loaders": "1.0.0", "naming": "1.0.0", "retry": "1.0.0"}; !maps.Equal(u.versions(t), want) {
		t.Fatalf("versions %v", u.versions(t))
	}
	if config, err := os.ReadFile(filepath.Join(u.directory, ".code-rules", "config.yaml")); err != nil || strings.Contains(string(config), "techs/go/errors:") {
		t.Fatalf("the rolled-back configuration kept the update's pin: %v\n%s", err, config)
	}
}

// requireRecoveredUpdate checks that no transaction remains and the project checks offline.
func requireRecoveredUpdate(t *testing.T, u updateFixture) {
	t.Helper()
	for _, name := range []string{".code-rules-transaction", ".code-rules-cleanup", ".code-rules-lock"} {
		if _, err := os.Stat(filepath.Join(u.directory, ".code-rules", name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s remains: %v", name, err)
		}
	}
	if out, diagnostic, code := runCLI(t, u.binary, u.directory, "project", "check"); code != 0 {
		t.Fatalf("offline check: exit %d\n%s%s", code, out, diagnostic)
	}
}
