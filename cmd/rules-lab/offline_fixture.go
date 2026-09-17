// Exercise offline project operations against disposable authored and imported fixtures.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
)

// offlineFixture supplies readable libraries and local files, plus optional changes after a seed build.
type offlineFixture struct {
	buildFixture
	Action     string             `json:"action"`
	ConfigName string             `json:"configName,omitempty"`
	SeedBuild  bool               `json:"seedBuild,omitempty"`
	Changes    map[string]*string `json:"changes,omitempty"`
}

// offlineObservation separates operation changes from lab-observed before/after filesystem content.
type offlineObservation struct {
	Changes project.FileChanges `json:"changes"`
	Before  map[string]string   `json:"before"`
	After   map[string]string   `json:"after"`
}

// offlineResponse renders the exact operation error with separately labelled filesystem observations.
func offlineResponse(input json.RawMessage) (response, error) {
	observed, err := invokeOffline(input)
	if err != nil {
		failure := describeProjectError(err)
		if failure == nil {
			return response{}, err
		}
		return response{Error: failure, Observation: observed}, nil
	}
	return response{OK: true, Value: observed}, nil
}

// invokeOffline seeds an isolated project then calls native Build or Check and reads the result.
func invokeOffline(input json.RawMessage) (*offlineObservation, error) {
	var fixture offlineFixture
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return nil, &rules.ValidationError{Location: "fixture", Problem: "expected an offline project fixture"}
	}
	if fixture.Action != "build" && fixture.Action != "check" {
		return nil, &rules.ValidationError{Location: "action", Problem: "expected build or check"}
	}
	config, err := rules.ParseConfiguration(fixture.Configuration)
	if err != nil {
		return nil, err
	}
	name := fixture.ConfigName
	if name == "" {
		name = "config.json"
	}
	if err := projectFixturePath(name); err != nil {
		return nil, err
	}
	if filepath.Base(name) != name {
		return nil, &rules.ValidationError{Location: "configName", Problem: "use a filename inside the disposable project"}
	}
	dir, err := os.MkdirTemp("", "rules-lab-offline-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	if err := root.WriteFile(name, fixture.Configuration, 0600); err != nil {
		return nil, err
	}
	for file, text := range fixture.LocalFiles {
		if err := projectFixturePath(file); err != nil {
			return nil, err
		}
		if text == nil {
			return nil, &rules.ValidationError{Location: file, Problem: "expected text file contents"}
		}
		if err := writeOfflineFixtureFile(root, "local/"+file, []byte(*text)); err != nil {
			return nil, err
		}
	}
	snapshots := map[string]project.Snapshot{}
	for _, source := range config.Sources {
		entry, ok := fixture.Libraries[source.Name]
		if !ok {
			return nil, &rules.ValidationError{Location: source.Name, Problem: "missing fixture library"}
		}
		textFiles := map[string]string{}
		for file, text := range entry.Files {
			if text == nil {
				return nil, &rules.ValidationError{Location: file, Problem: "expected text file contents"}
			}
			textFiles[file] = *text
		}
		groups, err := json.Marshal(source.Groups)
		if err != nil {
			return nil, err
		}
		catalog, err := readLibraryFixture(libraryFixture{Files: textFiles, Groups: groups, Source: source.Name})
		if err != nil {
			return nil, err
		}
		snapshot := project.Snapshot{Repository: source.Repository, Ref: source.Ref, Version: source.Version, Commit: entry.Commit, Tag: entry.Tag, Selection: source.Groups, Groups: []string{}, Files: map[string][]byte{}}
		if entry.Tag != "" {
			snapshot.ResolvedVersion, err = rules.TagVersion(entry.Tag, "fixture.tag")
			if err != nil {
				return nil, err
			}
		}
		for _, group := range catalog.Groups {
			snapshot.Groups = append(snapshot.Groups, group.ID)
		}
		for _, file := range catalog.Paths() {
			snapshot.Files[file] = []byte(textFiles[file])
		}
		snapshots[source.Name] = snapshot
	}
	vendor, err := project.EncodeSnapshots(config, snapshots)
	if err != nil {
		return nil, err
	}
	for file, data := range vendor {
		if err := writeOfflineFixtureFile(root, "vendor/"+file, data); err != nil {
			return nil, err
		}
	}
	options := project.Options{ConfigPath: filepath.Join(dir, name), ToolVersion: "go-migration-review"}
	ctx := context.Background()
	if fixture.SeedBuild {
		if _, err := project.Build(ctx, options); err != nil {
			return nil, fmt.Errorf("seed build: %w", err)
		}
	}
	for file, text := range fixture.Changes {
		// The pending-recovery preset may seed a journal, but never a path outside this disposable root.
		if !fixturePath(file) {
			return nil, &rules.ValidationError{Location: file, Problem: "expected a contained relative fixture path"}
		}
		if text == nil {
			if err := root.Remove(file); err != nil {
				return nil, err
			}
		} else if err := writeOfflineFixtureFile(root, file, []byte(*text)); err != nil {
			return nil, err
		}
	}
	before, err := project.ReadTree(ctx, root, ".")
	if err != nil {
		return nil, err
	}
	var changes project.FileChanges
	if fixture.Action == "build" {
		changes, err = project.Build(ctx, options)
	} else {
		changes, err = project.Check(ctx, options)
	}
	after, readErr := project.ReadTree(ctx, root, ".")
	if readErr != nil {
		return nil, readErr
	}
	return &offlineObservation{Changes: changes, Before: displayProjectTree(before), After: displayProjectTree(after)}, err
}

// writeOfflineFixtureFile writes only setup data inside an already isolated root.
func writeOfflineFixtureFile(root *os.Root, file string, data []byte) error {
	if err := root.MkdirAll(filepath.Dir(file), 0700); err != nil {
		return err
	}
	return root.WriteFile(file, data, 0600)
}
