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
	"github.com/fabricahq/code-rules/internal/rules"
)

const (
	// maxReleaseTags bounds the release tags one check reads, matching the tag-listing limit for imports.
	maxReleaseTags = 20_000
	// maxTagBatchBytes bounds the tag messages read by one Git process; larger histories use several.
	maxTagBatchBytes = 16 * 1024 * 1024
)

// objectID accepts a full SHA-1 or SHA-256 object name.
var objectID = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// shallowClone tells the author how to get the history that comparing with a library release needs.
const shallowClone = "this clone has partial history, so it can't be compared with the latest library release. Fetch the full history and tags, such as with git fetch --unshallow --tags; in CI, check out with fetch-depth: 0"

// libraryGit runs Git in the library author's own repository, honoring their Git configuration.
type libraryGit struct {
	runner gitexec.Runner
	dir    string
}

// releaseTag is an annotated release/<number> tag reachable from HEAD.
type releaseTag struct {
	number int
	// object is the tag object's ID; size is its length in bytes.
	object string
	size   int
	// commit is the tagged commit.
	commit string
}

// releaseHistory is what the library releases reachable from HEAD record.
type releaseHistory struct {
	// latest is nil before the first library release.
	latest *publishedRelease
	// retired maps every rule a reachable library release retired to that release's number.
	retired map[string]int
}

