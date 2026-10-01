// Check how a refused push's diagnostics, which are never shown, choose its static explanation.

package library

import (
	"strings"
	"testing"
)

// TestPushRefusal_ExplainsTheCauseWithoutTheServersText classifies what GitHub, GitLab, server hooks, and Git print
// when a tag push fails, and quotes none of it.
func TestPushRefusal_ExplainsTheCauseWithoutTheServersText(t *testing.T) {
	const marker = "EXTERNAL-TEXT-MARKER"
	for _, test := range []struct{ name, diagnostics, explanation string }{
		{"GitHub ruleset", "remote: error: GH013: Repository rule violations found for refs/tags/release/2.\nremote: Review all repository rules at https://github.com/acme/rules/rules?ref=refs%2Ftags%2Frelease%2F2\nremote: - Cannot create ref due to creations being restricted. " + marker + "\nTo github.com:acme/rules.git\n ! [remote rejected] release/2 -> release/2 (push declined due to repository rule violations)\nerror: failed to push some refs to 'github.com:acme/rules.git'\n",
			"a repository rule refused the tag (GitHub error GH013). Check the repository's rulesets and tag protection rules, and that they let you create release/<number> tags,"},
		{"GitHub protected branch or tag", "remote: error: GH006: Protected branch update failed for refs/tags/release/2. " + marker + "\n ! [remote rejected] release/2 -> release/2 (protected branch hook declined)\n",
			"a repository rule refused the tag (GitHub error GH006)."},
		{"GitHub large file", "remote: error: GH001: Large files detected. " + marker + "\n ! [remote rejected] release/2 -> release/2 (pre-receive hook declined)\n",
			"the server refused the tag for a reason Code Rules doesn't recognize (GitHub error GH001)."},
		{"GitHub rule violation without a code", " ! [remote rejected] release/2 -> release/2 (push declined due to repository rule violations " + marker + ")\n",
			"a repository rule or tag protection refused the tag."},
		{"GitLab protected tag", "remote: GitLab: You are not allowed to create protected tags on this project. " + marker + "\n ! [remote rejected] release/2 -> release/2 (pre-receive hook declined)\n",
			"a repository rule or tag protection refused the tag."},
		{"GitHub permission", "remote: Permission to acme/rules.git denied to " + marker + ".\nfatal: unable to access 'https://github.com/acme/rules.git/': The requested URL returned error: 403\n",
			"the server denied access, or authentication failed."},
		{"SSH key refused", marker + "@github.com: Permission denied (publickey).\nfatal: Could not read from remote repository.\n",
			"the server denied access, or authentication failed."},
		{"HTTPS authentication", "fatal: Authentication failed for 'https://" + marker + "@example.com/rules.git/'\n",
			"the server denied access, or authentication failed."},
		{"GitLab permission, declined by its hook", "remote: GitLab: You are not allowed to push code to this project. " + marker + "\n ! [remote rejected] release/2 -> release/2 (pre-receive hook declined)\n",
			"the server denied access, or authentication failed."},
		{"pre-receive hook", "remote: " + marker + "\n ! [remote rejected] release/2 -> release/2 (pre-receive hook declined)\n",
			"a hook on the server declined the tag."},
		{"update hook", "remote: " + marker + "\nremote: error: hook declined to update refs/tags/release/2\n ! [remote rejected] release/2 -> release/2 (hook declined)\n",
			"a hook on the server declined the tag."},
		{"unrecognized rejection", " ! [remote rejected] release/2 -> release/2 (" + marker + ")\n",
			"the server refused the tag for a reason Code Rules doesn't recognize."},
		{"untrusted certificate", "fatal: unable to access '" + marker + "': server certificate verification failed. CAfile: none CRLfile: none\n",
			"Git couldn't verify the server's TLS certificate."},
		{"unknown SSH host key", "Host key verification failed.\r\nfatal: Could not read from remote repository. " + marker + "\n",
			"Git couldn't verify the server's SSH host key."},
		{"unreachable host", "ssh: Could not resolve hostname " + marker + ": nodename nor servname provided, or not known\nfatal: Could not read from remote repository.\n",
			"Git couldn't connect to the server."},
		{"no server response", "error: failed to push some refs to '" + marker + "'\n",
			"Git couldn't push release/2 to origin. Check your network connection"},
	} {
		t.Run(test.name, func(t *testing.T) {
			message := pushRefusal("release/2", "origin", []byte(test.diagnostics))
			if !strings.Contains(message, test.explanation) || !strings.HasPrefix(message, "Git couldn't push release/2 to origin") || !strings.HasSuffix(message, "Code Rules doesn't show messages from Git or the server; to read them, push a test tag with git push.") {
				t.Fatalf("message:\n%s\nwant it to contain %q", message, test.explanation)
			}
			for _, external := range []string{marker, "remote", "declined to", "hook declined", "403", "github.com:", "publickey"} {
				if strings.Contains(message, external) {
					t.Errorf("message shows %q:\n%s", external, message)
				}
			}
		})
	}
}

// TestFetchFailure_ExplainsTheCauseWithoutGitsText classifies what Git prints when reading or fetching from the
// remote fails, and quotes none of it.
func TestFetchFailure_ExplainsTheCauseWithoutGitsText(t *testing.T) {
	const marker = "EXTERNAL-TEXT-MARKER"
	for _, test := range []struct{ name, diagnostics, explanation string }{
		{"untrusted certificate", "fatal: unable to access '" + marker + "': SSL certificate problem: unable to get local issuer certificate\n", "Git couldn't read origin: Git couldn't verify the server's TLS certificate."},
		{"certificate verification", "fatal: unable to access '" + marker + "': SSL: certificate verification failed (result: 5)\n", "Git couldn't read origin: Git couldn't verify the server's TLS certificate."},
		{"unknown SSH host key", "Host key verification failed.\nfatal: Could not read from remote repository. " + marker + "\n", "Git couldn't read origin: Git couldn't verify the server's SSH host key."},
		{"unreachable host", "fatal: unable to access '" + marker + "': Could not resolve host: example.com\n", "Git couldn't read origin: Git couldn't connect to the server."},
		{"missing repository", "remote: Repository not found. " + marker + "\n", "Git couldn't read origin: the server denied access, or the repository doesn't exist."},
		{"other", "fatal: " + marker + "\n", "Git couldn't read origin. Check your network connection"},
	} {
		t.Run(test.name, func(t *testing.T) {
			message := fetchFailure("read", "origin", []byte(test.diagnostics)).Error()
			if !strings.HasPrefix(message, test.explanation) || strings.Contains(message, marker) {
				t.Fatalf("message:\n%s\nwant it to start %q", message, test.explanation)
			}
		})
	}
}
