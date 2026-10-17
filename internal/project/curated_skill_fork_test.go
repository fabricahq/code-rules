// Test that a project can fork a skill its library curates, and that an update lists what the curated repository
// changed in the skill since the commit the fork was copied from.

package project

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/fabricahq/trailguide/internal/imports"
)

func TestPlanSkillFork_ForksACuratedSkillAndUpdateListsItsSourcesChanges(t *testing.T) {
	ctx := context.Background()
	b := newCuratingFrom(t, "    recommended: [pocock:skills/ops/deploy]\n", `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"team":{"repository":TEAM,"rules":{"include":["techs/go/*"]},"skills":{"include":["pocock:skills/ops/deploy"]}}}}`)
	if _, err := Sync(ctx, b.options, b.git); err != nil {
		t.Fatal(err)
	}
	forkSkill(t, b.options, "team:pocock:skills/ops/deploy", "We deploy with our pipeline.")
	config := readRepositoryFile(t, b.options, ".trailguide/config.yaml")
	for _, want := range []string{"pocock:skills/ops/deploy", "replacedBy: skills/deploy", b.pocock.FirstCommit} {
		if !strings.Contains(config, want) {
			t.Fatalf("config lacks %q:\n%s", want, config)
		}
	}
	if _, err := Build(ctx, b.options); err != nil {
		t.Fatal(err)
	}
	if installed := readRepositoryFile(t, b.options, ".claude/skills/deploy/SKILL.md"); strings.Contains(installed, "team:pocock:skills/ops/deploy") {
		t.Fatalf("installed the curated skill, not the fork:\n%s", installed)
	}
	if result, err := Check(ctx, b.options); err != nil || !result.Current() {
		t.Fatalf("check = %+v, %v", result, err)
	}

	moved, err := b.pocock.Commit(ctx, b.pocock.Worktree(), "Change deploy", map[string][]byte{"skills/ops/deploy/scripts/deploy.sh": []byte("#!/bin/sh\necho deploy safely\n")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.pocock.Command(ctx, "tag", "--force", "stable", moved); err != nil {
		t.Fatal(err)
	}
	_, preview := b.previewUpdate(t, imports.UpdateTarget{Source: "team"})
	curated := preview.Sources[0].Curated
	if len(curated) != 1 || len(curated[0].Changes) != 0 || len(curated[0].Replaced) != 1 {
		t.Fatalf("curated rows = %+v; want only the replaced deploy", curated)
	}
	replaced := curated[0].Replaced[0]
	if replaced.ID != "pocock:skills/ops/deploy" || replaced.LocalSkill != "skills/deploy" || replaced.BasedOn != b.pocock.FirstCommit || replaced.To != moved || !slices.Equal(replaced.Files, []string{"scripts/deploy.sh"}) {
		t.Fatalf("replaced row = %+v", replaced)
	}
}
