// Choose each imported rule's version from a source's configuration, its recorded snapshot, and its library releases.

package imports

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// sourcePlan is what one source imports: the revision that supplies its library-wide files, and each imported
// rule's version and the commit that supplies its files. Resolving it may read the library's release history;
// importing it fetches the commits it names and reads their files.
type sourcePlan struct {
	// release is the library release that supplies the library-wide files, or 0 when ref isn't a library release.
	// For a source without ref, it is never older than the library release of a rule the plan imports.
	release int
	commit  string
	rules   map[string]library.ImportedRule
	// individual lists the individually selected rules to load; it omits entries naming retired rules.
	individual []string
	warnings   []string
	// retired lists, sorted, the rules the library retired that the source selects; it is empty, never nil.
	retired []string
}

// planner chooses versions for one source, reading the library's release history only when a choice needs it.
type planner struct {
	ctx    context.Context
	repo   *repository
	source rules.Source
	// recorded is the source's snapshot from the last sync, or nil for a new source or a changed repository.
	recorded *library.Snapshot
	// history is nil until first needed.
	history *releaseHistory
}

// planSource chooses what source imports, as project sync does: recorded versions stay, and versions are chosen
// only for new sources, changed repositories, newly selected rules, added or changed pins, and a changed ref.
// It fails when a library release tag that a recorded rule names has moved.
func planSource(ctx context.Context, repo *repository, source rules.Source, recorded *library.Snapshot) (sourcePlan, error) {
	p := newPlanner(ctx, repo, source, recorded)
	plan, err := p.choose()
	if err != nil {
		return sourcePlan{}, err
	}
	return plan, p.requireUnmoved(plan)
}

// newPlanner plans source against recorded, which it ignores when it records another repository.
func newPlanner(ctx context.Context, repo *repository, source rules.Source, recorded *library.Snapshot) *planner {
	if recorded != nil && recorded.Repository != source.Repository {
		recorded = nil
	}
	return &planner{ctx: ctx, repo: repo, source: source, recorded: recorded}
}

// choose plans what the source imports as project sync does, without fetching the commits the plan names that
// choosing didn't need.
func (p *planner) choose() (sourcePlan, error) {
	choose := p.planVersions
	if !p.source.Ref.IsZero() {
		choose = p.planRevision
	}
	chosen, err := choose()
	if err != nil {
		return sourcePlan{}, err
	}
	chosen.warnings = append(redundantRules(p.source, chosen.rules), chosen.warnings...)
	chosen.retired, err = p.retiredRules(chosen)
	return chosen, err
}

// redundantRules warns about each imported rule the source selects individually although its groups already select
// the rule's group, since the entry changes nothing. An entry naming a retired rule gets its own warning instead.
func redundantRules(source rules.Source, imported map[string]library.ImportedRule) []string {
	warnings := []string{}
	for _, id := range source.Rules {
		if _, ok := imported[id]; ok && source.Groups.Includes(ruleGroup(id)) {
			group := ruleGroup(id)
			warnings = append(warnings, fmt.Sprintf("sources.%s.rules names %s, whose group %s sources.%s.groups already selects, so the entry changes nothing; delete it.", source.Name, id, group, source.Name))
		}
	}
	return warnings
}

// retiredRules returns the retired rules a sync that chose plan records. When the source selects what recorded did,
// from the same revision, with the same shared files, it keeps recorded's list, so that a sync changing nothing
// else, such as an exclusion the list already covers, never rewrites the record, whatever the planner happened to
// read; it adds only the retired rules of exclusions that sync just found retired, which the record didn't cover,
// so offline checks accept them too. Otherwise it returns the release history's, as freshRetiredRules does.
func (p *planner) retiredRules(plan sourcePlan) ([]string, error) {
	recorded := p.recorded
	if recorded == nil || !sameGroupSelection(recorded.Selection, p.source.Groups) || !slices.Equal(recorded.RuleSelection, p.source.Rules) || !p.source.Ref.Equal(recorded.Ref) || plan.release != recorded.Release || plan.commit != recorded.Commit {
		return p.freshRetiredRules()
	}
	retired := slices.Clone(recorded.RetiredRules)
	for _, id := range slices.Sorted(maps.Keys(p.source.Exclude)) {
		// Validating such an exclusion read the history, which records why it only warns.
		if _, imported := plan.rules[id]; !imported && !slices.Contains(retired, id) && p.history != nil && p.history.retired(id) {
			retired = append(retired, id)
		}
	}
	slices.Sort(retired)
	return retired, nil
}

