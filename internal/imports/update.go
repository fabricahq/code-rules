// Plan project update: move rules to their newest versions, preview each change, and lock the versions to install.

package imports

import (
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

// UpdateTarget names what an update moves: every rule of Source when Rule is empty, or only the library rule Rule.
type UpdateTarget struct {
	Source string
	Rule   string
}

// UpdateChange is how an update affects one rule, as the update preview lists it.
type UpdateChange string

// The update preview's changes, in the order it lists them.
const (
	UpdateMajor    UpdateChange = "major"
	UpdateMinor    UpdateChange = "minor"
	UpdatePatch    UpdateChange = "patch"
	UpdateNew      UpdateChange = "new"
	UpdateRetired  UpdateChange = "retired"
	UpdateReplaced UpdateChange = "replaced"
	UpdatePinned   UpdateChange = "pinned"
)

var updateChangeOrder = []UpdateChange{UpdateMajor, UpdateMinor, UpdatePatch, UpdateNew, UpdateRetired, UpdateReplaced, UpdatePinned}

// SourceUpdate is one source's part of the update preview.
type SourceUpdate struct {
	Name string `json:"name"`
	// Ref is the source's ref, which update doesn't move, so Rules is empty; it is empty for other sources.
	Ref string `json:"ref,omitempty"`
	// Rules is sorted by change, in the order of the UpdateChange constants, then by rule ID. It is empty, never
	// nil, when nothing changes.
	Rules []RuleUpdate `json:"rules"`
}

// RuleUpdate is one row of the update preview.
type RuleUpdate struct {
	ID     string       `json:"id"`
	Change UpdateChange `json:"change"`
	// From is the version the project imports; it is nil for a new rule.
	From *rules.RuleVersion `json:"from,omitempty"`
	// To is the version the update installs; it is nil for retired and pinned rules, which don't move.
	To *rules.RuleVersion `json:"to,omitempty"`
	// Newest is a pinned rule's newest version, and LastVersion a retired rule's; each is nil otherwise.
	Newest      *rules.RuleVersion `json:"newest,omitempty"`
	LastVersion *rules.RuleVersion `json:"lastVersion,omitempty"`
	// Summaries holds one line per change note, oldest first: every version after From up to To, every version of
	// a new rule, or the retirement. It is empty, never nil, for a pinned rule.
	Summaries []string `json:"summaries"`
	// ReplacedBy is the library rule that replaces a retired rule, when there is one.
	ReplacedBy string `json:"replacedBy,omitempty"`
	// LocalRule is the project's local rule that replaces a replaced rule, relative to the Code Rules directory.
	LocalRule string `json:"localRule,omitempty"`
	// Pin is the configured pin of a pinned rule, or of a retired rule a pin keeps.
	Pin *rules.Pin `json:"pin,omitempty"`
	// Decision is "keep" when the project pins the rule at From instead of applying the change, or "exclude" when it
	// excludes a new rule; Reason is recorded with it. Planning leaves both empty.
	Decision string `json:"decision,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// Update is a planned project update: each source's preview, and the versions that installing it restores.
type Update struct {
	// Sources lists, in configuration order, the sources the update names. A source that uses ref has no rows.
	Sources []SourceUpdate
	// Warnings explain configuration the update tolerates, as sync reports them, for the sources it names. It is
	// empty, never nil, when there are none.
	Warnings []string
	// plans holds the planned rules of each source the update names; recorded holds every recorded snapshot.
	plans    map[string]plannedSource
	recorded map[string]library.Snapshot
}

// plannedSource is one source before and after the update. Before is what project sync would import.
type plannedSource struct {
	before, after sourcePlan
}

// PlanUpdate plans moving the rules that targets name to their newest versions, reading each named source's
// release history without fetching rule files or writing anything. No targets names every source. It starts from
// what project sync would import from configuration and recorded, the snapshots from the last sync without
// files, so the preview never repeats changes sync makes on its own. It fails without a partial result when a
// target names an unknown source, a source that uses ref, or a rule the source doesn't import.
func PlanUpdate(ctx context.Context, configuration rules.Configuration, recorded map[string]library.Snapshot, targets []UpdateTarget, options Options) (Update, error) {
	if err := ctx.Err(); err != nil {
		return Update{}, gitexec.ContextFailure(err)
	}
	scopes, err := updateScopes(configuration, targets)
	if err != nil {
		return Update{}, err
	}
	update := Update{Sources: []SourceUpdate{}, Warnings: []string{}, plans: map[string]plannedSource{}, recorded: maps.Clone(recorded)}
	for _, source := range configuration.Sources {
		scope, named := scopes[source.Name]
		if !named {
			continue
		}
		var previous *library.Snapshot
		if snapshot, ok := recorded[source.Name]; ok {
			previous = &snapshot
		}
		planned, preview, err := planSourceUpdate(ctx, source, previous, scope, options)
		if sourceQualified(err, source.Name) {
			return Update{}, err
		}
		if err != nil {
			return Update{}, fmt.Errorf("update source %q: %w", source.Name, err)
		}
		update.plans[source.Name] = planned
		update.Sources = append(update.Sources, preview)
		update.Warnings = append(update.Warnings, planned.after.warnings...)
	}
	return update, nil
}

// sourceQualified reports whether err is a validation error of the source's configuration or of a SOURCE:RULE
// argument naming it, whose location already names the source. Other failures, such as an invalid release
// record in the library, need the source's name added.
func sourceQualified(err error, source string) bool {
	var validation *rules.ValidationError
	if !errors.As(err, &validation) || err != error(validation) {
		return false
	}
	location := validation.Location
	return location == "sources."+source || strings.HasPrefix(location, "sources."+source+".") || strings.HasPrefix(location, source+":")
}

// updateScopes maps each source the targets name to the rules they name, or to nil for every rule. No targets
// names every source.
func updateScopes(configuration rules.Configuration, targets []UpdateTarget) (map[string][]string, error) {
	scopes := map[string][]string{}
	whole := map[string]bool{}
	for _, source := range configuration.Sources {
		if len(targets) == 0 {
			scopes[source.Name] = nil
		}
	}
	for _, target := range targets {
		where := target.Source
		if target.Rule != "" {
			where += ":" + target.Rule
		}
		index := slices.IndexFunc(configuration.Sources, func(source rules.Source) bool { return source.Name == target.Source })
		if index < 0 {
			return nil, &rules.ValidationError{Location: where, Problem: "no source named " + target.Source + " in .code-rules/config.yaml"}
		}
		if configuration.Sources[index].Ref != "" {
			return nil, &rules.ValidationError{Location: where, Problem: "the source imports one revision with ref, so update doesn't move it; change sources." + target.Source + ".ref and run code-rules project sync"}
		}
		if target.Rule == "" {
			whole[target.Source] = true
			scopes[target.Source] = nil
		} else if !whole[target.Source] && !slices.Contains(scopes[target.Source], target.Rule) {
			scopes[target.Source] = append(scopes[target.Source], target.Rule)
		}
	}
	return scopes, nil
}

// planSourceUpdate plans one source within the same deadline as an import. A nil scope moves every rule and adds
// new ones; otherwise it moves only the rules scope names. A source that uses ref stays as sync would import it.
func planSourceUpdate(ctx context.Context, source rules.Source, recorded *library.Snapshot, scope []string, options Options) (_ plannedSource, _ SourceUpdate, err error) {
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	if timeout < 0 {
		return plannedSource{}, SourceUpdate{}, fail("invalid-options", "Import timeout must be positive or zero for the default.", nil)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	repo, err := openRepository(ctx, source, options)
	if err != nil {
		return plannedSource{}, SourceUpdate{}, err
	}
	// Joining only a failed Close keeps a validation error unwrapped, so its location reads on its own.
	defer func() {
		if closeErr := repo.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	p := newPlanner(ctx, repo, source, recorded)
	before, err := p.choose()
	if err != nil {
		return plannedSource{}, SourceUpdate{}, err
	}
	preview := SourceUpdate{Name: source.Name, Ref: source.Ref, Rules: []RuleUpdate{}}
	if source.Ref != "" {
		return plannedSource{before, before}, preview, nil
	}
	for _, id := range scope {
		if _, imported := before.rules[id]; !imported {
			return plannedSource{}, SourceUpdate{}, &rules.ValidationError{Location: source.Name + ":" + id, Problem: "source " + source.Name + " doesn't import this rule; name a rule it imports"}
		}
	}
	after, rows, err := p.update(before, scope)
	if err != nil {
		return plannedSource{}, SourceUpdate{}, err
	}
	if err := p.requireUnmoved(before); err != nil {
		return plannedSource{}, SourceUpdate{}, err
	}
	preview.Rules = rows
	return plannedSource{before, after}, preview, nil
}

// update moves the rules in scope, or every rule when scope is nil, to their newest versions and drops retired
// ones, except that pins keep rules where they are. A nil scope also adds the rules the library added to the
// selected groups. It returns what the source then imports, and a preview row for each change: none for a rule
// excluded without a replacement, and replaced for one with a replacement.
func (p *planner) update(before sourcePlan, scope []string) (sourcePlan, []RuleUpdate, error) {
	history, err := p.versioned()
	if err != nil {
		return sourcePlan{}, nil, err
	}
	newest := history.newest().record
	after := sourcePlan{rules: maps.Clone(before.rules), individual: []string{}, warnings: []string{}}
	rows := []RuleUpdate{}
	ids := scope
	if ids == nil {
		ids = slices.Sorted(maps.Keys(before.rules))
	}
	for _, id := range ids {
		current := before.rules[id]
		latest, published := newest.Rules[id]
		exclusion, excluded := p.source.Exclude[id]
		listed := !excluded || exclusion.ReplacedBy != ""
		pin, pinned := p.source.Pins[id]
		switch {
		case pinned && !listed:
		case pinned && !published:
			row, err := retiredRow(id, current, history)
			if err != nil {
				return sourcePlan{}, nil, err
			}
			row.Pin = &pin
			rows = append(rows, row)
		case pinned && latest.Compare(*current.Version) > 0:
			rows = append(rows, RuleUpdate{ID: id, Change: UpdatePinned, From: current.Version, Newest: &latest, Summaries: []string{}, Pin: &pin})
		case pinned:
		case !published:
			delete(after.rules, id)
			if listed {
				row, err := retiredRow(id, current, history)
				if err != nil {
					return sourcePlan{}, nil, err
				}
				rows = append(rows, row)
			}
		case latest.Compare(*current.Version) > 0:
			if after.rules[id], err = p.publishedVersion(id, latest); err != nil {
				return sourcePlan{}, nil, err
			}
			if !listed {
				continue
			}
			row := RuleUpdate{ID: id, Change: versionChange(*current.Version, latest), From: current.Version, To: &latest, Summaries: history.summaries(id, current.Version, latest)}
			if excluded {
				row.Change, row.LocalRule = UpdateReplaced, exclusion.ReplacedBy
			}
			rows = append(rows, row)
		}
	}
	if scope == nil {
		for _, id := range slices.Sorted(maps.Keys(newest.Rules)) {
			if _, imported := before.rules[id]; imported || !p.source.Groups.Includes(ruleGroup(id)) {
				continue
			}
			version := newest.Rules[id]
			if after.rules[id], err = p.publishedVersion(id, version); err != nil {
				return sourcePlan{}, nil, err
			}
			rows = append(rows, RuleUpdate{ID: id, Change: UpdateNew, To: &version, Summaries: history.summaries(id, nil, version)})
		}
	}
	slices.SortFunc(rows, func(a, b RuleUpdate) int {
		if order := slices.Index(updateChangeOrder, a.Change) - slices.Index(updateChangeOrder, b.Change); order != 0 {
			return order
		}
		return strings.Compare(a.ID, b.ID)
	})
	if err := p.settle(&after); err != nil {
		return sourcePlan{}, nil, err
	}
	return after, rows, nil
}

// settle completes a plan whose rules the update chose: individually selected rules to load, warnings for entries
// naming rules the library retired, and the library release that supplies library-wide files.
func (p *planner) settle(plan *sourcePlan) error {
	for _, id := range p.source.Rules {
		if _, imported := plan.rules[id]; imported {
			plan.individual = append(plan.individual, id)
		} else {
			plan.warnings = append(plan.warnings, p.retiredEntry("rules", id))
		}
	}
	if err := p.requireEntries(plan); err != nil {
		return err
	}
	newest := p.history.newest()
	plan.release, plan.commit = newest.number, newest.commit
	if len(plan.rules) > 0 {
		plan.release = 0
		for _, rule := range plan.rules {
			if rule.Release > plan.release {
				plan.release, plan.commit = rule.Release, rule.Commit
			}
		}
	}
	return nil
}

// retiredRow previews the retirement of rule id, which the project imports at current.
func retiredRow(id string, current library.ImportedRule, history releaseHistory) (RuleUpdate, error) {
	retired := history.retirement(id)
	if retired == nil {
		return RuleUpdate{}, fail("invalid-release-tag", fmt.Sprintf("Rule %s is missing from library release release/%d, but no library release retired it. Don't create or move release tags by hand.", id, history.newest().number), nil)
	}
	last := retired.LastVersion
	return RuleUpdate{ID: id, Change: UpdateRetired, From: current.Version, LastVersion: &last, Summaries: summaryLines(retired.Summary), ReplacedBy: retired.ReplacedBy}, nil
}

// versionChange classifies moving from one version to a newer one by the largest component that changed.
func versionChange(from, to rules.RuleVersion) UpdateChange {
	switch {
	case to.Major != from.Major:
		return UpdateMajor
	case to.Minor != from.Minor:
		return UpdateMinor
	}
	return UpdatePatch
}

// Lock returns the snapshots, without files, that ImportLibraries restores to install the update under
// configuration: the planned configuration plus the pins and exclusions the project decided on. Each named source
// gets its planned versions, except that a rule configuration pins keeps its version from before the update; any
// other source keeps its recorded snapshot, so it imports what sync would.
func (u Update) Lock(configuration rules.Configuration) map[string]library.Snapshot {
	result := map[string]library.Snapshot{}
	for _, source := range configuration.Sources {
		planned, named := u.plans[source.Name]
		if !named {
			if snapshot, ok := u.recorded[source.Name]; ok {
				result[source.Name] = snapshot
			}
			continue
		}
		imported := maps.Clone(planned.after.rules)
		for id := range source.Pins {
			if rule, ok := planned.before.rules[id]; ok {
				imported[id] = rule
			}
		}
		release, commit := planned.after.release, planned.after.commit
		if source.Ref == "" && len(imported) > 0 {
			release = 0
			for _, rule := range imported {
				if rule.Release > release {
					release, commit = rule.Release, rule.Commit
				}
			}
		}
		result[source.Name] = library.Snapshot{
			Repository:    source.Repository,
			Pins:          pinnedVersions(source.Pins),
			Exclude:       slices.Sorted(maps.Keys(source.Exclude)),
			Ref:           source.Ref,
			Release:       release,
			Commit:        commit,
			Selection:     rules.GroupSelection{Pattern: source.Groups.Pattern, Groups: slices.Clone(source.Groups.Groups)},
			Groups:        []string{},
			RuleSelection: slices.Clone(source.Rules),
			Rules:         imported,
		}
	}
	return result
}
