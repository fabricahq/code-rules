// Exercise real Git revision fetching through a disposable local SSH transport.

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

	"github.com/fabricahq/code-rules/internal/gitfixture"
	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/rules"
)

// gitRevisionFixture changes a selector or a named failure while keeping transport local and isolated.
type gitRevisionFixture struct {
	Files    map[string]string `json:"files"`
	Ref      string            `json:"ref,omitempty"`
	Version  string            `json:"version,omitempty"`
	Scenario string            `json:"scenario"`
}

// invokeGitRevision builds a real repository, fetches it, and closes every owned resource before returning.
func invokeGitRevision(input json.RawMessage) (_ any, err error) {
	var request gitRevisionFixture
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil {
		return nil, &rules.ValidationError{Location: "fixture", Problem: "expected files, selector, and Git scenario"}
	}
	switch request.Scenario {
	case "fetch", "commit", "ambiguous", "moved", "missingGit", "cancel", "timeout", "overflow":
	default:
		return nil, &rules.ValidationError{Location: "scenario", Problem: "unknown Git revision scenario"}
	}
	files := map[string][]byte{}
	for name, text := range request.Files {
		if err := projectFixturePath(name); err != nil {
			return nil, err
		}
		files[name] = []byte(text)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	fixture, err := gitfixture.New(ctx, files)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, fixture.Close()) }()
	source := rules.Source{Name: "team", Repository: fixture.Repository, Ref: request.Ref, Version: request.Version}
	options := imports.Options{GitPath: fixture.GitPath, Environment: fixture.Environment}
	switch request.Scenario {
	case "commit":
		source.Ref = fixture.FirstCommit
		source.Version = ""
	case "ambiguous":
		if _, err := fixture.Command(ctx, "tag", "1.2.0", fixture.FirstCommit); err != nil {
			return nil, err
		}
	case "moved":
		marker := filepath.Join(fixture.Directory, "served")
		repository := filepath.Join(fixture.Directory, "repository")
		script := "#!/bin/sh\nif [ -f " + gitfixture.Quote(marker) + " ]; then " + gitfixture.Quote(fixture.GitPath) + " -C " + gitfixture.Quote(repository) + " tag -f v1.2.0 " + fixture.FirstCommit + " >/dev/null; fi\ntouch " + gitfixture.Quote(marker) + "\nexec " + gitfixture.Quote(fixture.GitPath) + " upload-pack " + gitfixture.Quote(repository) + "\n"
		if err := os.WriteFile(filepath.Join(fixture.Directory, "ssh"), []byte(script), 0700); err != nil {
			return nil, err
		}
	case "missingGit":
		options.GitPath = filepath.Join(fixture.Directory, "missing-git")
	case "cancel":
		cancel()
	case "timeout", "overflow":
		helper := filepath.Join(fixture.Directory, "git-fault")
		body := "sleep 30 &\nwait\n"
		if request.Scenario == "timeout" {
			options.Timeout = 50 * time.Millisecond
		} else {
			body = "while :; do printf diagnostic >&2; done\n"
		}
		if err := os.WriteFile(helper, []byte("#!/bin/sh\n"+body), 0700); err != nil {
			return nil, err
		}
		options.GitPath = helper
	}
	revision, err := imports.FetchRevision(ctx, source, options)
	if err != nil {
		return nil, err
	}
	if err := revision.Close(); err != nil {
		return nil, err
	}
	return struct {
		Revision                   *imports.Revision `json:"revision"`
		First                      string            `json:"firstCommit"`
		Latest                     string            `json:"latestCommit"`
		TemporaryRepositoryRemoved bool              `json:"temporaryRepositoryRemoved"`
	}{revision, fixture.FirstCommit, fixture.LatestCommit, true}, nil
}

// gitFailure exposes typed native import errors while keeping unexpected fixture failures at HTTP 500.
func gitFailure(err error) *failure {
	var importErr *imports.Error
	if errors.As(err, &importErr) {
		return &failure{Name: "ImportError", Code: importErr.Code, Message: fmt.Sprint(importErr)}
	}
	return nil
}
