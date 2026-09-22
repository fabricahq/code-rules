// Plan library authoring with shared live-state checks and no locks held across prompts.

package library

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
)

// GroupPlan identifies a new group in an initialized library and owns no filesystem resources.
type GroupPlan struct {
	id      string
	options Options
}

// RulePlan identifies a new library rule; Commit rechecks that its group still exists.
type RulePlan struct {
	id      string
	options Options
}

// PlanGroup checks the group target before collecting metadata and anchors the library directory.
func PlanGroup(ctx context.Context, id string, options Options) (*GroupPlan, error) {
	options, err := planAuthoring(ctx, options, func(root *os.Root, _ Options) error { return checkNewGroup(ctx, root, id) })
	if err != nil {
		return nil, err
	}
	return &GroupPlan{id, options}, nil
}

// Commit validates metadata and repeats target checks under writer ownership before publishing.
func (p *GroupPlan) Commit(ctx context.Context, metadata rules.GroupMetadata) (AuthoringResult, error) {
	if p == nil || p.id == "" {
		return AuthoringResult{}, failure("invalid-operation", "expected a planned group", nil)
	}
	return AddGroup(ctx, p.id, metadata, p.options)
}

// PlanRule checks the path, parent group, and target before collecting rule details.
func PlanRule(ctx context.Context, id string, options Options) (*RulePlan, error) {
	options, err := planAuthoring(ctx, options, func(root *os.Root, options Options) error { return checkNewRule(ctx, root, id, options) })
	if err != nil {
		return nil, err
	}
	return &RulePlan{id, options}, nil
}

// Commit revalidates live state before creating a complete rule or marked draft when body is nil.
func (p *RulePlan) Commit(ctx context.Context, metadata rules.RuleMetadata, body *string) (AuthoringResult, error) {
	if p == nil || p.id == "" {
		return AuthoringResult{}, failure("invalid-operation", "expected a planned rule", nil)
	}
	return AddRule(ctx, p.id, metadata, RuleOptions{Options: p.options, Body: body})
}

// planAuthoring retains only an absolute location; no root or lock survives input collection.
func planAuthoring(ctx context.Context, options Options, check func(*os.Root, Options) error) (Options, error) {
	directory, err := filepath.Abs(options.Directory)
	if err != nil {
		return Options{}, err
	}
	options.Directory = directory
	root, err := openLibrary(ctx, options, false)
	if err != nil {
		return Options{}, err
	}
	defer root.Close()
	if _, _, err = libraryManifest(ctx, root); err != nil {
		return Options{}, err
	}
	return options, check(root, options)
}

func checkNewGroup(ctx context.Context, root *os.Root, id string) error {
	if err := rules.ValidateGroupID(id, "group"); err != nil {
		return err
	}
	return filetxn.RequireAbsent(ctx, root, id+"/_group.json", id+"/README.md")
}

func checkNewRule(ctx context.Context, root *os.Root, id string, options Options) error {
	if strings.HasSuffix(id, ".md") {
		return failure("invalid-rule-path", "use a rule path without the .md extension", nil)
	}
	group, err := rules.GroupFromPath(id+".md", "rule")
	if err != nil {
		return err
	}
	exists, err := hasGroupMetadata(ctx, root, group)
	if err != nil {
		return err
	}
	if !exists {
		directory, err := filepath.Abs(options.Directory)
		if err != nil {
			return err
		}
		command := "code-rules library add group " + group + " --directory='" + strings.ReplaceAll(directory, "'", "'\"'\"'") + "'"
		return failure("missing-group", "group "+group+" does not exist; create it first with "+command+", then retry adding the rule", nil)
	}
	return filetxn.RequireAbsent(ctx, root, id+".md")
}