// freshRetiredRules returns, sorted, the rules the library's release history retired that the source would
// otherwise import, as configuration entries naming them only warn: those its groups or rules list selects, and
// those recorded imported or listed as retired, so an entry left after deselecting a retired rule stays a warning
// offline too. It reads the history when the planner hasn't yet.
func (p *planner) freshRetiredRules() ([]string, error) {
	history, err := p.releases()
	if err != nil {
		return nil, err
	}
	retired := []string{}
	for _, release := range history.releases {
		for id := range release.record.Retired {
			if p.wouldImport(id) {
				retired = append(retired, id)
			}
		}
	}
	slices.Sort(retired)
	return slices.Compact(retired), nil
}

// releases returns the library's release history, reading it on first use; it may be empty.
func (p *planner) releases() (releaseHistory, error) {
	if p.history == nil {
		history, err := p.repo.loadHistory(p.ctx)
		if err != nil {
			return releaseHistory{}, err
		}
		p.history = &history
	}
	return *p.history, nil
}

// versioned returns the release history, failing with code releases-not-found before the first library release.
func (p *planner) versioned() (releaseHistory, error) {
	history, err := p.releases()
	if err != nil {
		return releaseHistory{}, err
	}
	if history.newest() == nil {
		return releaseHistory{}, fail("releases-not-found", fmt.Sprintf("The library that source %s imports has no release/<number> tags, because it hasn't published its first library release. Ask the maintainer to publish a library release, or import a commit with sources.%s.ref.", p.source.Name, p.source.Name), nil)
	}
	return history, nil
}

// planVersions plans a source that follows rule versions: recorded rules keep their versions unless a pin moves
// them or they have none, and newly selected rules get their pinned or newest version.
func (p *planner) planVersions() (sourcePlan, error) {
	plan := sourcePlan{rules: map[string]library.ImportedRule{}, individual: []string{}, warnings: []string{}}
	recorded := p.recorded
	if recorded != nil {
		for _, id := range slices.Sorted(maps.Keys(recorded.Rules)) {
			if p.selected(id) {
				plan.rules[id] = recorded.Rules[id]
			}
		}
	}
	for _, id := range slices.Sorted(maps.Keys(plan.rules)) {
		rule := plan.rules[id]
		if pin, pinned := p.source.Pins[id]; pinned && (rule.Version == nil || *rule.Version != pin.Version) {
			chosen, err := p.pinnedVersion(id, pin)
			if err != nil {
				return sourcePlan{}, err
			}
			plan.rules[id] = chosen
		} else if rule.Version == nil {
			// Only a removed ref leaves unreleased rules; each gets its newest version, if it has one.
			history, err := p.versioned()
			if err != nil {
				return sourcePlan{}, err
			}
			if _, published := history.newest().record.Rules[id]; !published {
				delete(plan.rules, id)
				continue
			}
			if plan.rules[id], err = p.newestVersion(id); err != nil {
				return sourcePlan{}, err
			}
		}
	}
	chosen := []string{}
	if recorded == nil || !sameGroupSelection(recorded.Selection, p.source.Groups) {
		history, err := p.versioned()
		if err != nil {
			return sourcePlan{}, err
		}
		// A group is newly selected when the snapshot didn't import it in full, whatever its selector matched:
		// a wildcard's snapshot lacks the groups later library releases added.
		for _, id := range slices.Sorted(maps.Keys(history.newest().record.Rules)) {
			group := ruleGroup(id)
			if _, imported := plan.rules[id]; !imported && p.source.Groups.Includes(group) && (recorded == nil || !slices.Contains(recorded.Groups, group)) {
				chosen = append(chosen, id)
			}
		}
	}
	for _, id := range p.source.Rules {
		if _, imported := plan.rules[id]; imported || slices.Contains(chosen, id) {
			plan.individual = append(plan.individual, id)
			continue
		}
		if recorded != nil && slices.Contains(recorded.RuleSelection, id) && !hasRule(recorded, id) {
			plan.warnings = append(plan.warnings, p.retiredEntry("rules", id))
			continue
		}
		history, err := p.versioned()
		if err != nil {
			return sourcePlan{}, err
		}
		if _, current := history.newest().record.Rules[id]; current {
			chosen = append(chosen, id)
			plan.individual = append(plan.individual, id)
			continue
		}
		if err := p.unimported("rules", id, &plan); err != nil {
			return sourcePlan{}, err
		}
	}
	for _, id := range chosen {
		var err error
		if pin, pinned := p.source.Pins[id]; pinned {
			plan.rules[id], err = p.pinnedVersion(id, pin)
		} else {
			plan.rules[id], err = p.newestVersion(id)
		}
		if err != nil {
			return sourcePlan{}, err
		}
	}
	if err := p.requireEntries(&plan); err != nil {
		return sourcePlan{}, err
	}
	if recorded == nil || len(chosen) > 0 || selectsMore(recorded, p.source) {
		return plan, p.newestSharedFiles(&plan)
	}
	plan.release, plan.commit = recorded.Release, recorded.Commit
	if plan.release == 0 {
		// Only a removed ref that named a revision other than a library release records none.
		return plan, p.newestSharedFiles(&plan)
	}
	raiseSharedFiles(&plan)
	return plan, nil
}

