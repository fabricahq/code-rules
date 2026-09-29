// Fetch exact revisions into owned bare repositories without checking out source files.

package imports

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fabricahq/code-rules/internal/gitexec"
	"github.com/fabricahq/code-rules/internal/rules"
)

// Options selects trusted process settings. Zero values use Git from PATH and a 120-second deadline.
// Environment, when nonnil, replaces the inherited environment before isolation overrides are applied.
// These are application settings, never remote-library or configuration-file fields.
type Options struct {
	GitPath     string
	Environment []string
	Timeout     time.Duration
}

// revision owns a temporary bare repository until Close. It is not safe for concurrent use with Close.
type revision struct {
	Commit    string `json:"commit"`
	directory string
	runner    gitexec.Runner
}

// Close removes all temporary repository state. Callers must handle cleanup failures.
func (r *revision) Close() error {
	if r == nil || r.directory == "" {
		return nil
	}
	if err := os.RemoveAll(r.directory); err != nil {
		return fail("cleanup-failed", "Could not remove the temporary Git repository.", err)
	}
	r.directory = ""
	return nil
}

// fetchRevision validates a source selector, fetches it, and verifies the resulting immutable identity.
// The caller owns Close on success. Failure removes temporary state and returns no partial revision.
func fetchRevision(ctx context.Context, source rules.Source, options Options) (_ *revision, err error) {
	if err := ctx.Err(); err != nil {
		return nil, gitexec.ContextFailure(err)
	}
	if options.Timeout < 0 {
		return nil, fail("invalid-options", "Git timeout must be positive or zero for the default.", nil)
	}
	if options.Timeout == 0 {
		options.Timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	address, _ := json.Marshal(source.Repository)
	if _, err := rules.ParseRepository(address, "sources."+source.Name+".repository"); err != nil {
		return nil, err
	}
	if source.Ref == "" {
		return nil, fail("invalid-source", "Specify an exact ref.", nil)
	}
	exact, err := rules.ParseGitRef(source.Ref, "sources."+source.Name+".ref")
	if err != nil {
		return nil, err
	}
	ref := exact.Name
	if exact.Kind == rules.GitRefCommit {
		ref = exact.SHA
	}
	runner, err := gitexec.Isolated(gitexec.Options{GitPath: options.GitPath, Environment: options.Environment})
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "code-rules-git-*")
	if err != nil {
		return nil, fail("temporary-storage", "Cannot create temporary Git storage.", err)
	}
	revision := &revision{directory: dir, runner: runner}
	defer func() {
		if err != nil {
			err = errors.Join(err, revision.Close())
		}
	}()
	if err := runner.RequireVersion(ctx, dir); err != nil {
		return nil, err
	}
	if _, err = runner.Output(ctx, dir, []string{"init", "--bare", "--quiet", "--template="}, 4096); err != nil {
		return nil, err
	}
	fetched, err := runner.Run(ctx, dir, []string{"fetch", "--quiet", "--depth=1", "--no-tags", "--no-auto-gc", "--no-recurse-submodules", source.Repository, ref}, 64<<10, nil)
	if err != nil {
		return nil, err
	}
	if fetched.Status != 0 {
		reachable, err := runner.Run(ctx, dir, []string{"ls-remote", "--exit-code", source.Repository, "HEAD"}, 64<<10, nil)
		if err != nil {
			return nil, err
		}
		if reachable.Status == 0 || reachable.Status == 2 {
			return nil, fail("ref-not-found", "Requested ref is missing or refused; no other revision was selected.", nil)
		}
		return nil, fail("not-found-or-no-access", "Repository not found or no access; check its address and Git credentials.", nil)
	}
	resolved, err := runner.Run(ctx, dir, []string{"rev-parse", "--verify", "FETCH_HEAD^{commit}"}, 4096, nil)
	if err != nil {
		return nil, err
	}
	if resolved.Status != 0 {
		return nil, fail("unsupported-content", "Requested tag does not resolve to a commit.", nil)
	}
	commit := strings.TrimSpace(string(resolved.Output))
	if !validObjectID(commit) || (exact.Kind == rules.GitRefCommit && commit != exact.SHA) {
		return nil, fail("git-failed", "Git returned a different or unsupported commit identity.", nil)
	}
	revision.Commit = commit
	return revision, nil
}

// validObjectID accepts canonical SHA-1 object IDs supported by the current source format.
func validObjectID(text string) bool {
	decoded, err := hex.DecodeString(text)
	return err == nil && len(decoded) == 20 && fmt.Sprintf("%x", decoded) == text
}
