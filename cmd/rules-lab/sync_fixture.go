// Show native sync changes and complete filesystem observations in a disposable project.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
)

// syncFixture supplies real Git libraries, authored local files, and optional post-seed edits.
type syncFixture struct {
	toolVersion string // Set only by the CLI adapter to match the executable used for setup output.
	importFixture
	LocalFiles map[string]string  `json:"localFiles,omitempty"`
	SeedSync   bool               `json:"seedSync,omitempty"`
	Changes    map[string]*string `json:"changes,omitempty"`
}

// syncResponse keeps typed failures visible alongside files left on disk after the attempted operation.
func syncResponse(input json.RawMessage) (response, error) {
	observed, err := invokeSync(input)
	if err != nil {
		failure := describeProjectError(err)
		if failure == nil {
			failure = gitFailure(err)
		}
		if failure == nil {
			return response{}, err
		}
		return response{Error: failure, Observation: observed}, nil
	}
	return response{OK: true, Value: observed}, nil
}

// invokeSync creates routed local Git fixtures and owns the temporary project for one request.
func invokeSync(input json.RawMessage) (any, error) {
	var fixture syncFixture
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return nil, &rules.ValidationError{Location: "fixture", Problem: "expected a sync fixture"}
	}
	if fixture.Scenario != "sync" && fixture.Scenario != "cancel" {
		return nil, &rules.ValidationError{Location: "scenario", Problem: "expected sync or cancel"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	return withImportFixture(ctx, fixture.importFixture, func(git imports.Options) (any, error) {
		return observeSync(ctx, cancel, fixture, git)
	})
}

// observeSync seeds authored inputs, optionally syncs once, then captures the requested sync and its files.
func observeSync(ctx context.Context, cancel context.CancelFunc, fixture syncFixture, git imports.Options) (any, error) {
	return withSyncProject(ctx, cancel, fixture, git, func(root *os.Root, options project.Options, before *project.Tree) (any, error) {
		changes, operationErr := project.Sync(ctx, options, git)
		after, err := project.ReadTree(context.Background(), root, ".")
		if err != nil {
			return nil, err
		}
		return &offlineObservation{Changes: changes, Before: displayProjectTree(before), After: displayProjectTree(after)}, operationErr
	})
}

// withSyncProject owns fixture setup and confines every configured source before running a reviewed operation.
func withSyncProject(ctx context.Context, cancel context.CancelFunc, fixture syncFixture, git imports.Options, operation func(*os.Root, project.Options, *project.Tree) (any, error)) (any, error) {
	dir, err := os.MkdirTemp("", "rules-lab-sync-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	if err := root.WriteFile("config.json", fixture.Configuration, 0600); err != nil {
		return nil, err
	}
	for file, text := range fixture.LocalFiles {
		if err := projectFixturePath(file); err != nil {
			return nil, err
		}
		if err := writeOfflineFixtureFile(root, "local/"+file, []byte(text)); err != nil {
			return nil, err
		}
	}
	options := project.Options{ConfigPath: filepath.Join(dir, "config.json"), ToolVersion: "go-migration-review"}
	if fixture.toolVersion != "" {
		options.ToolVersion = fixture.toolVersion
	}
	if fixture.SeedSync {
		if _, err := project.Sync(ctx, options, git); err != nil {
			return nil, err
		}
	}
	for file, text := range fixture.Changes {
		if err := projectFixturePath(file); err != nil {
			return nil, err
		}
		if text == nil {
			if err := root.Remove(file); err != nil {
				return nil, err
			}
		} else if err := writeOfflineFixtureFile(root, file, []byte(*text)); err != nil {
			return nil, err
		}
	}
	// Edits may retire sources, but must not route the demo outside its prepared Git fixtures.
	currentConfig, err := root.ReadFile("config.json")
	if err != nil {
		return nil, err
	}
	config, err := rules.ParseConfiguration(currentConfig)
	if err != nil {
		return nil, err
	}
	for _, source := range config.Sources {
		if _, exists := fixture.Libraries[source.Name]; !exists || source.Repository != "git@fixture.invalid:"+source.Name {
			return nil, &rules.ValidationError{Location: "configuration.sources." + source.Name, Problem: "sync walkthrough sources must use their supplied local Git fixture"}
		}
	}
	before, err := project.ReadTree(ctx, root, ".")
	if err != nil {
		return nil, err
	}
	if fixture.Scenario == "cancel" {
		cancel()
	}
	return operation(root, options, before)
}