// newestSharedFiles takes the plan's library-wide files from the newest library release, which is at least as new
// as every rule version.
func (p *planner) newestSharedFiles(plan *sourcePlan) error {
	history, err := p.versioned()
	if err != nil {
		return err
	}
	plan.release, plan.commit = history.newest().number, history.newest().commit
	return nil
}

// raiseSharedFiles moves the plan's library-wide files to the newest library release among its rule versions when
// that is newer, such as after a pin moved a rule up, because a rule version can rely on the shared files and group
// metadata its own library release published.
func raiseSharedFiles(plan *sourcePlan) {
	for _, id := range slices.Sorted(maps.Keys(plan.rules)) {
		if rule := plan.rules[id]; rule.Release > plan.release {
			plan.release, plan.commit = rule.Release, rule.Commit
		}
	}
}

// selectsMore reports whether source selects a rule or group that recorded's selection didn't: an individually
// selected rule recorded's lacks, a listed group recorded didn't import in full, or a wildcard wider than recorded's.
func selectsMore(recorded *library.Snapshot, source rules.Source) bool {
	for _, id := range source.Rules {
		if !slices.Contains(recorded.RuleSelection, id) {
			return true
		}
	}
	if sameGroupSelection(recorded.Selection, source.Groups) {
		return false
	}
	if source.Groups.Pattern != "" {
		return recorded.Selection.Pattern != "*"
	}
	for _, group := range source.Groups.Groups {
		if !slices.Contains(recorded.Groups, group) {
			return true
		}
	}
	return false
}

