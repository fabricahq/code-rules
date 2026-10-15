// Test the configuration fields that choose skills and where installed copies go.

package rules_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/errs"
	"github.com/fabricahq/code-rules/internal/rules"
)

func TestParseConfigurationReadsSkillTargetsAndSkillSelectors(t *testing.T) {
	config, err := rules.ParseConfigurationYAML([]byte(`schemaVersion: 2
skillTargets: [agents, claude]
sources:
  pstack:
    repository: https://example.invalid/pstack.git
    ref: v1.0.0
    skills:
      include: [skills/review, skills/engineering/to-spec]
  team:
    repository: https://example.invalid/team.git
    skills:
      include: ["*"]
`))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(config.SkillTargets, []string{"agents", "claude"}) || !reflect.DeepEqual(config.SkillFolders(), []string{".agents/skills", ".claude/skills"}) {
		t.Fatalf("skill targets = %v, folders %v", config.SkillTargets, config.SkillFolders())
	}
	if got := config.Sources[0]; !reflect.DeepEqual(got.Skills, []string{"skills/engineering/to-spec", "skills/review"}) || got.HasRules() {
		t.Fatalf("pstack = %+v", got)
	}
	if got := config.Sources[1]; !reflect.DeepEqual(got.Skills, []string{"*"}) || got.HasRules() || got.Pins == nil || got.Exclude == nil || got.Rules == nil {
		t.Fatalf("team = %+v", got)
	}
}

func TestParseConfigurationReadsSkillRenames(t *testing.T) {
	config, err := rules.ParseConfigurationYAML([]byte(`schemaVersion: 2
sources:
  team:
    repository: https://example.invalid/team.git
    ref: main
    skills:
      include: ["*"]
      rename:
        skills/review: team-review
        skills/engineering/to-spec: spec
  other:
    repository: https://example.invalid/other.git
    ref: main
    skills:
      include: ["*"]
`))
	if err != nil {
		t.Fatal(err)
	}
	if got := config.Sources[1].SkillRename; !reflect.DeepEqual(got, map[string]string{"skills/review": "team-review", "skills/engineering/to-spec": "spec"}) {
		t.Fatalf("team renames = %v", got)
	}
	if got := config.Sources[0].SkillRename; got != nil {
		t.Fatalf("other renames = %v, want none", got)
	}
}

func TestParseConfigurationTreatsMissingSkillTargetsAsNone(t *testing.T) {
	for name, input := range map[string]string{
		"missing": "schemaVersion: 2\nsources: {}\n",
		"empty":   "schemaVersion: 2\nskillTargets: []\nsources: {}\n",
	} {
		t.Run(name, func(t *testing.T) {
			config, err := rules.ParseConfigurationYAML([]byte(input))
			if err != nil || config.SkillTargets == nil || len(config.SkillTargets) != 0 || len(config.SkillFolders()) != 0 {
				t.Fatalf("config = %+v, %v", config, err)
			}
		})
	}
}

