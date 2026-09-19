// Exercise adoption policy using native parsed rules and source-labeled catalogs.

package build

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

const document = "---\ntitle: Return errors\nimpact: HIGH\nimpactDescription: Preserve failures.\nwhenToRead: When calling functions.\n---\n# Return errors\n\nReturn errors to the caller.\n"
const metadata = `{"name":"Go","description":"Go guidance.","whenToRead":"When editing Go."}`
const commit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

// fixture supplies parsed configuration and an immutable selected catalog for policy tests.
func fixture(t *testing.T, exclude, replace string) (rules.Configuration, map[string]Library) {
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
	return config, map[string]Library{"team": {Commit: commit, Catalog: library.Catalog{Selection: config.Sources[0].Groups, Groups: []library.Group{{ID: "techs/go", Metadata: meta, Rules: []rules.Rule{rule}}}, License: nil, SupportingFiles: map[string][]byte{"techs/go/_group.json": []byte(metadata)}}}}
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
			got, err := resolve(config, libraries, test.local)
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
	got, err := resolve(config, libraries, local)
	if err != nil {
		t.Fatal(err)
	}
	if guidance := got.Groups[0].EffectiveGuidance; len(guidance) != 1 || guidance[0].Source != "local" {
		t.Fatal("local guidance did not take precedence")
	}
}

