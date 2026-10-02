// Exercise adoption policy using native parsed rules and source-labeled catalogs.

package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/coderules"
	"github.com/fabricahq/code-rules/internal/errs"
	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

const document = "---\ntitle: Return errors\nimpact: HIGH\nimpactDescription: Preserve failures.\nwhenToRead: When calling functions.\n---\n# Return errors\n\nReturn errors to the caller.\n"
const metadata = `{"name":"Go","description":"Go guidance.","whenToRead":"When editing Go."}`
const commit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

// fixture supplies parsed configuration with the given exclude map and an immutable selected catalog for policy tests.
func fixture(t *testing.T, exclude string) (rules.Configuration, map[string]Library) {
	t.Helper()
	config, err := rules.ParseConfiguration(json.RawMessage(`{"schemaVersion":1,"sources":{"team":{"repository":"https://github.com/acme/rules","ref":"v1.0.0","groups":["techs/go"],"exclude":` + exclude + `}}}`))
	if err != nil {
		t.Fatal(err)
	}
	rule, err := coderules.ParseRule(document, "techs/go/errors.md", "team")
	if err != nil {
		t.Fatal(err)
	}
	meta, err := coderules.ParseGroupMetadata([]byte(metadata), "group")
	if err != nil {
		t.Fatal(err)
	}
	return config, map[string]Library{"team": versioned(library.Catalog{Selection: config.Sources[0].Groups, Groups: []library.Group{{ID: "techs/go", Metadata: meta, Rules: []coderules.Rule{rule}}}, License: nil, SupportingFiles: map[string][]byte{"techs/go/_group.yaml": []byte(metadata)}})}
}

// versioned supplies catalog with a snapshot record that imports each of its rules at version 1.0.0 from
// library release 1, at commit.
func versioned(catalog library.Catalog) Library {
	snapshot := library.Snapshot{Release: 1, Commit: commit, Rules: map[string]library.ImportedRule{}}
	for _, group := range catalog.Groups {
		for _, rule := range group.Rules {
			snapshot.Rules[strings.TrimSuffix(rule.Path, ".md")] = library.ImportedRule{Version: &coderules.FirstRuleVersion, Release: 1, Commit: commit}
		}
	}
	return Library{Catalog: catalog, Snapshot: snapshot}
}

