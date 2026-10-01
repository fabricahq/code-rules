// Preview an update of the project's library rules, then install exactly the previewed versions with the pins,
// exclusions, and forks the project decided on.

package project

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/rules"
)

// UpdatePlan is a previewed update. Planning reads the project and the libraries' release histories without
// writing anything; Apply installs exactly the planned versions.
type UpdatePlan struct {
	options Options
	git     imports.Options
	update  imports.Update
	// planned is the project the plan started from, and guide the managed guide's bytes then, nil when it was
	// absent; Apply refuses to install the plan when any of them changed.
	planned projectState
	guide   []byte
	// recovered reports that planning first recovered an interrupted earlier command, which changed files, and kept
	// the directories that recovery kept rather than remove.
	recovered bool
	kept      []string
	// forks holds each fork Preview or Apply read, by forkKey, so a fork is read from the library once.
	forks map[string]forkUpdate
}

// Recovered reports whether planning first recovered an interrupted earlier command, restoring or finishing its
// files, so the project changed even if the update then writes nothing.
func (p *UpdatePlan) Recovered() bool { return p.recovered }

// UpdateDecisionKind is what a decision does with its preview row.
type UpdateDecisionKind string

// The decisions an update accepts.
const (
	// DecisionKeep pins a changed or retired rule at its current version instead of applying the change.
	DecisionKeep UpdateDecisionKind = "keep"
	// DecisionExclude excludes a new rule instead of adding it.
	DecisionExclude UpdateDecisionKind = "exclude"
	// DecisionUpdateFork replaces a replaced rule's local rule and its asset directory with a fork of the library's
	// newest version, exactly as forking that version writes it, overwriting any edits, and sets the exclusion's
	// basedOn to that version, so later updates compare with it.
	DecisionUpdateFork UpdateDecisionKind = "update-fork"
)

// UpdateDecision answers one preview row. Reason is recorded with a pin or exclusion, and a fork update has none.
type UpdateDecision struct {
	Source string
	Rule   string
	Kind   UpdateDecisionKind
	Reason string
}

// UpdateResult is an update's preview and, once applied, the files it changed.
type UpdateResult struct {
	// Applied is false for a preview, which wrote nothing and so lists no changed files.
	Applied bool                   `json:"applied"`
	Sources []imports.SourceUpdate `json:"sources"`
	// FileChanges lists vendor and generated paths, config.yaml when the update wrote pins, exclusions, or basedOn
	// versions, and the local files of each fork it replaced.
	// Its warnings are the preview's until the update is applied, and then the installation's.
	FileChanges
}

// Moves reports whether the update has anything to apply: a source's library-wide files, or a rule of any row
// except pinned rules, retirements a pin keeps, and rows the project decided to keep. A replaced row counts even
// when its imported copy stays, since applying it can replace the local rule's fork.
func (r UpdateResult) Moves() bool {
	for _, source := range r.Sources {
		if source.SharedFiles != nil {
			return true
		}
		for _, row := range source.Rules {
			if row.Change == imports.UpdateReplaced || row.Change != imports.UpdatePinned && row.Pin == nil && row.Decision != string(DecisionKeep) {
				return true
			}
		}
	}
	return false
}

