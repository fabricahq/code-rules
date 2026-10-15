// Test that a source's skills.exclude leaves skills out, and that a fork installs in place of the skill it replaces
// and has its source's later changes listed by update.

package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/errs"
	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// skillConfig is a project configuration that installs every skill of the fixture's library into .claude/skills/,
// followed by exclude, the YAML of a skills.exclude list or empty.
func skillConfig(exclude string) string {
	return "schemaVersion: 2\nskillTargets: [claude]\nsources:\n  team:\n    repository: REPOSITORY\n    skills:\n      include: ['*']\n" + exclude
}

// forkSkill plans and commits a fork of from with reason.
func forkSkill(t *testing.T, options Options, from, reason string) (*SkillForkPlan, AuthoringResult) {
	t.Helper()
	source, err := ParseSkillForkSource(from)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanSkillFork(context.Background(), source, options)
	if err != nil {
		t.Fatal(err)
	}
	result, err := plan.Commit(context.Background(), reason)
	if err != nil {
		t.Fatal(err)
	}
	return plan, result
}

// TestSync_LeavesOutAnExcludedSkill installs the source's other skills, keeps the excluded skill's files in vendor/
// so updates can compare them, and lists it with its reason in SKILLS.md.
func TestSync_LeavesOutAnExcludedSkill(t *testing.T) {
	ctx := context.Background()
	f, _ := skillLibrary(t)
	options, git := skillProject(t, f, skillConfig("      exclude:\n        - id: skills/review\n          reason: We review in CI.\n"))
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	if exists(t, options, ".claude/skills/review") || !exists(t, options, ".claude/skills/build-feature/SKILL.md") {
		t.Fatal("sync installed the excluded skill, or not the other one")
	}
	if !exists(t, options, ".code-rules/vendor/team/skills/review/SKILL.md") {
		t.Fatal("sync didn't keep the excluded skill's files in vendor/")
	}
	index := readRepositoryFile(t, options, ".code-rules/generated/SKILLS.md")
	if !strings.Contains(index, "## Excluded skills") || !strings.Contains(index, "- `team:skills/review`: We review in CI.") || strings.Contains(index, "### review") {
		t.Fatalf("SKILLS.md doesn't list the excluded skill with its reason:\n%s", index)
	}
	if readme := readRepositoryFile(t, options, ".code-rules/generated/libraries/team/README.md"); !strings.Contains(readme, "- `skills/review`: excluded") {
		t.Fatalf("source README:\n%s", readme)
	}
	if result, err := Check(ctx, options); err != nil || !result.Current() {
		t.Fatalf("check after sync = %+v, %v", result, err)
	}
}

// TestSync_RejectsAnExclusionOfASkillTheSourceDoesNotInstall fails before writing, naming the entry.
func TestSync_RejectsAnExclusionOfASkillTheSourceDoesNotInstall(t *testing.T) {
	f, _ := skillLibrary(t)
	options, git := skillProject(t, f, "schemaVersion: 2\nskillTargets: [claude]\nsources:\n  team:\n    repository: REPOSITORY\n    skills:\n      include: [skills/review]\n      exclude:\n        - id: skills/build-feature\n          reason: Not ours.\n")
	_, err := Sync(context.Background(), options, git)
	var invalid errs.ValidationError
	if !errors.As(err, &invalid) || invalid.ValidationLocation() != "sources.team.skills.exclude.skills/build-feature" || !strings.Contains(invalid.ValidationProblem(), "the source doesn't install skill skills/build-feature") {
		t.Fatalf("got %v; want a validation error naming the exclusion", err)
	}
	if exists(t, options, ".claude/skills/review") {
		t.Fatal("refused sync installed skills")
	}
}

