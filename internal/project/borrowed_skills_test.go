// Test that a project installs the skills its library borrows from other repositories, from those repositories, at
// revisions the project resolves and records itself.

package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/syllabus/internal/errs"
	"github.com/fabricahq/syllabus/internal/imports"
	"github.com/fabricahq/syllabus/internal/test/gitfixture"
)

const (
	grillSkill  = "---\nname: grilling\ndescription: Ask hard questions about a plan.\n---\n# Grilling\n"
	deploySkill = "---\nname: deploy\ndescription: Deploy a service.\n---\n# Deploy\n\nRun scripts/deploy.sh.\n"
)

// borrowing is a project whose library, team, borrows skills from pocock, a repository that isn't a library, at its
// tag stable: deploy is required, to-spec recommended, and grilling offered. to-spec uses interview, which team
// doesn't list.
type borrowing struct {
	team, pocock *gitfixture.Fixture
	options      Options
	git          imports.Options
	root         *os.Root
}

// newBorrowing returns a borrowing whose project uses team with config, JSON in which TEAM and POCOCK stand for the
// two repositories, and has run no command yet.
func newBorrowing(t *testing.T, config string) *borrowing {
	t.Helper()
	ctx := context.Background()
	pocock, err := gitfixture.New(ctx, map[string][]byte{
		"skills/engineering/to-spec/SKILL.md":   usingSkill("to-spec", "skills/engineering/interview"),
		"skills/engineering/interview/SKILL.md": usingSkill("interview"),
		"skills/productivity/grilling/SKILL.md": []byte(grillSkill),
		"skills/ops/deploy/SKILL.md":            []byte(deploySkill),
		"skills/ops/deploy/scripts/deploy.sh":   []byte("#!/bin/sh\necho deploy\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pocock.Close() })
	if _, err := pocock.Command(ctx, "tag", "stable", pocock.FirstCommit); err != nil {
		t.Fatal(err)
	}
	team, err := gitfixture.New(ctx, gitfixture.Library(map[string][]byte{
		"config.yaml":          []byte("schemaVersion: 2\nsources:\n  pocock:\n    repository: git@fixture.invalid:pocock\n    ref: stable\nlibrary:\n  skills:\n    required: [pocock:skills/ops/deploy]\n    recommended: [pocock:skills/engineering/to-spec]\n    offered: [pocock:skills/productivity/grilling]\n"),
		"techs/go/_group.yaml": []byte(projectMetadata),
		"techs/go/errors.md":   []byte(projectRule),
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = team.Close() })
	if err := team.Release(ctx, 1, firstRelease); err != nil {
		t.Fatal(err)
	}
	environment, err := team.Route(map[string]*gitfixture.Fixture{"team": team, "pocock": pocock})
	if err != nil {
		t.Fatal(err)
	}
	b := &borrowing{team: team, pocock: pocock, git: imports.Options{GitPath: team.GitPath, Environment: environment}}
	b.root = openTestProject(t)
	b.options = Options{Directory: filepath.Dir(b.root.Name()), ToolVersion: "1.2.3"}
	if _, err := Initialize(ctx, b.options); err != nil {
		t.Fatal(err)
	}
	b.configure(t, config)
	return b
}

// configure writes the project's configuration, config, as newBorrowing takes it.
func (b *borrowing) configure(t *testing.T, config string) {
	t.Helper()
	config = strings.ReplaceAll(strings.ReplaceAll(config, "TEAM", quoteJSON("git@fixture.invalid:team")), "POCOCK", quoteJSON("git@fixture.invalid:pocock"))
	writeFixture(t, b.root, configurationFile, config)
}

// teamRecord returns team's source record, parsed.
func (b *borrowing) teamRecord(t *testing.T) map[string]any {
	t.Helper()
	var record map[string]any
	if err := json.Unmarshal([]byte(readRepositoryFile(t, b.options, ".syllabus/vendor/team/_source.json")), &record); err != nil {
		t.Fatal(err)
	}
	return record
}

func TestSync_InstallsBorrowedSkillsFromTheRepositoryTheyComeFrom(t *testing.T) {
	ctx := context.Background()
	b := newBorrowing(t, `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"team":{"repository":TEAM,"rules":{"include":["techs/go/*"]},"skills":{"include":["pocock:skills/productivity/grilling"]}}}}`)
	if _, err := Sync(ctx, b.options, b.git); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"deploy", "grilling"} {
		installed := readRepositoryFile(t, b.options, ".claude/skills/"+name+"/SKILL.md")
		if want := "<!-- syllabus:installed team:pocock:skills/"; !strings.Contains(installed, want) || !strings.Contains(installed, b.pocock.FirstCommit) {
			t.Errorf("%s:\n%s\nwant the borrowed skill's project ID and the commit its tag names", name, installed)
		}
	}
	if exists(t, b.options, ".claude/skills/to-spec") {
		t.Error("installed a recommended skill the project doesn't include")
	}
	if info, err := os.Stat(filepath.Join(b.options.Directory, ".claude/skills/deploy/scripts/deploy.sh")); err != nil || info.Mode().Perm()&0o100 != 0 {
		t.Errorf("deploy.sh: %v, %v; the fixture commits it without its executable bit", info, err)
	}
	if !exists(t, b.options, ".syllabus/vendor/team/borrowed/pocock/skills/ops/deploy/SKILL.md") {
		t.Error("didn't vendor the borrowed skill under borrowed/<source>/")
	}
	borrowed, _ := b.teamRecord(t)["borrowed"].(map[string]any)
	pocock, _ := borrowed["pocock"].(map[string]any)
	if pocock["resolvedCommit"] != b.pocock.FirstCommit || pocock["ref"] != "stable" || pocock["kind"] != "non-library" || pocock["repository"] != "git@fixture.invalid:pocock" {
		t.Fatalf("borrowed record: %v", borrowed)
	}
	skillsIndex := readRepositoryFile(t, b.options, ".syllabus/generated/SKILLS.md")
	for _, want := range []string{"`team:pocock:skills/ops/deploy`, borrowed by `team` from git@fixture.invalid:pocock, commit `" + b.pocock.FirstCommit + "`", "**Level:** required by `team`, release 1", "**Level:** offered by `team`, release 1"} {
		if !strings.Contains(skillsIndex, want) {
			t.Errorf("SKILLS.md lacks %q:\n%s", want, skillsIndex)
		}
	}
	provenance := readRepositoryFile(t, b.options, ".syllabus/generated/provenance.json")
	if !strings.Contains(provenance, `"borrowed": "pocock"`) || !strings.Contains(provenance, `"folder": "skills/ops/deploy"`) {
		t.Errorf("provenance doesn't name the borrowed source and the skill's folder there:\n%s", provenance)
	}
	if result, err := Check(ctx, b.options); err != nil || !result.Current() {
		t.Fatalf("check = %+v, %v", result, err)
	}
}

func TestSync_IncludesRecommendedBorrowedSkillsWithTheSkillsTheyUse(t *testing.T) {
	ctx := context.Background()
	b := newBorrowing(t, `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"team":{"repository":TEAM,"skills":{"include":["recommended"]}}}}`)
	if _, err := Sync(ctx, b.options, b.git); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"deploy", "to-spec", "interview"} {
		if !exists(t, b.options, ".claude/skills/"+name+"/SKILL.md") {
			t.Errorf("didn't install %s", name)
		}
	}
	if exists(t, b.options, ".claude/skills/grilling") {
		t.Error("installed an offered skill the project doesn't include")
	}
	skills, _ := b.teamRecord(t)["skills"].(map[string]any)
	interview, _ := skills["pocock:skills/engineering/interview"].(map[string]any)
	if interview["level"] != "offered" || interview["included"] != false || len(interview["usedBy"].([]any)) != 1 || interview["usedBy"].([]any)[0] != "pocock:skills/engineering/to-spec" {
		t.Fatalf("interview's record = %v; want a skill installed because to-spec uses it", interview)
	}
	if _, err := Check(ctx, b.options); err != nil {
		t.Fatalf("check after sync: %v", err)
	}
	provenance := readRepositoryFile(t, b.options, ".syllabus/generated/provenance.json")
	if !strings.Contains(provenance, `"team:pocock:skills/engineering/interview"`) {
		t.Fatalf("provenance doesn't list the used borrowed skill:\n%s", provenance)
	}
}

func TestSync_RefusesABorrowedSkillTheLibraryDoesNotList(t *testing.T) {
	ctx := context.Background()
	b := newBorrowing(t, `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"team":{"repository":TEAM,"skills":{"include":["pocock:skills/ops/unlisted"]}}}}`)
	_, err := Sync(ctx, b.options, b.git)
	var validation errs.ValidationError
	if !errors.As(err, &validation) || validation.ValidationLocation() != "sources.team.skills.include" || !strings.Contains(validation.ValidationProblem(), "the library doesn't list pocock:skills/ops/unlisted at any level") {
		t.Fatalf("got %v; want the unlisted borrowed skill refused", err)
	}
}

func TestSync_KeepsTheBorrowedRevisionAndUpdateResolvesTheLibrarysRefAgain(t *testing.T) {
	ctx := context.Background()
	b := newBorrowing(t, `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"team":{"repository":TEAM,"rules":{"include":["techs/go/*"]}}}}`)
	if _, err := Sync(ctx, b.options, b.git); err != nil {
		t.Fatal(err)
	}
	moved, err := b.pocock.Commit(ctx, b.pocock.Worktree(), "Improve deploy", map[string][]byte{"skills/ops/deploy/SKILL.md": []byte(strings.Replace(deploySkill, "# Deploy", "# Deploy safely", 1))})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.pocock.Command(ctx, "tag", "--force", "stable", moved); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(ctx, b.options, b.git); err != nil {
		t.Fatal(err)
	}
	if installed := readRepositoryFile(t, b.options, ".claude/skills/deploy/SKILL.md"); strings.Contains(installed, "Deploy safely") {
		t.Fatal("sync moved the borrowed revision")
	}
	plan, err := PlanUpdate(ctx, b.options, b.git, []imports.UpdateTarget{{Source: "team"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := plan.Preview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Sources) != 1 || len(preview.Sources[0].Borrowed) != 1 {
		t.Fatalf("preview = %+v; want one borrowed row", preview.Sources)
	}
	if row := preview.Sources[0].Borrowed[0]; row.Name != "pocock" || row.From != b.pocock.FirstCommit || row.To != moved || row.Ref != "stable" || strings.Join(row.Skills, ",") != "pocock:skills/ops/deploy" {
		t.Fatalf("borrowed row = %+v", row)
	}
	if _, err := plan.Apply(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if installed := readRepositoryFile(t, b.options, ".claude/skills/deploy/SKILL.md"); !strings.Contains(installed, "Deploy safely") || !strings.Contains(installed, moved) {
		t.Fatalf("update didn't take the borrowed skill from the tag's new commit:\n%s", installed)
	}
	if borrowed, _ := b.teamRecord(t)["borrowed"].(map[string]any); borrowed["pocock"].(map[string]any)["resolvedCommit"] != moved {
		t.Fatalf("borrowed record: %v", borrowed)
	}
}

func TestSync_AProjectSourceForTheSameRepositoryOverridesTheLibrarysRef(t *testing.T) {
	ctx := context.Background()
	b := newBorrowing(t, `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"team":{"repository":TEAM,"rules":{"include":["techs/go/*"]}},"mine":{"repository":POCOCK,"ref":"v1.2.0","skills":{"include":["skills/ops/deploy"]}}}}`)
	if _, err := Sync(ctx, b.options, b.git); err != nil {
		t.Fatal(err)
	}
	installed := readRepositoryFile(t, b.options, ".claude/skills/deploy/SKILL.md")
	if !strings.Contains(installed, "syllabus:installed team:pocock:skills/ops/deploy "+b.pocock.LatestCommit) {
		t.Fatalf("deploy:\n%s\nwant the borrowed skill once, from the overriding source's revision", installed)
	}
	borrowed, _ := b.teamRecord(t)["borrowed"].(map[string]any)
	if pocock := borrowed["pocock"].(map[string]any); pocock["resolvedCommit"] != b.pocock.LatestCommit || pocock["source"] != "mine" || pocock["ref"] != "v1.2.0" {
		t.Fatalf("borrowed record: %v", borrowed)
	}
	if result, err := Check(ctx, b.options); err != nil || !result.Current() {
		t.Fatalf("check = %+v, %v", result, err)
	}
}

func TestSync_ALibraryRepositoryInstallsTheSkillsItRequiresFromItsSources(t *testing.T) {
	ctx := context.Background()
	b := newBorrowing(t, `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"pocock":{"repository":POCOCK,"ref":"stable"}},"library":{"skills":{"required":["pocock:skills/ops/deploy"],"offered":["pocock:skills/productivity/grilling"]}}}`)
	if _, err := Sync(ctx, b.options, b.git); err != nil {
		t.Fatal(err)
	}
	if installed := readRepositoryFile(t, b.options, ".claude/skills/deploy/SKILL.md"); !strings.Contains(installed, "syllabus:installed pocock:skills/ops/deploy "+b.pocock.FirstCommit) {
		t.Fatalf("deploy:\n%s", installed)
	}
	if exists(t, b.options, ".claude/skills/grilling") {
		t.Error("installed a skill the library only offers")
	}
	if result, err := Check(ctx, b.options); err != nil || !result.Current() {
		t.Fatalf("check = %+v, %v", result, err)
	}
}

func TestSync_ALibraryRepositoryNeedsNoSkillsFromASourceItOnlyOffers(t *testing.T) {
	b := newBorrowing(t, `{"schemaVersion":2,"skillTargets":["claude"],"sources":{"pocock":{"repository":POCOCK,"ref":"stable"}},"library":{"skills":{"offered":["pocock:skills/productivity/grilling"]}}}`)
	if _, err := Sync(context.Background(), b.options, b.git); err != nil {
		t.Fatal(err)
	}
	if exists(t, b.options, ".claude/skills/grilling") {
		t.Error("installed a skill the library only offers")
	}
}