// PlanUpdate reads the project and plans moving the rules targets name, or every source's rules when there are
// no targets, to their newest versions. It reads the project under the writer, first recovering an interrupted
// operation as sync does, and releases the writer before planning, so prompts never hold it. It refuses a project
// another writer is changing. Before reading any library, it refuses with invalid-arguments a fork update among
// decisions that the configuration rules out; Preview and Apply check every decision against the preview.
func PlanUpdate(ctx context.Context, options Options, git imports.Options, targets []imports.UpdateTarget, decisions []UpdateDecision) (*UpdatePlan, error) {
	// recovered reports that the writer first recovered an interrupted earlier command, which changed files.
	recovered := false
	if err := ctx.Err(); err != nil {
		return nil, unchanged(err, recovered)
	}
	root, err := openProject(ctx, options, false)
	if err != nil {
		return nil, unchanged(err, recovered)
	}
	defer root.Close()
	var state projectState
	var guide []byte
	var kept []string
	err = filetxn.WithWriter(ctx, root, func(w *filetxn.Writer) error {
		recovered, kept = w.Recovered(), w.Kept()
		if state, err = readProject(ctx, root); err != nil {
			return err
		}
		name, _ := projectGuide()
		guide, err = filetxn.ReadOptional(ctx, root, name)
		return err
	})
	if err != nil {
		return nil, unchanged(err, recovered)
	}
	for _, decision := range decisions {
		if decision.Kind != DecisionUpdateFork {
			continue
		}
		if err := forkUpdateRefusal(state.config, decision); err != nil {
			return nil, unchanged(err, recovered)
		}
	}
	recorded, err := recordedSnapshots(state.config, treeFiles(state.vendor))
	if err != nil {
		return nil, unchanged(err, recovered)
	}
	// imports.PlanUpdate names the source or argument that failed, which is all the context the command needs.
	update, err := imports.PlanUpdate(ctx, state.config, recorded, targets, git)
	if err != nil {
		return nil, unchanged(err, recovered)
	}
	return &UpdatePlan{options: options, git: git, update: update, planned: state, guide: guide, recovered: recovered, kept: kept}, nil
}

// Preview returns the planned update with each decided row marked, without writing anything. It fails with
// invalid-arguments, naming the decision's flag and rule, when a decision names a rule twice, keeps a rule the
// update doesn't move or retire, excludes a rule the update doesn't add, or updates a fork the preview doesn't list
// as replaced, or when a pin or exclusion has no reason or a fork update has one.
//
// Preview reads the version each fork update forks, as Apply then installs it, to list the local files the fork
// overwrites and removes; it fails with invalid-release-tag when that version's library release tag moved after
// planning.
func (p *UpdatePlan) Preview(ctx context.Context, decisions []UpdateDecision) (UpdateResult, error) {
	sources, _, forks, err := p.decide(decisions)
	if err != nil {
		return UpdateResult{}, unchanged(err, p.recovered)
	}
	if err := p.readForks(ctx, sources, forks); err != nil {
		return UpdateResult{}, unchanged(err, p.recovered)
	}
	warnings := append(slices.Clone(p.update.Warnings), keptWarnings(p.kept)...)
	return UpdateResult{Sources: sources, FileChanges: FileChanges{Added: []string{}, Changed: []string{}, Removed: []string{}, Warnings: warnings}}, nil
}

