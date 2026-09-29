// Compare the library author's clone with its upstream remote, then create and push one release tag.

package library

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
)

// maxRemoteListing bounds the output of listing the remote's default branch and release tags.
const maxRemoteListing = 8 * 1024 * 1024

// upstream is where the checked-out branch publishes.
type upstream struct {
	// branch is the checked-out branch's short name, such as main.
	branch string
	// remote is the upstream remote's name, such as origin, and url its fetch URL after insteadOf rewriting.
	remote, url string
	// ref is the branch on the remote, such as refs/heads/main.
	ref string
	// tracking is the local remote-tracking branch, such as refs/remotes/origin/main, or "" when there is none.
	tracking string
}

// remoteState is what the upstream remote holds, read without fetching.
type remoteState struct {
	// defaultBranch is the ref the remote's HEAD names, such as refs/heads/main.
	defaultBranch string
	// head is the commit at the upstream branch, or "" when the remote lacks it.
	head string
	// tags maps each release/<number> tag's number to its object ID.
	tags map[int]string
}

// latest returns the highest release number on the remote, or 0 before the first library release.
func (r remoteState) latest() int {
	if len(r.tags) == 0 {
		return 0
	}
	return slices.Max(slices.Collect(maps.Keys(r.tags)))
}

// localTag is a release tag in the author's clone.
type localTag struct {
	object, commit string
}

// upstream finds the checked-out branch and its upstream remote. It fails on a detached HEAD, a branch
// without an upstream on a remote, and a remote without a URL.
func (g *libraryGit) upstream(ctx context.Context) (upstream, error) {
	symbolic, err := g.runner.Run(ctx, g.dir, []string{"symbolic-ref", "--quiet", "HEAD"}, 4096, nil)
	if err != nil {
		return upstream{}, fmt.Errorf("find the checked-out branch: %w", err)
	}
	ref := strings.TrimSpace(string(symbolic.Output))
	branch, onBranch := strings.CutPrefix(ref, "refs/heads/")
	if symbolic.Status != 0 || !onBranch {
		return upstream{}, failure("detached-head", "HEAD isn't on a branch. Check out the library's default branch, then run code-rules library release again.", nil)
	}
	listing, err := g.runner.Output(ctx, g.dir, []string{"for-each-ref", "--format=%(upstream:remotename)%00%(upstream:remoteref)%00%(upstream)", ref}, 64*1024)
	if err != nil {
		return upstream{}, fmt.Errorf("find the upstream of branch=%q: %w", branch, err)
	}
	fields := strings.Split(strings.TrimSuffix(string(listing), "\n"), "\x00")
	if len(fields) != 3 || fields[0] == "" || fields[0] == "." || !strings.HasPrefix(fields[1], "refs/heads/") {
		return upstream{}, failure("no-upstream", branch+" has no upstream branch on a remote, so there's nothing to compare it with. Push it with git push --set-upstream, then run code-rules library release again.", nil)
	}
	result := upstream{branch: branch, remote: fields[0], ref: fields[1]}
	if strings.HasPrefix(fields[2], "refs/remotes/") {
		result.tracking = fields[2]
	}
	url, err := g.runner.Output(ctx, g.dir, []string{"remote", "get-url", "--", result.remote}, 64*1024)
	if err != nil {
		return upstream{}, fmt.Errorf("find the URL of remote=%q: %w", result.remote, err)
	}
	result.url = strings.TrimSpace(string(url))
	return result, nil
}

// requireCommitterIdentity fails before anything changes when Git has no tagger identity for the tag.
func (g *libraryGit) requireCommitterIdentity(ctx context.Context) error {
	result, err := g.runner.Run(ctx, g.dir, []string{"var", "GIT_COMMITTER_IDENT"}, 64*1024, nil)
	if err != nil {
		return fmt.Errorf("find Git's committer identity: %w", err)
	}
	if result.Status != 0 {
		return failure("git-identity", "Git doesn't know your name and email, which the release tag records as its tagger. Set user.name and user.email, or GIT_COMMITTER_NAME and GIT_COMMITTER_EMAIL, then run code-rules library release again.", nil)
	}
	return nil
}

// headCommit returns the checked-out commit; a branch without commits fails.
func (g *libraryGit) headCommit(ctx context.Context) (string, error) {
	result, err := g.runner.Run(ctx, g.dir, []string{"rev-parse", "--verify", "--quiet", "HEAD^{commit}"}, 4096, nil)
	if err != nil {
		return "", fmt.Errorf("find the library's HEAD commit: %w", err)
	}
	head := strings.TrimSpace(string(result.Output))
	if result.Status != 0 || !objectID.MatchString(head) {
		return "", failure("no-commits", "the library has no commits yet. Commit and push it, then run code-rules library release again.", nil)
	}
	return head, nil
}

