// Exercise complete generation through the package interface, including ownership and failure results.

package build_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/rules"
)

func TestGenerateCompleteOutput(t *testing.T) {
	config, files := generationInputs(t)
	before, err := json.Marshal(files)
	if err != nil {
		t.Fatal(err)
	}
	options := build.Options{ToolVersion: "test"}
	output, err := build.Generate(config, nil, files, options)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"RULES.md", "groups/techs/go.md", "rules/local/techs/go/errors.md", "provenance.json"} {
		if len(output.Files[name]) == 0 {
			t.Errorf("missing generated file %s", name)
		}
	}
	if !strings.Contains(string(output.Files["rules/local/techs/go/errors.md"]), "Return the error.\r\n") {
		t.Fatal("rule body bytes changed")
	}
	if !strings.Contains(string(output.Files["groups/techs/go.md"]), "Return the error.") {
		t.Fatal("default generation did not include the complete group")
	}
	options.IndexMaxLines = 750
	again, err := build.Generate(config, nil, files, options)
	if err != nil || !reflect.DeepEqual(output, again) {
		t.Fatalf("default and explicit limits differ: %v", err)
	}
	// Callers own output; changing it must not change the source files.
	for _, data := range output.Files {
		if len(data) > 0 {
			data[0] ^= 0xff
		}
	}
	after, err := json.Marshal(files)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("generation or output mutation changed inputs: %v", err)
	}
}

func TestGenerateFailureReturnsNoFiles(t *testing.T) {
	for _, scenario := range []string{"invalid-rule", "invalid-line-limit", "invalid-inline-limit"} {
		t.Run(scenario, func(t *testing.T) {
			config, files := generationInputs(t)
			options := build.Options{ToolVersion: "test"}
			switch scenario {
			case "invalid-rule":
				files["techs/go/errors.md"] = []byte("missing rule metadata")
			case "invalid-line-limit":
				options.IndexMaxLines = -1
			case "invalid-inline-limit":
				limit := -1
				options.GroupInlineMaxBytes = &limit
			}
			output, err := build.Generate(config, nil, files, options)
			if err == nil || output.Files != nil {
				t.Fatalf("expected failure without partial output, got %v, %v", output, err)
			}
		})
	}
}

func generationInputs(t *testing.T) (rules.Configuration, map[string][]byte) {
	t.Helper()
	config, err := rules.ParseConfiguration([]byte(`{"schemaVersion":1,"sources":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	return config, map[string][]byte{
		"techs/go/_group.json": []byte(`{"name":"Go","description":"Go guidance.","whenToRead":"When writing Go."}`),
		"techs/go/errors.md":   []byte("---\ntitle: Return errors\nimpact: HIGH\nimpactDescription: Preserve failures.\nwhenToRead: When calling functions.\n---\n# Return errors\n\nReturn the error.\r\n\r\nKeep its context.\n"),
	}
}
