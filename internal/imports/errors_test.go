// Exercise how import failures explain Git's diagnostics, without showing them: unreachable hosts, servers that
// refuse objects by ID, and other failures.

package imports

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// TestGitFailure_ExplainsTheCauseWithoutGitsText classifies the diagnostics Git prints for each cause into a static
// explanation that quotes none of them.
func TestGitFailure_ExplainsTheCauseWithoutGitsText(t *testing.T) {
	const marker = "EXTERNAL-TEXT-MARKER"
	connection := "Could not connect to the library's repository. Check the repository address and your network connection."
	certificate := "The library's HTTPS server certificate couldn't be verified. Check that your system trusts it: Git's http.sslCAInfo setting, your system's certificate store, and any proxy that intercepts TLS."
	hostKey := "The library's SSH host key couldn't be verified. Check the server's entry in your known_hosts file."
	refused := "The library's server refused to send a file by its object ID, which Code Rules needs to read one version of each rule without downloading the whole repository. GitHub.com and GitLab.com allow it; ask the administrator of a self-hosted server to enable Git protocol version 2 or uploadpack.allowAnySHA1InWant."
	for _, test := range []struct {
		name, diagnostics, code, message string
	}{
		{"unresolvable HTTPS host", "fatal: unable to access 'https://reader:s3cret@example.com/acme/rules.git/': Could not resolve host: " + marker + "\n", "connection-failed", connection},
		{"unresolvable SSH host", "ssh: Could not resolve hostname " + marker + ": nodename nor servname provided, or not known\nfatal: Could not read from remote repository.\n", "connection-failed", connection},
		{"refused connection, with a terminal sequence inside the phrase", "fatal: unable to access '" + marker + "': Failed to connect to example.com port 443 after 3 ms: Connection \x1b[31mrefused\n", "connection-failed", connection},
		{"untrusted certificate", "fatal: unable to access '" + marker + "': SSL certificate problem: self-signed certificate in certificate chain\n", "https-certificate-failed", certificate},
		{"certificate verification with GnuTLS", "fatal: unable to access '" + marker + "': server certificate verification failed. CAfile: none CRLfile: none\n", "https-certificate-failed", certificate},
		{"unknown SSH host key", "Host key verification failed.\r\nfatal: Could not read from remote repository. " + marker + "\n", "ssh-host-key-failed", hostKey},
		{"object refused by ID", "error: Server does not allow request for unadvertised object " + marker + "\nfatal: could not fetch 33e1b731 from promisor remote\n", "object-fetch-refused", refused},
		{"object refused as not our ref", "fatal: remote error: upload-pack: not our ref " + marker + "\n", "object-fetch-refused", refused},
		{"other failure", "warning: something\nfatal: bad object " + marker + "\n", "git-failed", "Could not read library files."},
		{"no diagnostics", "", "git-failed", "Could not read library files."},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := gitFailure("git-failed", "Could not read library files.", []byte(test.diagnostics))
			requireCode(t, err, test.code)
			if err.Error() != test.message {
				t.Fatalf("message %q, want %q", err, test.message)
			}
		})
	}
	for _, test := range []struct{ diagnostics, code, message string }{
		{"remote: Repository not found " + marker + ".\nfatal: repository 'https://example.com/acme/rules.git/' not found\n", "not-found-or-no-access", "Could not find the library's repository, or your Git credentials can't read it. Check the repository address and your Git credentials."},
		{"fatal: unable to access 'https://example.com/': Could not resolve host: " + marker + "\n", "connection-failed", connection},
		{"fatal: unable to access '" + marker + "': SSL: certificate verification failed (result: 5)\n", "https-certificate-failed", certificate},
		{"Host key verification failed.\nfatal: Could not read from remote repository. " + marker + "\n", "ssh-host-key-failed", hostKey},
		{"fatal: Authentication failed for '" + marker + "'\n", "not-found-or-no-access", "Could not find the library's repository, or your Git credentials can't read it. Check the repository address and your Git credentials."},
		{"fatal: could not read Username for '" + marker + "': terminal prompts disabled\n", "not-found-or-no-access", "Could not find the library's repository, or your Git credentials can't read it. Check the repository address and your Git credentials."},
		{"fatal: " + marker + "\n", "git-failed", "Could not read the library's repository for a reason Code Rules doesn't recognize. Run git ls-remote with the repository address to read Git's message."},
		{"", "git-failed", "Could not read the library's repository for a reason Code Rules doesn't recognize. Run git ls-remote with the repository address to read Git's message."},
	} {
		err := remoteFailure([]byte(test.diagnostics))
		requireCode(t, err, test.code)
		if err.Error() != test.message {
			t.Errorf("message %q, want %q", err, test.message)
		}
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
	if !errors.As(err, &failure) || failure.Code != "connection-failed" || strings.Contains(err.Error(), "refused") {
		t.Fatalf("got %v, want connection-failed without Git's text", err)
	}
}
