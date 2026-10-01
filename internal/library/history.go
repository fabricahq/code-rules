// Read the library's own release tags and compare working-tree files with the latest library release.

package library

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/fabricahq/code-rules/internal/gitexec"
	"github.com/fabricahq/code-rules/internal/releasetag"
	"github.com/fabricahq/code-rules/libraryformat"
)

// maxReleaseTags bounds the release tags one check reads, matching the tag-listing limit for imports.
const maxReleaseTags = 20_000

// objectID accepts a full SHA-1 or SHA-256 object name.
var objectID = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// shallowClone tells the author how to get the history that comparing with a library release needs.
const shallowClone = "this clone has partial history, so it can't be compared with the latest library release. Fetch the full history and tags, such as with git fetch --unshallow --tags; in CI, check out with fetch-depth: 0."

// missingReleaseTags explains a clone whose commit has change notes but no release tags, such as a clone made with
// git clone --no-tags, which would otherwise look like a library that never published a library release.
const missingReleaseTags = "changes/ holds change notes, but this clone has no release/<number> tags, so it can't be compared with the latest library release. Fetch the tags, such as with git fetch --tags; in CI, check out with fetch-depth: 0. If the library has never published a library release, delete the notes in changes/, which the first library release doesn't need."

// libraryGit runs Git in the library author's own repository, honoring their Git configuration.
type libraryGit struct {
	runner gitexec.Runner
	dir    string
	// unpublished is the number of a release tag that exists only in this clone, which the release history leaves
	// out, so it's checked against the library instead of trusted; it's 0 when there is none.
	unpublished int
}

// withoutUnpublished returns g with a release history that leaves out release tag number, which exists only in
// this clone. A number of 0 leaves out nothing.
func (g *libraryGit) withoutUnpublished(number int) *libraryGit {
	copied := *g
	copied.unpublished = number
	return &copied
}

// releaseHistory is what the library releases reachable from HEAD record.
type releaseHistory struct {
	// latest is nil before the first library release.
	latest *publishedRelease
	// retired maps every rule a reachable library release retired to that release's number, and replacedBy maps
	// each of those that named a replacement to it.
	retired    map[string]int
	replacedBy map[string]string
}

// publishedRelease is the latest library release reachable from HEAD.
type publishedRelease struct {
	number int
	// object is the tag object's ID, and commit the commit it tags.
	object, commit string
	// notes are the release notes before the tag message's record, which a GitHub Release page repeats.
	notes  string
	record libraryformat.ReleaseRecord
	// files maps each file under practices/, techs/, and changes/ at the release's commit to its blob ID.
	files map[string]string
}

// tagName returns the release tag's short name, such as release/4.
func (r *publishedRelease) tagName() string { return "release/" + strconv.Itoa(r.number) }

// openLibraryGit returns nil when the library root isn't a Git repository's root, so it has no release history.
// The CLI always resolves a library inside Git to its repository root.
func openLibraryGit(ctx context.Context, dir string, options gitexec.Options) (*libraryGit, error) {
	if _, err := os.Lstat(filepath.Join(dir, ".git")); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, failure("git-failed", "inspect the library's Git repository: "+err.Error(), err)
	}
	runner, err := gitexec.Owned(options)
	if err != nil {
		return nil, err
	}
	if err := runner.RequireVersion(ctx, dir); err != nil {
		return nil, err
	}
	return &libraryGit{runner: runner, dir: dir}, nil
}

