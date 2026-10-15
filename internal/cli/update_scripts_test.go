// Exercise how code-rules project update shows the scripts an update adds or changes in a skill, in human output,
// JSON, and a terminal, against a library whose second release changes a skill's scripts.

package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/gitfixture"
	"github.com/fabricahq/code-rules/internal/test/terminalfixture"
)

// newScriptUpdateFixture syncs a project that installs the skill build-feature from a library's release/1, then
// publishes release/2, which moves techs/go/errors to 1.1.0 and, in build-feature, changes the executable run.sh
// and the reference notes.md, leaves the executable lint.sh as it was, and adds the executable setup.sh.
func newScriptUpdateFixture(t *testing.T) updateFixture {
	t.Helper()
	ctx := context.Background()
	f, err := gitfixture.New(ctx, gitfixture.Library(map[string][]byte{
		"config.yaml":                                          []byte(`{"schemaVersion":2,"sources":{},"library":{}}`),
		"techs/go/_group.yaml":                                 []byte(`{"name":"Go","description":"Go guidance.","whenToRead":"When writing Go."}`),
		"techs/go/errors.md":                                   updateRule("errors 1.0.0"),
		".code-rules/skills/build-feature/SKILL.md":            []byte("---\nname: build-feature\ndescription: Build a feature from a spec.\n---\n# Build a feature\n"),
		".code-rules/skills/build-feature/scripts/run.sh":      []byte("#!/bin/sh\necho build\n"),
		".code-rules/skills/build-feature/scripts/lint.sh":     []byte("#!/bin/sh\necho lint\n"),
		".code-rules/skills/build-feature/references/notes.md": []byte("Notes.\n"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	chmodExecutable(t, f, ".code-rules/skills/build-feature/scripts/run.sh", ".code-rules/skills/build-feature/scripts/lint.sh")
	if _, err := f.Commit(ctx, f.Worktree(), "First release", nil); err != nil {
		t.Fatal(err)
	}
	if err := f.Release(ctx, 1, "formatVersion: 1\nrelease: 1\nrules:\n  techs/go/errors: 1.0.0\nchanges:\n  techs/go/errors: {change: new, summaries: [Add the rule.]}\n"); err != nil {
		t.Fatal(err)
	}
	u := updateFixture{binary: buildCLI(t), directory: t.TempDir(), fixture: f}
	if out, diagnostic, code := runCLI(t, u.binary, u.directory, "project", "init", "--skill-targets", "claude"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	u.write(t, "config.yaml", "schemaVersion: 2\nskillTargets: [claude]\nsources:\n  team:\n    repository: "+f.Repository+"\n    rules:\n      include: [techs/go/*]\n    skills:\n      include: [skills/build-feature]\n")
	if out, diagnostic, code := u.run(t, "project", "sync"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	if _, err := f.Commit(ctx, f.Worktree(), "Change the scripts", gitfixture.Library(map[string][]byte{
		"techs/go/errors.md":                                   updateRule("errors 1.1.0"),
		".code-rules/skills/build-feature/scripts/run.sh":      []byte("#!/bin/sh\necho build everything\n"),
		".code-rules/skills/build-feature/scripts/setup.sh":    []byte("#!/bin/sh\necho setup\n"),
		".code-rules/skills/build-feature/references/notes.md": []byte("More notes.\n"),
	})); err != nil {
		t.Fatal(err)
	}
	chmodExecutable(t, f, ".code-rules/skills/build-feature/scripts/setup.sh")
	if _, err := f.Commit(ctx, f.Worktree(), "Make setup.sh executable", nil); err != nil {
		t.Fatal(err)
	}
	if err := f.Release(ctx, 2, "formatVersion: 1\nrelease: 2\nrules:\n  techs/go/errors: 1.1.0\nchanges:\n  techs/go/errors: {change: minor, from: 1.0.0, summaries: [Wrap errors.]}\n"); err != nil {
		t.Fatal(err)
	}
	return u
}

// chmodExecutable sets the executable bits of names, paths in the fixture's working tree.
func chmodExecutable(t *testing.T, f *gitfixture.Fixture, names ...string) {
	t.Helper()
	for _, name := range names {
		if err := os.Chmod(filepath.Join(f.Worktree(), filepath.FromSlash(name)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
}

// scriptUpdatePreview is how newScriptUpdateFixture's update previews its changes, before its closing advice.
const scriptUpdatePreview = "team\n" +
	"  minor     techs/go/errors  1.0.0 -> 1.1.0\n" +
	"            Wrap errors.\n" +
	"  changed   skills/build-feature\n" +
	"            Changed: references/notes.md, scripts/run.sh, scripts/setup.sh\n" +
	"            script changed: scripts/run.sh\n" +
	"            script added: scripts/setup.sh\n" +
	"  Shared files: release 1 -> 2\n" +
	"\n" +
	"Warning: some skills in this update add or change scripts. Code Rules never runs them, but your agents may; read them before you confirm.\n"

func TestUpdate_PreviewFlagsEachScriptAndWarns(t *testing.T) {
	u := newScriptUpdateFixture(t)
	out, diagnostic, code := u.run(t, "project", "update")
	want := scriptUpdatePreview + "\nThis is a preview; no files were written. To apply it, run the command again\nwith --yes, or in a terminal to answer each question and confirm.\n"
	if code != 0 || diagnostic != "" || out != want {
		t.Fatalf("exit %d, stderr %q, stdout:\n%s\nwant:\n%s", code, diagnostic, out, want)
	}
}

// TestUpdate_JSONReportsEachScriptAndTheWarning previews, then applies with --yes, which reports the scripts and
// warns again.
func TestUpdate_JSONReportsEachScriptAndTheWarning(t *testing.T) {
	u := newScriptUpdateFixture(t)
	wantSkills := []string{`{"id":"skills/build-feature","change":"changed","files":["references/notes.md","scripts/run.sh","scripts/setup.sh"],"scripts":[{"path":"scripts/run.sh","change":"changed"},{"path":"scripts/setup.sh","change":"added"}]}`}
	for _, run := range []struct {
		args    []string
		applied bool
		warning string
	}{
		{[]string{"project", "update", "--json"}, false, "some skills in this update add or change scripts. Code Rules never runs them, but your agents may; read them before you confirm."},
		{[]string{"project", "update", "--json", "--yes"}, true, "some skills in this update added or changed scripts. Code Rules never runs them, but your agents may; read them before you commit the update."},
	} {
		out, diagnostic, code := u.run(t, run.args...)
		var result struct {
			OK    bool
			Value struct {
				Applied bool
				Sources []struct {
					Skills []json.RawMessage
				}
				Warnings []string
			}
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil || code != 0 || diagnostic != "" || !result.OK || result.Value.Applied != run.applied || len(result.Value.Sources) != 1 {
			t.Fatalf("%v: exit %d, %v:\n%s%s", run.args, code, err, out, diagnostic)
		}
		got := []string{}
		for _, row := range result.Value.Sources[0].Skills {
			got = append(got, compactJSON(t, row))
		}
		if !reflect.DeepEqual(got, wantSkills) || !reflect.DeepEqual(result.Value.Warnings, []string{run.warning}) {
			t.Fatalf("%v: skills:\n%s\nwarnings %q", run.args, strings.Join(got, "\n"), result.Value.Warnings)
		}
	}
	info, err := os.Stat(filepath.Join(u.directory, ".claude", "skills", "build-feature", "scripts", "setup.sh"))
	if err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("installed setup.sh: %v, %v; want it executable", info, err)
	}
}

// TestUpdate_WarnsAboutScriptsBeforeAskingInATerminal declines the update after reading the warning.
func TestUpdate_WarnsAboutScriptsBeforeAskingInATerminal(t *testing.T) {
	u := newScriptUpdateFixture(t)
	before := projectFileContents(t, u.directory)
	result, err := terminalfixture.RunWithEnvironment(context.Background(), u.binary, u.directory, u.fixture.Environment, []string{"project", "update"}, []terminalfixture.Step{{Prompt: "Apply the update? [yes/no]:", Answer: "no"}})
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("exit %d, %v\n%s", result.ExitCode, err, result.Transcript)
	}
	transcript := strings.ReplaceAll(result.Transcript, "\r\n", "\n")
	if beforeQuestion, _, asked := strings.Cut(transcript, "Apply the update?"); !asked || !strings.Contains(beforeQuestion, scriptUpdatePreview) {
		t.Fatalf("the preview and its scripts warning didn't come before the confirmation:\n%s", result.Transcript)
	}
	if after := projectFileContents(t, u.directory); !reflect.DeepEqual(before, after) {
		t.Fatal("a declined update changed the project")
	}
}
