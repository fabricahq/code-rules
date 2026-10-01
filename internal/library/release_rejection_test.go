// Check which parts of a refused push's diagnostics reach the failure message.

package library

import (
	"os"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/gitexec"
)

// TestServerRejection_WithholdsRejectionLinesHoldingATokenAndItsFragments: Git's rejection lines carry the server's
// reason, which here holds a known token in full and wrapped across lines, so none of them is shown.
func TestServerRejection_WithholdsRejectionLinesHoldingATokenAndItsFragments(t *testing.T) {
	runner, err := gitexec.Owned(gitexec.Options{Environment: []string{"PATH=" + os.Getenv("PATH"), "GH_TOKEN=opaque-secret-value"}})
	if err != nil {
		t.Fatal(err)
	}
	git := &libraryGit{runner: runner}
	diagnostics := "To ssh://fixture.invalid/rules\n ! [remote rejected] release/2 -> release/2 (opaque-secret-value then opaque-)\n ! [remote rejected] release/2 -> release/2 (secret-value)\n"
	reason := git.serverRejection([]byte(diagnostics), upstream{remote: "origin", url: "ssh://fixture.invalid/rules", pushURL: "ssh://fixture.invalid/rules"})
	if strings.Contains(reason, "opaque") || strings.Contains(reason, "secret") || !strings.Contains(reason, "withheld") {
		t.Fatalf("reason:\n%s", reason)
	}
}
