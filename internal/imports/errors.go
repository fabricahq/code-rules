// Share the Git runner's failure type, so every import failure carries one stable code, and explain Git's failures
// with static messages chosen from its diagnostics, which are never shown.

package imports

import "github.com/fabricahq/code-rules/internal/gitexec"

// Error is the type of every import failure, including Git failures. Its Code is stable for callers.
type Error = gitexec.Error

// fail constructs an import failure with a stable category and safe explanation.
func fail(code, problem string, cause error) error {
	return gitexec.Fail(code, problem, cause)
}

// objectRefusals are the texts Git prints when a server refuses to send an object by its ID, which only servers
// that speak Git protocol version 0 or 1 without uploadpack.allowAnySHA1InWant do.
var objectRefusals = []string{"server does not allow request for unadvertised object", "not our ref"}

// connectionFailure explains a host Git couldn't reach.
const connectionFailure = "Could not connect to the library's repository. Check the repository address and your network connection."

// remoteFailure explains a Git command that couldn't read the library's repository, from its diagnostics, which it
// never shows: a host it couldn't reach, with code connection-failed, or otherwise a repository that doesn't exist or
// that the credentials can't read, with code not-found-or-no-access, since servers report both the same way.
func remoteFailure(diagnostics []byte) error {
	if gitexec.Mentions(diagnostics, gitexec.ConnectionFailures...) {
		return fail("connection-failed", connectionFailure, nil)
	}
	return fail("not-found-or-no-access", "Repository not found or no access; check its address and Git credentials.", nil)
}

// gitFailure explains a Git command that failed while reading the library, from its diagnostics, which it never
// shows: a host it couldn't reach, with code connection-failed; a server that refused to send files by object ID,
// with code object-fetch-refused; or otherwise problem, with code.
func gitFailure(code, problem string, diagnostics []byte) error {
	if gitexec.Mentions(diagnostics, gitexec.ConnectionFailures...) {
		return fail("connection-failed", connectionFailure, nil)
	}
	if gitexec.Mentions(diagnostics, objectRefusals...) {
		return fail("object-fetch-refused", "The library's server refused to send a file by its object ID, which Code Rules needs to read one version of each rule without downloading the whole repository. GitHub.com and GitLab.com allow it; ask the administrator of a self-hosted server to enable Git protocol version 2 or uploadpack.allowAnySHA1InWant.", nil)
	}
	return fail(code, problem, nil)
}
