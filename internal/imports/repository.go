// Fetch commits, tags, and selected blobs of one library into an owned partial repository, without a checkout.

package imports

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
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

// repository owns a temporary bare repository that fetches one library as a partial clone: commits and trees
// arrive with depth 1 and no blobs, and blobs arrive only when requested. It isn't safe for concurrent use.
type repository struct {
	// source names the configured source in diagnostics.
	source    string
	directory string
	runner    gitexec.Runner
	// present lists the commits already fetched, so later fetches request only missing ones.
	present map[string]bool
	// trees caches each fetched commit's parsed file tree, and owned indexes it by the rule each file belongs to.
	trees map[string]map[string]treeEntry
	owned map[string]map[string]map[string]treeEntry
}

// fetchRemote is the name of the library's remote; Git records a partial clone's promisor by remote name.
const fetchRemote = "origin"

// openRepository validates the source's address and creates the temporary repository. The caller owns Close
// on success; failure removes temporary state.
func openRepository(ctx context.Context, source rules.Source, options Options) (_ *repository, err error) {
	if err := ctx.Err(); err != nil {
		return nil, gitexec.ContextFailure(err)
	}
	address, _ := json.Marshal(source.Repository)
	if _, err := rules.ParseRepository(address, "sources."+source.Name+".repository"); err != nil {
		return nil, err
	}
	runner, err := gitexec.Isolated(gitexec.Options{GitPath: options.GitPath, Environment: options.Environment})
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "code-rules-git-*")
	if err != nil {
		return nil, fail("temporary-storage", "Cannot create temporary Git storage.", err)
	}
	repo := &repository{source: source.Name, directory: dir, runner: runner, present: map[string]bool{}, trees: map[string]map[string]treeEntry{}, owned: map[string]map[string]map[string]treeEntry{}}
	defer func() {
		if err != nil {
			err = errors.Join(err, repo.Close())
		}
	}()
	if err := runner.RequireVersion(ctx, dir); err != nil {
		return nil, err
	}
	if _, err = runner.Output(ctx, dir, []string{"init", "--bare", "--quiet", "--template="}, 4096); err != nil {
		return nil, err
	}
	if _, err = runner.Output(ctx, dir, []string{"remote", "add", fetchRemote, source.Repository}, 4096); err != nil {
		return nil, err
	}
	return repo, nil
}

// Close removes all temporary repository state. Callers must handle cleanup failures.
func (r *repository) Close() error {
	if r == nil || r.directory == "" {
		return nil
	}
	if err := os.RemoveAll(r.directory); err != nil {
		return fail("cleanup-failed", "Could not remove the temporary Git repository.", err)
	}
	r.directory = ""
	return nil
}

// fetch requests refspecs from the remote without blobs other than those it names, returning whether Git
// succeeded. With commits, commits arrive without their history. Refspecs travel on standard input, so their
// number doesn't bound the command line.
func (r *repository) fetch(ctx context.Context, refspecs []string, commits bool) (bool, error) {
	args := []string{"-c", "fetch.negotiationAlgorithm=noop", "fetch", "--quiet", "--filter=blob:none", "--no-tags", "--no-auto-gc", "--no-recurse-submodules", "--no-write-fetch-head", "--stdin", fetchRemote}
	if commits {
		args = slices.Insert(args, 4, "--depth=1")
	}
	result, err := r.runner.Run(ctx, r.directory, args, 1<<20, []byte(strings.Join(refspecs, "\n")+"\n"))
	if err != nil {
		return false, err
	}
	return result.Status == 0, nil
}

// unreachable reports why a failed fetch failed: a repository that can't be reached, or nil when the repository
// answers, so the requested revision is missing.
func (r *repository) unreachable(ctx context.Context) error {
	result, err := r.runner.Run(ctx, r.directory, []string{"ls-remote", "--exit-code", fetchRemote, "HEAD"}, 64<<10, nil)
	if err != nil {
		return err
	}
	if result.Status == 0 || result.Status == 2 {
		return nil
	}
	return fail("not-found-or-no-access", "Repository not found or no access; check its address and Git credentials.", nil)
}

// hasBranch reports whether the library has a branch named name, to explain a ref that names no tag.
func (r *repository) hasBranch(ctx context.Context, name string) (bool, error) {
	result, err := r.runner.Run(ctx, r.directory, []string{"ls-remote", "--exit-code", "--heads", fetchRemote, "refs/heads/" + name}, 64<<10, nil)
	if err != nil {
		return false, err
	}
	return result.Status == 0, nil
}

