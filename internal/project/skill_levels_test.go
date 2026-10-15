// Test that a library's skill levels decide which of its skills a project installs, records, lists, and previews.

package project

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/fabricahq/syllabus/internal/imports"
	"github.com/fabricahq/syllabus/internal/test/gitfixture"
)

// leveledLibrary returns a library whose release/1 publishes techs/go/errors and four skills: ship, required;
// review, recommended; deploy, listed as offered; and build-feature, offered because no list names it.
func leveledLibrary(t *testing.T) *gitfixture.Fixture {
	t.Helper()
	ctx := context.Background()
	f, err := gitfixture.New(ctx, gitfixture.Library(map[string][]byte{
		"config.yaml":                             []byte(`{"schemaVersion":2,"sources":{},"library":{"skills":{"required":["skills/ship"],"recommended":["skills/review"],"offered":["skills/deploy"]}}}`),
		"techs/go/_group.yaml":                    []byte(projectMetadata),
		"techs/go/errors.md":                      []byte(projectRule),
		".syllabus/skills/ship/SKILL.md":          []byte("---\nname: ship\ndescription: Ship a service.\n---\n# Ship\n"),
		".syllabus/skills/review/SKILL.md":        []byte(reviewSkill),
		".syllabus/skills/deploy/SKILL.md":        []byte("---\nname: deploy\ndescription: Deploy a preview.\n---\n# Deploy\n"),
		".syllabus/skills/build-feature/SKILL.md": []byte(buildSkill),
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if err := f.Release(ctx, 1, firstRelease); err != nil {
		t.Fatal(err)
	}
	return f
}

// releaseNewLevels publishes release/2 of a leveledLibrary, which makes review required and adds audit, a required
// skill, without changing a rule.
func releaseNewLevels(t *testing.T, f *gitfixture.Fixture) {
	t.Helper()
	ctx := context.Background()
	files := map[string][]byte{
		"config.yaml":                     []byte(`{"schemaVersion":2,"sources":{},"library":{"skills":{"required":["skills/audit","skills/review","skills/ship"],"offered":["skills/deploy"]}}}`),
		".syllabus/skills/audit/SKILL.md": []byte("---\nname: audit\ndescription: Audit a change.\n---\n# Audit\n"),
	}
	if _, err := f.Commit(ctx, f.Worktree(), "Require review and audit", gitfixture.Library(files)); err != nil {
		t.Fatal(err)
	}
	if err := f.Release(ctx, 2, "formatVersion: 1\nrelease: 2\nrules:\n  techs/go/errors: 1.0.0\nchanges: {}\n"); err != nil {
		t.Fatal(err)
	}
}

// installedSkills returns the names of the skills the project installed in .claude/skills/, sorted.
func installedSkills(t *testing.T, options Options) []string {
	t.Helper()
	var record struct {
		Skills []struct{ Name string } `json:"skills"`
	}
	if err := json.Unmarshal([]byte(readRepositoryFile(t, options, ".syllabus/generated/installed-skills.json")), &record); err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, skill := range record.Skills {
		names = append(names, skill.Name)
	}
	slices.Sort(names)
	return names
}

func TestSync_InstallsSkillsByLevel(t *testing.T) {
	f := leveledLibrary(t)
	for name, test := range map[string]struct {
		skills string
		want   []string
	}{
		"nothing included":       {``, []string{"ship"}},
		"rules only":             {`,"rules":{"include":["techs/go/*"]}`, []string{"ship"}},
		"recommended":            {`,"skills":{"include":["recommended"]}`, []string{"review", "ship"}},
		"an offered skill":       {`,"skills":{"include":["skills/build-feature"]}`, []string{"build-feature", "ship"}},
		"a required skill named": {`,"skills":{"include":["skills/ship"]}`, []string{"ship"}},
		"every skill":            {`,"skills":{"include":["*"]}`, []string{"build-feature", "deploy", "review", "ship"}},
	} {
		t.Run(name, func(t *testing.T) {
			options, git := skillProject(t, f, `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"team":{"repository":REPOSITORY`+test.skills+`}}}`)
			if _, err := Sync(context.Background(), options, git); err != nil {
				t.Fatal(err)
			}
			if got := installedSkills(t, options); !slices.Equal(got, test.want) {
				t.Fatalf("installed %v; want %v", got, test.want)
			}
			if !exists(t, options, ".claude/skills/ship/SKILL.md") {
				t.Fatal("the required skill isn't installed")
			}
			if _, err := Build(context.Background(), options); err != nil {
				t.Fatalf("build after sync: %v", err)
			}
		})
	}
}

func TestSync_RecordsWhichLibrarySetEachLevel(t *testing.T) {
	f := leveledLibrary(t)
	options, git := skillProject(t, f, `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"team":{"repository":REPOSITORY,"skills":{"include":["recommended","skills/deploy"]}}}}`)
	if _, err := Sync(context.Background(), options, git); err != nil {
		t.Fatal(err)
	}
	var provenance struct {
		Skills []struct {
			ID                                   string
			RequiredBy, RecommendedBy, OfferedBy *struct {
				Source  string
				Release *int
				Commit  string
			}
		}
	}
	if err := json.Unmarshal([]byte(readRepositoryFile(t, options, ".syllabus/generated/provenance.json")), &provenance); err != nil {
		t.Fatal(err)
	}
	levels := map[string]string{}
	for _, skill := range provenance.Skills {
		for level, by := range map[string]*struct {
			Source  string
			Release *int
			Commit  string
		}{"required": skill.RequiredBy, "recommended": skill.RecommendedBy, "offered": skill.OfferedBy} {
			if by == nil {
				continue
			}
			if _, set := levels[skill.ID]; set || by.Source != "team" || by.Release == nil || *by.Release != 1 || len(by.Commit) != 40 {
				t.Fatalf("%s: %s by %+v", skill.ID, level, by)
			}
			levels[skill.ID] = level
		}
	}
	if want := map[string]string{"team:skills/ship": "required", "team:skills/review": "recommended", "team:skills/deploy": "offered"}; !mapsEqual(levels, want) {
		t.Fatalf("provenance levels = %v; want %v", levels, want)
	}
	index := readRepositoryFile(t, options, ".syllabus/generated/SKILLS.md")
	for _, want := range []string{"**Level:** required by `team`, release 1", "**Level:** recommended by `team`, release 1", "**Level:** offered by `team`, release 1", "every project that uses the library installs its required skills"} {
		if !strings.Contains(index, want) {
			t.Errorf("SKILLS.md lacks %q:\n%s", want, index)
		}
	}
	var record struct {
		AvailableSkills map[string]struct{ Level string } `json:"availableSkills"`
	}
	if err := json.Unmarshal([]byte(readRepositoryFile(t, options, ".syllabus/vendor/team/_source.json")), &record); err != nil {
		t.Fatal(err)
	}
	if len(record.AvailableSkills) != 4 || record.AvailableSkills["skills/build-feature"].Level != "offered" || record.AvailableSkills["skills/ship"].Level != "required" {
		t.Fatalf("availableSkills = %v", record.AvailableSkills)
	}
}

// mapsEqual reports whether two string maps hold the same entries.
func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func TestSyncAndBuild_RefuseToExcludeOrForkARequiredSkill(t *testing.T) {
	const want = "team's library requires skills/ship in release 1, so every project that uses the library installs it, and the project can't exclude or fork it. To diverge from it, stop using sources.team"
	f := leveledLibrary(t)
	excluded := `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"team":{"repository":REPOSITORY,"skills":{"include":["recommended"],"exclude":[{"id":"skills/ship","reason":"We ship by hand."}]}}}}`
	options, git := skillProject(t, f, excluded)
	before := projectTree(t, options)
	_, err := Sync(context.Background(), options, git)
	var validation *imports.ValidationError
	if !errors.As(err, &validation) || validation.Location != "sources.team.skills.exclude.skills/ship" || !strings.Contains(validation.Problem, want) {
		t.Fatalf("sync: got %v; want %q at sources.team.skills.exclude.skills/ship", err, want)
	}
	if after := projectTree(t, options); after.Digest() != before.Digest() {
		t.Fatal("a refused sync wrote files")
	}
	// A configuration that excludes the skill after sync fails offline too.
	configure(t, options, f, `"skills":{"include":["recommended"]}`)
	if _, err := Sync(context.Background(), options, git); err != nil {
		t.Fatal(err)
	}
	configure(t, options, f, `"skills":{"include":["recommended"],"exclude":[{"id":"skills/ship","reason":"We ship by hand."}]}`)
	if _, err := Build(context.Background(), options); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("build: got %v; want %q", err, want)
	}
	configure(t, options, f, `"skills":{"include":["recommended"]}`)
	if _, err := PlanSkillFork(context.Background(), SkillForkSource{Source: "team", Skill: "skills/ship"}, options); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("fork: got %v; want %q", err, want)
	}
	if _, err := PlanSkillFork(context.Background(), SkillForkSource{Source: "team", Skill: "skills/review"}, options); err != nil {
		t.Fatalf("forking a recommended skill: %v", err)
	}
}

func TestSync_RefusesRecommendedFromARepositoryThatIsNotALibrary(t *testing.T) {
	f, err := gitfixture.New(context.Background(), map[string][]byte{"tools/review/SKILL.md": []byte(reviewSkill)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	commit, err := f.Command(context.Background(), "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	ref := quoteJSON(strings.TrimSpace(commit))
	for name, test := range map[string]struct{ fields, location, problem string }{
		"recommended":      {`,"skills":{"include":["recommended"]}`, "sources.tools.skills.include", "so it recommends no skills"},
		"nothing included": {``, "sources.tools", "so it isn't a library and requires no skills"},
	} {
		t.Run(name, func(t *testing.T) {
			options, git := skillProject(t, f, `{"schemaVersion":2,"sources":{"tools":{"repository":REPOSITORY,"ref":`+ref+test.fields+`}}}`)
			_, err := Sync(context.Background(), options, git)
			var validation *imports.ValidationError
			if !errors.As(err, &validation) || validation.Location != test.location || !strings.Contains(validation.Problem, test.problem) {
				t.Fatalf("got %v; want %q at %s", err, test.problem, test.location)
			}
		})
	}
}

func TestListSkills_ShowsEveryInstalledAndAvailableSkillWithItsLevel(t *testing.T) {
	f := leveledLibrary(t)
	options, git := skillProject(t, f, `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"team":{"repository":REPOSITORY,"skills":{"include":["recommended","skills/deploy"],"exclude":[{"id":"skills/deploy","reason":"We deploy by hand."}]}}}}`)
	root, err := openProject(context.Background(), options, false)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "skills/check-release/SKILL.md", "---\nname: check-release\ndescription: Check a release.\n---\n# Check\n")
	root.Close()
	if _, err := Sync(context.Background(), options, git); err != nil {
		t.Fatal(err)
	}
	listing, err := ListSkills(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, skill := range listing.Skills {
		row := skill.ID + " " + skill.Level + " " + skill.Status
		if skill.SetBy != nil {
			row += " " + skill.SetBy.Source
			if skill.SetBy.Release != 1 {
				t.Errorf("%s set by release %d; want 1", skill.ID, skill.SetBy.Release)
			}
		}
		got = append(got, row)
	}
	want := []string{
		"team:skills/build-feature offered available team",
		"team:skills/deploy offered excluded team",
		"team:skills/review recommended installed team",
		"team:skills/ship required installed team",
		"skills/check-release  installed",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("listing:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestUpdate_PreviewsLevelChangesAndInstallsNewRequiredSkills(t *testing.T) {
	ctx := context.Background()
	f := leveledLibrary(t)
	options, git := skillProject(t, f, `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"team":{"repository":REPOSITORY}}}`)
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	releaseNewLevels(t, f)
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	if got := installedSkills(t, options); !slices.Equal(got, []string{"ship"}) {
		t.Fatalf("sync moved the source: installed %v", got)
	}
	plan, err := PlanUpdate(ctx, options, git, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := plan.Preview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []imports.SkillLevelUpdate{{ID: "skills/audit", To: "required"}, {ID: "skills/review", From: "recommended", To: "required"}}
	if got := preview.Sources[0].SkillLevels; !slices.Equal(got, want) {
		t.Fatalf("level rows = %+v; want %+v", got, want)
	}
	if !preview.Moves() {
		t.Fatal("the preview has nothing to apply")
	}
	if _, err := plan.Apply(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if got := installedSkills(t, options); !slices.Equal(got, []string{"audit", "review", "ship"}) {
		t.Fatalf("after update, installed %v; want the new required skills too", got)
	}
}
