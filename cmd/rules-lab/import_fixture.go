// Exercise complete native Git imports against isolated local repositories.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/fabricahq/code-rules/internal/gitfixture"
	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
)

// importFixture gives each declared source editable original files and optional exact binary bytes.
type importFixture struct {
	Configuration json.RawMessage `json:"configuration"`
	Libraries     map[string]struct {
		Files       map[string]string `json:"files"`
		BinaryFiles map[string][]byte `json:"binaryFiles,omitempty"`
	} `json:"libraries"`
	Scenario string `json:"scenario"`
}

// invokeImports creates real tagged repositories, imports every source, and returns readable snapshot files.
func invokeImports(input json.RawMessage) (_ any, err error) {
	var request importFixture
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil {
		return nil, &rules.ValidationError{Location: "fixture", Problem: "expected configuration, libraries, and scenario"}
	}
	if request.Scenario != "import" && request.Scenario != "cancel" && request.Scenario != "symlink" {
		return nil, &rules.ValidationError{Location: "scenario", Problem: "expected import, cancel, or symlink"}
	}
	config, err := rules.ParseConfiguration(request.Configuration)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	fixtures := map[string]*gitfixture.Fixture{}
	defer func() {
		for _, fixture := range fixtures {
			if cleanupErr := fixture.Close(); cleanupErr != nil {
				err = fmt.Errorf("clean up Git fixture: %v", errors.Join(err, cleanupErr))
			}
		}
	}()
	options := imports.Options{}
	var owner *gitfixture.Fixture
	for _, source := range config.Sources {
		entry, exists := request.Libraries[source.Name]
		if !exists || source.Repository != "git@fixture.invalid:"+source.Name {
			return nil, &rules.ValidationError{Location: source.Name, Problem: "provide a library fixture with repository git@fixture.invalid:" + source.Name}
		}
		files := map[string][]byte{}
		for name, text := range entry.Files {
			files[name] = []byte(text)
		}
		for name, data := range entry.BinaryFiles {
			if _, duplicate := files[name]; duplicate {
				return nil, &rules.ValidationError{Location: name, Problem: "file appears in text and binary inputs"}
			}
			files[name] = data
		}
		for name := range files {
			if err := projectFixturePath(name); err != nil {
				return nil, err
			}
		}
		fixture, err := gitfixture.New(ctx, files)
		if err != nil {
			return nil, err
		}
		fixtures[source.Name] = fixture
		if owner == nil {
			owner = fixture
		}
		if request.Scenario == "symlink" {
			if err := os.Symlink("errors.md", filepath.Join(fixture.Directory, "repository", "techs/go/link.md")); err != nil {
				return nil, err
			}
			for _, args := range [][]string{{"add", "--all"}, {"-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "Linked rule"}, {"tag", "-f", "v1.2.0"}} {
				if _, err := fixture.Command(ctx, args...); err != nil {
					return nil, err
				}
			}
		}
	}
	if len(fixtures) != len(request.Libraries) {
		return nil, &rules.ValidationError{Location: "libraries", Problem: "every fixture must have a configured source"}
	}
	if owner != nil {
		options.GitPath = owner.GitPath
		options.Environment, err = owner.Route(fixtures)
		if err != nil {
			return nil, err
		}
	}
	if request.Scenario == "cancel" {
		cancel()
	}
	imported, err := imports.ImportLibraries(ctx, config, options)
	if err != nil {
		return nil, err
	}
	snapshots := map[string]project.Snapshot{}
	catalogs := map[string]library.Catalog{}
	for alias, item := range imported {
		snapshots[alias] = item.Snapshot
		catalogs[alias] = item.Catalog
	}
	vendor, err := project.EncodeSnapshots(config, snapshots)
	if err != nil {
		return nil, err
	}
	display := map[string]any{}
	for name, data := range vendor {
		if utf8.Valid(data) {
			display[name] = string(data)
		} else {
			display[name] = map[string]any{"encoding": "base64", "bytes": data}
		}
	}
	return struct {
		Catalogs    map[string]library.Catalog `json:"catalogs"`
		VendorFiles map[string]any             `json:"vendorFiles"`
	}{catalogs, display}, nil
}