// readRemote lists the remote's default branch, the upstream branch's commit, and its release tags, as
// parseRemoteListing reads them.
func (g *libraryGit) readRemote(ctx context.Context, u upstream) (remoteState, error) {
	result, err := g.runner.Run(ctx, g.dir, []string{"ls-remote", "--symref", "--", u.remote, "HEAD", u.ref, "refs/tags/release/*"}, maxRemoteListing, nil)
	if err != nil {
		return remoteState{}, fmt.Errorf("list the branches and release tags of remote=%q: %w", u.remote, err)
	}
	if result.Status != 0 {
		return remoteState{}, failure("fetch-failed", "Git couldn't read "+u.remote+". Check your network connection and access to the repository, then run code-rules library release again.", nil)
	}
	return parseRemoteListing(string(result.Output), u)
}

// parseRemoteListing reads git ls-remote --symref output for the remote's HEAD, the upstream branch, and its
// release tags. It fails when the listing is malformed, has more than 20,000 tag records, or shows that the
// upstream branch isn't the remote's default branch.
func parseRemoteListing(listing string, u upstream) (remoteState, error) {
	state := remoteState{tags: map[int]string{}}
	records := 0
	for line := range strings.Lines(listing) {
		value, name, ok := strings.Cut(strings.TrimSuffix(line, "\n"), "\t")
		if !ok {
			return remoteState{}, failure("git-failed", "Git listed "+u.remote+"'s references in an unexpected format", nil)
		}
		if target, symbolic := strings.CutPrefix(value, "ref: "); symbolic {
			if name == "HEAD" {
				state.defaultBranch = target
			}
			continue
		}
		if !objectID.MatchString(value) {
			return remoteState{}, failure("git-failed", "Git listed "+u.remote+"'s references in an unexpected format", nil)
		}
		if name == u.ref {
			state.head = value
		}
		// Like imports, count every tag record, including the peeled ^{} records of annotated tags.
		if strings.HasPrefix(name, "refs/tags/") {
			if records++; records > maxReleaseTags {
				return remoteState{}, failure("limit-exceeded", u.remote+"'s listing of release tags exceeds 20,000 records", nil)
			}
		}
		// Code Rules ignores other tags under release/, such as release/01, and peeled ^{} entries.
		if number, err := rules.ParseReleaseTag(strings.TrimPrefix(name, "refs/tags/")); err == nil && strings.HasPrefix(name, "refs/tags/") {
			state.tags[number] = value
		}
	}
	if state.defaultBranch == "" {
		return remoteState{}, failure("not-default-branch", u.remote+" doesn't report a default branch, so code-rules library release can't confirm it's publishing from it.", nil)
	}
	if u.ref != state.defaultBranch {
		return remoteState{}, failure("not-default-branch", "library releases are published from "+u.remote+"'s default branch, "+strings.TrimPrefix(state.defaultBranch, "refs/heads/")+", but "+u.branch+" tracks "+strings.TrimPrefix(u.ref, "refs/heads/")+". Check out "+strings.TrimPrefix(state.defaultBranch, "refs/heads/")+", then run code-rules library release again.", nil)
	}
	return state, nil
}

// localReleaseTags maps each release/<number> tag in the clone, reachable or not, to its object and commit.
func (g *libraryGit) localReleaseTags(ctx context.Context) (map[int]localTag, error) {
	listing, err := g.runner.Output(ctx, g.dir, []string{"for-each-ref", "--format=%(refname)%00%(objectname)%00%(*objectname)", "refs/tags/release/"}, maxRemoteListing)
	if err != nil {
		return nil, fmt.Errorf("list the library's release tags: %w", err)
	}
	tags := map[int]localTag{}
	for line := range strings.Lines(string(listing)) {
		fields := strings.Split(strings.TrimSuffix(line, "\n"), "\x00")
		if len(fields) != 3 {
			return nil, failure("git-failed", "Git listed release tags in an unexpected format", nil)
		}
		if number, err := rules.ParseReleaseTag(strings.TrimPrefix(fields[0], "refs/tags/")); err == nil {
			tags[number] = localTag{object: fields[1], commit: fields[2]}
			if len(tags) > maxReleaseTags {
				return nil, failure("limit-exceeded", "library has more than 20,000 release tags", nil)
			}
		}
	}
	return tags, nil
}

