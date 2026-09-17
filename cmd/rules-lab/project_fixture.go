// Demonstrate managed writes on disposable project trees and retain observations even on failure.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
)

// projectWriteFixture controls only disposable lab files, never a user-supplied filesystem root.
type projectWriteFixture struct {
	Files    map[string]string                    `json:"files"`
	Output   map[project.Target]map[string]string `json:"output"`
	Scenario string                               `json:"scenario"`
}

// projectObservation displays actual reads before and after the native writer call.
type projectObservation struct {
	Before  map[string]string `json:"before"`
	After   map[string]string `json:"after"`
	Changes []string          `json:"changes"`
}

// projectWriteResponse preserves the function's error alongside separately labelled filesystem observations.
func projectWriteResponse(input json.RawMessage) (response, error) {
	observed, err := invokeProjectWrite(input)
	if err == nil {
		return response{OK: true, Value: observed}, nil
	}
	failure := &failure{Name: "Error", Message: err.Error()}
	var projectErr *project.Error
	if errors.As(err, &projectErr) {
		failure.Name = "ProjectError"
		failure.Code = projectErr.Code
	}
	var validation *rules.ValidationError
	if errors.As(err, &validation) {
		failure.Name = "ValidationError"
		failure.Location = validation.Location
	}
	return response{Error: failure, Observation: observed}, nil
}

// invokeProjectWrite invokes real filesystem writes and captures state before deleting its temporary root.
func invokeProjectWrite(input json.RawMessage) (*projectObservation, error) {
	var fixture projectWriteFixture
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return nil, &rules.ValidationError{Location: "fixture", Problem: "expected a project-write fixture"}
	}
	switch fixture.Scenario {
	case "apply", "repeat", "busy", "concurrentEdit", "renameFailure", "cancel", "recover", "editedRecovery":
	default:
		return nil, &rules.ValidationError{Location: "scenario", Problem: "expected a supported project-write scenario"}
	}
	dir, err := os.MkdirTemp("", "rules-lab-project-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	for file, text := range fixture.Files {
		if err := projectFixturePath(file); err != nil {
			return nil, err
		}
		if err := root.MkdirAll(path.Dir(file), 0700); err != nil {
			return nil, err
		}
		if err := root.WriteFile(file, []byte(text), 0600); err != nil {
			return nil, err
		}
	}
	output := map[project.Target]map[string][]byte{}
	for target, files := range fixture.Output {
		output[target] = map[string][]byte{}
		for file, text := range files {
			output[target][file] = []byte(text)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if fixture.Scenario == "recover" || fixture.Scenario == "editedRecovery" {
		if err := stageRecoveryFixture(root, output[project.Generated]); err != nil {
			return nil, err
		}
		if fixture.Scenario == "editedRecovery" {
			if err := root.WriteFile("generated/RULES.md", []byte("A user edit after interruption.\n"), 0600); err != nil {
				return nil, err
			}
		}
	}
	before, err := project.ReadTree(ctx, root, ".")
	if err != nil {
		return nil, err
	}
	if fixture.Scenario == "recover" || fixture.Scenario == "editedRecovery" {
		err = project.WithWriter(ctx, root, func(*project.Writer) error { return nil })
	} else if fixture.Scenario == "busy" {
		err = project.WithWriter(ctx, root, func(*project.Writer) error {
			return project.WithWriter(ctx, root, func(w *project.Writer) error { return w.Apply(output, nil) })
		})
	} else {
		err = project.WithWriter(ctx, root, func(w *project.Writer) error {
			if fixture.Scenario == "repeat" {
				if err := w.Apply(output, nil); err != nil {
					return err
				}
			}
			return w.Apply(output, func() error {
				switch fixture.Scenario {
				case "concurrentEdit":
					return root.WriteFile("generated/RULES.md", []byte("A later user edit.\n"), 0600)
				case "renameFailure":
					return root.RemoveAll(".code-rules-transaction/new-generated")
				case "cancel":
					cancel()
				}
				return nil
			})
		})
	}
	after, readErr := project.ReadTree(context.Background(), root, ".")
	if readErr != nil {
		return nil, errors.Join(err, readErr)
	}
	observed := &projectObservation{Before: displayProjectTree(before), After: displayProjectTree(after), Changes: []string{}}
	for file, data := range after.Files {
		if old, ok := before.Files[file]; !ok {
			observed.Changes = append(observed.Changes, "added "+file)
		} else if !bytes.Equal(old, data) {
			observed.Changes = append(observed.Changes, "changed "+file)
		}
	}
	for file := range before.Files {
		if _, ok := after.Files[file]; !ok {
			observed.Changes = append(observed.Changes, "removed "+file)
		}
	}
	slices.Sort(observed.Changes)
	return observed, err
}

// displayProjectTree keeps text fixture content readable instead of base64 encoding the observation.
func displayProjectTree(tree *project.Tree) map[string]string {
	out := map[string]string{}
	if tree != nil {
		for name, data := range tree.Files {
			out[name] = string(data)
		}
	}
	return out
}

// fixturePath confines authored lab files and protects writer-owned control paths from fixture setup.
func projectFixturePath(file string) error {
	if !fs.ValidPath(file) || file == "." || strings.ContainsAny(file, "\\\x00:") || strings.HasPrefix(file, ".code-rules-") {
		return &rules.ValidationError{Location: "fixture.files", Problem: fmt.Sprintf("%q: expected a contained project file", file)}
	}
	return nil
}

// stageRecoveryFixture seeds a version-1 interrupted transaction; recovery itself runs native Go.
// Actual process death is covered by the project's child-process Apply regression test.
func stageRecoveryFixture(root *os.Root, files map[string][]byte) error {
	ctx := context.Background()
	before, err := project.ReadTree(ctx, root, "generated")
	if err != nil {
		return err
	}
	tx := ".code-rules-transaction"
	if err := root.Mkdir(tx, 0700); err != nil {
		return err
	}
	staged := tx + "/new-generated"
	if err := root.Mkdir(staged, 0700); err != nil {
		return err
	}
	for file, data := range files {
		if err := projectFixturePath(file); err != nil {
			return err
		}
		if err := root.MkdirAll(path.Dir(staged+"/"+file), 0700); err != nil {
			return err
		}
		if err := root.WriteFile(staged+"/"+file, data, 0600); err != nil {
			return err
		}
	}
	after, err := project.ReadTree(ctx, root, staged)
	if err != nil {
		return err
	}
	record := map[string]any{"formatVersion": 1, "entries": []any{map[string]any{"name": "generated", "before": before.Digest(), "after": after.Digest(), "existed": before != nil}}}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err := root.WriteFile(tx+"/journal.json", append(data, '\n'), 0600); err != nil {
		return err
	}
	if before != nil {
		if err := root.Rename("generated", tx+"/old-generated"); err != nil {
			return err
		}
	}
	return root.Rename(staged, "generated")
}
