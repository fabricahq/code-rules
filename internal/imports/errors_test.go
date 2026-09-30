// Exercise how import failures explain Git's diagnostics: unreachable hosts, servers that refuse objects by ID,
// and other failures, without showing credentials.

package imports

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestGitFailure_ExplainsTheCauseFromGitsDiagnostics classifies the diagnostics Git prints for each cause and
// quotes only the part that explains it, with credentials redacted.
func TestGitFailure_ExplainsTheCauseFromGitsDiagnostics(t *testing.T) {
	repo := &repository{url: "https://reader:s3cret-token@example.com/acme/rules.git"}
	for _, test := range []struct {
		name, diagnostics, code, message string
	}{
		{"unresolvable HTTPS host", "fatal: unable to access 'https://reader:s3cret-token@example.com/acme/rules.git/': Could not resolve host: example.com\n", "connection-failed", "Could not connect to the library's repository: Could not resolve host: example.com. Check the repository address and your network connection."},
		{"unresolvable SSH host", "ssh: Could not resolve hostname example.com: nodename nor servname provided, or not known\nfatal: Could not read from remote repository.\n", "connection-failed", "Could not connect to the library's repository: Could not resolve hostname example.com: nodename nor servname provided, or not known."},
		{"refused connection", "fatal: unable to access 'https://example.com/': Failed to connect to example.com port 443 after 3 ms: Connection refused\n", "connection-failed", "Could not connect to the library's repository: Failed to connect to example.com port 443 after 3 ms: Connection refused."},
		{"object refused by ID", "error: Server does not allow request for unadvertised object 33e1b731e5757e44091d8e77afa12997b2f73a3b\nfatal: could not fetch 33e1b731 from promisor remote\n", "object-fetch-refused", "The library's server refused to send a file by its object ID (Server does not allow request for unadvertised object 33e1b731e5757e44091d8e77afa12997b2f73a3b)"},
		{"object refused as not our ref", "fatal: remote error: upload-pack: not our ref 33e1b731e5757e44091d8e77afa12997b2f73a3b\n", "object-fetch-refused", "(not our ref 33e1b731e5757e44091d8e77afa12997b2f73a3b)"},
		{"other failure quotes Git's last line", "warning: something\nfatal: bad object in https://reader:s3cret-token@example.com/x\n", "git-failed", "Could not read library files: bad object in https://[redacted]@example.com/x."},
		{"no diagnostics", "", "git-failed", "Could not read library files."},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := repo.gitFailure("git-failed", "Could not read library files.", []byte(test.diagnostics))
			requireCode(t, err, test.code)
			if !strings.Contains(err.Error(), test.message) || strings.Contains(err.Error(), "s3cret") {
				t.Fatalf("message %q, want it to contain %q and no credentials", err, test.message)
			}
		})
	}
	requireCode(t, repo.remoteFailure([]byte("remote: Repository not found.\nfatal: repository 'https://example.com/acme/rules.git/' not found\n")), "not-found-or-no-access")
	requireCode(t, repo.remoteFailure([]byte("fatal: unable to access 'https://example.com/': Could not resolve host: example.com\n")), "connection-failed")
}

// TestImport_ReportsAHostItCantConnectToApartFromAMissingRepository syncs a source whose host refuses connections,
// on this machine only, and gets connection-failed rather than not-found-or-no-access.
func TestImport_ReportsAHostItCantConnectToApartFromAMissingRepository(t *testing.T) {
	h := newHistory(t)
	config, err := rules.ParseConfiguration(json.RawMessage(`{"schemaVersion":1,"sources":{"team":{"repository":"https://127.0.0.1:1/acme/rules.git","groups":["techs/go"]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = ImportLibraries(context.Background(), config, nil, Options{GitPath: h.fixture.GitPath, Environment: h.fixture.Environment})
	var failure *Error
	if !errors.As(err, &failure) || failure.Code != "connection-failed" || !strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("got %v, want connection-failed naming the host", err)
	}
}
