// Check that a terminal run ends, with an error, when its prompt never appears.

package terminalfixture

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestRun_ReturnsWhenAPromptNeverAppearsAndTheProcessKeepsWriting keeps reading the terminal after giving up, so
// a process that writes more than the terminal buffers can still exit.
func TestRun_ReturnsWhenAPromptNeverAppearsAndTheProcessKeepsWriting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := RunWithEnvironment(ctx, "/bin/sh", t.TempDir(), os.Environ(), []string{"-c", "exec head -c 10000000 /dev/zero >&2"}, []Step{{Prompt: "never shown", Answer: "x"}})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "exceeds 2 MiB") {
			t.Fatalf("got %v, want the output limit", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the run never returned")
	}
}
