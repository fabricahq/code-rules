// Verify complete project status through its caller-facing operation, without CLI orchestration.

package project_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/project"
)

func TestCheckOwnsGuideAndGeneratedFreshness(t *testing.T) {
	for _, location := range []struct{ config, guide string }{
		{".code-rules/config.json", "README.md"},
		{"custom/project.json", "CODE_RULES.md"},
	} {
		for _, state := range []string{"current", "missing-guide", "edited-guide", "both-stale"} {
			t.Run(location.config+"/"+state, func(t *testing.T) {
				ctx := context.Background()
				options := project.Options{ConfigPath: filepath.Join(t.TempDir(), location.config)}
				if _, err := project.Initialize(ctx, options); err != nil {
					t.Fatal(err)
				}
				if _, err := project.Build(ctx, options); err != nil {
					t.Fatal(err)
				}
				root, err := os.OpenRoot(filepath.Dir(options.ConfigPath))
				if err != nil {
					t.Fatal(err)
				}
				defer root.Close()
				switch state {
				case "missing-guide":
					err = root.Remove(location.guide)
				case "edited-guide", "both-stale":
					err = root.WriteFile(location.guide, []byte("User notes.\n"), 0600)
				}
				if err != nil {
					t.Fatal(err)
				}
				if state == "both-stale" {
					if err := root.WriteFile("generated/RULES.md", []byte("Stale output.\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				before, err := filetxn.ReadTree(ctx, root, ".")
				if err != nil {
					t.Fatal(err)
				}
				report, err := project.Check(ctx, options)
				if err != nil {
					t.Fatal(err)
				}
				want := []project.Problem{}
				if state == "both-stale" {
					want = append(want, project.Problem{Kind: project.StaleContents, Path: "generated/RULES.md", Repair: project.Rebuild})
				}
				if state != "current" {
					want = append(want, project.Problem{Kind: project.OutdatedGuide, Path: location.guide, Repair: project.RefreshGuide})
				}
				if !reflect.DeepEqual(report.Problems, want) || report.Current() != (state == "current") {
					t.Fatalf("got %+v; want %+v", report, want)
				}
				after, err := filetxn.ReadTree(ctx, root, ".")
				if err != nil || before.Digest() != after.Digest() {
					t.Fatal("check modified project state", err)
				}
			})
		}
	}
}
