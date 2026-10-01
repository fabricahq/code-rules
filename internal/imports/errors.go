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

// remoteFailure explains a Git command that couldn't read the library's repository, from its diagnostics, which it
// never shows: a server it couldn't verify or reach, as unreachable explains; a repository that doesn't exist or
// that the credentials can't read, with code not-found-or-no-access, since servers report both the same way; or
// otherwise a failure it doesn't recognize, with code git-failed.
func remoteFailure(diagnostics []byte) error {
	if err := unreachable(diagnostics); err != nil {
		return err
	}
	if gitexec.Mentions(diagnostics, gitexec.AccessFailures...) {
		return fail("not-found-or-no-access", "Repository not found or no access; check its address and Git credentials.", nil)
	}
	return fail("git-failed", "Could not read the library's repository for a reason Code Rules doesn't recognize. Run git ls-remote with the repository address to read Git's message.", nil)
}

// gitFailure explains a Git command that failed while reading the library, from its diagnostics, which it never
// shows: a server it couldn't verify or reach, as unreachable explains; a server that refused to send files by
// object ID, with code object-fetch-refused; or otherwise problem, with code.
func gitFailure(code, problem string, diagnostics []byte) error {
	if err := unreachable(diagnostics); err != nil {
		return err
	}
	if gitexec.Mentions(diagnostics, objectRefusals...) {
		return fail("object-fetch-refused", "The library's server refused to send a file by its object ID, which Code Rules needs to read one version of each rule without downloading the whole repository. GitHub.com and GitLab.com allow it; ask the administrator of a self-hosted server to enable Git protocol version 2 or uploadpack.allowAnySHA1InWant.", nil)
	}
	return fail(code, problem, nil)
}

// unreachable explains diagnostics that show Git never reached a server it could trust: a TLS certificate it
// couldn't verify, with code certificate-failed; an SSH host key it couldn't verify, with code host-key-failed; or a
// host it couldn't connect to, with code connection-failed. It returns nil for other diagnostics.
func unreachable(diagnostics []byte) error {
	switch {
	case gitexec.Mentions(diagnostics, gitexec.CertificateFailures...):
		return fail("certificate-failed", "Git couldn't verify the library server's TLS certificate. Check that your system trusts it: Git's http.sslCAInfo setting, your system's certificate store, and any proxy that intercepts TLS.", nil)
	case gitexec.Mentions(diagnostics, gitexec.HostKeyFailures...):
		return fail("host-key-failed", "Git couldn't verify the library server's SSH host key. Check the server's entry in your known_hosts file.", nil)
	case gitexec.Mentions(diagnostics, gitexec.ConnectionFailures...):
		return fail("connection-failed", "Could not connect to the library's repository. Check the repository address and your network connection.", nil)
	}
	return nil
}
