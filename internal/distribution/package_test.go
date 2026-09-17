// Exercise packaged native binaries after extraction outside their source checkout.

package distribution

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNativeInstallUpgradeRollback runs two packaged versions with an empty PATH and keeps both installations usable.
func TestNativeInstallUpgradeRollback(t *testing.T) {
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
		project := filepath.Join(parent, "project-"+version)
		if err := os.Mkdir(project, 0700); err != nil {
			t.Fatal(err)
		}
		run := func(args ...string) string {
			t.Helper()
			cmd := exec.Command(binary, args...)
			cmd.Dir = project
			cmd.Env = []string{"PATH=" + parent + "/no-runtime"}
			data, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatal(args, err, string(data))
			}
			return string(data)
		}
		if got := strings.TrimSpace(run("--version")); got != version {
			t.Fatal(got)
		}
		run("init")
		run("local", "add", "group", "techs/go", "--name", "Go", "--description", "Go guidance.", "--when-to-read", "When editing Go.")
		if err := os.WriteFile(filepath.Join(project, "body.md"), []byte("Return failures to the caller.\n"), 0600); err != nil {
			t.Fatal(err)
		}
		run("local", "add", "rule", "techs/go/errors", "--title", "Return errors", "--impact", "HIGH", "--impact-description", "Preserve failures.", "--when-to-read", "When calling functions.", "--body-file", "body.md")
		run("build")
		run("check")
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
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "package.json"), []byte(`{"version":"1.0.0","license":"UNLICENSED"}`), 0600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "release")
	if _, err := Build(context.Background(), Options{Source: source, Output: output}); err == nil {
		t.Fatal("expected licensing gate")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("release gate wrote files", err)
	}
}