// TestPlanSkillFork_InstallsTheForkInPlaceOfTheSkill copies the vendored skill with its executable bit, records the
// exclusion in the same write, and installs the fork in the original's place on build, offline.
func TestPlanSkillFork_InstallsTheForkInPlaceOfTheSkill(t *testing.T) {
	ctx := context.Background()
	f, commit := skillLibrary(t)
	options, git := skillProject(t, f, skillConfig(""))
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	plan, result := forkSkill(t, options, "team:skills/build-feature", "We build with make.")
	if plan.Name() != "build-feature" || plan.BasedOn() != commit {
		t.Fatalf("plan = %s based on %s", plan.Name(), plan.BasedOn())
	}
	codeRules := filepath.Join(options.Directory, ".code-rules")
	if want := []string{filepath.Join(codeRules, "skills/build-feature/SKILL.md"), filepath.Join(codeRules, "skills/build-feature/scripts/run.sh")}; !reflect.DeepEqual(result.Added, want) || !reflect.DeepEqual(result.Changed, []string{filepath.Join(codeRules, "config.yaml")}) {
		t.Fatalf("result = %+v", result)
	}
	if fork := readRepositoryFile(t, options, ".code-rules/skills/build-feature/SKILL.md"); fork != buildSkill {
		t.Fatalf("fork's SKILL.md:\n%s", fork)
	}
	if info, err := os.Stat(filepath.Join(options.Directory, ".code-rules/skills/build-feature/scripts/run.sh")); err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("the fork's script isn't executable: %v %v", info, err)
	}
	data := []byte(readRepositoryFile(t, options, ".code-rules/config.yaml"))
	if want := "      exclude:\n        - id: skills/build-feature\n          reason: We build with make.\n          replacedBy: skills/build-feature\n          basedOn: \"" + commit + "\"\n"; !strings.Contains(string(data), want) {
		t.Fatalf("config.yaml:\n%s\nwant:\n%s", data, want)
	}
	if _, err := Build(ctx, options); err != nil {
		t.Fatal(err)
	}
	if installed := readRepositoryFile(t, options, ".claude/skills/build-feature/SKILL.md"); !strings.Contains(installed, "<!-- code-rules:installed skills/build-feature. Do not edit; edit .code-rules/skills/build-feature/ instead. -->") {
		t.Fatalf("installed copy isn't the fork:\n%s", installed)
	}
	index := readRepositoryFile(t, options, ".code-rules/generated/SKILLS.md")
	for _, want := range []string{"**Skill:** `skills/build-feature`, authored in this project", "**Replaces:** `team:skills/build-feature`, which the project excludes, based on its copy at commit `" + commit + "`", "**Reason:** We build with make."} {
		if !strings.Contains(index, want) {
			t.Errorf("SKILLS.md lacks %q:\n%s", want, index)
		}
	}
	var provenance struct {
		Skills []struct {
			ID       string `json:"id"`
			Upstream *struct {
				Source string `json:"source"`
				Commit string `json:"resolvedCommit"`
				Folder string `json:"folder"`
			} `json:"upstream"`
			Reason  *string `json:"replacementReason"`
			BasedOn *string `json:"basedOn"`
		} `json:"skills"`
	}
	if err := json.Unmarshal([]byte(readRepositoryFile(t, options, ".code-rules/generated/provenance.json")), &provenance); err != nil {
		t.Fatal(err)
	}
	if len(provenance.Skills) != 2 {
		t.Fatalf("provenance skills = %+v", provenance.Skills)
	}
	for _, skill := range provenance.Skills {
		switch skill.ID {
		case "skills/build-feature":
			if skill.Upstream == nil || skill.Upstream.Source != "team" || skill.Upstream.Commit != commit || skill.Upstream.Folder != "skills/build-feature" || *skill.Reason != "We build with make." || *skill.BasedOn != commit {
				t.Errorf("provenance of the fork = %+v", skill)
			}
		case "team:skills/review":
			if skill.Upstream != nil || skill.Reason != nil || skill.BasedOn != nil {
				t.Errorf("provenance of %s = %+v", skill.ID, skill)
			}
		default:
			t.Errorf("unexpected skill %s in provenance", skill.ID)
		}
	}
	if result, err := Check(ctx, options); err != nil || !result.Current() {
		t.Fatalf("check after build = %+v, %v", result, err)
	}
	// A teammate's sync takes the same commit and finds nothing to change.
	if again, err := Sync(ctx, options, git); err != nil || len(again.Added)+len(again.Changed)+len(again.Removed) != 0 {
		t.Fatalf("sync after build = %+v, %v; want no changes", again, err)
	}
}

