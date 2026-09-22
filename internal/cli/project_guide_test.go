// Exercise the agent-facing project guide through the native executable.

package cli

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// TestInitCreatesAgentGuide verifies init exposes the main agent entry point and a read-only freshness check.
func TestInitCreatesAgentGuide(t *testing.T) {
	binary := buildCLI(t)
	directory := t.TempDir()
	out, diagnostic, code := runCLI(t, binary, directory, "project", "init")
	if code != 0 {
		t.Fatal(out, diagnostic)
	}
	guidePath := filepath.Join(directory, ".code-rules", "README.md")
	guide, err := os.ReadFile(guidePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Add a group", "Add a rule", "Add a third-party library", "code-rules project check"} {
		if !strings.Contains(string(guide), text) {
			t.Fatalf("guide missing %q", text)
		}
	}
	if out, diagnostic, code := runCLI(t, binary, directory, "project", "build"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	out, diagnostic, code = runCLI(t, binary, directory, "project", "check")
	if code != 0 || !strings.Contains(out, "up to date") {
		t.Fatal(code, out, diagnostic)
	}
	if err := os.WriteFile(guidePath, append(guide, []byte("\nMy custom text.\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	out, diagnostic, code = runCLI(t, binary, directory, "project", "check", "--json")
	if code != 1 || !strings.Contains(out, `"ok": false`) || diagnostic != "" {
		t.Fatal(code, out, diagnostic)
	}
	out, diagnostic, code = runCLI(t, binary, directory, "project", "init")
	if code != 1 || !strings.Contains(diagnostic, "README.md") {
		t.Fatal(code, out, diagnostic)
	}
	after, err := os.ReadFile(guidePath)
	if err != nil || !strings.HasSuffix(string(after), "My custom text.\n") {
		t.Fatal("init overwrote custom text", err)
	}
}

// TestProjectGuideExamples executes every shell example from the generated README against the real CLI and an isolated Git publisher.
func TestProjectGuideExamples(t *testing.T) {
	binary := buildCLI(t)
	fixture, err := gitfixture.New(context.Background(), map[string][]byte{
		"rule-library.json":    []byte(`{"formatVersion":1}`),
		"techs/go/_group.json": []byte(`{"name":"Go","description":"Shared Go guidance.","whenToRead":"When writing Go."}`),
		"techs/go/shared.md":   []byte("---\ntitle: Preserve errors\nimpact: HIGH\nimpactDescription: Keep failures visible.\nwhenToRead: When calling fallible functions.\n---\nReturn errors to the caller.\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, projectName := range []string{"project", "project 'quoted'"} {
		t.Run(projectName, func(t *testing.T) {
			directory := filepath.Join(t.TempDir(), projectName)
			if err := os.Mkdir(directory, 0700); err != nil {
				t.Fatal(err)
			}
			readme := "# My project\nKeep this authored README unchanged.\n"
			if err := os.WriteFile(filepath.Join(directory, "README.md"), []byte(readme), 0600); err != nil {
				t.Fatal(err)
			}
			out, diagnostic, code := runCLI(t, binary, directory, "project", "init")
			if code != 0 {
				t.Fatal(out, diagnostic)
			}
			guide, err := os.ReadFile(filepath.Join(directory, ".code-rules", "README.md"))
			if err != nil {
				t.Fatal(err)
			}
			blocks := regexp.MustCompile("(?ms)^```sh\\n(.*?)^```$").FindAllSubmatch(guide, -1)
			if len(blocks) == 0 {
				t.Fatal("no executable README examples")
			}
			for index, block := range blocks {
				script := strings.ReplaceAll(string(block[1]), "code-rules ", gitfixture.Quote(binary)+" ")
				script = strings.ReplaceAll(script, "https://github.com/example/engineering-rules", fixture.Repository)
				command := exec.Command("/bin/sh", "-eu", "-c", script)
				command.Dir = directory
				command.Env = fixture.Environment
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("README example %d failed: %v\n%s", index+1, err, output)
				}
			}
			authored, err := os.ReadFile(filepath.Join(directory, "README.md"))
			if err != nil || string(authored) != readme {
				t.Fatal("changed authored README", err)
			}
			for _, file := range []string{"local/README.md", "local/techs/go/README.md"} {
				content, err := os.ReadFile(filepath.Join(directory, ".code-rules", file))
				if err != nil || !strings.Contains(string(content), "/README.md)") {
					t.Fatalf("%s does not link to the managed guide: %v", file, err)
				}
			}
			for _, path := range []string{"local/techs/go/return-errors.md", "vendor/team/techs/go/shared.md", "generated/RULES.md", "generated/groups/techs/go.md"} {
				if _, err := os.Stat(filepath.Join(directory, ".code-rules", path)); err != nil {
					t.Fatalf("README did not produce %s: %v", path, err)
				}
			}
		})
	}
}

// TestCheckVerifiesGuideAndGeneratedOutput exercises both independent checks without allowing writes.
func TestCheckVerifiesGuideAndGeneratedOutput(t *testing.T) {
	binary := buildCLI(t)
	for _, guideState := range []string{"current", "missing", "edited"} {
		for _, stale := range []bool{false, true} {
			name := guideState
			if stale {
				name += "/stale-output"
			}
			t.Run(name, func(t *testing.T) {
				directory := t.TempDir()
				for _, command := range []string{"init", "build"} {
					if out, diagnostic, code := runCLI(t, binary, directory, "project", command); code != 0 {
						t.Fatal(code, out, diagnostic)
					}
				}
				root := filepath.Join(directory, ".code-rules")
				guideName := "README.md"
				guidePath := filepath.Join(root, guideName)
				if guideState == "missing" {
					if err := os.Remove(guidePath); err != nil {
						t.Fatal(err)
					}
				} else if guideState == "edited" {
					if err := os.WriteFile(guidePath, []byte("Manually edited guide."), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if stale {
					if err := os.WriteFile(filepath.Join(root, "generated", "RULES.md"), []byte("Stale output."), 0600); err != nil {
						t.Fatal(err)
					}
				}
				before := projectFileContents(t, directory)
				for _, jsonMode := range []bool{false, true} {
					args := []string{"project", "check"}
					if jsonMode {
						args = append(args, "--json")
					}
					out, diagnostic, code := runCLI(t, binary, directory, args...)
					wantCode := 0
					if stale || guideState != "current" {
						wantCode = 1
					}
					if code != wantCode {
						t.Fatal(code, out, diagnostic)
					}
					if jsonMode {
						var result struct {
							OK    bool
							Value projectCheckResult
							Error *responseError
						}
						if err := json.Unmarshal([]byte(out), &result); err != nil {
							t.Fatal(err, out)
						}
						if diagnostic != "" || result.OK != (wantCode == 0) || (result.Error != nil) != (wantCode != 0) {
							t.Fatal(out, diagnostic)
						}
						kinds := map[string]bool{}
						for _, problem := range result.Value.Problems {
							kinds[problem.Kind] = true
						}
						if kinds["outdated_readme"] != (guideState != "current") || kinds["stale_contents"] != stale {
							t.Fatal(out)
						}
					} else {
						if !strings.Contains(out, "Status:") {
							t.Fatal(out, diagnostic)
						}
						if wantCode == 0 && !strings.Contains(out, "the Code Rules guide are current") {
							t.Fatal(out)
						}
						if guideState != "current" && !strings.Contains(out, guideName) {
							t.Fatal(out, diagnostic)
						}
					}
					if after := projectFileContents(t, directory); !reflect.DeepEqual(before, after) {
						t.Fatal("check changed project files")
					}
				}
			})
		}
	}
	if out, diagnostic, code := runCLI(t, binary, t.TempDir(), "project", "init", "--check"); code != 2 || !strings.Contains(diagnostic, "unknown flag") {
		t.Fatal(code, out, diagnostic)
	}
}

// projectFileContents captures all file bytes to detect writes or extra files during read-only CLI checks.
func projectFileContents(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// TestCheckDoesNotInitialize leaves a missing project absent when the actual command fails.
func TestCheckDoesNotInitialize(t *testing.T) {
	directory := t.TempDir()
	out, diagnostic, code := runCLI(t, buildCLI(t), directory, "project", "check")
	if code != 1 {
		t.Fatal(code, out, diagnostic)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatal(entries, err)
	}
}