// Apply installs the planned versions under the writer, adding to config.yaml a pin for each kept rule, an
// exclusion for each excluded one, and the new basedOn of each replaced fork, in the same transaction that replaces
// vendor and generated output and each replaced fork's local rule and asset directory, so recovery restores or
// finishes them together. A kept rule stays at its current version. Each fork is read from the library before the
// transaction, as code-rules project add rule --from reads it. It fails with concurrent-change, writing nothing,
// when the configuration, local rules, vendor or generated output, or managed guide changed after planning.
func (p *UpdatePlan) Apply(ctx context.Context, decisions []UpdateDecision) (UpdateResult, error) {
	// recovered reports that the writer first recovered an interrupted earlier command, which changed files.
	recovered := false
	sources, edits, forks, err := p.decide(decisions)
	if err != nil {
		return UpdateResult{}, unchanged(err, p.recovered)
	}
	if err := ctx.Err(); err != nil {
		return UpdateResult{}, unchanged(err, p.recovered || recovered)
	}
	if err := p.readForks(ctx, sources, forks); err != nil {
		return UpdateResult{}, unchanged(err, p.recovered || recovered)
	}
	forked := []imports.ForkedRule{}
	for _, fork := range forks {
		forked = append(forked, imports.ForkedRule{Source: fork.source, ID: fork.id, Version: fork.version})
	}
	root, err := openProject(ctx, p.options, false)
	if err != nil {
		return UpdateResult{}, unchanged(err, p.recovered || recovered)
	}
	defer root.Close()
	var changes FileChanges
	var kept []string
	err = filetxn.WithWriter(ctx, root, func(w *filetxn.Writer) error {
		recovered, kept = w.Recovered(), w.Kept()
		before, err := readProject(ctx, root)
		if err != nil {
			return err
		}
		guide, err := planProjectGuide(ctx, root)
		if err != nil {
			return err
		}
		if !sameProject(p.planned, before) || (guide.Previous == nil) != (p.guide == nil) || !bytes.Equal(guide.Previous, p.guide) {
			return failure("concurrent-change", "the project's configuration, local rules, imported files, generated output, or Code Rules guide changed after the update preview; run code-rules project update again", nil)
		}
		in := installation{guide: guide, config: before.config, options: p.options, forks: forks}
		if len(edits) > 0 {
			if in.edited, in.config, err = editConfiguration(before.configBytes, edits); err != nil {
				return err
			}
		}
		git := p.git
		git.GroupMetadata = groupsWithoutLocalMetadata(before)
		if in.imported, err = p.update.Import(ctx, in.config, git, forked); err != nil {
			return err
		}
		changes, err = install(ctx, root, w, before, in)
		return err
	})
	if err != nil {
		return UpdateResult{}, unchanged(err, p.recovered || recovered)
	}
	changes.Recovered = p.recovered || recovered
	changes.Warnings = append(changes.Warnings, keptWarnings(append(slices.Clone(p.kept), kept...))...)
	return UpdateResult{Applied: true, Sources: sources, FileChanges: changes}, nil
}

// decide checks decisions against the planned preview and returns a copy of the preview with each decided row
// marked, the pins, exclusions, and basedOn versions to set in each source, and the forks to replace, not yet read.
func (p *UpdatePlan) decide(decisions []UpdateDecision) ([]imports.SourceUpdate, map[string]rules.SourceEdit, []forkUpdate, error) {
	sources := make([]imports.SourceUpdate, len(p.update.Sources))
	for i, source := range p.update.Sources {
		sources[i] = source
		sources[i].Rules = slices.Clone(source.Rules)
		for j := range sources[i].Rules {
			sources[i].Rules[j].Overwrites, sources[i].Rules[j].Removes = []string{}, []string{}
		}
	}
	edits := map[string]rules.SourceEdit{}
	forks := []forkUpdate{}
	decided := map[string]bool{}
	for _, decision := range decisions {
		where := decision.Source + ":" + decision.Rule
		// refuse names the flag that makes the decision, so the refusal reads like the command line that gave it.
		refuse := func(problem string) error {
			return invalidArguments(decisionFlags[decision.Kind] + " " + where + ": " + problem)
		}
		if decided[where] {
			return nil, nil, nil, refuse("the update received more than one decision for this rule")
		}
		decided[where] = true
		if decision.Kind != DecisionUpdateFork && strings.TrimSpace(decision.Reason) == "" {
			return nil, nil, nil, refuse("give a reason to record with the decision")
		}
		row := previewRow(sources, decision.Source, decision.Rule)
		edit := edits[decision.Source]
		switch decision.Kind {
		case DecisionKeep:
			if row == nil || !keepable(*row) {
				return nil, nil, nil, refuse("the update doesn't move or retire this rule, so there's nothing to keep; name a rule the preview lists as major, minor, patch, retired, or replaced")
			}
			row.Decision, row.Reason = string(DecisionKeep), decision.Reason
			if edit.Pins == nil {
				edit.Pins = map[string]rules.Pin{}
			}
			edit.Pins[decision.Rule] = rules.Pin{Version: *row.From, Reason: decision.Reason}
		case DecisionUpdateFork:
			if err := forkUpdateRefusal(p.planned.config, decision); err != nil {
				return nil, nil, nil, err
			}
			switch {
			case row != nil && row.Change == imports.UpdateRetired:
				return nil, nil, nil, refuse("the library retired this rule, so it has no newest version to fork; keep your local rule as it is")
			case row == nil || row.Change != imports.UpdateReplaced:
				return nil, nil, nil, refuse("the update doesn't list this rule as replaced, because the library has no version newer than the one your local rule is based on, or the update doesn't include the rule; name a rule the preview lists as replaced")
			}
			version := *row.ReviewedVersion()
			row.Decision = string(DecisionUpdateFork)
			if edit.BasedOn == nil {
				edit.BasedOn = map[string]rules.RuleVersion{}
			}
			edit.BasedOn[decision.Rule] = version
			forks = append(forks, forkUpdate{source: decision.Source, id: decision.Rule, version: version, file: strings.TrimPrefix(row.LocalRule, "local/")})
		case DecisionExclude:
			if row == nil || row.Change != imports.UpdateNew {
				return nil, nil, nil, refuse("the update doesn't add this rule, so there's nothing to exclude; name a rule the preview lists as new")
			}
			row.Decision, row.Reason = string(DecisionExclude), decision.Reason
			if edit.Exclude == nil {
				edit.Exclude = map[string]rules.Exclusion{}
			}
			edit.Exclude[decision.Rule] = rules.Exclusion{Reason: decision.Reason}
		default:
			return nil, nil, nil, refuse(fmt.Sprintf("unknown update decision %q", decision.Kind))
		}
		edits[decision.Source] = edit
	}
	// Keeping a rule a scoped update would move can leave the shared files where they are, so they follow the
	// decisions.
	for i := range sources {
		sources[i].SharedFiles = p.update.SharedFiles(sources[i].Name, slices.Collect(maps.Keys(edits[sources[i].Name].Pins)))
	}
	return sources, edits, forks, nil
}