// TestPlanSkillFork_RefusesWhatItCannotFork fails before writing for each target it never replaces or can't read.
func TestPlanSkillFork_RefusesWhatItCannotFork(t *testing.T) {
	ctx := context.Background()
	f, _ := skillLibrary(t)
	options, git := skillProject(t, f, skillConfig("      exclude:\n        - id: skills/review\n          reason: We review in CI.\n"))
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	root, err := openProject(ctx, options, false)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	writeFixture(t, root, "skills/build-feature/SKILL.md", "---\nname: build-feature\ndescription: Ours.\n---\n# Build\n")
	for _, test := range []struct {
		from, location, problem string
	}{
		{"other:skills/review", "--from", "no source named other in .code-rules/config.yaml"},
		{"team:skills/missing", "--from", "source team doesn't install skill skills/missing, as its last sync recorded"},
		{"team:skills/review", "sources.team.skills.exclude.skills/review", "the source already excludes this skill"},
	} {
		source, err := ParseSkillForkSource(test.from)
		if err != nil {
			t.Fatal(err)
		}
		_, err = PlanSkillFork(ctx, source, options)
		var invalid errs.ValidationError
		if !errors.As(err, &invalid) || invalid.ValidationLocation() != test.location || !strings.Contains(invalid.ValidationProblem(), test.problem) {
			t.Errorf("fork %s: got %v; want %s: %s", test.from, err, test.location, test.problem)
		}
	}
	source, _ := ParseSkillForkSource("team:skills/build-feature")
	_, err = PlanSkillFork(ctx, source, options)
	projectCode(t, err, "already-exists")
	if _, err := ParseSkillForkSource("skills/review"); err == nil || !strings.Contains(err.Error(), "expected SOURCE:SKILL") {
		t.Fatalf("parse without a source = %v", err)
	}
}

// TestPlanSkillFork_CommitRefusesABlankReasonAndAConcurrentChange writes nothing in either case.
func TestPlanSkillFork_CommitRefusesABlankReasonAndAConcurrentChange(t *testing.T) {
	ctx := context.Background()
	f, _ := skillLibrary(t)
	options, git := skillProject(t, f, skillConfig(""))
	if _, err := Sync(ctx, options, git); err != nil {
		t.Fatal(err)
	}
	source, _ := ParseSkillForkSource("team:skills/review")
	plan, err := PlanSkillFork(ctx, source, options)
	if err != nil {
		t.Fatal(err)
	}
	var invalid errs.ValidationError
	if _, err := plan.Commit(ctx, " "); !errors.As(err, &invalid) || invalid.ValidationLocation() != "--reason" {
		t.Fatalf("blank reason = %v", err)
	}
	root, err := openProject(ctx, options, false)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	writeFixture(t, root, configurationFile, strings.ReplaceAll("# Edited.\n"+skillConfig(""), "REPOSITORY", quoteJSON(f.Repository)))
	_, err = plan.Commit(ctx, "Ours.")
	projectCode(t, err, "concurrent-change")
	if exists(t, options, ".code-rules/skills/review") {
		t.Fatal("a refused fork wrote files")
	}
}