// releaseTags lists the annotated release tags of commits reachable from HEAD in ascending number order, except
// the unpublished one. It fails in a shallow clone, which may lack tags and history, and with missing-release-tags
// when none is reachable but HEAD has change notes, which only a library release makes necessary. It returns none
// on an unborn branch. A nil receiver, a library outside Git, has no release tags.
func (g *libraryGit) releaseTags(ctx context.Context) ([]releasetag.Tag, error) {
	if g == nil {
		return nil, nil
	}
	if err := g.requireFullHistory(ctx); err != nil {
		return nil, err
	}
	head, err := g.runner.Run(ctx, g.dir, []string{"rev-parse", "--verify", "--quiet", "HEAD^{commit}"}, 4096, nil)
	if err != nil {
		return nil, fmt.Errorf("find the library's HEAD commit: %w", err)
	}
	if head.Status != 0 {
		return nil, nil
	}
	listed, err := g.listReleaseTags(ctx, "HEAD")
	if err != nil {
		return nil, err
	}
	tags := []releasetag.Tag{}
	for _, tag := range listed {
		if tag.Number == g.unpublished {
			continue
		}
		if tag.Type != "tag" || tag.TargetType != "commit" {
			return nil, failure("invalid-release-tag", tag.Name()+": expected an annotated tag on a commit, with a release record. Don't create release tags by hand; if you created "+tag.Name()+" by hand, delete it with git tag --delete "+tag.Name()+", then run the command again.", nil)
		}
		if tag.Size > releasetag.MaxBytes {
			return nil, failure("limit-exceeded", tag.Name()+": tag message exceeds 8 MiB", nil)
		}
		tags = append(tags, tag)
	}
	if len(tags) == 0 && g.unpublished == 0 {
		notes, err := g.runner.Run(ctx, g.dir, []string{"ls-tree", "--name-only", "HEAD", "--", changesDirectory}, 4096, nil)
		if err != nil {
			return nil, fmt.Errorf("list the library's committed change notes: %w", err)
		}
		if notes.Status == 0 && len(bytes.TrimSpace(notes.Output)) > 0 {
			return nil, failure("missing-release-tags", missingReleaseTags, nil)
		}
	}
	return tags, nil
}

// listReleaseTags lists the clone's release tags, or only those of commits reachable from merged when it isn't
// empty, in ascending number order. It fails with limit-exceeded past 20,000.
func (g *libraryGit) listReleaseTags(ctx context.Context, merged string) ([]releasetag.Tag, error) {
	tags, err := releasetag.List(ctx, g.runner, g.dir, merged)
	if err != nil {
		return nil, fmt.Errorf("list the library's release tags: %w", err)
	}
	if len(tags) > maxReleaseTags {
		return nil, failure("limit-exceeded", "library has more than 20,000 release tags", nil)
	}
	return tags, nil
}

// requireFullHistory fails with shallow-clone in a clone that may lack history and tags.
func (g *libraryGit) requireFullHistory(ctx context.Context) error {
	shallow, err := g.runner.Output(ctx, g.dir, []string{"rev-parse", "--is-shallow-repository"}, 4096)
	if err != nil {
		return fmt.Errorf("check whether the library is a shallow clone: %w", err)
	}
	if strings.TrimSpace(string(shallow)) != "false" {
		return failure("shallow-clone", shallowClone, nil)
	}
	return nil
}

// history reads every reachable release record and lists the latest release's rule and note files.
func (g *libraryGit) history(ctx context.Context) (releaseHistory, error) {
	history := releaseHistory{retired: map[string]int{}, replacedBy: map[string]string{}}
	tags, err := g.releaseTags(ctx)
	if err != nil || len(tags) == 0 {
		return history, err
	}
	// Only the latest library release's notes and record are kept; the others contribute their retirements.
	var latestRelease releasetag.Release
	err = releasetag.Read(ctx, g.runner, g.dir, tags, func(i int, release releasetag.Release) error {
		for id, retired := range release.Record.Retired {
			if _, ok := history.retired[id]; !ok {
				history.retired[id] = tags[i].Number
				if retired.ReplacedBy != "" {
					history.replacedBy[id] = retired.ReplacedBy
				}
			}
		}
		if i == len(tags)-1 {
			latestRelease = release
		}
		return nil
	})
	var invalid *releasetag.RecordError
	var unsupported *libraryformat.UnsupportedReleaseRecordError
	switch {
	case errors.As(err, &invalid) && errors.As(invalid.Err, &unsupported):
		return releaseHistory{}, unsupportedRecord(invalid.Tag, unsupported.FormatVersion)
	case invalid != nil:
		return releaseHistory{}, failure("invalid-release-tag", invalid.Error()+". Don't create or move release tags by hand; if you created "+invalid.Tag+" by hand, delete it with git tag --delete "+invalid.Tag+", then run the command again.", invalid.Err)
	}
	if err != nil {
		return releaseHistory{}, fmt.Errorf("read the library's release tags: %w", err)
	}
	latest := tags[len(tags)-1]
	files, err := g.releaseFiles(ctx, latest.Object)
	if err != nil {
		return releaseHistory{}, err
	}
	history.latest = &publishedRelease{number: latest.Number, object: latest.Object, commit: latest.Target, notes: latestRelease.Notes, record: latestRelease.Record, files: files}
	return history, nil
}

