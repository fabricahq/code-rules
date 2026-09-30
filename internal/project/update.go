// Preview an update of the project's library rules, then install exactly the previewed versions with the pins and
// exclusions the project decided on.

package project

import (
	"bytes"
	"context"
	"fmt"
	"maps"
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
}

// UpdateDecision answers one preview row: Keep pins a changed or retired rule at its current version instead of
// applying the change; otherwise the decision excludes a new rule. Reason is recorded with the pin or exclusion.
type UpdateDecision struct {
	Source string
	Rule   string
	Keep   bool
	Reason string
}

// UpdateResult is an update's preview and, once applied, the files it changed.
type UpdateResult struct {
	// Applied is false for a preview, which wrote nothing and so lists no changed files.
	Applied bool                   `json:"applied"`
	Sources []imports.SourceUpdate `json:"sources"`
	// FileChanges lists vendor and generated paths, and config.yaml when the update wrote pins or exclusions.
	// Its warnings are the preview's until the update is applied, and then the installation's.
	FileChanges
}

// Moves reports whether the update changes any imported rule: every row except pinned rules, retirements a pin
// keeps, and rows the project decided to keep.
func (r UpdateResult) Moves() bool {
	for _, source := range r.Sources {
		for _, row := range source.Rules {
			if row.Change != imports.UpdatePinned && row.Pin == nil && row.Decision != "keep" {
				return true
			}
		}
	}
	return false
}

// PlanUpdate reads the project and plans moving the rules targets name, or every source's rules when there are
// no targets, to their newest versions. It reads the project under the writer, first recovering an interrupted
// operation as sync does, and releases the writer before planning, so prompts never hold it. It refuses a project
// another writer is changing.
func PlanUpdate(ctx context.Context, options Options, git imports.Options, targets []imports.UpdateTarget) (*UpdatePlan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := openProject(ctx, options, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	var state projectState
	var guide []byte
	err = filetxn.WithWriter(ctx, root, func(*filetxn.Writer) error {
		if state, err = readProject(ctx, root); err != nil {
			return err
		}
		name, _ := projectGuide()
		guide, err = filetxn.ReadOptional(ctx, root, name)
		return err
	})
	if err != nil {
		return nil, err
	}
	recorded, err := recordedSnapshots(state.config, treeFiles(state.vendor))
	if err != nil {
		return nil, err
	}
	update, err := imports.PlanUpdate(ctx, state.config, recorded, targets, git)
	if err != nil {
		return nil, fmt.Errorf("plan project update: %w", err)
	}
	return &UpdatePlan{options: options, git: git, update: update, planned: state, guide: guide}, nil
}

// Preview returns the planned update with each decided row marked, without writing anything. It fails when a
// decision has no reason, names a rule twice, keeps a rule the update doesn't move or retire, or excludes a rule
// the update doesn't add.
func (p *UpdatePlan) Preview(decisions []UpdateDecision) (UpdateResult, error) {
	sources, _, err := p.decide(decisions)
	if err != nil {
		return UpdateResult{}, err
	}
	return UpdateResult{Sources: sources, FileChanges: FileChanges{Added: []string{}, Changed: []string{}, Removed: []string{}, Warnings: p.update.Warnings}}, nil
}

// Apply installs the planned versions under the writer, adding a pin for each kept rule and an exclusion for each
// excluded one to config.yaml in the same transaction that replaces vendor and generated output, so recovery
// restores or finishes all three together. A kept rule stays at its current version. It fails with
// concurrent-change, writing nothing, when the configuration, local rules, vendor or generated output, or managed
// guide changed after planning.
func (p *UpdatePlan) Apply(ctx context.Context, decisions []UpdateDecision) (UpdateResult, error) {
	sources, edits, err := p.decide(decisions)
	if err != nil {
		return UpdateResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return UpdateResult{}, err
	}
	root, err := openProject(ctx, p.options, false)
	if err != nil {
		return UpdateResult{}, err
	}
	defer root.Close()
	var changes FileChanges
	err = filetxn.WithWriter(ctx, root, func(w *filetxn.Writer) error {
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
		in := installation{guide: guide, config: before.config, git: p.git, options: p.options}
		if len(edits) > 0 {
			if in.edited, in.config, err = editConfiguration(before.configBytes, edits); err != nil {
				return err
			}
		}
		in.recorded = p.update.Lock(in.config)
		changes, err = install(ctx, root, w, before, in)
		return err
	})
	if err != nil {
		return UpdateResult{}, fmt.Errorf("update project: %w", err)
	}
	return UpdateResult{Applied: true, Sources: sources, FileChanges: changes}, nil
}

// decide checks decisions against the planned preview and returns a copy of the preview with each decided row
// marked, and the pins and exclusions to add to each source.
func (p *UpdatePlan) decide(decisions []UpdateDecision) ([]imports.SourceUpdate, map[string]rules.SourceEdit, error) {
	sources := make([]imports.SourceUpdate, len(p.update.Sources))
	for i, source := range p.update.Sources {
		sources[i] = source
		sources[i].Rules = slices.Clone(source.Rules)
	}
	edits := map[string]rules.SourceEdit{}
	decided := map[string]bool{}
	for _, decision := range decisions {
		where := decision.Source + ":" + decision.Rule
		if decided[where] {
			return nil, nil, &rules.ValidationError{Location: where, Problem: "the update received more than one decision for this rule"}
		}
		decided[where] = true
		if strings.TrimSpace(decision.Reason) == "" {
			return nil, nil, &rules.ValidationError{Location: where, Problem: "give a reason to record with the decision"}
		}
		row := previewRow(sources, decision.Source, decision.Rule)
		edit := edits[decision.Source]
		if decision.Keep {
			if row == nil || !keepable(*row) {
				return nil, nil, &rules.ValidationError{Location: where, Problem: "the update doesn't move or retire this rule, so there's nothing to keep; name a rule the preview lists as major, minor, patch, retired, or replaced"}
			}
			row.Decision, row.Reason = "keep", decision.Reason
			if edit.Pins == nil {
				edit.Pins = map[string]rules.Pin{}
			}
			edit.Pins[decision.Rule] = rules.Pin{Version: *row.From, Reason: decision.Reason}
		} else {
			if row == nil || row.Change != imports.UpdateNew {
				return nil, nil, &rules.ValidationError{Location: where, Problem: "the update doesn't add this rule, so there's nothing to exclude; name a rule the preview lists as new"}
			}
			row.Decision, row.Reason = "exclude", decision.Reason
			if edit.Exclude == nil {
				edit.Exclude = map[string]rules.Exclusion{}
			}
			edit.Exclude[decision.Rule] = rules.Exclusion{Reason: decision.Reason}
		}
		edits[decision.Source] = edit
	}
	return sources, edits, nil
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
	case imports.UpdateMajor, imports.UpdateMinor, imports.UpdatePatch, imports.UpdateReplaced:
		return true
	case imports.UpdateRetired:
		return row.Pin == nil
	}
	return false
}

// editConfiguration adds each source's pins and exclusions to the configuration bytes, preserving comments, and
// returns the edited bytes and their parsed configuration.
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
