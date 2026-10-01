// Read one published version of a library rule, with the shared assets it links to, for a project to fork.

package imports

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/fabricahq/code-rules/coderules"
	"github.com/fabricahq/code-rules/internal/gitexec"
	"github.com/fabricahq/code-rules/internal/rules"
)

// PublishedRule is one published version of a library rule, as the library release that published it holds it.
type PublishedRule struct {
	// Release is the number of the library release that published the version, and Commit the commit its tag names.
	Release int
	Commit  string
	// Files holds original bytes by library path: the rule's Markdown file, its asset directory's files, and each
	// file in the shared assets/ directory that those files link to, directly or through other shared assets.
	// Declared license and notice files are left out, even when they link to them.
	Files map[string][]byte
	// GroupMetadata is the original _group.yaml of the rule's group at Commit, or nil when Commit has none.
	GroupMetadata []byte
}

// ReadPublishedRule reads version of rule id from the library at source's repository, within options' deadline,
// without writing outside temporary storage. The library release that published the version is found from the
// release records in its release/<number> tags. It fails with code releases-not-found before the library's first
// library release, and version-not-found when the rule never published version.
func ReadPublishedRule(ctx context.Context, source rules.Source, id string, version coderules.RuleVersion, options Options) (PublishedRule, error) {
	return readPublishedRule(ctx, source, id, version, options, nil)
}

// readPublishedRule is ReadPublishedRule, failing with expect's error, when expect isn't nil, before reading any rule
// file of the library release it finds.
func readPublishedRule(ctx context.Context, source rules.Source, id string, version coderules.RuleVersion, options Options, expect func(*libraryRelease) error) (_ PublishedRule, err error) {
	ctx, cancel, err := withTimeout(ctx, options)
	if err != nil {
		return PublishedRule{}, err
	}
	defer cancel()
	repo, err := openRepository(ctx, source, options)
	if err != nil {
		return PublishedRule{}, err
	}
	// Joining only a failed Close keeps err itself, so callers still see its identity, such as a validation error.
	defer func() {
		if closeErr := repo.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	history, err := repo.loadHistory(ctx)
	if err != nil {
		return PublishedRule{}, err
	}
	release, err := publishingRelease(history, id, version)
	if err != nil {
		return PublishedRule{}, err
	}
	if expect != nil {
		if err := expect(release); err != nil {
			return PublishedRule{}, err
		}
	}
	tree, err := repo.tree(ctx, release.commit)
	if err != nil {
		return PublishedRule{}, err
	}
	terms, err := repo.terms(ctx, source.Name, tree)
	if err != nil {
		return PublishedRule{}, err
	}
	owned, err := repo.ruleFiles(ctx, release.commit, id)
	if err != nil {
		return PublishedRule{}, err
	}
	if _, ok := owned[id+".md"]; !ok {
		return PublishedRule{}, fail("unsupported-content", fmt.Sprintf("Library release release/%d has no rule %s, although its release record publishes it. Don't create or move release tags by hand.", release.number, id), nil)
	}
	reader := newGitFiles(ctx, repo, tree)
	files, err := readEntries(ctx, repo, reader, owned)
	if err != nil {
		return PublishedRule{}, err
	}
	if err := readLinkedSharedAssets(ctx, repo, reader, tree, terms, files); err != nil {
		return PublishedRule{}, err
	}
	result := PublishedRule{Release: release.number, Commit: release.commit, Files: files}
	if entry, ok := tree[ruleGroup(id)+"/_group.yaml"]; ok {
		metadata, err := readEntries(ctx, repo, reader, map[string]treeEntry{entry.name: entry})
		if err != nil {
			return PublishedRule{}, err
		}
		result.GroupMetadata = metadata[entry.name]
	}
	if err := ctx.Err(); err != nil {
		return PublishedRule{}, gitexec.ContextFailure(err)
	}
	return result, nil
}

// publishingRelease returns the library release that published version of rule id. Its errors name the versions
// the rule did publish.
func publishingRelease(history releaseHistory, id string, version coderules.RuleVersion) (*libraryRelease, error) {
	if history.newest() == nil {
		return nil, fail("releases-not-found", "The library has no release/<number> tags, because it hasn't published its first library release, so its rules have no versions to fork. Ask the maintainer to publish a library release.", nil)
	}
	if release := history.publisher(id, version); release != nil {
		return release, nil
	}
	versions := history.versionList(id)
	if versions == "" {
		return nil, fail("version-not-found", fmt.Sprintf("The library never published a rule %s; check the rule ID.", id), nil)
	}
	return nil, fail("version-not-found", fmt.Sprintf("Rule %s never published version %s. Its published versions, newest first: %s.", id, version, versions), nil)
}

// readEntries fetches the blobs of entries in one request and returns their bytes by path.
func readEntries(ctx context.Context, repo *repository, reader *gitFiles, entries map[string]treeEntry) (map[string][]byte, error) {
	if err := repo.prefetch(ctx, slices.Collect(maps.Values(entries))); err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for _, name := range slices.Sorted(maps.Keys(entries)) {
		data, err := reader.ReadFile(name)
		if err != nil {
			return nil, err
		}
		files[name] = data
	}
	return files, nil
}

// readLinkedSharedAssets adds to files each file in tree's shared assets/ directory that a Markdown file in files
// links to, repeating for the Markdown files it adds. Links to anything else, including missing shared assets and
// declared terms, are left for the caller to judge.
func readLinkedSharedAssets(ctx context.Context, repo *repository, reader *gitFiles, tree map[string]treeEntry, terms []string, files map[string][]byte) error {
	checked := map[string]bool{}
	for {
		wanted := map[string]treeEntry{}
		for _, file := range slices.Sorted(maps.Keys(files)) {
			if checked[file] || !strings.HasSuffix(file, ".md") {
				continue
			}
			checked[file] = true
			if !utf8.Valid(files[file]) {
				return fail("unsupported-content", file+": expected UTF-8 Markdown.", nil)
			}
			targets, err := rules.MarkdownTargets(string(files[file]), file)
			if err != nil {
				return err
			}
			for _, target := range targets {
				entry, ok := tree[target]
				if _, have := files[target]; ok && !have && rules.AssetDirectory(target) == "assets/" && !slices.Contains(terms, target) {
					wanted[target] = entry
				}
			}
		}
		if len(wanted) == 0 {
			return nil
		}
		added, err := readEntries(ctx, repo, reader, wanted)
		if err != nil {
			return err
		}
		maps.Copy(files, added)
	}
}