// planRevision plans a source that imports one revision with ref. A recorded snapshot of the same ref, however it
// is written, keeps its commit and rule versions; otherwise the ref is resolved again. Rules newly selected at the
// revision record the version its library release record lists, or, for any other revision, the newest published
// version whose files match, or none.
func (p *planner) planRevision() (sourcePlan, error) {
	plan := sourcePlan{rules: map[string]library.ImportedRule{}, individual: []string{}, warnings: []string{}}
	recorded := p.recorded
	if recorded != nil && p.source.Ref.Equal(recorded.Ref) {
		plan.release, plan.commit = recorded.Release, recorded.Commit
		if err := p.repo.fetchCommits(p.ctx, []string{plan.commit}, fmt.Sprintf("The commit that vendor/%s/_source.json records for sources.%s.ref is missing from the library's repository. Delete vendor/%s and run code-rules project sync to resolve the ref again.", p.source.Name, p.source.Name, p.source.Name)); err != nil {
			return sourcePlan{}, err
		}
	} else {
		recorded = nil
		var err error
		if plan.release, plan.commit, err = p.resolveRef(); err != nil {
			return sourcePlan{}, err
		}
	}
	tree, err := p.repo.tree(p.ctx, plan.commit)
	if err != nil {
		return sourcePlan{}, err
	}
	present := rulesInTree(tree)
	imported := []string{}
	for _, id := range present {
		if p.source.Groups.Includes(ruleGroup(id)) {
			imported = append(imported, id)
		}
	}
	for _, id := range p.source.Rules {
		switch {
		case slices.Contains(present, id):
			plan.individual = append(plan.individual, id)
			if !slices.Contains(imported, id) {
				imported = append(imported, id)
			}
		case recorded != nil && slices.Contains(recorded.RuleSelection, id) && !hasRule(recorded, id):
			plan.warnings = append(plan.warnings, p.retiredEntry("rules", id))
		default:
			if err := p.unimported("rules", id, &plan); err != nil {
				return sourcePlan{}, err
			}
		}
	}
	slices.Sort(imported)
	for _, id := range imported {
		if recorded != nil && hasRule(recorded, id) {
			plan.rules[id] = recorded.Rules[id]
			continue
		}
		if plan.rules[id], err = p.revisionVersion(id, plan); err != nil {
			return sourcePlan{}, err
		}
	}
	if err := p.requireEntries(&plan); err != nil {
		return sourcePlan{}, err
	}
	if plan.release == 0 {
		if warning := unreleasedWarning(p.source, plan.rules); warning != "" {
			plan.warnings = append(plan.warnings, warning)
		}
	}
	return plan, nil
}

// resolveRef returns the commit the source's ref names and its library release number, or 0 when it isn't a
// library release tag. A missing tag or commit fails with code version-not-found.
func (p *planner) resolveRef() (int, string, error) {
	if ref := p.source.Ref; ref.Kind() == rules.GitRefTag {
		if number, err := rules.ParseReleaseTag(strings.TrimPrefix(ref.Canonical(), "refs/tags/")); err == nil {
			history, err := p.releases()
			if err != nil {
				return 0, "", err
			}
			release := history.release(number)
			if release == nil {
				return 0, "", fail("version-not-found", fmt.Sprintf("sources.%s.ref: the library has no %s; check the ref.", p.source.Name, p.source.Ref), nil)
			}
			return number, release.commit, nil
		}
	}
	commit, err := p.repo.fetchRef(p.ctx, p.source)
	return 0, commit, err
}

// revisionVersion chooses the version of rule id imported at the plan's revision.
func (p *planner) revisionVersion(id string, plan sourcePlan) (library.ImportedRule, error) {
	history, err := p.releases()
	if err != nil {
		return library.ImportedRule{}, err
	}
	if plan.release != 0 {
		release := history.release(plan.release)
		if release == nil {
			return library.ImportedRule{}, fail("version-not-found", fmt.Sprintf("sources.%s.ref names release/%d, which the library no longer has, so newly selected rule %s has no version to import. Ask the library's maintainer to restore the tag, or change sources.%s.ref.", p.source.Name, plan.release, id, p.source.Name), nil)
		}
		version, listed := release.record.Rules[id]
		if !listed {
			return library.ImportedRule{}, fail("invalid-release-tag", fmt.Sprintf("Library release release/%d contains rule %s, but its release record doesn't list it. Don't create or move release tags by hand.", plan.release, id), nil)
		}
		return p.publishedVersion(id, version)
	}
	files, err := p.repo.ruleFiles(p.ctx, plan.commit, id)
	if err != nil {
		return library.ImportedRule{}, err
	}
	for _, release := range history.published(id) {
		published, err := p.repo.ruleFiles(p.ctx, release.commit, id)
		if err != nil {
			return library.ImportedRule{}, err
		}
		if maps.Equal(files, published) {
			version := release.record.Rules[id]
			return library.ImportedRule{Version: &version, Release: release.number, Commit: release.commit}, nil
		}
	}
	return library.ImportedRule{Commit: plan.commit}, nil
}

