// Compose real parsers and catalog loading for build walkthrough requests.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/rules"
)

// buildFixture keeps all walkthrough input inside disposable libraries and in-memory local files.
type buildFixture struct {
	Configuration json.RawMessage `json:"configuration"`
	Libraries     map[string]struct {
		Files  map[string]*string `json:"files"`
		Commit string             `json:"commit"`
	} `json:"libraries"`
	LocalFiles map[string]*string `json:"localFiles"`
}

// resolveBuildFixture parses configuration and loads each selected library before rule resolution.
func resolveBuildFixture(input json.RawMessage) (build.Resolved, error) {
	var fixture buildFixture
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return build.Resolved{}, err
	}
	config, err := rules.ParseConfiguration(fixture.Configuration)
	if err != nil {
		return build.Resolved{}, err
	}
	supplied := map[string]build.Library{}
	for _, source := range config.Sources {
		entry, ok := fixture.Libraries[source.Name]
		if !ok {
			return build.Resolved{}, fmt.Errorf("missing fixture library %s", source.Name)
		}
		groups, err := json.Marshal(source.Groups)
		if err != nil {
			return build.Resolved{}, err
		}
		files := map[string]string{}
		for file, text := range entry.Files {
			if text == nil {
				return build.Resolved{}, fmt.Errorf("%s: fixture contents must be a string", file)
			}
			files[file] = *text
		}
		catalog, err := readLibraryFixture(libraryFixture{Files: files, Groups: groups, Source: source.Name})
		if err != nil {
			return build.Resolved{}, err
		}
		supplied[source.Name] = build.Library{Catalog: catalog, Commit: entry.Commit}
	}
	for alias := range fixture.Libraries {
		if _, ok := supplied[alias]; !ok {
			return build.Resolved{}, fmt.Errorf("undeclared fixture library %s", alias)
		}
	}
	local := map[string][]byte{}
	for file, text := range fixture.LocalFiles {
		if text == nil {
			return build.Resolved{}, fmt.Errorf("%s: fixture contents must be a string", file)
		}
		local[file] = []byte(*text)
	}
	return build.Resolve(config, supplied, local)
}
