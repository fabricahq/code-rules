// Coordinate complete source imports and preserve original bytes with verified Git provenance.

package imports

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/fabricahq/code-rules/internal/gitexec"
	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// Library keeps the parsed catalog and its byte-preserving, verified source snapshot together.
type Library struct {
	Catalog  library.Catalog  `json:"catalog"`
	Snapshot library.Snapshot `json:"snapshot"`
	// Warnings explain configuration the import tolerated: entries naming retired rules, and a ref that isn't a
	// library release. It is empty, never nil, when there are none.
	Warnings []string `json:"warnings"`
}

// ImportLibraries imports every configured source or returns no partial result. recorded holds each source's
// snapshot from the last sync, without files, keyed by source name; its versions are kept, and versions are
// chosen only where configuration asks for something a snapshot doesn't have. It never installs files in a
// consuming project. Source temporary state is closed on every path.
func ImportLibraries(ctx context.Context, configuration rules.Configuration, recorded map[string]library.Snapshot, options Options) (map[string]Library, error) {
	if err := ctx.Err(); err != nil {
		return nil, gitexec.ContextFailure(err)
	}
	result := make(map[string]Library, len(configuration.Sources))
	for _, source := range configuration.Sources {
		if _, exists := result[source.Name]; exists {
			return nil, fail("invalid-configuration", "Duplicate source alias.", nil)
		}
		var previous *library.Snapshot
		if snapshot, ok := recorded[source.Name]; ok {
			previous = &snapshot
		}
		imported, err := importLibrary(ctx, source, previous, options)
		if err != nil {
			return nil, fmt.Errorf("import source %q failed (no libraries were returned because all configured sources must succeed): %w", source.Name, err)
		}
		result[source.Name] = imported
	}
	return result, nil
}

// importLibrary applies one deadline across choosing versions, fetching, blob reads, validation, and snapshot
// construction.
func importLibrary(ctx context.Context, source rules.Source, recorded *library.Snapshot, options Options) (_ Library, err error) {
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	if timeout < 0 {
		return Library{}, fail("invalid-options", "Import timeout must be positive or zero for the default.", nil)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	repo, err := openRepository(ctx, source, options)
	if err != nil {
		return Library{}, err
	}
	defer func() { err = errors.Join(err, repo.Close()) }()
	plan, err := planSource(ctx, repo, source, recorded)
	if err != nil {
		return Library{}, err
	}
	input, err := repo.snapshotFiles(ctx, source, plan)
	if err != nil {
		return Library{}, err
	}
	catalog, err := library.LoadSource(ctx, input, source.Name, source.Groups, plan.individual)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Library{}, gitexec.ContextFailure(err)
		}
		return Library{}, err
	}
	if loaded := catalogRules(catalog); !slices.Equal(loaded, slices.Sorted(maps.Keys(plan.rules))) {
		return Library{}, fail("unsupported-content", "The library's files don't match the rules its release records list. Don't create or move release tags by hand.", nil)
	}
	snapshot := library.Snapshot{
		Repository:    source.Repository,
		Pins:          maps.Clone(source.Pins),
		Exclude:       slices.Sorted(maps.Keys(source.Exclude)),
		Ref:           source.Ref,
		Release:       plan.release,
		Commit:        plan.commit,
		Selection:     rules.GroupSelection{Pattern: source.Groups.Pattern, Groups: slices.Clone(source.Groups.Groups)},
		Groups:        []string{},
		RuleSelection: slices.Clone(source.Rules),
		Rules:         plan.rules,
	}
	for _, group := range catalog.Groups {
		if source.Groups.Includes(group.ID) {
			snapshot.Groups = append(snapshot.Groups, group.ID)
		}
	}
	snapshot.Files = map[string][]byte{}
	for file, data := range catalog.Files() {
		snapshot.Files[file] = bytes.Clone(data)
	}
	if err := ctx.Err(); err != nil {
		return Library{}, gitexec.ContextFailure(err)
	}
	return Library{Catalog: catalog, Snapshot: snapshot, Warnings: plan.warnings}, nil
}