// publishedRelease is the latest library release reachable from HEAD.
type publishedRelease struct {
	number int
	// object is the tag object's ID, and commit the commit it tags.
	object, commit string
	// notes are the release notes before the tag message's record, which a GitHub Release page repeats.
	notes  string
	record rules.ReleaseRecord
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

// releaseTags lists the annotated release tags reachable from HEAD in ascending number order.
// It fails in a shallow clone, which may lack tags and history, and returns none on an unborn branch.
// A nil receiver, a library outside Git, has no release tags.
func (g *libraryGit) releaseTags(ctx context.Context) ([]releaseTag, error) {
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
	listing, err := g.runner.Output(ctx, g.dir, []string{"for-each-ref", "--merged=HEAD", "--format=%(refname)%00%(objecttype)%00%(objectname)%00%(objectsize)%00%(*objecttype)%00%(*objectname)", "refs/tags/release/"}, 8*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("list the library's release tags: %w", err)
	}
	tags := []releaseTag{}
	for line := range strings.Lines(string(listing)) {
		fields := strings.Split(strings.TrimSuffix(line, "\n"), "\x00")
		if len(fields) != 6 {
			return nil, failure("git-failed", "Git listed release tags in an unexpected format", nil)
		}
		// Code Rules ignores other tags under release/, such as release/01 or release/v2.
		name := strings.TrimPrefix(fields[0], "refs/tags/")
		number, err := rules.ParseReleaseTag(name)
		if err != nil {
			continue
		}
		if fields[1] != "tag" || fields[4] != "commit" {
			return nil, failure("invalid-release-tag", name+": expected an annotated tag on a commit, with a release record; don't create release tags by hand", nil)
		}
		size, err := strconv.Atoi(fields[3])
		if !objectID.MatchString(fields[2]) || !objectID.MatchString(fields[5]) || err != nil || size < 0 {
			return nil, failure("git-failed", "Git listed release tags in an unexpected format", nil)
		}
		if size > maxFileBytes {
			return nil, failure("limit-exceeded", name+": tag message exceeds 8 MiB", nil)
		}
		tags = append(tags, releaseTag{number: number, object: fields[2], size: size, commit: fields[5]})
		if len(tags) > maxReleaseTags {
			return nil, failure("limit-exceeded", "library has more than 20,000 release tags", nil)
		}
	}
	slices.SortFunc(tags, func(a, b releaseTag) int { return a.number - b.number })
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
	history := releaseHistory{retired: map[string]int{}}
	tags, err := g.releaseTags(ctx)
	if err != nil || len(tags) == 0 {
		return history, err
	}
	notes, records, err := g.releaseRecords(ctx, tags)
	if err != nil {
		return releaseHistory{}, err
	}
	for i, record := range records {
		for id := range record.Retired {
			if _, ok := history.retired[id]; !ok {
				history.retired[id] = tags[i].number
			}
		}
	}
	latest := tags[len(tags)-1]
	files, err := g.releaseFiles(ctx, latest.object)
	if err != nil {
		return releaseHistory{}, err
	}
	history.latest = &publishedRelease{number: latest.number, object: latest.object, commit: latest.commit, notes: notes, record: records[len(records)-1], files: files}
	return history, nil
}

// releaseRecords parses each tag's release record, in tag order, reading messages in bounded batches.
// It also returns the last tag's release notes.
func (g *libraryGit) releaseRecords(ctx context.Context, tags []releaseTag) (string, []rules.ReleaseRecord, error) {
	var notes string
	records := make([]rules.ReleaseRecord, 0, len(tags))
	for start := 0; start < len(tags); {
		end, total := start, 0
		var input strings.Builder
		for end < len(tags) && (end == start || total+tags[end].size <= maxTagBatchBytes) {
			total += tags[end].size + 128
			input.WriteString(tags[end].object + "\n")
			end++
		}
		result, err := g.runner.Run(ctx, g.dir, []string{"cat-file", "--batch"}, total+4096, []byte(input.String()))
		if err != nil {
			return "", nil, fmt.Errorf("read the library's release tags: %w", err)
		}
		if result.Status != 0 {
			return "", nil, failure("git-failed", "Git could not read the library's release tags", nil)
		}
		remaining := result.Output
		for _, tag := range tags[start:end] {
			var body []byte
			body, remaining, err = batchObject(remaining, tag.object, "tag", tag.size)
			if err != nil {
				return "", nil, err
			}
			var record rules.ReleaseRecord
			notes, record, err = parseReleaseTag(body, tag.number)
			if err != nil {
				return "", nil, err
			}
			records = append(records, record)
		}
		start = end
	}
	return notes, records, nil
}

// batchObject splits one object from git cat-file --batch output, verifying its header, and returns the rest.
func batchObject(output []byte, object, kind string, size int) ([]byte, []byte, error) {
	header, rest, ok := bytes.Cut(output, []byte{'\n'})
	if !ok || string(header) != object+" "+kind+" "+strconv.Itoa(size) || len(rest) < size+1 || rest[size] != '\n' {
		return nil, nil, failure("git-failed", "Git returned inconsistent or incomplete release tags", nil)
	}
	return rest[:size], rest[size+1:], nil
}

// signatureHeaders begin a signature that Git appends to a signed tag's message.
var signatureHeaders = []string{"-----BEGIN PGP SIGNATURE-----", "-----BEGIN PGP MESSAGE-----", "-----BEGIN SSH SIGNATURE-----", "-----BEGIN SIGNED MESSAGE-----"}

// parseReleaseTag reads the release notes and record from a raw tag object, ignoring any signature. An invalid
// record, including one whose number differs from the tag's, fails with code invalid-release-tag and keeps the
// parser's validation error, with its location, as the cause.
func parseReleaseTag(object []byte, number int) (string, rules.ReleaseRecord, error) {
	name := "release/" + strconv.Itoa(number)
	_, message, ok := bytes.Cut(object, []byte("\n\n"))
	if !ok {
		message = nil
	}
	// Git treats the last line starting a signature as its start.
	end := len(message)
	for offset := 0; offset < len(message); {
		line := message[offset:]
		if slices.ContainsFunc(signatureHeaders, func(header string) bool { return bytes.HasPrefix(line, []byte(header)) }) {
			end = offset
		}
		next := bytes.IndexByte(line, '\n')
		if next < 0 {
			break
		}
		offset += next + 1
	}
	notes, record, err := rules.ParseReleaseMessage(name, message[:end])
	if err != nil {
		return "", rules.ReleaseRecord{}, failure("invalid-release-tag", "invalid release record in "+name+": "+err.Error()+". Don't create or move release tags by hand", err)
	}
	return notes, record, nil
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
