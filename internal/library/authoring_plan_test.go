// Check library plan/commit parity when another writer changes authoring targets.

package library

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

func TestAuthoringPlansRevalidateLiveState(t *testing.T) {
	ctx := context.Background()
	options := Options{Directory: filepath.Join(t.TempDir(), "team's library")}
	if _, err := Initialize(ctx, options, nil); err != nil {
		t.Fatal(err)
	}
	metadata := rules.GroupMetadata{Name: "Go", Description: "Go guidance.", WhenToRead: "When editing Go."}
	group, err := PlanGroup(ctx, "techs/go", options)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AddGroup(ctx, "techs/go", metadata, options); err != nil {
		t.Fatal(err)
	}
	_, commitErr := group.Commit(ctx, metadata)
	_, planErr := PlanGroup(ctx, "techs/go", options)
	if commitErr == nil || planErr == nil || commitErr.Error() != planErr.Error() {
		t.Fatal(commitErr, planErr)
	}
	plan, err := PlanRule(ctx, "techs/go/errors", options)
	if err != nil {
		t.Fatal(err)
	}
	rule := rules.RuleMetadata{Title: "Return errors", Impact: "HIGH", ImpactDescription: "Preserve failures.", WhenToRead: "When calling functions."}
	if err := os.Remove(filepath.Join(options.Directory, "techs/go/_group.json")); err != nil {
		t.Fatal(err)
	}
	_, commitErr = plan.Commit(ctx, rule, nil)
	_, planErr = PlanRule(ctx, "techs/go/errors", options)
	if commitErr == nil || planErr == nil || commitErr.Error() != planErr.Error() {
		t.Fatal(commitErr, planErr)
	}
	quoted := "--directory='" + strings.ReplaceAll(options.Directory, "'", "'\"'\"'") + "'"
	if !strings.Contains(commitErr.Error(), quoted) {
		t.Fatal("repair lost library location", commitErr)
	}
	if _, err := os.Stat(filepath.Join(options.Directory, "techs/go/errors.md")); !os.IsNotExist(err) {
		t.Fatal("wrote rule after group disappeared", err)
	}
}
