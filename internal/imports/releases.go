// List, fetch, and read a library's release tags, and answer questions about its rules' versions.

package imports

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/fabricahq/code-rules/internal/gitexec"
	"github.com/fabricahq/code-rules/internal/rules"
)

// maxTagBatchBytes bounds the tag messages one Git process reads; more releases use several.
const maxTagBatchBytes = 16 << 20

// libraryRelease is one library release, read from its release/<number> tag.
type libraryRelease struct {
	number int
	// tag is the annotated tag object's ID; commit is the commit it tags.
	tag    string
	commit string
	record rules.ReleaseRecord
}

// releaseHistory holds every library release in ascending number order; it is empty before the first one.
type releaseHistory struct {
	releases []libraryRelease
}

// newest returns the library release with the highest number, or nil when there is none.
func (h releaseHistory) newest() *libraryRelease {
	if len(h.releases) == 0 {
		return nil
	}
	return &h.releases[len(h.releases)-1]
}

// release returns the library release numbered number, or nil when there is none.
func (h releaseHistory) release(number int) *libraryRelease {
	for i := range h.releases {
		if h.releases[i].number == number {
			return &h.releases[i]
		}
	}
	return nil
}

// publisher returns the library release that published version of rule id: the one whose changes entry for the
// rule leads to that version. It returns nil when the rule never published the version.
func (h releaseHistory) publisher(id string, version rules.RuleVersion) *libraryRelease {
	for i := range h.releases {
		record := h.releases[i].record
		if _, changed := record.Changes[id]; changed && record.Rules[id] == version {
			return &h.releases[i]
		}
	}
	return nil
}

// published returns, newest first, each library release that published a version of rule id.
func (h releaseHistory) published(id string) []*libraryRelease {
	result := []*libraryRelease{}
	for i := len(h.releases) - 1; i >= 0; i-- {
		if _, changed := h.releases[i].record.Changes[id]; changed {
			result = append(result, &h.releases[i])
		}
	}
	return result
}

// retired reports whether any library release retired rule id.
func (h releaseHistory) retired(id string) bool {
	return h.retirement(id) != nil
}

// retirement returns how a library release retired rule id, or nil when none did.
func (h releaseHistory) retirement(id string) *rules.RetiredRule {
	for _, release := range h.releases {
		if retired, ok := release.record.Retired[id]; ok {
			return &retired
		}
	}
	return nil
}

// summaries returns, oldest first, the summaries of every version of rule id newer than from, up to and including
// to, one per change note. A nil from includes every version up to to. The result is empty, never nil, when there
// are none.
func (h releaseHistory) summaries(id string, from *rules.RuleVersion, to rules.RuleVersion) []string {
	result := []string{}
	for _, release := range h.releases {
		change, changed := release.record.Changes[id]
		version := release.record.Rules[id]
		if changed && (from == nil || version.Compare(*from) > 0) && version.Compare(to) <= 0 {
			result = append(result, change.Summaries...)
		}
	}
	return result
}

// loadHistory lists the library's release/<number> tags, fetches them without history or blobs, and reads their
// release records. Tags under release/ with other names are ignored. A lightweight or unreadable release tag,
// or one that moved while it was fetched, fails with code invalid-release-tag.
func (r *repository) loadHistory(ctx context.Context) (releaseHistory, error) {
	advertised, err := r.listReleases(ctx)
	if err != nil || len(advertised) == 0 {
		return releaseHistory{}, err
	}
	refspecs := make([]string, 0, len(advertised))
	for _, release := range advertised {
		name := "refs/tags/release/" + strconv.Itoa(release.number)
		refspecs = append(refspecs, "+"+name+":"+name)
	}
	ok, err := r.fetch(ctx, refspecs, true)
	if err != nil {
		return releaseHistory{}, err
	}
	if !ok {
		if err := r.unreachable(ctx); err != nil {
			return releaseHistory{}, err
		}
		return releaseHistory{}, fail("git-failed", "Could not fetch the library's release tags.", nil)
	}
	sizes, err := r.fetchedReleases(ctx, advertised)
	if err != nil {
		return releaseHistory{}, err
	}
	if err := r.readRecords(ctx, advertised, sizes); err != nil {
		return releaseHistory{}, err
	}
	for _, release := range advertised {
		r.present[release.commit] = true
	}
	return releaseHistory{releases: advertised}, nil
}

