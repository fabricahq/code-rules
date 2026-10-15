// Test that Check reports every problem in an authored skill's name and Code Rules metadata keys.

package skills_test

import (
	"reflect"
	"testing"

	"github.com/fabricahq/code-rules/internal/skills"
)

func TestCheckReportsEachProblemInAnAuthoredSkillsFrontmatter(t *testing.T) {
	valid := skills.Metadata{Name: "fix-bug", Frontmatter: true}
	with := func(change func(*skills.Metadata)) skills.Metadata {
		metadata := valid
		change(&metadata)
		return metadata
	}
	tests := map[string]struct {
		metadata  skills.Metadata
		qualified bool
		want      []string
	}{
		"valid without metadata": {valid, false, []string{}},
		"each type":              {with(func(m *skills.Metadata) { m.Type = "review" }), false, []string{}},
		"playbook":               {with(func(m *skills.Metadata) { m.Type = "playbook" }), false, []string{}},
		"citations of a library skill": {with(func(m *skills.Metadata) {
			m.RulesUsed, m.SkillsUsed = []string{"practices/testing/verify-retries", "techs/go/errors"}, []string{"skills/tdd"}
		}), false, []string{}},
		"citations of a project skill": {with(func(m *skills.Metadata) {
			m.RulesUsed, m.SkillsUsed = []string{"practices/release/tag-from-main", "fabrica:practices/testing/verify-retries"}, []string{"skills/tdd", "fabrica:skills/engineering/to-spec"}
		}), true, []string{}},
		"no frontmatter":           {skills.Metadata{}, false, []string{"expected YAML frontmatter at the start of the file"}},
		"missing name":             {with(func(m *skills.Metadata) { m.Name = "" }), false, []string{"name is required, and must be the folder's name, fix-bug"}},
		"name isn't folder":        {with(func(m *skills.Metadata) { m.Name = "fix-bugs" }), false, []string{`name is "fix-bugs", but the folder is fix-bug; they must match`}},
		"unknown type":             {with(func(m *skills.Metadata) { m.Type = "design" }), false, []string{`syllabus-skill-type is "design"; use playbook or review, or leave it out`}},
		"type spelled in capitals": {with(func(m *skills.Metadata) { m.Type = "Review" }), false, []string{`syllabus-skill-type is "Review"; use playbook or review, or leave it out`}},
		"reserved key":             {with(func(m *skills.Metadata) { m.ReservedKeys = []string{"syllabus-groups"} }), false, []string{`metadata."syllabus-groups" isn't a Code Rules key; use syllabus-skill-type, syllabus-rules-used, or syllabus-skills-used`}},
		"qualified rule in a library": {with(func(m *skills.Metadata) { m.RulesUsed = []string{"fabrica:techs/go/errors"} }), false, []string{
			`syllabus-rules-used names "fabrica:techs/go/errors"; cite a rule by its ID in this library, without a source name, such as practices/testing/verify-retries`,
		}},
		"rule file name": {with(func(m *skills.Metadata) { m.RulesUsed = []string{"techs/go/errors.md"} }), true, []string{
			`syllabus-rules-used names "techs/go/errors.md"; cite a rule by its ID, without .md, such as techs/go/errors`,
		}},
		"group instead of rule": {with(func(m *skills.Metadata) { m.RulesUsed = []string{"techs/go"} }), true, []string{
			`syllabus-rules-used names "techs/go"; expected a rule ID, a rule's path in rules/ without .md, such as practices/testing/verify-retries`,
		}},
		"malformed source name": {with(func(m *skills.Metadata) { m.RulesUsed = []string{"Fabrica:techs/go/errors"} }), true, []string{
			`syllabus-rules-used names "Fabrica:techs/go/errors"; expected a source name, a colon, and a rule ID, such as fabrica:practices/testing/verify-retries`,
		}},
		"repeated rule":     {with(func(m *skills.Metadata) { m.RulesUsed = []string{"techs/go/errors", "techs/go/errors"} }), false, []string{"syllabus-rules-used names techs/go/errors twice"}},
		"skill uses itself": {with(func(m *skills.Metadata) { m.SkillsUsed = []string{"skills/fix-bug"} }), false, []string{"syllabus-skills-used names the skill itself"}},
		"skill by name only": {with(func(m *skills.Metadata) { m.SkillsUsed = []string{"tdd"} }), false, []string{
			`syllabus-skills-used names "tdd"; expected skills/ and a skill's name, such as skills/tdd`,
		}},
		"qualified skill in a library": {with(func(m *skills.Metadata) { m.SkillsUsed = []string{"fabrica:skills/tdd"} }), false, []string{
			`syllabus-skills-used names "fabrica:skills/tdd"; name a skill in this library by its ID, without a source name, such as skills/tdd`,
		}},
		"qualified skill without an ID": {with(func(m *skills.Metadata) { m.SkillsUsed = []string{"fabrica:"} }), true, []string{
			`syllabus-skills-used names "fabrica:"; expected a source name, a colon, and the skill's ID in that source, such as fabrica:skills/tdd`,
		}},
		"repeated skill": {with(func(m *skills.Metadata) { m.SkillsUsed = []string{"skills/tdd", "skills/tdd"} }), false, []string{"syllabus-skills-used names skills/tdd twice"}},
		"several problems in order": {with(func(m *skills.Metadata) {
			m.Name, m.Type, m.RulesUsed, m.SkillsUsed = "other", "design", []string{"techs/go"}, []string{"skills/fix-bug"}
		}), false, []string{
			`name is "other", but the folder is fix-bug; they must match`,
			`syllabus-skill-type is "design"; use playbook or review, or leave it out`,
			`syllabus-rules-used names "techs/go"; expected a rule ID, a rule's path in rules/ without .md, such as practices/testing/verify-retries`,
			"syllabus-skills-used names the skill itself",
		}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := skills.Check("fix-bug", test.metadata, test.qualified); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Check =\n%q\nwant\n%q", got, test.want)
			}
		})
	}
}
