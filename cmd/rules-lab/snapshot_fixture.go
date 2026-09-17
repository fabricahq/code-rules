// Exercise native snapshot encoding and verification without writing project files.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
)

// snapshotFixture exposes authored files and explicit post-encoding changes for integrity scenarios.
type snapshotFixture struct {
	Configuration json.RawMessage    `json:"configuration"`
	Files         map[string]*string `json:"files"`
	Commit        string             `json:"commit"`
	Tag           string             `json:"tag,omitempty"`
	Groups        []string           `json:"groups"`
	// BinaryFiles uses JSON base64 strings to preserve non-text bytes.
	BinaryFiles   map[string][]byte          `json:"binaryFiles,omitempty"`
	FileChanges   map[string]*string         `json:"fileChanges,omitempty"`
	RecordChanges map[string]json.RawMessage `json:"recordChanges,omitempty"`
}

// invokeSnapshots encodes one source and verifies optional corruption using the real project package.
func invokeSnapshots(input json.RawMessage) (any, error) {
	var fixture snapshotFixture
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return nil, &rules.ValidationError{Location: "fixture", Problem: "expected a snapshot fixture"}
	}
	config, err := rules.ParseConfiguration(fixture.Configuration)
	if err != nil {
		return nil, err
	}
	if len(config.Sources) != 1 {
		return nil, &rules.ValidationError{Location: "fixture", Problem: "use exactly one source in this walkthrough"}
	}
	source := config.Sources[0]
	files := map[string][]byte{}
	for file, text := range fixture.Files {
		if text == nil {
			return nil, &rules.ValidationError{Location: file, Problem: "file contents must be text"}
		}
		files[file] = []byte(*text)
	}
	for file, data := range fixture.BinaryFiles {
		if _, ok := files[file]; ok {
			return nil, &rules.ValidationError{Location: file, Problem: "file appears in both text and binary inputs"}
		}
		files[file] = data
	}
	version := ""
	if fixture.Tag != "" {
		version, err = rules.TagVersion(fixture.Tag, "fixture.tag")
		if err != nil {
			return nil, err
		}
	}
	snapshot := project.Snapshot{Repository: source.Repository, Ref: source.Ref, Version: source.Version, Tag: fixture.Tag, ResolvedVersion: version, Commit: fixture.Commit, Selection: source.Groups, Groups: fixture.Groups, Files: files}
	vendor, err := project.EncodeSnapshots(config, map[string]project.Snapshot{source.Name: snapshot})
	if err != nil {
		return nil, err
	}
	recordPath := source.Name + "/_source.json"
	if len(fixture.RecordChanges) > 0 {
		var record map[string]json.RawMessage
		if err := json.Unmarshal(vendor[recordPath], &record); err != nil {
			return nil, fmt.Errorf("read fixture record: %v", err)
		}
		for key, value := range fixture.RecordChanges {
			if bytes.Equal(value, []byte("null")) {
				delete(record, key)
			} else {
				record[key] = value
			}
		}
		vendor[recordPath], err = json.MarshalIndent(record, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("edit fixture record: %v", err)
		}
	}
	for file, text := range fixture.FileChanges {
		if text == nil {
			delete(vendor, file)
		} else {
			vendor[file] = []byte(*text)
		}
	}
	decoded, err := project.DecodeSnapshots(config, vendor)
	if err != nil {
		return nil, err
	}
	// The review surface preserves readable text and explicitly labels binary bytes.
	display := map[string]any{}
	for file, data := range vendor {
		if utf8.Valid(data) {
			display[file] = string(data)
		} else {
			display[file] = map[string]any{"encoding": "base64", "bytes": data}
		}
	}
	return struct {
		Files    map[string]any              `json:"vendorFiles"`
		Verified map[string]project.Snapshot `json:"verifiedSnapshots"`
	}{display, decoded}, nil
}