// secondSkillRelease publishes release/2, which changes build-feature's SKILL.md and adds a script to it.
func secondSkillRelease(t *testing.T, f *gitfixture.Fixture) string {
	t.Helper()
	ctx := context.Background()
	commit, err := f.Commit(ctx, f.Worktree(), "Change a skill", map[string][]byte{
		".code-rules/skills/build-feature/SKILL.md":         []byte(strings.Replace(buildSkill, "Run scripts/run.sh.", "Run scripts/check.sh first.", 1)),
		".code-rules/skills/build-feature/scripts/check.sh": []byte("#!/bin/sh\necho check\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Release(ctx, 2, "formatVersion: 1\nrelease: 2\nrules:\n  techs/go/errors: 1.0.0\n"); err != nil {
		t.Fatal(err)
	}
	return commit
}

// TestUpdate_ListsChangesToAReplacedSkill compares the source's newer copy of a replaced skill with the copy its
// replacement is based on, or, without basedOn, the locked copy, and leaves the replacement and basedOn alone.
func TestUpdate_ListsChangesToAReplacedSkill(t *testing.T) {
	ctx := context.Background()
	for _, recorded := range []bool{true, false} {
		t.Run(map[bool]string{true: "basedOn", false: "locked commit"}[recorded], func(t *testing.T) {
			f, commit := skillLibrary(t)
			options, git := skillProject(t, f, skillConfig(""))
			if _, err := Sync(ctx, options, git); err != nil {
				t.Fatal(err)
			}
			forkSkill(t, options, "team:skills/build-feature", "We build with make.")
			basedOn := commit
			if !recorded {
				root, err := openProject(ctx, options, false)
				if err != nil {
					t.Fatal(err)
				}
				config := readRepositoryFile(t, options, ".code-rules/config.yaml")
				writeFixture(t, root, configurationFile, strings.Replace(config, "          basedOn: \""+commit+"\"\n", "", 1))
				root.Close()
				basedOn = ""
			}
			if _, err := Build(ctx, options); err != nil {
				t.Fatal(err)
			}
			later := secondSkillRelease(t, f)
			plan, err := PlanUpdate(ctx, options, git, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			preview, err := plan.Preview(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			want := []imports.ReplacedSkillUpdate{{ID: "skills/build-feature", LocalSkill: "skills/build-feature", BasedOn: basedOn, From: commit, To: later, Files: []string{"SKILL.md", "scripts/check.sh"}}}
			if len(preview.Sources) != 1 || !reflect.DeepEqual(preview.Sources[0].ReplacedSkills, want) {
				t.Fatalf("preview = %+v\nwant replaced skills %+v", preview.Sources, want)
			}
			fork := readRepositoryFile(t, options, ".code-rules/skills/build-feature/SKILL.md")
			config := readRepositoryFile(t, options, ".code-rules/config.yaml")
			if _, err := plan.Apply(ctx, nil); err != nil {
				t.Fatal(err)
			}
			if readRepositoryFile(t, options, ".code-rules/skills/build-feature/SKILL.md") != fork || readRepositoryFile(t, options, ".code-rules/config.yaml") != config {
				t.Fatal("the update changed the fork or its exclusion")
			}
			if installed := readRepositoryFile(t, options, ".claude/skills/build-feature/SKILL.md"); !strings.Contains(installed, "edit .code-rules/skills/build-feature/ instead") {
				t.Fatalf("after the update, the installed copy isn't the fork:\n%s", installed)
			}
		})
	}
}

// TestSync_ChecksTheCommitAReplacementIsBasedOn fails with version-not-found for a commit the repository lacks, and
// with a validation error for a commit without the skill.
func TestSync_ChecksTheCommitAReplacementIsBasedOn(t *testing.T) {
	ctx := context.Background()
	f, _ := skillLibrary(t)
	// A commit after release/1 removes the skill, so the repository has the commit but the commit lacks the skill.
	without, err := f.Commit(ctx, f.Worktree(), "Remove a skill", map[string][]byte{".code-rules/skills/build-feature/SKILL.md": nil, ".code-rules/skills/build-feature/scripts/run.sh": nil})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, basedOn string
		check         func(t *testing.T, err error)
	}{
		{"missing commit", strings.Repeat("a", 40), func(t *testing.T, err error) {
			var failure *imports.Error
			if !errors.As(err, &failure) || failure.Code != "version-not-found" || !strings.Contains(err.Error(), "sources.team.skills.exclude.skills/build-feature.basedOn: the repository has no commit "+strings.Repeat("a", 40)) {
				t.Fatalf("got %v; want version-not-found naming basedOn", err)
			}
		}},
		{"commit without the skill", without, func(t *testing.T, err error) {
			var invalid errs.ValidationError
			if !errors.As(err, &invalid) || invalid.ValidationLocation() != "sources.team.skills.exclude.skills/build-feature.basedOn" || !strings.Contains(invalid.ValidationProblem(), "commit "+without+" has no skill skills/build-feature") {
				t.Fatalf("got %v; want a validation error naming basedOn", err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			options, git := skillProject(t, f, skillConfig("      exclude:\n        - id: skills/build-feature\n          reason: Ours.\n          replacedBy: skills/build-feature\n          basedOn: \""+test.basedOn+"\"\n"))
			root, err := openProject(ctx, options, false)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			writeFixture(t, root, "skills/build-feature/SKILL.md", buildSkill)
			_, err = Sync(ctx, options, git)
			test.check(t, err)
			if exists(t, options, ".claude/skills") {
				t.Fatal("refused sync installed skills")
			}
		})
	}
}