// ReplacedFiles returns the local files that replacing the fork at localRule, a replaced row's local rule,
// overwrites or removes, as the project was when planned: the rule and every file in its asset directory, relative
// to the Code Rules directory and sorted. It is empty when none of them exist.
func (p *UpdatePlan) ReplacedFiles(localRule string) []string {
	overwritten := slices.Sorted(maps.Keys(localPaths(forkPaths(treeFiles(p.planned.local), strings.TrimPrefix(localRule, "local/")))))
	if overwritten == nil {
		return []string{}
	}
	return overwritten
}

// readForks reads each of forks that p hasn't read yet, filling its files, and lists in its row of sources the
// local files the new fork overwrites, which it has too, and removes, which it doesn't.
func (p *UpdatePlan) readForks(ctx context.Context, sources []imports.SourceUpdate, forks []forkUpdate) error {
	if p.forks == nil {
		p.forks = map[string]forkUpdate{}
	}
	for i := range forks {
		key := forks[i].source + ":" + forks[i].id + "@" + forks[i].version.String()
		if read, ok := p.forks[key]; ok {
			forks[i].files = read.files
		} else if err := forks[i].read(ctx, p.update, p.planned.config, p.git); err != nil {
			return err
		}
		p.forks[key] = forks[i]
		row := previewRow(sources, forks[i].source, forks[i].id)
		for name := range forkPaths(treeFiles(p.planned.local), forks[i].file) {
			if _, kept := forks[i].files[name]; kept {
				row.Overwrites = append(row.Overwrites, path.Join("local", name))
			} else {
				row.Removes = append(row.Removes, path.Join("local", name))
			}
		}
		slices.Sort(row.Overwrites)
		slices.Sort(row.Removes)
	}
	return nil
}