// pinnedVersion returns the pinned version of rule id, failing with code version-not-found when the rule never
// published it.
func (p *planner) pinnedVersion(id string, pin rules.Pin) (library.ImportedRule, error) {
	history, err := p.versioned()
	if err != nil {
		return library.ImportedRule{}, err
	}
	if history.publisher(id, pin.Version) == nil {
		return library.ImportedRule{}, fail("version-not-found", fmt.Sprintf("sources.%s.pins.%s: the rule never published version %s; check the pin. Its published versions, newest first: %s.", p.source.Name, id, pin.Version, history.versionList(id)), nil)
	}
	return p.publishedVersion(id, pin.Version)
}

// newestVersion returns the version of rule id in the newest library release.
func (p *planner) newestVersion(id string) (library.ImportedRule, error) {
	history, err := p.versioned()
	if err != nil {
		return library.ImportedRule{}, err
	}
	return p.publishedVersion(id, history.newest().record.Rules[id])
}

// publishedVersion returns version of rule id with the library release that published it.
func (p *planner) publishedVersion(id string, version rules.RuleVersion) (library.ImportedRule, error) {
	history, err := p.releases()
	if err != nil {
		return library.ImportedRule{}, err
	}
	release := history.publisher(id, version)
	if release == nil {
		return library.ImportedRule{}, fail("invalid-release-tag", fmt.Sprintf("No library release's record publishes version %s of rule %s, although a later record lists it. Don't create or move release tags by hand.", version, id), nil)
	}
	return library.ImportedRule{Version: &version, Release: release.number, Commit: release.commit}, nil
}

// requireEntries checks that each pin and exclusion names an imported rule. An entry naming a retired rule adds a
// warning; any other fails.
func (p *planner) requireEntries(plan *sourcePlan) error {
	for _, id := range slices.Sorted(maps.Keys(p.source.Pins)) {
		if _, imported := plan.rules[id]; imported {
			continue
		}
		// A pin that the last sync recorded without importing its rule already named a retired rule.
		if recorded := p.recorded; recorded != nil && !hasRule(recorded, id) && hasPin(recorded, id, p.source.Pins[id].Version) {
			plan.warnings = append(plan.warnings, p.retiredEntry("pins", id))
			continue
		}
		if err := p.unimported("pins", id, plan); err != nil {
			return err
		}
	}
	for _, id := range slices.Sorted(maps.Keys(p.source.Exclude)) {
		if _, imported := plan.rules[id]; !imported {
			if err := p.unimported("exclude", id, plan); err != nil {
				return err
			}
		}
	}
	return nil
}

// unimported handles a configuration entry naming a rule the source doesn't import: a warning for a retired rule
// the source would otherwise import, through its groups, its rules list, or its recorded snapshot, and a
// validation error for any other.
func (p *planner) unimported(field, id string, plan *sourcePlan) error {
	history, err := p.releases()
	if err != nil {
		return err
	}
	if p.wouldImport(id) && history.retired(id) {
		plan.warnings = append(plan.warnings, p.retiredEntry(field, id))
		return nil
	}
	location := "sources." + p.source.Name + "." + field + "." + id
	if field == "rules" {
		return &rules.ValidationError{Location: "sources." + p.source.Name + ".rules", Problem: "the library has no rule " + id + " to import; check the rule ID"}
	}
	return &rules.ValidationError{Location: location, Problem: "rule is not imported by this source; name a rule that its groups or rules select"}
}

// wouldImport reports whether the source would import rule id if the library still published it: its groups or rules
// list selects it, or recorded imported it or listed it as retired.
func (p *planner) wouldImport(id string) bool {
	return p.selected(id) || p.recorded != nil && (hasRule(p.recorded, id) || slices.Contains(p.recorded.RetiredRules, id))
}