func TestParseConfigurationRejectsInvalidSkillFields(t *testing.T) {
	const source = "schemaVersion: 2\nsources:\n  team:\n    repository: https://example.invalid/team.git\n"
	tests := map[string]struct{ input, location, problem string }{
		"unknown target":     {"schemaVersion: 2\nskillTargets: [cursor]\nsources: {}\n", "skillTargets[0]", "expected claude"},
		"repeated target":    {"schemaVersion: 2\nskillTargets: [claude, claude]\nsources: {}\n", "skillTargets[1]", "more than once"},
		"targets not a list": {"schemaVersion: 2\nskillTargets: claude\nsources: {}\n", "skillTargets", "expected a list"},
		"nothing included":   {source, "sources.team", "include rules or skills"},
		"empty include":      {source + "    skills:\n      include: []\n", "sources.team.skills.include", "select skills"},
		"missing include":    {source + "    skills: {}\n", "sources.team.skills.include", "select skills"},
		"unknown field":      {source + "    skills:\n      include: [skills/x]\n      pins: {}\n", "sources.team.skills", "pins"},
		"rename not a map":   {source + "    skills:\n      include: [\"*\"]\n      rename: [skills/x]\n", "sources.team.skills.rename", "expected a mapping"},
		"rename wildcard":    {source + "    skills:\n      include: [\"*\"]\n      rename: {\"*\": x}\n", "sources.team.skills.rename.*", "names one skill"},
		"rename invalid ID":  {source + "    skills:\n      include: [\"*\"]\n      rename: {skills/../x: y}\n", "sources.team.skills.rename.skills/../x", "expected a skill ID"},
		"rename bad name":    {source + "    skills:\n      include: [\"*\"]\n      rename: {skills/x: Team_Review}\n", "sources.team.skills.rename.skills/x", "name grammar"},
		"rename excluded":    {source + "    skills:\n      include: [\"*\"]\n      exclude: [{id: skills/x, reason: No.}]\n      rename: {skills/x: y}\n", "sources.team.skills.rename.skills/x", "excludes this skill"},
		"levels":             {source + "    skills:\n      include: [recommended]\n", "sources.team.skills.include[0]", "skill levels aren't supported yet"},
		"prefix":             {source + "    skills:\n      include: [skills/*]\n", "sources.team.skills.include[0]", "no prefixes"},
		"wildcard and IDs":   {source + "    skills:\n      include: [\"*\", skills/x]\n", "sources.team.skills.include", "already includes every skill"},
		"repeated ID":        {source + "    skills:\n      include: [skills/x, skills/x]\n", "sources.team.skills.include[1]", "more than once"},
		"invalid ID":         {source + "    skills:\n      include: [skills/../x]\n", "sources.team.skills.include[0]", "expected a skill ID"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := rules.ParseConfigurationYAML([]byte(test.input))
			var validation errs.ValidationError
			if !errors.As(err, &validation) || validation.ValidationLocation() != test.location || !strings.Contains(validation.ValidationProblem(), test.problem) {
				t.Fatalf("got %v; want %q at %s", err, test.problem, test.location)
			}
		})
	}
}

func TestAppendConfigurationSourceWritesSkillSelectors(t *testing.T) {
	out, err := rules.AppendConfigurationSource([]byte("schemaVersion: 2\nsources: {}\n"), "pstack", rules.Source{Repository: "https://example.invalid/pstack.git", Ref: mustRef(t, "v1.0.0"), Skills: []string{"*"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "rules:") || !strings.Contains(string(out), "    skills:\n      include:\n        - '*'\n") {
		t.Fatalf("configuration:\n%s", out)
	}
	config, err := rules.ParseConfigurationYAML(out)
	if err != nil || !reflect.DeepEqual(config.Sources[0].Skills, []string{"*"}) {
		t.Fatalf("parsed %+v, %v", config, err)
	}
}

func TestSetConfigurationSkillTargetsKeepsOtherFields(t *testing.T) {
	input := "# Team configuration.\nschemaVersion: 2\nsources: {} # none yet\n"
	added, err := rules.SetConfigurationSkillTargets([]byte(input), []string{"claude", "agents"})
	if err != nil {
		t.Fatal(err)
	}
	if want := "# Team configuration.\nschemaVersion: 2\nskillTargets: [claude, agents]\nsources: {} # none yet\n"; string(added) != want {
		t.Fatalf("added:\n%s\nwant:\n%s", added, want)
	}
	replaced, err := rules.SetConfigurationSkillTargets(added, []string{})
	if err != nil {
		t.Fatal(err)
	}
	if want := "# Team configuration.\nschemaVersion: 2\nskillTargets: []\nsources: {} # none yet\n"; string(replaced) != want {
		t.Fatalf("replaced:\n%s\nwant:\n%s", replaced, want)
	}
}

func TestEditConfigurationSourceRefusesRuleEditsWithoutRules(t *testing.T) {
	input := []byte("schemaVersion: 2\nsources:\n  team:\n    repository: https://example.invalid/team.git\n    skills:\n      include: [skills/x]\n")
	_, err := rules.EditConfigurationSource(input, "team", rules.SourceEdit{Exclude: map[string]rules.Exclusion{"techs/go/x": {Reason: "No."}}})
	var validation errs.ValidationError
	if !errors.As(err, &validation) || validation.ValidationLocation() != "sources.team.rules" {
		t.Fatalf("got %v", err)
	}
}

// mustRef parses text as a source ref.
func mustRef(t *testing.T, text string) rules.GitRef {
	t.Helper()
	ref, err := rules.ParseGitRef(text, "ref")
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
