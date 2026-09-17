// Inspect and run verified native artifacts without using the checkout executable at runtime.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fabricahq/code-rules/internal/distribution"
	"github.com/fabricahq/code-rules/internal/project"
)

// packageFixture selects a bounded artifact installation or integrity failure scenario.
type packageFixture struct {
	Scenario string `json:"scenario"`
}

// packageObservation combines actual installed-command results with readable artifact metadata.
type packageObservation struct {
	authoringObservation
	Artifacts map[string]string `json:"artifacts"`
}

// packageResponse verifies the trusted sibling artifact collection and runs the extracted host binary with an empty PATH.
func packageResponse(input json.RawMessage) (response, error) {
	executable, err := os.Executable()
	if err != nil {
		return response{}, err
	}
	return packageFromArtifacts(input, filepath.Join(filepath.Dir(executable), "native-artifacts"))
}

// packageFromArtifacts preserves collection-level completion checks before copying a host archive into the disposable lab.
func packageFromArtifacts(input json.RawMessage, artifacts string) (response, error) {
	var fixture packageFixture
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return adapterError("expected a packaging scenario"), nil
	}
	if fixture.Scenario != "install" && fixture.Scenario != "corrupt" && fixture.Scenario != "existing" && fixture.Scenario != "incomplete" {
		return adapterError("use install, corrupt, existing, or incomplete"), nil
	}
	if _, err := os.Lstat(filepath.Join(artifacts, "INCOMPLETE")); !os.IsNotExist(err) {
		return response{Error: &failure{Name: "ArtifactError", Message: "artifact build is incomplete or unreadable"}}, nil
	}
	data, err := os.ReadFile(filepath.Join(artifacts, "manifest.json"))
	if err != nil {
		return adapterError("build native candidate artifacts into native-artifacts beside rules-lab before this walkthrough"), nil
	}
	var manifest distribution.Manifest
	if err = json.Unmarshal(data, &manifest); err != nil {
		return response{}, err
	}
	target := runtime.GOOS + "/" + runtime.GOARCH
	var selected distribution.Artifact
	for _, artifact := range manifest.Artifacts {
		if artifact.Target == target {
			selected = artifact
		}
	}
	if selected.File == "" || filepath.Base(selected.File) != selected.File {
		return adapterError("no host artifact in manifest"), nil
	}
	temporary, err := os.MkdirTemp("", "rules-lab-package-")
	if err != nil {
		return response{}, err
	}
	defer os.RemoveAll(temporary)
	copied := filepath.Join(temporary, "artifacts")
	if err = os.Mkdir(copied, 0700); err != nil {
		return response{}, err
	}
	if err = os.WriteFile(filepath.Join(copied, "manifest.json"), data, 0600); err != nil {
		return response{}, err
	}
	archive, err := os.ReadFile(filepath.Join(artifacts, selected.File))
	if err != nil {
		return response{}, err
	}
	if fixture.Scenario == "corrupt" {
		archive = []byte("Changed archive bytes")
	}
	if err = os.WriteFile(filepath.Join(copied, selected.File), archive, 0600); err != nil {
		return response{}, err
	}
	if fixture.Scenario == "incomplete" {
		if err = os.WriteFile(filepath.Join(copied, "INCOMPLETE"), []byte("Build interrupted"), 0600); err != nil {
			return response{}, err
		}
	}
	installed := filepath.Join(temporary, "installed")
	if _, err = distribution.Install(copied, target, installed); err != nil {
		return response{Error: &failure{Name: "ArtifactError", Message: err.Error()}}, nil
	}
	if fixture.Scenario == "existing" {
		_, err = distribution.Install(copied, target, installed)
		return response{Error: &failure{Name: "ArtifactError", Message: err.Error()}}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	projectDir := filepath.Join(temporary, "project")
	if err = os.Mkdir(projectDir, 0700); err != nil {
		return response{}, err
	}
	root, err := os.OpenRoot(projectDir)
	if err != nil {
		return response{}, err
	}
	defer root.Close()
	observed := packageObservation{authoringObservation: authoringObservation{Before: map[string]string{}, After: map[string]string{}, Commands: []*cliObservation{}}, Artifacts: map[string]string{"manifest.json": string(data), "installed/code-rules": fmt.Sprintf("Verified %s executable: %d bytes. No Node/Bun interpreter.", target, selected.BinaryBytes)}}
	for _, name := range []string{"README.txt", "LICENSE.md"} {
		content, err := os.ReadFile(filepath.Join(installed, name))
		if err == nil {
			observed.Artifacts["installed/"+name] = string(content)
		} else if !os.IsNotExist(err) {
			return response{}, err
		}
	}
	for _, args := range [][]string{{"--version"}, {"init"}, {"build"}, {"check"}} {
		command := exec.CommandContext(ctx, filepath.Join(installed, "code-rules"), args...)
		command.Dir = projectDir
		command.Env = []string{"PATH=" + temporary + "/no-runtime"}
		var out, diagnostic bytes.Buffer
		command.Stdout = &out
		command.Stderr = &diagnostic
		err := command.Run()
		code := 0
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				return response{}, err
			}
		}
		observed.Commands = append(observed.Commands, &cliObservation{Arguments: args, Stdout: out.String(), Stderr: diagnostic.String(), ExitCode: code})
		if code != 0 {
			break
		}
	}
	tree, err := project.ReadTree(ctx, root, ".")
	if err != nil {
		return response{}, err
	}
	observed.After = displayProjectTree(tree)
	return response{OK: true, Value: observed}, nil
}
