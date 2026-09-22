// Exercise packaged native binaries after extraction outside their source checkout.

package distribution

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"github.com/fabricahq/code-rules/internal/test/gitfixture"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNativeInstallUpgradeRollback runs two packaged versions with an empty PATH and keeps both installations usable.
func TestNativeInstallUpgradeRollback(t *testing.T) {
	t.Setenv("GOAMD64", "v3")
	t.Setenv("GOARM64", "v8.2")
	source, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	target := runtime.GOOS + "/" + runtime.GOARCH
	for _, version := range []string{"0.1.0-review.1", "0.1.0-review.2"} {
		output := filepath.Join(parent, "artifacts-"+version)
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		relativeOutput, err := filepath.Rel(cwd, output)
		if err != nil {
			t.Fatal(err)
		}
		manifest, err := Build(context.Background(), Options{Source: source, Output: relativeOutput, Version: version, Targets: []string{target}, Candidate: true})
		if err != nil {
			t.Fatal(err)
		}
		if len(manifest.Artifacts) != 1 || manifest.Artifacts[0].Bytes == 0 {
			t.Fatal(manifest)
		}
		installed := filepath.Join(parent, "installed-"+version)
		if _, err := Install(output, target, installed); err != nil {
			t.Fatal(err)
		}
		binary := filepath.Join(installed, "code-rules")
		info, err := buildinfo.ReadFile(binary)
		if err != nil {
			t.Fatal(err)
		}
		for _, setting := range info.Settings {
			if setting.Key == "GOAMD64" && setting.Value != "v1" || setting.Key == "GOARM64" && setting.Value != "v8.0" {
				t.Fatalf("ambient target tuning leaked: %+v", setting)
			}
		}
		project := filepath.Join(parent, "project-"+version)
		if err := os.Mkdir(project, 0700); err != nil {
			t.Fatal(err)
		}
		run := func(args ...string) string {
			t.Helper()
			cmd := exec.Command(binary, args...)
			cmd.Dir = project
			cmd.Env = []string{"PATH=" + parent + "/no-runtime"}
			var diagnostic bytes.Buffer
			cmd.Stderr = &diagnostic
			data, err := cmd.Output()
			if err != nil {
				t.Fatal(args, err, string(data), diagnostic.String())
			}
			return string(data)
		}
		if got := strings.TrimSpace(run("--version")); got != version {
			t.Fatal(got)
		}
		run("project", "init")
		run("project", "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When editing Go.")
		if err := os.WriteFile(filepath.Join(project, "body.md"), []byte("Return failures to the caller.\n"), 0600); err != nil {
			t.Fatal(err)
		}
		run("project", "add", "rule", "techs/go/errors", "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions.", "--body-file", "body.md")
		run("project", "build")
		run("project", "check")
		if _, err := Install(output, target, installed); err == nil {
			t.Fatal("overwrote an existing installation")
		}
		archive := filepath.Join(output, manifest.Artifacts[0].File)
		if err := os.WriteFile(archive, []byte("corrupt"), 0600); err != nil {
			t.Fatal(err)
		}
		rejected := filepath.Join(parent, "rejected-"+version)
		if _, err := Install(output, target, rejected); err == nil {
			t.Fatal("accepted corrupt artifact")
		}
		if _, err := os.Stat(rejected); !os.IsNotExist(err) {
			t.Fatal("corrupt archive changed install", err)
		}
	}
	// Rollback selects the retained previous binary, without a runtime download or reinstall.
	previous := exec.Command(filepath.Join(parent, "installed-0.1.0-review.1/code-rules"), "--version")
	previous.Env = []string{"PATH=" + parent + "/no-runtime"}
	data, err := previous.Output()
	if err != nil || strings.TrimSpace(string(data)) != "0.1.0-review.1" {
		t.Fatal(string(data), err)
	}
}

// TestPackagingRefusesUnapprovedRelease rejects missing tool terms before creating an output directory.
func TestPackagingRefusesUnapprovedRelease(t *testing.T) {
	fixture, err := gitfixture.New(context.Background(), map[string][]byte{"README.md": []byte("Fixture")})
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	source := filepath.Join(fixture.Directory, "repository")
	output := filepath.Join(t.TempDir(), "release")
	if _, err := Build(context.Background(), Options{Source: source, Output: output, Version: "0.1.0"}); err == nil || !strings.Contains(err.Error(), "LICENSE.md") {
		t.Fatal("expected licensing gate")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("release gate wrote files", err)
	}
}

// TestCandidateBuildExcludesUncommittedEdits verifies that a manifest's recorded commit owns the executable inputs.
func TestCandidateBuildExcludesUncommittedEdits(t *testing.T) {
	fixture, err := gitfixture.New(context.Background(), map[string][]byte{
		"go.mod":                 []byte("module example.invalid/fixture\n\ngo 1.27.1\n"),
		"cmd/code-rules/main.go": []byte("package main\nimport \"fmt\"\nvar version string\nfunc main(){fmt.Println(version)}\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	source := filepath.Join(fixture.Directory, "repository")
	if err := os.WriteFile(filepath.Join(source, "cmd/code-rules/main.go"), []byte("invalid uncommitted Go source"), 0600); err != nil {
		t.Fatal(err)
	}
	artifacts := filepath.Join(t.TempDir(), "artifacts")
	target := runtime.GOOS + "/" + runtime.GOARCH
	manifest, err := Build(context.Background(), Options{Source: source, Output: artifacts, Candidate: true, Targets: []string{target}})
	if err != nil {
		t.Fatal(err)
	}
	if !manifest.SourceDirty || manifest.SourceRevision != fixture.LatestCommit {
		t.Fatal(manifest)
	}
	installed := filepath.Join(t.TempDir(), "installed")
	if _, err := Install(artifacts, target, installed); err != nil {
		t.Fatal(err)
	}
	data, err := exec.Command(filepath.Join(installed, "code-rules")).Output()
	if err != nil || strings.TrimSpace(string(data)) != "0.0.0-dev.g"+fixture.LatestCommit[:12] {
		t.Fatal(string(data), err)
	}
}

// TestLicensedRelease packages an explicit version without any package metadata file.
func TestLicensedRelease(t *testing.T) {
	fixture, err := gitfixture.New(context.Background(), map[string][]byte{
		"LICENSE.md":             []byte("MIT fixture terms\n"),
		"go.mod":                 []byte("module example.invalid/fixture\n\ngo 1.27.1\n"),
		"cmd/code-rules/main.go": []byte("package main\nimport \"fmt\"\nvar version string\nfunc main(){fmt.Println(version)}\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	source := filepath.Join(fixture.Directory, "repository")
	target := runtime.GOOS + "/" + runtime.GOARCH
	output := filepath.Join(t.TempDir(), "release")
	manifest, err := Build(context.Background(), Options{Source: source, Output: output, Version: "0.1.0", Targets: []string{target}})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "0.1.0" || manifest.Candidate || manifest.SourceDirty || manifest.License != "SEE LICENSE.md" {
		t.Fatal(manifest)
	}
	installed := filepath.Join(t.TempDir(), "installed")
	if _, err := Install(output, target, installed); err != nil {
		t.Fatal(err)
	}
	data, err := exec.Command(filepath.Join(installed, "code-rules")).Output()
	if err != nil || strings.TrimSpace(string(data)) != "0.1.0" {
		t.Fatal(string(data), err)
	}
}