// snapshotFiles assembles the files the plan imports: library-wide files from the plan's commit, and each rule's
// Markdown file and asset directory from its own commit. Other rules' files are left out. It first fetches, in
// one request, the blobs the catalog loader will read, except shared assets, which rules name only in their text:
// the manifest, terms, the metadata of the groups the source imports, and the imported rules' files.
func (r *repository) snapshotFiles(ctx context.Context, source rules.Source, plan sourcePlan) (*gitFiles, error) {
	main, err := r.tree(ctx, plan.commit)
	if err != nil {
		return nil, err
	}
	terms, err := r.terms(ctx, source.Name, main)
	if err != nil {
		return nil, err
	}
	imported := map[string]bool{}
	for id := range plan.rules {
		imported[ruleGroup(id)+"/_group.yaml"] = true
	}
	files := map[string]treeEntry{}
	prefetch := []treeEntry{}
	for file, entry := range main {
		if _, versioned := rules.VersionedRule(file); versioned {
			continue
		}
		files[file] = entry
		group, metadata := strings.CutSuffix(file, "/_group.yaml")
		if file == "rule-library.yaml" || slices.Contains(terms, file) || metadata && (imported[file] || source.Groups.Includes(group)) {
			prefetch = append(prefetch, entry)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(plan.rules)) {
		owned, err := r.ruleFiles(ctx, plan.rules[id].Commit, id)
		if err != nil {
			return nil, err
		}
		if _, ok := owned[id+".md"]; !ok {
			return nil, fail("unsupported-content", fmt.Sprintf("The library's commit %s has no rule %s, although its release record lists the rule.", plan.rules[id].Commit, id), nil)
		}
		for file, entry := range owned {
			files[file] = entry
			prefetch = append(prefetch, entry)
		}
	}
	restored, err := r.missingGroupMetadata(ctx, files, plan)
	if err != nil {
		return nil, err
	}
	for file, entry := range restored {
		files[file] = entry
		prefetch = append(prefetch, entry)
	}
	if err := r.prefetch(ctx, prefetch); err != nil {
		return nil, err
	}
	return newGitFiles(ctx, r, files), nil
}

// missingGroupMetadata returns, by path, the metadata of each imported rule's group that files lacks, such as a
// retired rule's group that the library-wide release removed. Each comes from the newest library release among
// the group's imported rules that still has it; a group none of them has stays missing, for the loader to report.
func (r *repository) missingGroupMetadata(ctx context.Context, files map[string]treeEntry, plan sourcePlan) (map[string]treeEntry, error) {
	byGroup := map[string][]library.ImportedRule{}
	for id, rule := range plan.rules {
		byGroup[ruleGroup(id)] = append(byGroup[ruleGroup(id)], rule)
	}
	restored := map[string]treeEntry{}
	for _, group := range slices.Sorted(maps.Keys(byGroup)) {
		metadata := group + "/_group.yaml"
		if _, ok := files[metadata]; ok {
			continue
		}
		candidates := byGroup[group]
		slices.SortFunc(candidates, func(a, b library.ImportedRule) int { return b.Release - a.Release })
		for _, rule := range candidates {
			tree, err := r.tree(ctx, rule.Commit)
			if err != nil {
				return nil, err
			}
			if entry, ok := tree[metadata]; ok {
				restored[metadata] = entry
				break
			}
		}
	}
	return restored, nil
}

// catalogRules returns the library rule IDs of every rule in catalog, sorted.
func catalogRules(catalog library.Catalog) []string {
	ids := []string{}
	for _, group := range catalog.Groups {
		for _, rule := range group.Rules {
			ids = append(ids, strings.TrimSuffix(rule.Path, ".md"))
		}
	}
	slices.Sort(ids)
	return ids
}