// forkUpdateRefusal returns the invalid-arguments refusal of decision, a fork update, that configuration alone
// decides, or nil: it has a reason, or names a source configuration doesn't have or a rule that source doesn't
// exclude with replacedBy.
func forkUpdateRefusal(configuration rules.Configuration, decision UpdateDecision) error {
	where := "--update-fork " + decision.Source + ":" + decision.Rule + ": "
	if decision.Reason != "" {
		return invalidArguments(where + "replacing a fork records no reason; give --reason only with --keep or --exclude")
	}
	index := slices.IndexFunc(configuration.Sources, func(source rules.Source) bool { return source.Name == decision.Source })
	if index < 0 {
		return invalidArguments(where + "no source named " + decision.Source + " in .code-rules/config.yaml")
	}
	if exclusion := configuration.Sources[index].Exclude[decision.Rule]; exclusion.ReplacedBy == "" {
		return invalidArguments(where + "sources." + decision.Source + " doesn't exclude this rule with replacedBy, so no local rule replaces it; to fork the rule, run code-rules project add rule " + decision.Rule + " --from " + decision.Source + "@VERSION")
	}
	return nil
}

// decisionFlags names the code-rules project update flag that makes each kind of decision; a terminal answer
// makes the same decision.
var decisionFlags = map[UpdateDecisionKind]string{DecisionKeep: "--keep", DecisionExclude: "--exclude", DecisionUpdateFork: "--update-fork"}

// invalidArguments is a refusal of the command's arguments, which the CLI reports as a usage error.
func invalidArguments(problem string) error { return failure("invalid-arguments", problem, nil) }

// forkUpdate is a fork the update replaces: the local rule at file, relative to local/, which replaces rule id of
// source, becomes a fork of version. read fills files.
type forkUpdate struct {
	source, id, file string
	version          rules.RuleVersion
	// files holds the new fork's files by path relative to local/: the rule at file and its asset directory.
	files map[string][]byte
}

// read reads the forked version from the library of the source in configuration, from the library release that
// planning the update found published it, and prepares the fork's files at the local rule's path, exactly as
// code-rules project add rule --from writes a fork there.
func (f *forkUpdate) read(ctx context.Context, update imports.Update, configuration rules.Configuration, options imports.Options) error {
	index := slices.IndexFunc(configuration.Sources, func(source rules.Source) bool { return source.Name == f.source })
	library := rules.Source{Name: f.source, Repository: configuration.Sources[index].Repository}
	published, err := update.ReadFork(ctx, library, f.id, f.version, options)
	if err != nil {
		return fmt.Errorf("fork %s@%s: %w", f.id, f.version, err)
	}
	attribution, err := forkAttribution(library.Repository, f.id, f.version, published)
	if err != nil {
		return err
	}
	if f.files, err = forkFiles(f.id, f.file, published, attribution); err != nil {
		return fmt.Errorf("fork %s@%s: %w", f.id, f.version, err)
	}
	return nil
}

// previewRow returns the preview row of rule in source, or nil when the preview doesn't list it.
func previewRow(sources []imports.SourceUpdate, source, rule string) *imports.RuleUpdate {
	for i := range sources {
		if sources[i].Name != source {
			continue
		}
		for j := range sources[i].Rules {
			if sources[i].Rules[j].ID == rule {
				return &sources[i].Rules[j]
			}
		}
	}
	return nil
}

// keepable reports whether a pin at the row's current version would keep the update from moving or dropping it.
func keepable(row imports.RuleUpdate) bool {
	switch row.Change {
	case imports.UpdateMajor, imports.UpdateMinor, imports.UpdatePatch:
		return true
	case imports.UpdateReplaced, imports.UpdateRetired:
		return row.Pin == nil
	}
	return false
}

// editConfiguration applies each source's pins, exclusions, and basedOn versions to the configuration bytes,
// preserving comments, and returns the edited bytes and their parsed configuration.
func editConfiguration(data []byte, edits map[string]rules.SourceEdit) ([]byte, rules.Configuration, error) {
	for _, name := range slices.Sorted(maps.Keys(edits)) {
		var err error
		if data, err = rules.EditConfigurationSource(data, name, edits[name]); err != nil {
			return nil, rules.Configuration{}, err
		}
	}
	config, err := rules.ParseConfigurationYAML(data)
	if err != nil {
		return nil, rules.Configuration{}, err
	}
	return data, config, nil
}