// syncReleaseTags fetches the upstream branch and every release tag the clone lacks. It refuses when a
// release tag in the clone differs from the remote's, or when the clone has a release tag the remote lacks,
// unless that tag is on head and numbered one past the remote's latest: a run that stopped before pushing its
// tag. It returns that tag's number, or 0.
func (g *libraryGit) syncReleaseTags(ctx context.Context, u upstream, remote remoteState, head string) (int, error) {
	local, err := g.localReleaseTags(ctx)
	if err != nil {
		return 0, err
	}
	refspecs := []string{u.ref}
	if u.tracking != "" {
		refspecs[0] = "+" + u.ref + ":" + u.tracking
	}
	for _, number := range slices.Sorted(maps.Keys(remote.tags)) {
		name := "release/" + strconv.Itoa(number)
		tag, ok := local[number]
		switch {
		case !ok:
			refspecs = append(refspecs, "refs/tags/"+name+":refs/tags/"+name)
		case tag.object != remote.tags[number]:
			return 0, failure("release-tag-mismatch", name+" in this clone differs from "+name+" on "+u.remote+". Release tags must never change, because projects may have imported them. Find out which one is the original and restore it; to discard this clone's copy, run git tag --delete "+name+".", nil)
		}
	}
	unpublished := 0
	for _, number := range slices.Sorted(maps.Keys(local)) {
		if _, published := remote.tags[number]; published {
			continue
		}
		name := "release/" + strconv.Itoa(number)
		if number != remote.latest()+1 || local[number].commit != head {
			return 0, failure("release-tag-mismatch", name+" exists in this clone but not on "+u.remote+". Only code-rules library release creates release tags; if you created it by hand, delete it with git tag --delete "+name+", then run code-rules library release again.", nil)
		}
		unpublished = number
	}
	result, err := g.runner.Run(ctx, g.dir, []string{"fetch", "--no-tags", "--stdin", "--", u.remote}, 1024*1024, []byte(strings.Join(refspecs, "\n")+"\n"))
	if err != nil {
		return 0, fmt.Errorf("fetch remote=%q: %w", u.remote, err)
	}
	if result.Status != 0 {
		return 0, failure("fetch-failed", "Git couldn't fetch from "+u.remote+". Check your network connection and access to the repository, then run code-rules library release again.", nil)
	}
	return unpublished, nil
}

// requireCurrent refuses unless head is the commit at the remote's branch, explaining how they differ.
func (g *libraryGit) requireCurrent(ctx context.Context, u upstream, remoteHead, head string) error {
	remoteName := u.remote + "/" + strings.TrimPrefix(u.ref, "refs/heads/")
	if remoteHead == "" {
		return failure("branch-differs", remoteName+" doesn't exist. Push "+u.branch+", then run code-rules library release again.", nil)
	}
	if remoteHead == head {
		return nil
	}
	counts, err := g.runner.Output(ctx, g.dir, []string{"rev-list", "--left-right", "--count", head + "..." + remoteHead}, 4096)
	if err != nil {
		return fmt.Errorf("compare branch=%q with %s: %w", u.branch, remoteName, err)
	}
	fields := strings.Fields(string(counts))
	if len(fields) != 2 {
		return failure("git-failed", "Git compared the branches in an unexpected format", nil)
	}
	ahead, behind := fields[0], fields[1]
	switch {
	case behind == "0":
		return failure("branch-differs", u.branch+" has "+commits(ahead)+" that "+remoteName+" doesn't. A library release publishes only pushed commits: push them, then run code-rules library release again.", nil)
	case ahead == "0":
		return failure("branch-differs", remoteName+" has "+commits(behind)+" that "+u.branch+" doesn't. Pull them, then run code-rules library release again.", nil)
	}
	return failure("branch-differs", u.branch+" and "+remoteName+" have diverged: each has commits the other doesn't. Reconcile them, then run code-rules library release again.", nil)
}

// commits counts commits in prose, such as "1 commit" or "3 commits".
func commits(count string) string {
	if count == "1" {
		return "1 commit"
	}
	return count + " commits"
}

// requireCommitted refuses when the library files check read differ from head's, so the library release
// publishes exactly what check validated. It returns head's library-owned files, mapped to their blob IDs.
func (g *libraryGit) requireCommitted(ctx context.Context, head string, input checkInput) (map[string]string, error) {
	committed, err := g.treeFiles(ctx, head, libraryPaths(input.license))
	if err != nil {
		return nil, fmt.Errorf("list the files of commit=%s: %w", head, err)
	}
	captured := slices.Sorted(maps.Keys(input.tree.Files))
	captured = append(captured, slices.Sorted(maps.Keys(input.notes))...)
	hashes, err := g.hashFiles(ctx, captured)
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, name := range captured {
		switch blob, ok := committed[name]; {
		case !ok:
			problems = append(problems, name+" isn't committed")
		case blob != hashes[name]:
			problems = append(problems, name+" has uncommitted changes")
		}
	}
	for name := range committed {
		if _, ok := hashes[name]; !ok {
			problems = append(problems, name+" was deleted, and the deletion isn't committed")
		}
	}
	if len(problems) > 0 {
		slices.Sort(problems)
		return nil, failure("uncommitted-changes", "a library release publishes the checked-out commit, but the library has changes that aren't committed:\n  - "+strings.Join(problems, "\n  - ")+"\nCommit and push them, or discard them, then run code-rules library release again.", nil)
	}
	return committed, nil
}

