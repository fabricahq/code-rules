// Verify plans hold no locks and revalidate targets and groups after input collection.

package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

func TestAuthoringPlansRevalidateLiveState(t *testing.T) {
	ctx := context.Background()
	options := Options{Directory: t.TempDir()}
	if _, err := Initialize(ctx, options); err != nil {
		t.Fatal(err)
	}
	metadata := rules.GroupMetadata{Name: "Go", Description: "Go guidance.", WhenToRead: "When editing Go."}
	group, err := PlanLocalGroup(ctx, "techs/go", options)
	if err != nil {
		t.Fatal(err)
	}
	// A competing writer succeeds while this plan is pending, then both checks report the same conflict.
	if _, err := AddLocalGroup(ctx, "techs/go", metadata, options); err != nil {
		t.Fatal(err)
	}
	_, commitErr := group.Commit(ctx, metadata)
	_, planErr := PlanLocalGroup(ctx, "techs/go", options)
	if commitErr == nil || planErr == nil || commitErr.Error() != planErr.Error() {
		t.Fatal(commitErr, planErr)
	}

	rule, err := PlanLocalRule(ctx, "techs/go/errors", options)
	if err != nil {
		t.Fatal(err)
	}
	ruleMetadata := rules.RuleMetadata{Title: "Return errors", Impact: "HIGH", ImpactDescription: "Preserve failures.", WhenToRead: "When calling functions."}
	if err := os.Remove(filepath.Join(options.Directory, ".code-rules/local/techs/go/_group.yaml")); err != nil {
		t.Fatal(err)
	}
	_, commitErr = rule.Commit(ctx, ruleMetadata, nil)
	_, planErr = PlanLocalRule(ctx, "techs/go/errors", options)
	if commitErr == nil || planErr == nil || commitErr.Error() != planErr.Error() {
		t.Fatal(commitErr, planErr)
	}
	if _, err := os.Stat(filepath.Join(options.Directory, ".code-rules/local/techs/go/errors.md")); !os.IsNotExist(err) {
		t.Fatal("wrote rule after group disappeared", err)
	}

	source, err := PlanSource(ctx, "team", options)
	if err != nil {
		t.Fatal(err)
	}
	declaration := SourceInput{Repository: "https://example.invalid/rules.git", Ref: "v1.0.0", Groups: []string{"*"}}
	if _, err := AddSource(ctx, "team", declaration, options); err != nil {
		t.Fatal(err)
	}
	_, commitErr = source.Commit(ctx, declaration)
	_, planErr = PlanSource(ctx, "team", options)
	if commitErr == nil || planErr == nil || commitErr.Error() != planErr.Error() {
		t.Fatal(commitErr, planErr)
	}
}

func TestRulePlanPreservesCompetingFile(t *testing.T) {
	ctx := context.Background()
	options := Options{Directory: t.TempDir()}
	if _, err := Initialize(ctx, options); err != nil {
		t.Fatal(err)
	}
	if _, err := AddLocalGroup(ctx, "techs/go", rules.GroupMetadata{Name: "Go", Description: "Go guidance.", WhenToRead: "When editing Go."}, options); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanLocalRule(ctx, "techs/go/errors", options)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(options.Directory, ".code-rules/local/techs/go/errors.md")
	if err := os.WriteFile(name, []byte("Concurrent author's content"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = plan.Commit(ctx, rules.RuleMetadata{Title: "Return errors", Impact: "HIGH", ImpactDescription: "Preserve failures.", WhenToRead: "When calling functions."}, nil)
	if err == nil {
		t.Fatal("replaced competing rule")
	}
	data, err := os.ReadFile(name)
	if err != nil || string(data) != "Concurrent author's content" {
		t.Fatal(string(data), err)
	}
}
