// Fetch exact revisions into owned bare repositories without checking out source files.

package imports

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

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

// Revision owns a temporary bare repository until Close. It is not safe for concurrent use with Close.
// Tag and Version are present only for a version-constraint selection.
type Revision struct {
	Commit    string `json:"commit"`
	Tag       string `json:"resolvedTag,omitempty"`
	Version   string `json:"resolvedVersion,omitempty"`
	directory string
	runner    gitRunner
}

// Close removes all temporary repository state. Callers must handle cleanup failures.
func (r *Revision) Close() error {
	if r == nil || r.directory == "" {
		return nil
	}
	if err := os.RemoveAll(r.directory); err != nil {
		return fail("cleanup-failed", "Could not remove the temporary Git repository.", err)
	}
	r.directory = ""
	return nil
}

var gitVersion = regexp.MustCompile(`^git version ([0-9]+)\.([0-9]+)`)

// FetchRevision validates a source selector, fetches it, and verifies the resulting immutable identity.
// The caller owns Close on success. Failure removes temporary state and returns no partial revision.
func FetchRevision(ctx context.Context, source rules.Source, options Options) (_ *Revision, err error) {
	if err := ctx.Err(); err != nil {
		return nil, fail("cancelled", "Git operation cancelled or timed out.", err)
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
	if (source.Ref == "") == (source.Version == "") {
		return nil, fail("invalid-source", "Specify exactly one ref or version constraint.", nil)
	}
	var ref string
	var exact rules.GitRef
	var constraint rules.VersionConstraint
	if source.Ref != "" {
		exact, err = rules.ParseGitRef(source.Ref, "sources."+source.Name+".ref")
		if err != nil {
			return nil, err
		}
		ref = exact.Name
		if exact.Kind == rules.GitRefCommit {
			ref = exact.SHA
		}
	} else {
		constraint, err = rules.ParseVersionConstraint(source.Version, "sources."+source.Name+".version")
		if err != nil {
			return nil, err
		}
	}
	if options.GitPath == "" {
		options.GitPath = "git"
	}
	env := options.Environment
	if env == nil {
		env = os.Environ()
	}
	runner := gitRunner{options.GitPath, gitEnvironment(env)}
	dir, err := os.MkdirTemp("", "code-rules-git-*")
	if err != nil {
		return nil, fail("temporary-storage", "Cannot create temporary Git storage.", err)
	}
	revision := &Revision{directory: dir, runner: runner}
	defer func() {
		if err != nil {
			err = errors.Join(err, revision.Close())
		}
	}()
	version, err := runner.command(ctx, dir, []string{"--version"}, 4096)
	if err != nil {
		return nil, err
	}
	parts := gitVersion.FindStringSubmatch(string(version))
	major, minor := 0, 0
	if parts != nil {
		major, _ = strconv.Atoi(parts[1])
		minor, _ = strconv.Atoi(parts[2])
	}
	if major < 2 || (major == 2 && minor < 30) {
		return nil, fail("git-unavailable", "Git 2.30 or later is required.", nil)
	}
	if _, err = runner.command(ctx, dir, []string{"init", "--bare", "--quiet", "--template="}, 4096); err != nil {
		return nil, err
	}
	var selection rules.VersionSelection
	if source.Version != "" {
		available, err := runner.run(ctx, dir, []string{"ls-remote", "--tags", source.Repository}, 8<<20, nil)
		if err != nil {
			return nil, err
		}
		if available.status != 0 {
			return nil, fail("not-found-or-no-access", "Repository not found or no access; check its address and Git credentials.", nil)
		}
		selection, err = rules.SelectReleaseTag(string(available.output), constraint)
		if err != nil {
			return nil, err
		}
		ref = "refs/tags/" + selection.Tag
	}
	fetched, err := runner.run(ctx, dir, []string{"fetch", "--quiet", "--depth=1", "--no-tags", "--no-auto-gc", "--no-recurse-submodules", source.Repository, ref}, 64<<10, nil)
	if err != nil {
		return nil, err
	}
	if fetched.status != 0 {
		reachable, err := runner.run(ctx, dir, []string{"ls-remote", "--exit-code", source.Repository, "HEAD"}, 64<<10, nil)
		if err != nil {
			return nil, err
		}
		if reachable.status == 0 || reachable.status == 2 {
			return nil, fail("ref-not-found", "Requested ref is missing or refused; no other revision was selected.", nil)
		}
		return nil, fail("not-found-or-no-access", "Repository not found or no access; check its address and Git credentials.", nil)
	}
	if selection.Tag != "" {
		object, err := runner.command(ctx, dir, []string{"rev-parse", "--verify", "FETCH_HEAD"}, 4096)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(object)) != selection.Object {
			return nil, fail("ref-changed", "Selected version tag changed during import; retry to resolve it again.", nil)
		}
	}
	resolved, err := runner.run(ctx, dir, []string{"rev-parse", "--verify", "FETCH_HEAD^{commit}"}, 4096, nil)
	if err != nil {
		return nil, err
	}
	if resolved.status != 0 {
		return nil, fail("unsupported-content", "Requested tag does not resolve to a commit.", nil)
	}
	commit := strings.TrimSpace(string(resolved.output))
	if !validObjectID(commit) || (exact.Kind == rules.GitRefCommit && commit != exact.SHA) {
		return nil, fail("git-failed", "Git returned a different or unsupported commit identity.", nil)
	}
	revision.Commit = commit
	revision.Tag = selection.Tag
	revision.Version = selection.Version
	return revision, nil
}

// validObjectID accepts canonical SHA-1 object IDs supported by the current source format.
func validObjectID(text string) bool {
	decoded, err := hex.DecodeString(text)
	return err == nil && len(decoded) == 20 && fmt.Sprintf("%x", decoded) == text
}