// libraryPaths are the paths holding every library-owned file and change note, given the declared license.
func libraryPaths(license *rules.LicenseDeclaration) []string {
	return append([]string{"rule-library.yaml", "assets", "practices", "techs", changesDirectory}, rules.LicensePaths(license)...)
}

// changedLibraryFiles lists the library-wide files that differ between the latest library release and head,
// including added and deleted files, in path order. Before the first library release, released is empty, so
// every library-wide file is listed. terms are the declared license and notice files.
func changedLibraryFiles(head, released map[string]string, terms []string) []string {
	files := []string{}
	for _, name := range slices.Sorted(maps.Keys(head)) {
		if libraryWide(name, terms) && head[name] != released[name] {
			files = append(files, name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(released)) {
		if _, kept := head[name]; !kept && libraryWide(name, terms) {
			files = append(files, name)
		}
	}
	slices.Sort(files)
	return files
}

// libraryWide reports whether a file is one projects receive outside any rule's version: the manifest, the
// declared license and notice files, files in the library-root assets/ directory, and group metadata in
// _group.yaml. Group READMEs are authoring notes that projects never receive, so they're neither.
func libraryWide(name string, terms []string) bool {
	if name == "rule-library.yaml" || slices.Contains(terms, name) || strings.HasPrefix(name, "assets/") {
		return true
	}
	parts := strings.Split(name, "/")
	return len(parts) == 3 && (parts[0] == "practices" || parts[0] == "techs") && parts[2] == "_group.yaml"
}

// createTag writes an annotated tag on commit with exactly message, using the author's tagger identity and
// tag signing configuration, and returns the tag object's ID.
func (g *libraryGit) createTag(ctx context.Context, name, commit string, message []byte) (string, error) {
	result, err := g.runner.Run(ctx, g.dir, []string{"tag", "--annotate", "--cleanup=verbatim", "--file=-", "--", name, commit}, 64*1024, message)
	if err != nil {
		return "", fmt.Errorf("create tag=%q: %w", name, err)
	}
	if result.Status != 0 {
		return "", failure("tag-failed", "Git couldn't create the tag "+name+". Check that it doesn't already exist and that tag signing, if configured, works, then run code-rules library release again.", nil)
	}
	object, err := g.runner.Output(ctx, g.dir, []string{"rev-parse", "--verify", "refs/tags/" + name}, 4096)
	if err != nil {
		return "", fmt.Errorf("read tag=%q: %w", name, err)
	}
	return strings.TrimSpace(string(object)), nil
}

// pushTag pushes only the tag, running the author's pre-push hook. If the push fails, it deletes the local
// tag, so the clone never keeps a release tag the remote lacks; a remote that already has a different tag of
// that name fails with release-conflict.
func (g *libraryGit) pushTag(ctx context.Context, u upstream, name, object string) error {
	ref := "refs/tags/" + name
	pushed, err := g.runner.Run(ctx, g.dir, []string{"push", "--no-follow-tags", "--", u.remote, ref + ":" + ref}, 1024*1024, nil)
	if err == nil && pushed.Status == 0 {
		return nil
	}
	problem := failure("push-failed", "Git couldn't push "+name+" to "+u.remote+". Check your access to the repository and any pre-push hook, then run code-rules library release again.", err)
	if err == nil {
		switch remote, listErr := g.runner.Run(ctx, g.dir, []string{"ls-remote", "--", u.remote, ref}, 64*1024, nil); {
		case listErr != nil || remote.Status != 0:
		case strings.HasPrefix(string(remote.Output), object+"\t"):
			return nil
		case len(remote.Output) > 0:
			problem = failure("release-conflict", "someone else published "+name+" to "+u.remote+" first. Pull their changes, then run code-rules library release again if anything remains to publish.", nil)
		}
	}
	// Delete the tag even after cancellation, which may have interrupted the push.
	deleted, deleteErr := g.runner.Run(context.WithoutCancel(ctx), g.dir, []string{"update-ref", "-d", ref, object}, 4096, nil)
	if deleteErr != nil || deleted.Status != 0 {
		return fmt.Errorf("%w The local tag remains; delete it with git tag --delete %s", problem, name)
	}
	return problem
}
