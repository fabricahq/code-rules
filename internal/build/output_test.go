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
	supplied.Catalog.License = &rules.LicenseDeclaration{Files: []string{"LICENSE"}, AttributionFiles: []string{"NOTICE"}}
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
	resolved.Sources[0].License = &rules.LicenseDeclaration{Files: []string{"missing"}}
	output, err := build.Prepare(resolved, build.Options{ToolVersion: "test", IndexMaxBytes: 8000})
	if err == nil || output.Files != nil {
		t.Fatal("accepted missing terms")
	}
}

// TestProvenanceCompatibility keeps source-relative terms and explicit absent identity fields.
func TestProvenanceCompatibility(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	lib := libraries["team"]
	lib.Catalog.License = &rules.LicenseDeclaration{Files: []string{"LICENSE"}, AttributionFiles: []string{"NOTICE"}}
	lib.Catalog.SupportingFiles["LICENSE"] = []byte("Terms")
	lib.Catalog.SupportingFiles["NOTICE"] = []byte("Notice")
	libraries["team"] = lib
	resolved, err := build.Resolve(config, libraries, map[string][]byte{"techs/go/local.md": []byte(document)})
	if err != nil {
		t.Fatal(err)
	}
	output, err := build.Prepare(resolved, build.Options{ToolVersion: "test", IndexMaxBytes: 8000})
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Sources []struct {
			LicenseFiles []string
			License      *struct {
				Files            []string
				AttributionFiles []string
			}
		}
		Rules []struct {
			ID                string
			Origin            map[string]any
			ReplacementReason json.RawMessage
			License           *struct{ Files []string }
		}
	}
	if err := json.Unmarshal(output.Files["provenance.json"], &data); err != nil {
		t.Fatal(err)
	}
	source := data.Sources[0]
	if source.License == nil || !reflect.DeepEqual(source.LicenseFiles, []string{"LICENSE", "NOTICE"}) || !reflect.DeepEqual(source.License.Files, []string{"LICENSE"}) || !reflect.DeepEqual(source.License.AttributionFiles, []string{"NOTICE"}) {
		t.Fatalf("source paths: %+v", source)
	}
	for _, rule := range data.Rules {
		if string(rule.ReplacementReason) != "null" {
			t.Fatalf("reason missing for %s", rule.ID)
		}
		if strings.HasPrefix(rule.ID, "local:") {
			if rule.License != nil {
				t.Fatal("local rule inherited a library license")
			}
			for _, field := range []string{"repository", "ref", "resolvedCommit"} {
				value, present := rule.Origin[field]
				if !present || value != nil {
					t.Fatalf("%s: %v", field, rule.Origin)
				}
			}
		} else if rule.License == nil || !reflect.DeepEqual(rule.License.Files, []string{"vendor/team/LICENSE"}) {
			t.Fatalf("rule paths: %+v", rule)
		}
	}
	if strings.Contains(string(output.Files["provenance.json"]), `"licenses":`) {
		t.Fatal("obsolete licenses array in provenance")
	}
	resolved.Sources[0].License = nil
	output, err = build.Prepare(resolved, build.Options{ToolVersion: "test", IndexMaxBytes: 8000})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output.Files["provenance.json"]), `"licenseFiles": []`) {
		t.Fatal("missing empty license inventory")
	}
	if err := json.Unmarshal(output.Files["provenance.json"], &data); err != nil || data.Sources[0].License != nil {
		t.Fatalf("undeclared source license must be null: %s, %v", output.Files["provenance.json"], err)
	}
}

// TestProvenanceGuidanceOrder retains library-first provenance without changing effective local priority.
func TestProvenanceGuidanceOrder(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	resolved, err := build.Resolve(config, libraries, map[string][]byte{"techs/go/_group.json": []byte(`{"name":"Local Go","description":"Local policy","whenToRead":"When editing Go"}`)})
	if err != nil {
		t.Fatal(err)
	}
	original := append([]build.Guidance(nil), resolved.Groups[0].Guidance...)
	output, err := build.Prepare(resolved, build.Options{ToolVersion: "test", IndexMaxBytes: 8000})
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Groups []struct {
			Guidance                 []build.Guidance
			EffectiveGuidanceSources []string
		}
	}
	if err := json.Unmarshal(output.Files["provenance.json"], &data); err != nil {
		t.Fatal(err)
	}
	group := data.Groups[0]
	if group.Guidance[0].Source != "team" || group.Guidance[1].Source != "local" || !reflect.DeepEqual(group.EffectiveGuidanceSources, []string{"local"}) {
		t.Fatalf("%+v", group)
	}
	if !reflect.DeepEqual(resolved.Groups[0].Guidance, original) {
		t.Fatal("mutated input guidance")
	}
}
