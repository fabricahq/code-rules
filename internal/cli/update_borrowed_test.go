// Exercise how syllabus project update previews the skills a library borrows from another repository, in human
// output and JSON, when that repository's tag moves and changes a borrowed skill's script.

package cli

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/syllabus/internal/test/gitfixture"
)

// newBorrowedUpdateFixture syncs a project that uses team, a library that requires deploy, a skill it borrows from
// pocock, a repository that isn't a library, at its tag stable, then moves stable to a commit that changes deploy's
// script and makes it executable.
func newBorrowedUpdateFixture(t *testing.T) (updateFixture, string, string) {
	t.Helper()
	ctx := context.Background()
	pocock, err := gitfixture.New(ctx, map[string][]byte{
		"skills/ops/deploy/SKILL.md":          []byte("---\nname: deploy\ndescription: Deploy a service.\n---\n# Deploy\n"),
		"skills/ops/deploy/scripts/deploy.sh": []byte("#!/bin/sh\necho deploy\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pocock.Close() })
	if _, err := pocock.Command(ctx, "tag", "stable", pocock.FirstCommit); err != nil {
		t.Fatal(err)
	}
	team, err := gitfixture.New(ctx, gitfixture.Library(map[string][]byte{
		"config.yaml":          []byte("schemaVersion: 2\nsources:\n  pocock:\n    repository: git@fixture.invalid:pocock\n    ref: stable\nlibrary:\n  skills:\n    required: [pocock:skills/ops/deploy]\n"),
		"techs/go/_group.yaml": []byte(`{"name":"Go","description":"Go guidance.","whenToRead":"When writing Go."}`),
		"techs/go/errors.md":   updateRule("errors 1.0.0"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = team.Close() })
	if err := team.Release(ctx, 1, "formatVersion: 1\nrelease: 1\nrules:\n  techs/go/errors: 1.0.0\nchanges:\n  techs/go/errors: {change: new, summaries: [Add the rule.]}\n"); err != nil {
		t.Fatal(err)
	}
	if team.Environment, err = team.Route(map[string]*gitfixture.Fixture{"team": team, "pocock": pocock}); err != nil {
		t.Fatal(err)
	}
	u := updateFixture{binary: buildCLI(t), directory: t.TempDir(), fixture: team}
	if out, diagnostic, code := runCLI(t, u.binary, u.directory, "project", "init", "--skill-targets", "claude"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	u.write(t, "config.yaml", "schemaVersion: 2\nskillTargets: [claude]\nsources:\n  team:\n    repository: git@fixture.invalid:team\n    rules:\n      include: [techs/go/*]\n")
	if out, diagnostic, code := u.run(t, "project", "sync"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	worktree := pocock.Worktree()
	if _, err := pocock.Commit(ctx, worktree, "Deploy more", map[string][]byte{"skills/ops/deploy/scripts/deploy.sh": []byte("#!/bin/sh\necho deploy everything\n")}); err != nil {
		t.Fatal(err)
	}
	chmodExecutable(t, pocock, "skills/ops/deploy/scripts/deploy.sh")
	moved, err := pocock.Commit(ctx, worktree, "Make deploy.sh executable", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pocock.Command(ctx, "tag", "--force", "stable", moved); err != nil {
		t.Fatal(err)
	}
	return u, pocock.FirstCommit, moved
}

func TestUpdate_PreviewsBorrowedSkillsAndTheirScripts(t *testing.T) {
	u, from, to := newBorrowedUpdateFixture(t)
	out, diagnostic, code := u.run(t, "project", "update", "team")
	want := "team\n" +
		"  borrowed  pocock  commit " + from[:12] + " -> " + to[:12] + "\n" +
		"            Borrowed from git@fixture.invalid:pocock, resolved from ref stable.\n" +
		"            Skills: pocock:skills/ops/deploy\n" +
		"  changed   pocock:skills/ops/deploy\n" +
		"            Changed: scripts/deploy.sh\n" +
		"            script changed: scripts/deploy.sh\n" +
		"\n" +
		"Warning: some skills in this update add or change scripts. Syllabus never runs them, but your agents may; read them before you confirm.\n"
	if code != 0 || diagnostic != "" || !strings.HasPrefix(out, want) {
		t.Fatalf("exit %d, stderr %q, stdout:\n%s\nwant it to start:\n%s", code, diagnostic, out, want)
	}
}

func TestUpdate_JSONReportsBorrowedSkillsAndTheirScripts(t *testing.T) {
	u, from, to := newBorrowedUpdateFixture(t)
	out, diagnostic, code := u.run(t, "project", "update", "team", "--json")
	var result struct {
		OK    bool
		Value struct {
			Sources []struct {
				Borrowed []json.RawMessage
			}
			Warnings []string
		}
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil || code != 0 || diagnostic != "" || !result.OK || len(result.Value.Sources) != 1 || len(result.Value.Sources[0].Borrowed) != 1 {
		t.Fatalf("exit %d, %v:\n%s%s", code, err, out, diagnostic)
	}
	want := `{"name":"pocock","repository":"git@fixture.invalid:pocock","ref":"stable","from":"` + from + `","to":"` + to + `","skills":["pocock:skills/ops/deploy"],"changes":[{"id":"pocock:skills/ops/deploy","change":"changed","files":["scripts/deploy.sh"],"scripts":[{"path":"scripts/deploy.sh","change":"changed"}]}]}`
	if got := compactJSON(t, result.Value.Sources[0].Borrowed[0]); got != want {
		t.Fatalf("borrowed row:\n%s\nwant:\n%s", got, want)
	}
	if wantWarnings := []string{"some skills in this update add or change scripts. Syllabus never runs them, but your agents may; read them before you confirm."}; !reflect.DeepEqual(result.Value.Warnings, wantWarnings) {
		t.Fatalf("warnings %q", result.Value.Warnings)
	}
}