// fetchCommits fetches each commit that isn't present yet. A commit the repository doesn't have fails with
// code version-not-found and missing's explanation.
func (r *repository) fetchCommits(ctx context.Context, commits []string, missing string) error {
	wanted := []string{}
	for _, commit := range commits {
		if !r.present[commit] && !slices.Contains(wanted, commit) {
			wanted = append(wanted, commit)
		}
	}
	if len(wanted) == 0 {
		return nil
	}
	slices.Sort(wanted)
	ok, err := r.fetch(ctx, wanted, true)
	if err != nil {
		return err
	}
	if !ok {
		if err := r.unreachable(ctx); err != nil {
			return err
		}
		return fail("version-not-found", missing, nil)
	}
	for _, commit := range wanted {
		if _, err := r.commit(ctx, commit); err != nil {
			return err
		}
	}
	return nil
}

// fetchRef fetches ref, the source's parsed ref, and returns its commit. A revision the repository doesn't have
// fails with code version-not-found.
func (r *repository) fetchRef(ctx context.Context, source rules.Source, ref rules.GitRef) (string, error) {
	refspec := ref.SHA
	if ref.Kind == rules.GitRefTag {
		refspec = "+" + ref.Name + ":" + ref.Name
	}
	ok, err := r.fetch(ctx, []string{refspec}, true)
	if err != nil {
		return "", err
	}
	if !ok {
		if err := r.unreachable(ctx); err != nil {
			return "", err
		}
		if ref.Kind == rules.GitRefTag {
			branch, err := r.hasBranch(ctx, strings.TrimPrefix(ref.Name, "refs/tags/"))
			if err != nil {
				return "", err
			}
			if branch {
				return "", fail("version-not-found", fmt.Sprintf("sources.%s.ref: %s is a branch; ref accepts a tag or a full commit SHA, not a branch, so every import can be reproduced.", source.Name, source.Ref), nil)
			}
		}
		return "", fail("version-not-found", fmt.Sprintf("sources.%s.ref: the library has no tag or commit %s; check the ref.", source.Name, source.Ref), nil)
	}
	resolve := ref.SHA
	if ref.Kind == rules.GitRefTag {
		resolve = ref.Name
	}
	commit, err := r.commit(ctx, resolve)
	if err != nil {
		return "", err
	}
	if ref.Kind == rules.GitRefCommit && commit != ref.SHA {
		return "", fail("git-failed", "Git returned a different or unsupported commit identity.", nil)
	}
	return commit, nil
}

// commit resolves a fetched revision to its commit and marks it present; a tag of something else is unsupported.
func (r *repository) commit(ctx context.Context, revision string) (string, error) {
	resolved, err := r.runner.Run(ctx, r.directory, []string{"rev-parse", "--verify", "--quiet", "--end-of-options", revision + "^{commit}"}, 4096, nil)
	if err != nil {
		return "", err
	}
	if resolved.Status != 0 {
		return "", fail("unsupported-content", "Requested tag does not resolve to a commit.", nil)
	}
	commit := strings.TrimSpace(string(resolved.Output))
	if !validObjectID(commit) {
		return "", fail("git-failed", "Git returned a different or unsupported commit identity.", nil)
	}
	r.present[commit] = true
	return commit, nil
}

// tree returns the parsed file tree of a fetched commit, listing it once. Listing needs no blobs.
func (r *repository) tree(ctx context.Context, commit string) (map[string]treeEntry, error) {
	if entries, ok := r.trees[commit]; ok {
		return entries, nil
	}
	data, err := r.runner.Output(ctx, r.directory, []string{"ls-tree", "-r", "-z", "--full-tree", commit}, maxTreeBytes)
	if err != nil {
		return nil, err
	}
	entries, err := parseTree(data)
	if err != nil {
		return nil, err
	}
	r.trees[commit] = entries
	return entries, nil
}

// prefetch fetches the blobs among entries that aren't present yet in one request, so reading them later needs
// no request per file. Entries that aren't ordinary files, such as submodules, are skipped for the loader to
// reject if it reads them. Reading a blob that wasn't prefetched still fetches it on its own.
func (r *repository) prefetch(ctx context.Context, entries []treeEntry) error {
	objects := []string{}
	for _, entry := range entries {
		if entry.mode.IsRegular() {
			objects = append(objects, entry.object)
		}
	}
	if len(objects) == 0 {
		return nil
	}
	listing, err := r.runner.Output(ctx, r.directory, []string{"cat-file", "--batch-all-objects", "--batch-check=%(objectname)"}, 64<<20)
	if err != nil {
		return err
	}
	present := map[string]bool{}
	for line := range bytes.Lines(listing) {
		present[string(bytes.TrimSuffix(line, []byte("\n")))] = true
	}
	missing := []string{}
	for _, object := range objects {
		if !present[object] {
			present[object] = true
			missing = append(missing, object)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	slices.Sort(missing)
	ok, err := r.fetch(ctx, missing, false)
	if err != nil {
		return err
	}
	if !ok {
		return fail("git-failed", "Could not fetch library files.", nil)
	}
	return nil
}

// validObjectID accepts canonical SHA-1 object IDs supported by the current source format.
func validObjectID(text string) bool {
	decoded, err := hex.DecodeString(text)
	return err == nil && len(decoded) == 20 && fmt.Sprintf("%x", decoded) == text
}
