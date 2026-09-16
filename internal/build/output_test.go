// Verify byte-preserving terms, policy provenance, deterministic output, and failure atomicity.

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

// TestPrepareRetainsTermsWithoutActiveRules copies binary terms unchanged even after every upstream rule is excluded.
func TestPrepareRetainsTermsWithoutActiveRules(t *testing.T) {
	config, libraries := fixture(t, `{"techs/go/errors":"Use local policy"}`, `{}`)
	supplied := libraries["team"]
	terms := []byte{'x', '\r', '\n', 0, 255}
	supplied.Catalog.Licenses = []rules.LicenseDeclaration{{Files: []string{"LICENSE"}, AttributionFiles: []string{"NOTICE"}}}
	supplied.Catalog.SupportingFiles["LICENSE"] = terms
	supplied.Catalog.SupportingFiles["NOTICE"] = []byte("Notice\r\n")
	libraries["team"] = supplied
	resolved, err := build.Resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	options := build.Options{ToolVersion: "test", IndexMaxBytes: 8000}
	got, err := build.Prepare(resolved, options)
	if err != nil {
		t.Fatal(err)
	}
	file := "libraries/team/licenses/LICENSE.md"
	if !bytes.Equal(got.Files[file], terms) || string(got.Files["libraries/team/licenses/notices/001.md"]) != "Notice\r\n" {
		t.Fatal("changed terms")
	}
	if _, ok := got.Files["rules/team/techs/go/errors.md"]; ok {
		t.Fatal("rendered excluded rule")
	}
	var provenance struct {
		Sources []json.RawMessage
		Rules   []json.RawMessage
	}
	if err := json.Unmarshal(got.Files["provenance.json"], &provenance); err != nil {
		t.Fatal(err)
	}
	if len(provenance.Sources) != 1 || len(provenance.Rules) != 0 {
		t.Fatal("lost excluded-source provenance")
	}
	again, err := build.Prepare(resolved, options)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("nondeterministic output")
	}
	got.Files[file][0] = 'z'
	if terms[0] != 'x' {
		t.Fatal("output aliases original term bytes")
	}
}

// TestPrepareReplacementProvenance retains upstream identity but assigns only the local definition's license basis.
func TestPrepareReplacementProvenance(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{"techs/go/errors":{"file":"local/techs/go/custom.md","reason":"Project policy"}}`)
	resolved, err := build.Resolve(config, libraries, map[string][]byte{"techs/go/custom.md": []byte(document)})
	if err != nil {
		t.Fatal(err)
	}
	got, err := build.Prepare(resolved, build.Options{ToolVersion: "test", IndexMaxBytes: 8000})
	if err != nil {
		t.Fatal(err)
	}
	var provenance struct {
		Rules []struct {
			ID                string
			Origin            build.Origin
			Upstream          *build.Origin
			ReplacementReason string
			LicenseBasis      string
		}
	}
	if err := json.Unmarshal(got.Files["provenance.json"], &provenance); err != nil {
		t.Fatal(err)
	}
	rule := provenance.Rules[0]
	if rule.ID != "local:techs/go/custom" || rule.Origin.Source != "local" || rule.Upstream == nil || rule.Upstream.Source != "team" || rule.ReplacementReason != "Project policy" || rule.LicenseBasis != "undeclared" {
		t.Fatalf("%+v", rule)
	}
	text := string(got.Files["provenance.json"])
	for _, field := range []string{`"document":`, `"body":`} {
		if strings.Contains(text, field) {
			t.Fatal("redundant source text in provenance")
		}
	}
}

// TestPrepareNoPartialOutput rejects missing term bytes and invalid budgets after resolution.
func TestPrepareNoPartialOutput(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	resolved, err := build.Resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, options := range []build.Options{{ToolVersion: "", IndexMaxBytes: 8000}, {ToolVersion: "test", IndexMaxBytes: 0}} {
		output, err := build.Prepare(resolved, options)
		if err == nil || output.Files != nil {
			t.Fatal("accepted invalid options or returned partial output")
		}
	}
	resolved.Sources[0].Licenses = []rules.LicenseDeclaration{{Files: []string{"missing"}}}
	output, err := build.Prepare(resolved, build.Options{ToolVersion: "test", IndexMaxBytes: 8000})
	if err == nil || output.Files != nil {
		t.Fatal("accepted missing terms")
	}
}