// TestResolveAdoption covers imported definitions, exclusions, replacements, local additions, and guidance precedence.
func TestResolveAdoption(t *testing.T) {
	for _, test := range []struct {
		name, exclude string
		local         map[string][]byte
		ids           []string
	}{
		{"import", `{}`, nil, []string{"team:techs/go/errors"}},
		{"exclude", `{"techs/go/errors":{"reason":"Project policy"}}`, nil, []string{}},
		{"replace", `{"techs/go/errors":{"reason":"Project policy","replacedBy":"local/techs/go/custom.md"}}`, map[string][]byte{"techs/go/custom.md": []byte(document)}, []string{"local:techs/go/custom"}},
		{"addition", `{}`, map[string][]byte{"techs/go/custom.md": []byte(document)}, []string{"local:techs/go/custom", "team:techs/go/errors"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, libraries := fixture(t, test.exclude)
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
	config, libraries := fixture(t, `{}`)
	local := map[string][]byte{"techs/go/_group.yaml": []byte(strings.ReplaceAll(metadata, "Go guidance.", "Local guidance."))}
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
		name, exclude string
		local         map[string][]byte
		alter         func(map[string]Library)
	}{
		{name: "missing library", exclude: `{}`, alter: func(l map[string]Library) { delete(l, "team") }},
		{name: "invalid excluded rule", exclude: `{"techs/go/errors":{"reason":"Not needed"}}`, alter: func(l map[string]Library) { l["team"].Catalog.Groups[0].Rules[0].Document = "broken" }},
		{name: "missing replacement", exclude: `{"techs/go/errors":{"reason":"Project policy","replacedBy":"local/techs/go/custom.md"}}`},
		{name: "wrong group", exclude: `{"techs/go/errors":{"reason":"Project policy","replacedBy":"local/techs/rust/custom.md"}}`, local: map[string][]byte{"techs/rust/custom.md": []byte(document)}},
		{name: "bad local path", exclude: `{}`, local: map[string][]byte{"../assets/data": []byte("x")}},
		{name: "missing local metadata", exclude: `{}`, local: map[string][]byte{"practices/testing/a.md": []byte(document)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, libraries := fixture(t, test.exclude)
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

// TestResolveReplacementFileReuse refuses one local file replacing two rules, within one source or across
// sources, and accepts distinct replacement files.
func TestResolveReplacementFileReuse(t *testing.T) {
	meta, err := coderules.ParseGroupMetadata([]byte(metadata), "group")
	if err != nil {
		t.Fatal(err)
	}
	catalog := func(source string, paths ...string) Library {
		group := library.Group{ID: "techs/go", Metadata: meta, Rules: []coderules.Rule{}}
		for _, path := range paths {
			rule, err := coderules.ParseRule(document, path, source)
			if err != nil {
				t.Fatal(err)
			}
			group.Rules = append(group.Rules, rule)
		}
		return versioned(library.Catalog{Selection: rules.GroupSelection{Groups: []string{"techs/go"}}, Groups: []library.Group{group}, SupportingFiles: map[string][]byte{"techs/go/_group.yaml": []byte(metadata)}})
	}
	replaced := func(file string) string {
		return `{"reason":"Project policy","replacedBy":"local/techs/go/` + file + `"}`
	}
	local := map[string][]byte{"techs/go/custom.md": []byte(document), "techs/go/second.md": []byte(document)}
	for _, test := range []struct {
		name      string
		sources   string
		libraries map[string]Library
		reused    bool
	}{
		{"same source", `"team":{"repository":"https://github.com/acme/rules","ref":"v1.0.0","groups":["techs/go"],"exclude":{"techs/go/errors":` + replaced("custom.md") + `,"techs/go/other":` + replaced("custom.md") + `}}`,
			map[string]Library{"team": catalog("team", "techs/go/errors.md", "techs/go/other.md")}, true},
		{"two sources", `"acme":{"repository":"https://github.com/acme/other","ref":"v1.0.0","groups":["techs/go"],"exclude":{"techs/go/errors":` + replaced("custom.md") + `}},` +
			`"team":{"repository":"https://github.com/acme/rules","ref":"v1.0.0","groups":["techs/go"],"exclude":{"techs/go/errors":` + replaced("custom.md") + `}}`,
			map[string]Library{"acme": catalog("acme", "techs/go/errors.md"), "team": catalog("team", "techs/go/errors.md")}, true},
		{"distinct files", `"team":{"repository":"https://github.com/acme/rules","ref":"v1.0.0","groups":["techs/go"],"exclude":{"techs/go/errors":` + replaced("custom.md") + `,"techs/go/other":` + replaced("second.md") + `}}`,
			map[string]Library{"team": catalog("team", "techs/go/errors.md", "techs/go/other.md")}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, err := rules.ParseConfiguration(json.RawMessage(`{"schemaVersion":1,"sources":{` + test.sources + `}}`))
			if err != nil {
				t.Fatal(err)
			}
			got, err := resolve(config, test.libraries, local)
			if !test.reused {
				if err != nil || len(got.Groups) != 1 || len(got.Groups[0].Rules) != 2 || got.Groups[0].Rules[0].Rule.ID != "local:techs/go/custom" || got.Groups[0].Rules[1].Rule.ID != "local:techs/go/second" {
					t.Fatalf("got %+v, %v", got.Groups, err)
				}
				return
			}
			var validation errs.ValidationError
			if !errors.As(err, &validation) || validation.ValidationLocation() != "local/techs/go/custom.md" || validation.ValidationProblem() != "replacement file is reused for multiple targets" || got.Groups != nil || got.Sources != nil {
				t.Fatalf("got %+v, %v; want reuse refusal and no result", got, err)
			}
		})
	}
}

// TestResolveRecordsEachRulesVersionAndCommit gives each imported rule the version and commit its snapshot records.
func TestResolveRecordsEachRulesVersionAndCommit(t *testing.T) {
	config, libraries := fixture(t, `{}`)
	older, version := strings.Repeat("b", 40), coderules.RuleVersion{Major: 1, Minor: 2}
	team := libraries["team"]
	team.Snapshot.Release = 3
	team.Snapshot.Rules["techs/go/errors"] = library.ImportedRule{Version: &version, Release: 2, Commit: older}
	libraries["team"] = team
	got, err := resolve(config, libraries, nil)
	if err != nil {
		t.Fatal(err)
	}
	origin := got.Groups[0].Rules[0].Origin
	if origin.Commit != older || origin.Version == nil || *origin.Version != version || origin.Release != 2 {
		t.Fatalf("origin %+v", origin)
	}
	delete(team.Snapshot.Rules, "techs/go/errors")
	if _, err := resolve(config, libraries, nil); err == nil {
		t.Fatal("resolved a rule without a version record")
	}
}

// TestResolveLocalOnly preserves an empty local group and validates its standalone rule.
func TestResolveLocalOnly(t *testing.T) {
	config, err := rules.ParseConfiguration(json.RawMessage(`{"schemaVersion":1,"sources":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolve(config, nil, map[string][]byte{"techs/go/_group.yaml": []byte(metadata), "techs/go/errors.md": []byte(document)})
	if err != nil || len(got.Groups) != 1 || len(got.Groups[0].Rules) != 1 || got.Groups[0].Rules[0].Rule.ID != "local:techs/go/errors" {
		t.Fatalf("%+v, %v", got, err)
	}
}

// TestLocalSupportingFilesRemainSupport accepts inert support while rejecting unsafe names before asset classification.
func TestLocalSupportingFilesRemainSupport(t *testing.T) {
	config, libraries := fixture(t, `{}`)
	for _, file := range []string{"techs/go/_notes.md", "techs/go/notes.txt", "assets/examples/_group.yaml", "techs/go/assets/example/_group.yaml"} {
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
	config, libraries := fixture(t, `{}`)
	for _, file := range []string{"techs/assets/_group.yaml", "practices/assets/_group.yaml"} {
		if _, err := resolve(config, libraries, map[string][]byte{file: []byte(metadata)}); err == nil {
			t.Fatalf("accepted %s", file)
		}
	}
}

// TestResolveBindsSelection rejects an otherwise valid catalog loaded with a narrower selector.
func TestResolveBindsSelection(t *testing.T) {
	config, libraries := fixture(t, `{}`)
	config.Sources[0].Groups = rules.GroupSelection{Pattern: "*"}
	if _, err := resolve(config, libraries, nil); err == nil {
		t.Fatal("accepted narrower catalog for wildcard")
	}
}

// TestResolveRetainsInactiveDocuments preserves hidden upstream text without duplicating active documents.
func TestResolveRetainsInactiveDocuments(t *testing.T) {
	for _, policy := range []struct {
		exclude string
		local   map[string][]byte
	}{
		{`{"techs/go/errors":{"reason":"Not applicable"}}`, nil},
		{`{"techs/go/errors":{"reason":"Project policy","replacedBy":"local/techs/go/custom.md"}}`, map[string][]byte{"techs/go/custom.md": []byte(document)}},
	} {
		config, libraries := fixture(t, policy.exclude)
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
	config, libraries := fixture(t, `{}`)
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
			config, libraries := fixture(t, `{}`)
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
	config, libraries := fixture(t, `{}`)
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
			config, libraries := fixture(t, `{}`)
			other := config.Sources[0]
			other.Name = "aaa"
			other.Repository = "https://github.com/acme/other"
			config.Sources = append([]rules.Source{other}, config.Sources...)
			imported := libraries["team"].Catalog.Groups[0].Metadata
			libraries["aaa"] = libraries["team"]
			local := map[string][]byte{}
			localMeta := coderules.GroupMetadata{Name: "Project Go", Description: "Project policy.", WhenToRead: "When editing this project."}
			if withLocal {
				data, err := json.Marshal(localMeta)
				if err != nil {
					t.Fatal(err)
				}
				local["techs/go/_group.yaml"] = data
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
			config, libraries := fixture(t, "{}")
			lib := libraries["team"]
			lib.Catalog.Groups[0].Rules = nil
			for i := range names {
				if reverse {
					i = len(names) - 1 - i
				}
				rule, err := coderules.ParseRule(document, "techs/go/"+names[i]+".md", "team")
				if err != nil {
					t.Fatal(err)
				}
				lib.Catalog.Groups[0].Rules = append(lib.Catalog.Groups[0].Rules, rule)
			}
			libraries["team"] = versioned(lib.Catalog)
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
