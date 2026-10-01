// Exercise how import failures explain Git's diagnostics: unreachable hosts, servers that refuse objects by ID,
// and other failures, without showing credentials.

package imports

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/gitexec"
	"github.com/fabricahq/code-rules/internal/rules"
)

// TestGitFailure_ExplainsTheCauseFromGitsDiagnostics classifies the diagnostics Git prints for each cause and
// quotes only the part that explains it, with credentials redacted.
func TestGitFailure_ExplainsTheCauseFromGitsDiagnostics(t *testing.T) {
	repo := &repository{url: "https://reader:s3cret-token@example.com/acme/rules.git"}
	for _, test := range []struct {
		name, diagnostics, code, message string
	}{
		{"unresolvable HTTPS host", "fatal: unable to access 'https://example.com/acme/rules.git/': Could not resolve host: example.com\n", "connection-failed", "Could not connect to the library's repository: Could not resolve host: example.com. Check the repository address and your network connection."},
		{"unresolvable host in text holding the address's password", "fatal: unable to access 'https://reader:s3cret-token@example.com/acme/rules.git/': Could not resolve host: example.com\n", "connection-failed", "Could not connect to the library's repository: Git's message was withheld because it contained a credential. Check the repository address and your network connection."},
		{"unresolvable SSH host", "ssh: Could not resolve hostname example.com: nodename nor servname provided, or not known\nfatal: Could not read from remote repository.\n", "connection-failed", "Could not connect to the library's repository: Could not resolve hostname example.com: nodename nor servname provided, or not known."},
		{"refused connection", "fatal: unable to access 'https://example.com/': Failed to connect to example.com port 443 after 3 ms: Connection refused\n", "connection-failed", "Could not connect to the library's repository: Failed to connect to example.com port 443 after 3 ms: Connection refused."},
		{"object refused by ID", "error: Server does not allow request for unadvertised object 33e1b731e5757e44091d8e77afa12997b2f73a3b\nfatal: could not fetch 33e1b731 from promisor remote\n", "object-fetch-refused", "The library's server refused to send a file by its object ID (Server does not allow request for unadvertised object 33e1b731e5757e44091d8e77afa12997b2f73a3b)"},
		{"object refused as not our ref", "fatal: remote error: upload-pack: not our ref 33e1b731e5757e44091d8e77afa12997b2f73a3b\n", "object-fetch-refused", "(not our ref 33e1b731e5757e44091d8e77afa12997b2f73a3b)"},
		{"other failure quotes Git's last line", "warning: something\nfatal: bad object in https://someone:other@example.com/x\n", "git-failed", "Could not read library files: bad object in https://[redacted]@example.com/x."},
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

// TestGitFailure_QuotesNoKnownCredentialHoweverItIsSplit: a token split by a control character is redacted after
// the control character goes, and a token wrapped across lines withholds Git's text rather than quote a fragment.
func TestGitFailure_QuotesNoKnownCredentialHoweverItIsSplit(t *testing.T) {
	runner, err := gitexec.Isolated(gitexec.Options{Environment: []string{"PATH=" + os.Getenv("PATH"), "GH_TOKEN=opaque-secret-value"}})
	if err != nil {
		t.Fatal(err)
	}
	repo := &repository{url: "https://reader:pw@example.com/acme/rules.git", runner: runner}
	for _, diagnostics := range []string{
		"fatal: token opaque-\tsecret-value rejected\n",
		"error: token opaque-\nfatal: secret-value rejected\n",
		"fatal: password pw rejected\n",
		"fatal: unable to access 'https://example.com/': Could not resolve host: opaque-\x1bsecret-value\n",
		"error: token opaque-\nfatal\t: secret-value rejected\n",
	} {
		err := repo.gitFailure("git-failed", "Could not read library files.", []byte(diagnostics))
		for _, fragment := range []string{"opaque", "secret-value", " pw "} {
			if strings.Contains(err.Error(), fragment) {
				t.Errorf("diagnostics %q: the failure quotes %q: %v", diagnostics, fragment, err)
			}
		}
	}
	// An HTTPS user name can itself be a token.
	tokenUser := &repository{url: "https://opaque-user-token@example.com/acme/rules.git", runner: runner}
	if err := tokenUser.gitFailure("git-failed", "Could not read library files.", []byte("error: token opaque-\nfatal: user-token rejected\n")); strings.Contains(err.Error(), "user-token") {
		t.Errorf("the failure quotes the user name token's fragment: %v", err)
	}
}

// TestGitFailure_WithholdsLargeDiagnosticsWithoutScanningThemAtLength: a megabyte of lines repeating a known token
// is withheld, with the failure's code, rather than redacted line by line.
func TestGitFailure_WithholdsLargeDiagnosticsWithoutScanningThemAtLength(t *testing.T) {
	runner, err := gitexec.Isolated(gitexec.Options{Environment: []string{"PATH=" + os.Getenv("PATH"), "GH_TOKEN=opaque-secret-value"}})
	if err != nil {
		t.Fatal(err)
	}
	repo := &repository{url: "https://example.com/acme/rules.git", runner: runner}
	diagnostics := strings.Repeat("error: token opaque-secret-value rejected\n", 1<<20/42)
	err = repo.gitFailure("git-failed", "Could not read library files.", []byte(diagnostics))
	requireCode(t, err, "git-failed")
	if strings.Contains(err.Error(), "opaque") || strings.Contains(err.Error(), "[redacted]") || !strings.Contains(err.Error(), "withheld") {
		t.Fatalf("got %v", err)
	}
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
