// Exercise repository-root initialization and subdirectory commands through the real executable.

package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func locationGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = directory
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "GIT_") {
			cmd.Env = append(cmd.Env, item)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid", "GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func TestInitRequiresRepositoryRoot(t *testing.T) {
	binary := buildCLI(t)
	for _, scope := range []string{"project", "library"} {
		t.Run(scope, func(t *testing.T) {
			directory := t.TempDir()
			locationGit(t, directory, "init", "--quiet", "--template=")
			child := filepath.Join(directory, "src")
			if err := os.Mkdir(child, 0700); err != nil {
				t.Fatal(err)
			}
			for _, structured := range []bool{false, true} {
				args := []string{scope, "init"}
				if structured {
					args = append(args, "--json")
				}
				out, diagnostic, code := runCLI(t, binary, child, args...)
				if code != 1 {
					t.Fatalf("init succeeded below repository root: %d %s %s", code, out, diagnostic)
				}
				message := diagnostic
				if structured {
					var result response
					if err := json.Unmarshal([]byte(out), &result); err != nil || result.Error == nil || diagnostic != "" {
						t.Fatal(out, diagnostic, err)
					}
					message = result.Error.Message
					if result.Error.Code != "not-repository-root" {
						t.Fatal(out)
					}
				}
				if !strings.Contains(message, "repository root") || !strings.Contains(message, "cd ") || !strings.Contains(message, scope+" init") {
					t.Fatal(message)
				}
			}
			entries, err := os.ReadDir(child)
			if err != nil || len(entries) != 0 {
				t.Fatal("init created nested files", entries, err)
			}
			if scope == "library" {
				target := filepath.Join(child, "not-created")
				out, diagnostic, code := runCLI(t, binary, directory, "library", "init", "--directory", target)
				if code != 1 || !strings.Contains(diagnostic, "repository root") {
					t.Fatal(code, out, diagnostic)
				}
				if _, err := os.Stat(target); !os.IsNotExist(err) {
					t.Fatal("created target", err)
				}
			}
			if out, diagnostic, code := runCLI(t, binary, directory, scope, "init"); code != 0 {
				t.Fatal(code, out, diagnostic)
			}
		})
	}
}

func TestCommandsDiscoverRepositoryRoot(t *testing.T) {
	binary := buildCLI(t)
	for _, scope := range []string{"project", "library"} {
		t.Run(scope, func(t *testing.T) {
			root := t.TempDir()
			locationGit(t, root, "init", "--quiet", "--template=")
			child := filepath.Join(root, "src", "nested")
			if err := os.MkdirAll(child, 0700); err != nil {
				t.Fatal(err)
			}
			if out, diagnostic, code := runCLI(t, binary, root, scope, "init"); code != 0 {
				t.Fatal(code, out, diagnostic)
			}
			run := func(args ...string) {
				t.Helper()
				out, diagnostic, code := runCLI(t, binary, child, append([]string{scope}, args...)...)
				if code != 0 {
					t.Fatal(args, code, out, diagnostic)
				}
			}
			run("add", "group", "techs/go", "--name", "Go", "--description", "Go rules", "--when-to-read", "Writing Go")
			if err := os.WriteFile(filepath.Join(child, "body.md"), []byte("Return errors to callers.\n"), 0600); err != nil {
				t.Fatal(err)
			}
			run("add", "rule", "techs/go/errors", "--title", "Errors", "--when-to-read", "Writing Go", "--impact", "HIGH", "--impact-description", "Avoid lost failures", "--body-file", "body.md")
			if scope == "project" {
				run("build")
				run("sync")
			}
			run("check")
			prefix := root
			if scope == "project" {
				prefix = filepath.Join(root, ".code-rules", "local")
			}
			data, err := os.ReadFile(filepath.Join(prefix, "techs/go/errors.md"))
			if err != nil || !strings.Contains(string(data), "Return errors to callers.") {
				t.Fatal(string(data), err)
			}
			if _, err := os.Stat(filepath.Join(child, ".code-rules")); !os.IsNotExist(err) {
				t.Fatal("nested project created", err)
			}
		})
	}
}

func TestRepositoryBoundariesAndWorktrees(t *testing.T) {
	binary := buildCLI(t)
	outer := t.TempDir()
	locationGit(t, outer, "init", "--quiet", "--template=")
	locationGit(t, outer, "commit", "--quiet", "--allow-empty", "-m", "fixture")
	if out, diagnostic, code := runCLI(t, binary, outer, "project", "init"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	worktree := filepath.Join(t.TempDir(), "worktree")
	locationGit(t, outer, "worktree", "add", "--detach", worktree)
	nested := filepath.Join(outer, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	locationGit(t, nested, "init", "--quiet", "--template=")
	submodule := filepath.Join(outer, "module")
	locationGit(t, outer, "-c", "protocol.file.allow=always", "submodule", "add", outer, "module")
	for _, root := range []string{worktree, nested, submodule} {
		child := filepath.Join(root, "src")
		if err := os.Mkdir(child, 0700); err != nil {
			t.Fatal(err)
		}
		out, diagnostic, code := runCLI(t, binary, child, "project", "build", "--json")
		var result response
		if json.Unmarshal([]byte(out), &result) != nil || code != 1 || result.Error == nil || result.Error.Code != "needs-init" || diagnostic != "" {
			t.Fatal(code, out, diagnostic)
		}
		physical, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Error.Message, physical) {
			t.Fatal("missing root in recovery", out)
		}
		if out, diagnostic, code := runCLI(t, binary, root, "project", "init"); code != 0 {
			t.Fatal(code, out, diagnostic)
		}
		if out, diagnostic, code := runCLI(t, binary, child, "project", "build"); code != 0 {
			t.Fatal(code, out, diagnostic)
		}
	}
}

func TestLibraryMissingDirectoryDoesNotSelectAncestor(t *testing.T) {
	binary := buildCLI(t)
	root := t.TempDir()
	locationGit(t, root, "init", "--quiet", "--template=")
	if out, diagnostic, code := runCLI(t, binary, root, "library", "init"); code != 0 {
		t.Fatal(code, out, diagnostic)
	}
	before := projectFileContents(t, root)
	for _, args := range [][]string{
		{"library", "check"},
		{"library", "add", "group", "techs/go", "--name", "Go", "--description", "Go rules", "--when-to-read", "Writing Go"},
		{"library", "add", "rule", "techs/go/errors"},
	} {
		args = append(args, "--directory", filepath.Join(root, "does-not-exist"), "--json")
		out, diagnostic, code := runCLI(t, binary, root, args...)
		var result response
		if json.Unmarshal([]byte(out), &result) != nil || code != 1 || result.Error == nil || !strings.Contains(result.Error.Message, "does not exist") || diagnostic != "" {
			t.Fatal(args, code, out, diagnostic)
		}
		if !reflect.DeepEqual(before, projectFileContents(t, root)) {
			t.Fatal("changed ancestor library")
		}
	}
}
