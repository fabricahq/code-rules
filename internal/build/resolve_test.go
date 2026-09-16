// Exercise adoption policy using native parsed rules and source-labeled catalogs.

package build_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

const document = "---\ntitle: Return errors\nimpact: HIGH\nimpactDescription: Preserve failures.\nwhenToRead: When calling functions.\n---\n# Return errors\n\nReturn errors to the caller.\n"
const metadata = `{"name":"Go","description":"Go guidance.","whenToRead":"When editing Go."}`
const commit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

// fixture supplies parsed configuration and an immutable selected catalog for policy tests.
func fixture(t *testing.T, exclude, replace string) (rules.Configuration, map[string]build.Library) {
	t.Helper()
	config, err := rules.ParseConfiguration(json.RawMessage(`{"schemaVersion":1,"sources":{"team":{"repository":"https://github.com/acme/rules","ref":"v1.0.0","groups":["techs/go"],"exclude":` + exclude + `,"replace":` + replace + `}}}`))
	if err != nil {
		t.Fatal(err)
	}
	rule, err := rules.Parse(document, "techs/go/errors.md", "team")
	if err != nil {
		t.Fatal(err)
	}
	meta, err := rules.ParseGroupMetadata(json.RawMessage(metadata), "group")
	if err != nil {
		t.Fatal(err)
	}
	return config, map[string]build.Library{"team": {Commit: commit, Catalog: library.Catalog{Groups: []library.Group{{ID: "techs/go", Metadata: meta, Rules: []rules.Rule{rule}}}, Licenses: []rules.LicenseDeclaration{}, SupportingFiles: map[string][]byte{"techs/go/_group.json": []byte(metadata)}}}}
}

// TestResolveAdoption covers imported definitions, exclusions, replacements, local additions, and guidance precedence.
func TestResolveAdoption(t *testing.T) {
	for _, test := range []struct {
		name, exclude, replace string
		local                  map[string][]byte
		ids                    []string
	}{
		{"import", `{}`, `{}`, nil, []string{"team:techs/go/errors"}},
		{"exclude", `{"techs/go/errors":"Project policy"}`, `{}`, nil, []string{}},
		{"replace", `{}`, `{"techs/go/errors":{"file":"local/techs/go/custom.md","reason":"Project policy"}}`, map[string][]byte{"techs/go/custom.md": []byte(document)}, []string{"local:techs/go/custom"}},
		{"addition", `{}`, `{}`, map[string][]byte{"techs/go/custom.md": []byte(document)}, []string{"local:techs/go/custom", "team:techs/go/errors"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, libraries := fixture(t, test.exclude, test.replace)
			before, _ := json.Marshal(libraries)
			got, err := build.Resolve(config, libraries, test.local)
			if err != nil {
				t.Fatal(err)
			}
			ids := []string{}
			for _, active := range got.Groups[0].Rules {
				ids = append(ids, active.Rule.ID)
				if active.Rule.Document != document {
					t.Fatal("changed source document")
				}
				if test.name == "replace" && (active.Upstream == nil || active.Upstream.Source != "team" || active.Origin.Source != "local" || active.Reason != "Project policy") {
					t.Fatal("lost replacement provenance")
				}
			}
			if !reflect.DeepEqual(ids, test.ids) {
				t.Fatalf("IDs %v", ids)
			}
			after, _ := json.Marshal(libraries)
			if string(before) != string(after) {
				t.Fatal("mutated input catalog")
			}
			if len(got.Sources[0].Paths) != 2 {
				t.Fatal("lost full retained inventory")
			}
		})
	}
	config, libraries := fixture(t, `{}`, `{}`)
	local := map[string][]byte{"techs/go/_group.json": []byte(strings.ReplaceAll(metadata, "Go guidance.", "Local guidance."))}
	got, err := build.Resolve(config, libraries, local)
	if err != nil {
		t.Fatal(err)
	}
	if guidance := build.EffectiveGuidance(got.Groups[0]); len(guidance) != 1 || guidance[0].Source != "local" {
		t.Fatal("local guidance did not take precedence")
	}
}

// TestResolveFailures rejects missing inputs and validates candidates before exclusions can conceal them.
func TestResolveFailures(t *testing.T) {
	for _, test := range []struct {
		name, exclude, replace string
		local                  map[string][]byte
		alter                  func(map[string]build.Library)
	}{
		{name: "missing library", exclude: `{}`, replace: `{}`, alter: func(l map[string]build.Library) { delete(l, "team") }},
		{name: "invalid excluded rule", exclude: `{"techs/go/errors":"Not needed"}`, replace: `{}`, alter: func(l map[string]build.Library) { l["team"].Catalog.Groups[0].Rules[0].Document = "broken" }},
		{name: "missing exception", exclude: `{"techs/go/missing":"Not needed"}`, replace: `{}`},
		{name: "missing replacement", exclude: `{}`, replace: `{"techs/go/errors":{"file":"local/techs/go/custom.md","reason":"Project policy"}}`},
		{name: "wrong group", exclude: `{}`, replace: `{"techs/go/errors":{"file":"local/techs/rust/custom.md","reason":"Project policy"}}`, local: map[string][]byte{"techs/rust/custom.md": []byte(document)}},
		{name: "bad local path", exclude: `{}`, replace: `{}`, local: map[string][]byte{"../assets/data": []byte("x")}},
		{name: "missing local metadata", exclude: `{}`, replace: `{}`, local: map[string][]byte{"practices/testing/a.md": []byte(document)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, libraries := fixture(t, test.exclude, test.replace)
			if test.alter != nil {
				test.alter(libraries)
			}
			got, err := build.Resolve(config, libraries, test.local)
			if err == nil || got.Groups != nil {
				t.Fatalf("partial or successful result: %+v, %v", got, err)
			}
		})
	}
}

// TestResolveLocalOnly preserves an empty local group and validates its standalone rule.
func TestResolveLocalOnly(t *testing.T) {
	config, err := rules.ParseConfiguration(json.RawMessage(`{"schemaVersion":1,"sources":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	got, err := build.Resolve(config, nil, map[string][]byte{"techs/go/_group.json": []byte(metadata), "techs/go/errors.md": []byte(document)})
	if err != nil || len(got.Groups) != 1 || len(got.Groups[0].Rules) != 1 || got.Groups[0].Rules[0].Rule.ID != "local:techs/go/errors" {
		t.Fatalf("%+v, %v", got, err)
	}
}

// TestLocalSupportingFilesRemainSupport accepts inert support while rejecting unsafe names before asset classification.
func TestLocalSupportingFilesRemainSupport(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	for _, file := range []string{"techs/go/_notes.md", "techs/go/notes.txt"} {
		got, err := build.Resolve(config, libraries, map[string][]byte{file: []byte("support")})
		if err != nil || string(got.LocalFiles[file]) != "support" || len(got.Groups[0].Rules) != 1 {
			t.Fatalf("%s: %+v, %v", file, got, err)
		}
	}
	for _, file := range []string{"assets/new\nline.txt", "assets/del\x7ffile.txt"} {
		if _, err := build.Resolve(config, libraries, map[string][]byte{file: []byte("x")}); err == nil {
			t.Fatalf("accepted %q", file)
		}
	}
}