// retiredEntry warns that a configuration entry names a retired rule.
func (p *planner) retiredEntry(field, id string) string {
	return fmt.Sprintf("sources.%s.%s names %s, a rule the library retired, so the entry no longer does anything; delete it.", p.source.Name, field, id)
}

// requireUnmoved fails when the library release tag that supplies the plan's library-wide files, or a rule's, now
// names a different commit than the plan records.
func (p *planner) requireUnmoved(plan sourcePlan) error {
	if p.history == nil {
		return nil
	}
	if release := p.history.release(plan.release); plan.release != 0 && release != nil && release.commit != plan.commit {
		return fail("invalid-release-tag", fmt.Sprintf("Library release tag release/%d now names a different commit than vendor/%s/_source.json records for the library's shared files. Library release tags must not move; ask the library's maintainer, or delete vendor/%s and run code-rules project sync to use the tag's current commit.", plan.release, p.source.Name, p.source.Name), nil)
	}
	for _, id := range slices.Sorted(maps.Keys(plan.rules)) {
		rule := plan.rules[id]
		if release := p.history.release(rule.Release); release != nil && release.commit != rule.Commit {
			return fail("invalid-release-tag", fmt.Sprintf("Library release tag release/%d now names a different commit than vendor/%s/_source.json records for %s. Library release tags must not move; ask the library's maintainer, or delete vendor/%s and run code-rules project sync to use the tag's current commit.", rule.Release, p.source.Name, id, p.source.Name), nil)
		}
	}
	return nil
}

// selected reports whether the source's configuration selects rule id, through its groups or its rules list.
func (p *planner) selected(id string) bool {
	return p.source.Groups.Includes(ruleGroup(id)) || slices.Contains(p.source.Rules, id)
}

// unreleasedWarning names a source that imports a revision other than a library release, and its unreleased rules.
// It is empty when every imported rule matches a published version, as at a library release's commit.
func unreleasedWarning(source rules.Source, imported map[string]library.ImportedRule) string {
	unreleased := []string{}
	for _, id := range slices.Sorted(maps.Keys(imported)) {
		if imported[id].Version == nil {
			unreleased = append(unreleased, id)
		}
	}
	if len(unreleased) == 0 {
		return ""
	}
	return fmt.Sprintf("Source %s imports %s, which isn't a library release, so rules with unreleased changes have no version. Unreleased rules: %s.", source.Name, source.Ref, strings.Join(unreleased, ", "))
}

// ruleGroup returns the group of a valid library rule ID.
func ruleGroup(id string) string {
	parts := strings.SplitN(id, "/", 3)
	return parts[0] + "/" + parts[1]
}

// hasRule reports whether the snapshot imported rule id.
func hasRule(snapshot *library.Snapshot, id string) bool {
	_, ok := snapshot.Rules[id]
	return ok
}

// hasPin reports whether the snapshot recorded a pin of rule id to version.
func hasPin(snapshot *library.Snapshot, id string, version rules.RuleVersion) bool {
	pinned, ok := snapshot.Pins[id]
	return ok && pinned == version
}

// pinnedVersions returns the version of each pin, the part of a pin a snapshot records; it is empty, never nil.
func pinnedVersions(pins map[string]rules.Pin) map[string]rules.RuleVersion {
	versions := make(map[string]rules.RuleVersion, len(pins))
	for id, pin := range pins {
		versions[id] = pin.Version
	}
	return versions
}

// sameGroupSelection reports whether two group selections request the same groups.
func sameGroupSelection(a, b rules.GroupSelection) bool {
	return a.Pattern == b.Pattern && slices.Equal(a.Groups, b.Groups)
}

// rulesInTree returns, sorted, the IDs of the rules whose Markdown files tree holds.
func rulesInTree(tree map[string]treeEntry) []string {
	ids := []string{}
	for file := range tree {
		if id, ok := rules.VersionedRule(file); ok && file == id+".md" {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}