// unsupportedRecord fails with unsupported-release-record for release tag tag, whose record a newer Code Rules
// wrote in release record format version, asking the author to upgrade.
func unsupportedRecord(tag string, version int) error {
	return failure("unsupported-release-record", tag+" uses release record format "+strconv.Itoa(version)+", which this version of Code Rules can't read. Upgrade Code Rules, then run the command again.", nil)
}

// releaseFiles lists the blobs under practices/, techs/, and changes/ at a release tag's commit.
func (g *libraryGit) releaseFiles(ctx context.Context, tag string) (map[string]string, error) {
	files, err := g.treeFiles(ctx, tag, []string{"practices", "techs", "changes"})
	if err != nil {
		return nil, fmt.Errorf("list the latest library release's files: %w", err)
	}
	return files, nil
}

// treeFiles maps each file at or under paths in a commit's tree, or a tag's, to its blob ID, leaving out other
// entries, such as submodules.
func (g *libraryGit) treeFiles(ctx context.Context, treeish string, paths []string) (map[string]string, error) {
	entries, err := g.treeEntries(ctx, treeish, paths)
	if err != nil {
		return nil, err
	}
	files := map[string]string{}
	for name, entry := range entries {
		if entry.kind == "blob" {
			files[name] = entry.object
		}
	}
	return files, nil
}

// treeEntry is one entry of a recursive tree listing: a blob, including a symbolic link, or a submodule.
type treeEntry struct {
	// mode is the octal Git file mode, such as 100644 for a file or 120000 for a symbolic link.
	mode, kind, object string
}

// treeEntries maps each entry at or under paths in a commit's tree, or a tag's, to its mode, kind, and object.
func (g *libraryGit) treeEntries(ctx context.Context, treeish string, paths []string) (map[string]treeEntry, error) {
	listing, err := g.runner.Output(ctx, g.dir, append([]string{"ls-tree", "-r", "-z", "--full-tree", treeish, "--"}, paths...), 32*1024*1024)
	if err != nil {
		return nil, err
	}
	entries := map[string]treeEntry{}
	for record := range bytes.SplitSeq(bytes.TrimSuffix(listing, []byte{0}), []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, name, ok := strings.Cut(string(record), "\t")
		fields := strings.Fields(header)
		if !ok || len(fields) != 3 || !objectID.MatchString(fields[2]) {
			return nil, failure("git-failed", "Git listed the library's files in an unexpected format", nil)
		}
		entries[name] = treeEntry{mode: fields[0], kind: fields[1], object: fields[2]}
		if len(entries) > maxFiles {
			return nil, failure("limit-exceeded", treeish+": library exceeds 10,000 files", nil)
		}
	}
	return entries, nil
}

// changedRules reports, for each rule in working, whether its versioned files differ from the ones the latest
// library release published, comparing content as Git would store it. working maps a rule ID to the sorted
// working-tree paths of its Markdown file and asset directory's files.
func (g *libraryGit) changedRules(ctx context.Context, latest *publishedRelease, working map[string][]string) (map[string]bool, error) {
	paths := []string{}
	for _, names := range working {
		paths = append(paths, names...)
	}
	hashes, err := g.hashFiles(ctx, paths)
	if err != nil {
		return nil, err
	}
	changed := map[string]bool{}
	for id, names := range working {
		released := ruleFiles(id, maps.Keys(latest.files))
		changed[id] = !slices.Equal(names, released) || slices.ContainsFunc(names, func(name string) bool { return hashes[name] != latest.files[name] })
	}
	return changed, nil
}

// hashFiles returns the blob ID Git would store for each working-tree file, applying the repository's
// line-ending and filter attributes, so content compares equal to a committed copy.
func (g *libraryGit) hashFiles(ctx context.Context, paths []string) (map[string]string, error) {
	hashes := map[string]string{}
	if len(paths) == 0 {
		return hashes, nil
	}
	input := strings.Join(paths, "\n") + "\n"
	output, err := g.runner.Run(ctx, g.dir, []string{"hash-object", "--stdin-paths"}, len(paths)*65+4096, []byte(input))
	if err != nil {
		return nil, fmt.Errorf("hash the library's files: %w", err)
	}
	lines := strings.Split(strings.TrimSuffix(string(output.Output), "\n"), "\n")
	if output.Status != 0 || len(lines) != len(paths) {
		return nil, failure("git-failed", "Git could not hash the library's files", nil)
	}
	for i, line := range lines {
		if !objectID.MatchString(line) {
			return nil, failure("git-failed", "Git could not hash the library's files", nil)
		}
		hashes[paths[i]] = line
	}
	return hashes, nil
}
