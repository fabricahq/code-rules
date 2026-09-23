// Plan project authoring without retaining roots or locks; commit revalidates live state.

package project

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
)

// LocalGroupPlan identifies a new project-only group. It owns no filesystem resources.
type LocalGroupPlan struct {
	id      string
	options Options
}

// LocalRulePlan identifies a new project-only rule in an existing local or imported group.
type LocalRulePlan struct {
	id      string
	options Options
}

// SourcePlan reserves no alias; Commit refuses an alias taken after planning.
type SourcePlan struct {
	alias   string
	options Options
}

// PlanLocalGroup checks the target before input collection and anchors the project directory.
func PlanLocalGroup(ctx context.Context, id string, options Options) (*LocalGroupPlan, error) {
	options, err := planProjectAuthoring(ctx, options, func(root *os.Root, _ rules.Configuration) error { return checkLocalGroup(ctx, root, id) })
	if err != nil {
		return nil, err
	}
	return &LocalGroupPlan{id, options}, nil
}

// Commit validates metadata and repeats target checks under writer ownership before publishing.
func (p *LocalGroupPlan) Commit(ctx context.Context, metadata rules.GroupMetadata) (AuthoringResult, error) {
	if p == nil || p.id == "" {
		return AuthoringResult{}, failure("invalid-operation", "expected a planned group", nil)
	}
	return AddLocalGroup(ctx, p.id, metadata, p.options)
}

// PlanLocalRule checks path, group availability, and target absence before collecting metadata.
func PlanLocalRule(ctx context.Context, id string, options Options) (*LocalRulePlan, error) {
	options, err := planProjectAuthoring(ctx, options, func(root *os.Root, config rules.Configuration) error { return checkLocalRule(ctx, root, config, id) })
	if err != nil {
		return nil, err
	}
	return &LocalRulePlan{id, options}, nil
}

// Commit rechecks the group and target before creating a complete rule or draft when body is nil.
func (p *LocalRulePlan) Commit(ctx context.Context, metadata rules.RuleMetadata, body *string) (AuthoringResult, error) {
	if p == nil || p.id == "" {
		return AuthoringResult{}, failure("invalid-operation", "expected a planned rule", nil)
	}
	return AddLocalRule(ctx, p.id, metadata, RuleOptions{Options: p.options, Body: body})
}

// PlanSource rejects an occupied library alias before collecting its declaration; it never fetches.
func PlanSource(ctx context.Context, alias string, options Options) (*SourcePlan, error) {
	options, err := planProjectAuthoring(ctx, options, func(_ *os.Root, config rules.Configuration) error { return checkSourceAlias(config, alias) })
	if err != nil {
		return nil, err
	}
	return &SourcePlan{alias, options}, nil
}

// Commit validates and stores a source declaration, rechecking the alias under writer ownership.
func (p *SourcePlan) Commit(ctx context.Context, declaration SourceInput) (AuthoringResult, error) {
	if p == nil || p.alias == "" {
		return AuthoringResult{}, failure("invalid-operation", "expected a planned library alias", nil)
	}
	return AddSource(ctx, p.alias, declaration, p.options)
}

// planProjectAuthoring closes its advisory snapshot before prompts; only commit owns recovery and writes.
func planProjectAuthoring(ctx context.Context, options Options, check func(*os.Root, rules.Configuration) error) (Options, error) {
	directory, err := filepath.Abs(options.Directory)
	if err != nil {
		return Options{}, err
	}
	options.Directory = directory
	root, err := openProject(ctx, options, false)
	if err != nil {
		return Options{}, err
	}
	defer root.Close()
	_, config, err := configuration(ctx, root)
	if err != nil {
		return Options{}, err
	}
	return options, check(root, config)
}

func checkLocalGroup(ctx context.Context, root *os.Root, id string) error {
	if err := rules.ValidateGroupID(id, "group"); err != nil {
		return err
	}
	return filetxn.RequireAbsent(ctx, root, path.Join("local", id, "_group.yaml"), path.Join("local", id, "README.md"))
}

func checkLocalRule(ctx context.Context, root *os.Root, config rules.Configuration, id string) error {
	if strings.HasSuffix(id, ".md") {
		return failure("invalid-rule-path", "use a rule path without the .md extension", nil)
	}
	group, err := rules.GroupFromPath(id+".md", "rule")
	if err != nil {
		return err
	}
	available, err := groupAvailable(ctx, root, config, group)
	if err != nil {
		return err
	}
	if !available {
		return failure("missing-group", "group "+group+" does not exist; create it first with code-rules project add group "+group+", then retry adding the rule", nil)
	}
	return filetxn.RequireAbsent(ctx, root, path.Join("local", id+".md"))
}

func checkSourceAlias(config rules.Configuration, alias string) error {
	for _, source := range config.Sources {
		if source.Name == alias {
			return failure("source-exists", "library alias "+alias+" already exists; edit .code-rules/config.yaml or choose a different alias", nil)
		}
	}
	return nil
}
