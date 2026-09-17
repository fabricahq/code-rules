// Package acceptance runs disposable, real-CLI migration pilots for tests and human walkthroughs.
package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fabricahq/code-rules/internal/gitfixture"
	"github.com/fabricahq/code-rules/internal/project"
)

// Step records an actual command or a deliberate fixture edit and its observable outcome.
type Step struct {
	Label     string   `json:"label"`
	Arguments []string `json:"arguments,omitempty"`
	Stdout    string   `json:"stdout,omitempty"`
	Stderr    string   `json:"stderr,omitempty"`
	ExitCode  int      `json:"exitCode"`
}

// Report retains commands, verified invariants, and exact project bytes for inspection.
type Report struct {
	Scenario string            `json:"scenario"`
	Steps    []Step            `json:"steps"`
	Verified []string          `json:"verified"`
	Files    map[string][]byte `json:"files"`
}

// Run exercises a real native CLI from library authoring through Git sync and offline checks.
// All writes and Git history belong to disposable directories; no external network or global install is used.
func Run(ctx context.Context, binary, scenario string) (report Report, err error) {
	report = Report{Scenario: scenario, Steps: []Step{}, Verified: []string{}}
	if scenario != "lifecycle" && scenario != "update" && scenario != "changed-vendor" && scenario != "failed-sync" {
		return report, fmt.Errorf("unknown acceptance scenario")
	}
	directory, err := os.MkdirTemp("", "code-rules-acceptance-")
	if err != nil {
		return report, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(directory)) }()
	libraryDir := filepath.Join(directory, "library")
	consumer := filepath.Join(directory, "consumer")
	for _, dir := range []string{libraryDir, consumer} {
		if err := os.Mkdir(dir, 0700); err != nil {
			return report, err
		}
	}
	// Keep the observed project even when a command or invariant fails, before temporary cleanup.
	defer func() {
		captureCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		final, captureErr := readTree(captureCtx, consumer)
		if captureErr != nil {
			err = errors.Join(err, fmt.Errorf("capture acceptance project files: %w", captureErr))
			return
		}
		report.Files = final.Files
	}()
	offline := []string{"PATH=" + filepath.Join(directory, "no-runtime")}
	// invoke captures real exit status and fails the pilot when it differs from the stated scenario.
	invoke := func(label, dir string, env []string, want int, args ...string) error {
		var original *project.Tree
		var err error
		if len(args) > 0 && args[0] == "check" {
			args = append(args, "--json")
			original, err = readTree(ctx, dir)
			if err != nil {
				return err
			}
		}
		command := exec.CommandContext(ctx, binary, args...)
		command.Dir = dir
		command.Env = env
		var out, diagnostic bytes.Buffer
		command.Stdout = &out
		command.Stderr = &diagnostic
		runErr := command.Run()
		code := 0
		if runErr != nil {
			var exit *exec.ExitError
			if !errors.As(runErr, &exit) {
				return runErr
			}
			code = exit.ExitCode()
		}
		report.Steps = append(report.Steps, Step{label, args, out.String(), diagnostic.String(), code})
		if code != want {
			return fmt.Errorf("%s: exit %d, expected %d: %s", label, code, want, diagnostic.String())
		}
		if want != 0 && strings.TrimSpace(diagnostic.String()) == "" && !(len(args) > 0 && args[0] == "check" && reportsCheckProblems(out.Bytes())) {
			return fmt.Errorf("%s: refusal returned no diagnostic", label)
		}
		if original != nil {
			after, err := readTree(ctx, dir)
			if err != nil {
				return err
			}
			if !equalTrees(original, after) {
				return fmt.Errorf("%s: read-only check changed project", label)
			}
		}
		return ctx.Err()
	}
	terms := []byte("Original library terms.\r\nPreserve these bytes.\r\n")
	notice := []byte("Original notice.\r\n")
	for name, data := range map[string][]byte{"terms.txt": terms, "notice.txt": notice, "body.md": []byte("Return every failure.\n\n![diagram](assets/errors/diagram.bin)\n")} {
		if err := os.WriteFile(filepath.Join(libraryDir, name), data, 0600); err != nil {
			return report, err
		}
	}
	commands := [][]string{
		{"library", "init", "--spdx", "MIT", "--license-file", "terms.txt", "--notice-file", "notice.txt"},
		{"library", "add", "group", "techs/go", "--name", "Go", "--description", "Shared Go guidance.", "--when-to-read", "When editing Go."},
		{"library", "add", "rule", "techs/go/errors", "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions.", "--body-file", "body.md"},
	}
	for _, args := range commands {
		if err := invoke("Author library", libraryDir, offline, 0, args...); err != nil {
			return report, err
		}
	}
	asset := []byte{0, 255, 13, 10, 42}
	assetPath := filepath.Join(libraryDir, "techs/go/assets/errors")
	if err := os.MkdirAll(assetPath, 0700); err != nil {
		return report, err
	}
	if err := os.WriteFile(filepath.Join(assetPath, "diagram.bin"), asset, 0600); err != nil {
		return report, err
	}
	if err := invoke("Validate complete library", libraryDir, offline, 0, "library", "check"); err != nil {
		return report, err
	}
	libraryTree, err := readTree(ctx, libraryDir)
	if err != nil {
		return report, err
	}
	fixture, err := gitfixture.New(ctx, libraryTree.Files)
	if err != nil {
		return report, err
	}
	defer func() { err = errors.Join(err, fixture.Close()) }()
	gitBin := filepath.Join(directory, "git-only")
	if err := os.Mkdir(gitBin, 0700); err != nil {
		return report, err
	}
	if err := os.Symlink(fixture.GitPath, filepath.Join(gitBin, "git")); err != nil {
		return report, err
	}
	online := []string{"PATH=" + gitBin}
	for _, value := range fixture.Environment {
		key, _, _ := strings.Cut(value, "=")
		if strings.HasPrefix(key, "GIT_") || key == "HOME" || key == "XDG_CONFIG_HOME" {
			online = append(online, value)
		}
	}
	for _, args := range [][]string{{"init"}, {"add", "source", "team", "--repository", fixture.Repository, "--version", ">= 1.0.0, < 2.0.0", "--groups", "techs/go"}} {
		if err := invoke("Configure consumer", consumer, offline, 0, args...); err != nil {
			return report, err
		}
	}
	if err := invoke("Sync from real Git with no Node/Bun", consumer, online, 0, "sync"); err != nil {
		return report, err
	}
	files, err := readTree(ctx, consumer)
	if err != nil {
		return report, err
	}
	for name, want := range map[string][]byte{".code-rules/vendor/team/LICENSE.md": terms, ".code-rules/vendor/team/NOTICE.md": notice, ".code-rules/vendor/team/techs/go/assets/errors/diagram.bin": asset} {
		if !bytes.Equal(files.Files[name], want) {
			return report, fmt.Errorf("original bytes not preserved: %s", name)
		}
	}
	provenance := string(files.Files[".code-rules/generated/provenance.json"])
	if !strings.Contains(provenance, fixture.LatestCommit) || !strings.Contains(provenance, "v1.2.0") {
		return report, fmt.Errorf("selected revision missing from provenance")
	}
	report.Verified = append(report.Verified, "Annotated v1.2.0 selected using HashiCorp constraints", "License, notice, and binary asset bytes preserved exactly")
	if err := invoke("Local group overrides imported guidance", consumer, offline, 0, "local", "add", "group", "techs/go", "--name", "Project Go", "--description", "Project-specific Go guidance.", "--when-to-read", "When changing this project."); err != nil {
		return report, err
	}
	if err := invoke("Build offline without Git, Node, or Bun", consumer, offline, 0, "build"); err != nil {
		return report, err
	}
	if err := invoke("Check offline", consumer, offline, 0, "check"); err != nil {
		return report, err
	}
	before, err := readTree(ctx, consumer)
	if err != nil {
		return report, err
	}
	if !strings.Contains(string(before.Files[".code-rules/generated/RULES.md"]), "Project-specific Go guidance.") {
		return report, fmt.Errorf("local group guidance missing")
	}
	if err := invoke("Repeat build", consumer, offline, 0, "build"); err != nil {
		return report, err
	}
	after, err := readTree(ctx, consumer)
	if err != nil {
		return report, err
	}
	if !equalTrees(before, after) {
		return report, fmt.Errorf("repeat build changed bytes")
	}
	report.Verified = append(report.Verified, "Local group guidance wins", "Offline build/check work with no runtimes on PATH", "Repeated build is byte-for-byte stable")
	switch scenario {
	case "lifecycle":
		name := ".code-rules/generated/RULES.md"
		if err := os.WriteFile(filepath.Join(consumer, name), []byte("Stale output"), 0600); err != nil {
			return report, err
		}
		report.Steps = append(report.Steps, Step{Label: "Fixture edit: replace generated RULES.md with stale text"})
		if err := invoke("Read-only check detects stale output", consumer, offline, 1, "check"); err != nil {
			return report, err
		}
		if err := invoke("Rebuild repairs generated output", consumer, offline, 0, "build"); err != nil {
			return report, err
		}
		if err := invoke("Final offline check", consumer, offline, 0, "check"); err != nil {
			return report, err
		}
		report.Verified = append(report.Verified, "Check reports stale output without changing it; rebuild repairs it")
	case "changed-vendor":
		name := filepath.Join(consumer, ".code-rules/vendor/team/LICENSE.md")
		if err := os.WriteFile(name, []byte("Manual edit"), 0600); err != nil {
			return report, err
		}
		report.Steps = append(report.Steps, Step{Label: "Fixture edit: manually change vendored LICENSE.md"})
		before, err = readTree(ctx, consumer)
		if err != nil {
			return report, err
		}
		if err := invoke("Refuse changed snapshot", consumer, offline, 1, "build"); err != nil {
			return report, err
		}
		after, err = readTree(ctx, consumer)
		if err != nil {
			return report, err
		}
		if !equalTrees(before, after) {
			return report, fmt.Errorf("failed build changed files")
		}
		report.Verified = append(report.Verified, "Changed vendor bytes produce an error and preserve all project files")
	case "update", "failed-sync":
		rulePath := filepath.Join(fixture.Directory, "repository/techs/go/errors.md")
		document := libraryTree.Files["techs/go/errors.md"]
		if scenario == "update" {
			document = bytes.ReplaceAll(document, []byte("Return every failure."), []byte("Return updated failures."))
		} else {
			document = []byte("Invalid rule without metadata")
		}
		if err := os.WriteFile(rulePath, document, 0600); err != nil {
			return report, err
		}
		for _, args := range [][]string{{"add", "--all"}, {"-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "Next fixture revision"}, {"tag", "v1.3.0"}} {
			if _, err := fixture.Command(ctx, args...); err != nil {
				return report, err
			}
		}
		report.Steps = append(report.Steps, Step{Label: "Fixture edit: publish local Git tag v1.3.0 with " + scenario + " content"})
		want := 0
		if scenario == "failed-sync" {
			want = 1
		}
		if err := invoke("Sync newer selected release", consumer, online, want, "sync"); err != nil {
			return report, err
		}
		after, err = readTree(ctx, consumer)
		if err != nil {
			return report, err
		}
		if want == 1 {
			if !equalTrees(before, after) {
				return report, fmt.Errorf("failed sync changed project")
			}
			report.Verified = append(report.Verified, "Failed import preserves vendor and generated bytes")
		} else {
			if !bytes.Equal(after.Files[".code-rules/vendor/team/techs/go/errors.md"], document) {
				return report, fmt.Errorf("sync did not retain new rule")
			}
			if !strings.Contains(string(after.Files[".code-rules/generated/provenance.json"]), "v1.3.0") {
				return report, fmt.Errorf("new tag missing from provenance")
			}
			if err := invoke("Check updated release offline", consumer, offline, 0, "check"); err != nil {
				return report, err
			}
			report.Verified = append(report.Verified, "New release updates retained bytes and provenance, then checks offline")
		}
	}
	return report, nil
}

// readTree captures file bytes and empty directories through the confined project reader.
func readTree(ctx context.Context, directory string) (*project.Tree, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	tree, err := project.ReadTree(ctx, root, ".")
	if err != nil {
		return nil, err
	}
	return tree, nil
}

// equalTrees compares original bytes and directory inventory, including empty directories.
func equalTrees(before, after *project.Tree) bool {
	return maps.EqualFunc(before.Files, after.Files, bytes.Equal) && slices.Equal(before.Directories, after.Directories)
}

// reportsCheckProblems recognizes a read-only check report whose problems explain the nonzero exit.
func reportsCheckProblems(data []byte) bool {
	var result struct {
		OK    bool
		Value struct {
			Status   string
			Problems []struct{ Kind, Path, NextStep string }
		}
		Error struct{ Kind string }
	}
	return json.Unmarshal(data, &result) == nil && !result.OK && result.Error.Kind == "out_of_date" && result.Value.Status == "out_of_date" && len(result.Value.Problems) > 0
}
