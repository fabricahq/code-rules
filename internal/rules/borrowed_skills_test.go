// Test that a library lists borrowed skills only from the sources its configuration declares.

package rules_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/fabricahq/trailguide/internal/errs"
	"github.com/fabricahq/trailguide/internal/rules"
	"github.com/fabricahq/trailguide/internal/skills"
)

func TestParseConfiguration_ReadsBorrowedSkillsAndTheirSources(t *testing.T) {
	config, err := rules.ParseConfigurationYAML([]byte("schemaVersion: 2\nsources:\n  pocock:\n    repository: https://github.com/mattpocock/skills.git\n    ref: main-2026\nlibrary:\n  skills:\n    required: [pocock:skills/ops/deploy]\n    recommended: [pocock:skills/engineering/to-spec, skills/build-feature]\n"))
	if err != nil {
		t.Fatal(err)
	}
	listed := config.Library.Skills
	if listed.Level("pocock:skills/ops/deploy") != skills.LevelRequired || listed.Level("skills/build-feature") != skills.LevelRecommended {
		t.Fatalf("levels = %+v", listed)
	}
	if got := listed.Borrowed(); !slices.Equal(got, []string{"pocock:skills/engineering/to-spec", "pocock:skills/ops/deploy"}) {
		t.Fatalf("borrowed = %v", got)
	}
	if source := config.Library.Borrowed["pocock"]; source.Repository != "https://github.com/mattpocock/skills.git" || source.Ref.String() != "main-2026" {
		t.Fatalf("borrowed source = %+v", source)
	}
	if requires := config.Sources[0].LibraryRequires; !slices.Equal(requires, []string{"skills/ops/deploy"}) {
		t.Fatalf("the library repository's own source requires %v; want the required borrowed skill", requires)
	}
}

func TestParseConfiguration_RejectsInvalidBorrowedSkills(t *testing.T) {
	for name, test := range map[string]struct {
		config, location, problem string
	}{
		"undeclared source": {
			config:   "schemaVersion: 2\nsources: {}\nlibrary:\n  skills:\n    offered: [pocock:skills/to-spec]\n",
			location: "library.skills.offered[0]",
			problem:  "pocock:skills/to-spec borrows from pocock, which isn't a source in sources; add sources.pocock with the repository the skill comes from, and a ref unless the repository publishes library releases",
		},
		"listed twice": {
			config:   "schemaVersion: 2\nsources:\n  pocock: {repository: https://github.com/mattpocock/skills.git, ref: v1}\nlibrary:\n  skills:\n    required: [pocock:skills/to-spec]\n    offered: [pocock:skills/to-spec]\n",
			location: "library.skills.offered[0]",
			problem:  "pocock:skills/to-spec is already required; give each skill one level",
		},
		"invalid source name": {
			config:   "schemaVersion: 2\nsources: {}\nlibrary:\n  skills:\n    offered: [Pocock:skills/to-spec]\n",
			location: "library.skills.offered[0]",
			problem:  `expected a borrowed skill as the name of a source in sources, a colon, and the skill's ID in that source's repository, such as mattpocock:skills/to-spec, got "Pocock:skills/to-spec"`,
		},
		"replacing a borrowed skill": {
			config:   "schemaVersion: 2\nsources:\n  team:\n    repository: https://github.com/acme/team.git\n    skills:\n      include: [pocock:skills/to-spec]\n      exclude:\n        - id: pocock:skills/to-spec\n          reason: Ours is stricter.\n          replacedBy: skills/to-spec\n",
			location: "sources.team.skills.exclude.pocock:skills/to-spec.replacedBy",
			problem:  "a borrowed skill can be excluded but not replaced, since its revision isn't the library's; to fork it, declare the repository it comes from as a source of your own and fork the skill from that source",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := rules.ParseConfigurationYAML([]byte(test.config))
			var validation errs.ValidationError
			if !errors.As(err, &validation) || validation.ValidationLocation() != test.location || validation.ValidationProblem() != test.problem {
				t.Fatalf("got %v; want %s: %s", err, test.location, test.problem)
			}
		})
	}
}
