// Test how project list skills and the update preview show skill levels.

package cli

import (
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/project"
)

func TestFormatSkillListing_AlignsLevelsAndWhoSetThem(t *testing.T) {
	commit := strings.Repeat("a", 40)
	listing := project.SkillListing{Skills: []project.ListedSkill{
		{ID: "team:skills/ship", Source: "team", Level: "required", SetBy: &project.LevelSetter{Source: "team", Release: 3, Commit: commit}, Status: project.SkillInstalled},
		{ID: "team:skills/deploy", Source: "team", Level: "offered", SetBy: &project.LevelSetter{Source: "team", Commit: commit}, Status: project.SkillReplaced, ReplacedBy: "skills/deploy"},
		{ID: "tools:tools/review", Source: "tools", Status: project.SkillAvailable},
		{ID: "skills/check-release", Status: project.SkillInstalled},
	}}
	want := `SKILL                 LEVEL     SET BY                     STATUS
team:skills/ship      required  team, release 3            installed
team:skills/deploy    offered   team, commit aaaaaaaaaaaa  replaced by skills/deploy
tools:tools/review    -         tools, not a library       available
skills/check-release  -         this project               installed
`
	if got := formatSkillListing(listing); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if got := formatSkillListing(project.SkillListing{}); got != "The project has no skills, and its sources offer none.\n" {
		t.Fatalf("empty listing: %q", got)
	}
}

func TestFormatUpdatePreview_ShowsSkillLevelChanges(t *testing.T) {
	var out strings.Builder
	formatUpdatePreview(&out, []imports.SourceUpdate{{Name: "team", Rules: []imports.RuleUpdate{}, SharedFiles: &imports.SharedFilesUpdate{From: 1, To: 2}, SkillLevels: []imports.SkillLevelUpdate{
		{ID: "skills/audit", To: "required"},
		{ID: "skills/review", From: "recommended", To: "required"},
		{ID: "skills/ship", From: "required", To: "offered"},
	}}})
	for _, want := range []string{
		"  level     skills/audit   none -> required\n            New required skill: it installs whatever skills.include says.\n",
		"  level     skills/review  recommended -> required\n            Now required: it installs whatever skills.include says, and the project can't exclude or fork it.\n",
		"  level     skills/ship    required -> offered\n            No longer required: it stays installed only when skills.include selects it.\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("preview lacks %q:\n%s", want, out.String())
		}
	}
}
