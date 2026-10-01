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
			"a repository rule refused the tag (GitHub error GH013)."},
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