// listReleases reads the remote's release/<number> tags, within the tag-listing limits, in ascending order.
func (r *repository) listReleases(ctx context.Context) ([]libraryRelease, error) {
	result, err := r.runner.Run(ctx, r.directory, []string{"ls-remote", "--tags", fetchRemote, "refs/tags/release/*"}, 8<<20+64<<10, nil)
	if err != nil {
		return nil, err
	}
	if result.Status != 0 {
		return nil, fail("not-found-or-no-access", "Repository not found or no access; check its address and Git credentials.", nil)
	}
	tags, err := rules.ParseTagAdvertisement(string(result.Output))
	if err != nil {
		var advertisement *rules.TagAdvertisementError
		if errors.As(err, &advertisement) {
			return nil, fail(string(advertisement.Kind), advertisement.Problem, nil)
		}
		return nil, err
	}
	releases := []libraryRelease{}
	for _, name := range slices.Sorted(maps.Keys(tags)) {
		number, err := rules.ParseReleaseTag(name)
		if err != nil {
			continue
		}
		commit, annotated := tags[name+"^{}"]
		if !annotated {
			return nil, fail("invalid-release-tag", fmt.Sprintf("Library release tag %s isn't an annotated tag with a release record. Don't create or move release tags by hand.", name), nil)
		}
		releases = append(releases, libraryRelease{number: number, tag: tags[name], commit: commit})
	}
	slices.SortFunc(releases, func(a, b libraryRelease) int { return a.number - b.number })
	return releases, nil
}

// fetchedReleases checks that each fetched release tag is the tag object the listing advertised and that it tags
// the advertised object, which must be a commit, so a tag moved in between or tagging a tree or blob fails. It
// returns each tag object's size in bytes.
func (r *repository) fetchedReleases(ctx context.Context, advertised []libraryRelease) (map[string]int, error) {
	listing, err := r.runner.Output(ctx, r.directory, []string{"for-each-ref", "--format=%(refname) %(objecttype) %(objectname) %(objectsize) %(*objectname) %(*objecttype)", "refs/tags/release/"}, 8<<20)
	if err != nil {
		return nil, err
	}
	fetched := map[string][]string{}
	for line := range strings.Lines(string(listing)) {
		fields := strings.Fields(line)
		if len(fields) == 6 {
			fetched[strings.TrimPrefix(fields[0], "refs/tags/")] = fields[1:]
		}
	}
	sizes := map[string]int{}
	for _, release := range advertised {
		name := "release/" + strconv.Itoa(release.number)
		fields := fetched[name]
		if len(fields) != 5 || fields[0] != "tag" || fields[1] != release.tag || fields[3] != release.commit {
			return nil, fail("invalid-release-tag", fmt.Sprintf("Library release tag %s changed while Code Rules read it; retry. Don't create or move release tags by hand.", name), nil)
		}
		if fields[4] != "commit" {
			return nil, fail("invalid-release-tag", fmt.Sprintf("Library release tag %s tags a %s, not a commit. Don't create or move release tags by hand.", name, fields[4]), nil)
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size < 0 || size > maxBlobBytes {
			return nil, fail("limit-exceeded", fmt.Sprintf("Library release tag %s exceeds 8 MiB.", name), nil)
		}
		sizes[release.tag] = size
	}
	return sizes, nil
}

// readRecords reads and parses each release tag's record into releases, reading messages in bounded batches. It
// stops between records when ctx ends, since parsing thousands of large records takes long.
func (r *repository) readRecords(ctx context.Context, releases []libraryRelease, sizes map[string]int) error {
	for start := 0; start < len(releases); {
		end, total := start, 0
		var input strings.Builder
		for end < len(releases) && (end == start || total+sizes[releases[end].tag] <= maxTagBatchBytes) {
			total += sizes[releases[end].tag] + 128
			input.WriteString(releases[end].tag + "\n")
			end++
		}
		result, err := r.runner.Run(ctx, r.directory, []string{"cat-file", "--batch"}, total+4096, []byte(input.String()))
		if err != nil {
			return err
		}
		if result.Status != 0 {
			return fail("git-failed", "Git could not read the library's release tags.", nil)
		}
		remaining := result.Output
		for i := start; i < end; i++ {
			if err := ctx.Err(); err != nil {
				return gitexec.ContextFailure(err)
			}
			release := &releases[i]
			name := "release/" + strconv.Itoa(release.number)
			header, rest, ok := bytes.Cut(remaining, []byte{'\n'})
			size := sizes[release.tag]
			if !ok || string(header) != release.tag+" tag "+strconv.Itoa(size) || len(rest) < size+1 || rest[size] != '\n' {
				return fail("git-failed", "Git returned inconsistent or incomplete release tags.", nil)
			}
			_, record, err := rules.ParseReleaseTagObject(name, rest[:size])
			var unsupported *rules.UnsupportedReleaseRecordError
			if errors.As(err, &unsupported) {
				return fail("unsupported-release-record", fmt.Sprintf("Library release tag %s uses release record format %d, which this version of Code Rules can't read. Upgrade Code Rules, then run the command again.", name, unsupported.FormatVersion), nil)
			}
			if err != nil {
				return fail("invalid-release-tag", fmt.Sprintf("Invalid release record in library release tag %s: %v. Don't create or move release tags by hand.", name, err), err)
			}
			release.record = record
			remaining = rest[size+1:]
		}
		start = end
	}
	return nil
}