// TestResolveFailures rejects missing inputs and validates candidates before exclusions can conceal them.
func TestResolveFailures(t *testing.T) {
	for _, test := range []struct {
		name, exclude, replace string
		local                  map[string][]byte
		alter                  func(map[string]Library)
	}{
		{name: "missing library", exclude: `{}`, replace: `{}`, alter: func(l map[string]Library) { delete(l, "team") }},
		{name: "invalid excluded rule", exclude: `{"techs/go/errors":"Not needed"}`, replace: `{}`, alter: func(l map[string]Library) { l["team"].Catalog.Groups[0].Rules[0].Document = "broken" }},
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
			got, err := resolve(config, libraries, test.local)
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
	got, err := resolve(config, nil, map[string][]byte{"techs/go/_group.json": []byte(metadata), "techs/go/errors.md": []byte(document)})
	if err != nil || len(got.Groups) != 1 || len(got.Groups[0].Rules) != 1 || got.Groups[0].Rules[0].Rule.ID != "local:techs/go/errors" {
		t.Fatalf("%+v, %v", got, err)
	}
}

// TestLocalSupportingFilesRemainSupport accepts inert support while rejecting unsafe names before asset classification.
func TestLocalSupportingFilesRemainSupport(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	for _, file := range []string{"techs/go/_notes.md", "techs/go/notes.txt", "assets/examples/_group.json", "techs/go/assets/example/_group.json"} {
		got, err := resolve(config, libraries, map[string][]byte{file: []byte("support")})
		if err != nil || string(got.LocalFiles[file]) != "support" || len(got.Groups[0].Rules) != 1 || !reflect.DeepEqual(got.LocalPaths, []string{file}) {
			t.Fatalf("%s: %+v, %v", file, got, err)
		}
	}
	for _, file := range []string{"assets/new\nline.txt", "assets/del\x7ffile.txt"} {
		if _, err := resolve(config, libraries, map[string][]byte{file: []byte("x")}); err == nil {
			t.Fatalf("accepted %q", file)
		}
	}
}

// TestReservedLocalGroupMetadata cannot hide reserved group names in the asset classification.
func TestReservedLocalGroupMetadata(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	for _, file := range []string{"techs/assets/_group.json", "practices/assets/_group.json"} {
		if _, err := resolve(config, libraries, map[string][]byte{file: []byte(metadata)}); err == nil {
			t.Fatalf("accepted %s", file)
		}
	}
}

// TestResolveBindsSelection rejects an otherwise valid catalog loaded with a narrower selector.
func TestResolveBindsSelection(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	config.Sources[0].Groups = rules.GroupSelection{Pattern: "*"}
	if _, err := resolve(config, libraries, nil); err == nil {
		t.Fatal("accepted narrower catalog for wildcard")
	}
}

// TestResolveVersionProvenance requires a matching release and retains its tag and normalized version.
func TestResolveVersionProvenance(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	config.Sources[0].Ref = ""
	config.Sources[0].ParsedRef = nil
	config.Sources[0].Version = ">= 1.0.0, < 2.0.0"
	supplied := libraries["team"]
	for _, tag := range []string{"", "v0.9.0", "v1.2.3"} {
		supplied.Tag = tag
		libraries["team"] = supplied
		got, err := resolve(config, libraries, nil)
		if tag != "v1.2.3" {
			if err == nil {
				t.Fatalf("accepted %q", tag)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if got.Sources[0].Tag != tag || got.Sources[0].ResolvedVersion != "1.2.3" || got.Groups[0].Rules[0].Origin.Ref != tag {
			t.Fatal("lost selected version")
		}
	}
}

// TestResolveRetainsInactiveDocuments preserves hidden upstream text without duplicating active documents.
func TestResolveRetainsInactiveDocuments(t *testing.T) {
	for _, policy := range []struct {
		exclude, replace string
		local            map[string][]byte
	}{
		{`{"techs/go/errors":"Not applicable"}`, `{}`, nil},
		{`{}`, `{"techs/go/errors":{"file":"local/techs/go/custom.md","reason":"Project policy"}}`, map[string][]byte{"techs/go/custom.md": []byte(document)}},
	} {
		config, libraries := fixture(t, policy.exclude, policy.replace)
		resolved, err := resolve(config, libraries, policy.local)
		if err != nil {
			t.Fatal(err)
		}
		if string(resolved.Sources[0].Files["techs/go/errors.md"]) != document {
			t.Fatal("lost inactive original")
		}
		if _, exists := libraries["team"].Catalog.SupportingFiles["techs/go/errors.md"]; exists {
			t.Fatal("mutated input catalog")
		}
	}
	config, libraries := fixture(t, `{}`, `{}`)
	resolved, err := resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := resolved.Sources[0].Files["techs/go/errors.md"]; exists {
		t.Fatal("duplicated active document")
	}
}

// TestResolveRejectsLocalRuleLinks applies the library link policy to local rules and retained Markdown attachments.
func TestResolveRejectsLocalRuleLinks(t *testing.T) {
	for _, from := range []string{"techs/go/one.md", "assets/guide.md", "techs/go/assets/one/guide.md"} {
		t.Run(from, func(t *testing.T) {
			config, libraries := fixture(t, `{}`, `{}`)
			local := map[string][]byte{"techs/go/one.md": []byte(document), "techs/go/two.md": []byte(document)}
			local[from] = append(local[from], []byte("\n[other](/techs/go/two.md)\n")...)
			got, err := resolve(config, libraries, local)
			if err == nil || !strings.Contains(err.Error(), "links to other rule documents are not allowed") || got.Groups != nil {
				t.Fatalf("expected rule-link error with no partial result: %+v, %v", got, err)
			}
		})
	}
}

// TestResolveAllowsLocalSupportingLinks keeps same-document anchors, shared explanations, and code examples valid.
func TestResolveAllowsLocalSupportingLinks(t *testing.T) {
	config, libraries := fixture(t, `{}`, `{}`)
	local := map[string][]byte{
		"techs/go/one.md": []byte(document + "\n[self](#details) [guide](/assets/guide.md) `\x5bother](two.md)`\n"),
		"assets/guide.md": []byte("[next](next.md) [external](https://example.com/reference)"),
		"assets/next.md":  []byte("Supporting text."),
	}
	if _, err := resolve(config, libraries, local); err != nil {
		t.Fatal(err)
	}
}

// TestResolveGroupGuidance resolves local metadata as a whole while preserving imported rules and provenance.
func TestResolveGroupGuidance(t *testing.T) {
	for _, withLocal := range []bool{false, true} {
		t.Run(fmt.Sprint("local=", withLocal), func(t *testing.T) {
			config, libraries := fixture(t, `{}`, `{}`)
			other := config.Sources[0]
			other.Name = "aaa"
			other.Repository = "https://github.com/acme/other"
			config.Sources = append([]rules.Source{other}, config.Sources...)
			imported := libraries["team"].Catalog.Groups[0].Metadata
			libraries["aaa"] = libraries["team"]
			local := map[string][]byte{}
			localMeta := rules.GroupMetadata{Name: "Project Go", Description: "Project policy.", WhenToRead: "When editing this project."}
			if withLocal {
				data, err := json.Marshal(localMeta)
				if err != nil {
					t.Fatal(err)
				}
				local["techs/go/_group.json"] = data
			}
			got, err := resolve(config, libraries, local)
			if err != nil {
				t.Fatal(err)
			}
			group := got.Groups[0]
			expected := []groupGuidance{{Source: "aaa", Metadata: imported}, {Source: "team", Metadata: imported}}
			if withLocal {
				expected = []groupGuidance{{Source: "local", Metadata: localMeta}}
			}
			if !reflect.DeepEqual(group.EffectiveGuidance, expected) {
				t.Fatalf("effective guidance: %+v", group.EffectiveGuidance)
			}
			wantDefinitions := 2
			if withLocal {
				wantDefinitions++
			}
			if len(group.Guidance) != wantDefinitions || len(group.Rules) != 2 {
				t.Fatalf("lost provenance or imported rules: %+v", group)
			}
		})
	}
}

// TestResolveNumericRuleOrder keeps numbered IDs in human reading order through summary rendering.
func TestResolveNumericRuleOrder(t *testing.T) {
	for _, names := range [][]string{
		{"rule-0", "rule-1", "rule-2", "rule-9", "rule-10", "rule-11"},
		{"rule-02", "rule-2", "rule-10"},
		{"rule-2-part-9", "rule-2-part-10", "rule-10-part-1"},
		{"rule-99999999999999999999", "rule-100000000000000000000"},
	} {
		for _, reverse := range []bool{false, true} {
			config, libraries := fixture(t, "{}", "{}")
			lib := libraries["team"]
			lib.Catalog.Groups[0].Rules = nil
			for i := range names {
				if reverse {
					i = len(names) - 1 - i
				}
				rule, err := rules.Parse(document, "techs/go/"+names[i]+".md", "team")
				if err != nil {
					t.Fatal(err)
				}
				lib.Catalog.Groups[0].Rules = append(lib.Catalog.Groups[0].Rules, rule)
			}
			libraries["team"] = lib
			resolved, err := resolve(config, libraries, nil)
			if err != nil {
				t.Fatal(err)
			}
			pages, err := renderIndexes(resolved, defaultIndexMaxLines, 0)
			if err != nil {
				t.Fatal(err)
			}
			previous := -1
			for i, name := range names {
				id := "team:techs/go/" + name
				if got := resolved.Groups[0].Rules[i].Rule.ID; got != id {
					t.Fatalf("reverse=%v: position %d: got %s, want %s", reverse, i, got, id)
				}
				position := strings.Index(pages["groups/techs/go.md"], "`"+id+"`")
				if position <= previous {
					t.Fatalf("rendered order lost %s", id)
				}
				previous = position
			}
		}
	}
}
