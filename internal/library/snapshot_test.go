// Check individually selected rules and snapshots that store rules from several library releases.

package library_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// ruleDocument is a valid rule whose body identifies it, so tests can tell copies apart.
func ruleDocument(body string) string {
	return strings.Replace(document, "Return the error.", body, 1)
}

// selectionFiles is a library with two groups, each with two rules, one of them with assets.
func selectionFiles() map[string][]byte {
	return map[string][]byte{
		"rule-library.yaml":                  []byte(`{"formatVersion":1}`),
		"techs/go/_group.yaml":               []byte(metadata),
		"techs/go/errors.md":                 []byte(ruleDocument("Errors.")),
		"techs/go/assets/errors/diagram.bin": {0, 1},
		"techs/go/naming.md":                 []byte(ruleDocument("Naming.")),
		"techs/go/assets/naming/example.bin": {2, 3},
		"practices/testing/_group.yaml":      []byte(metadata),
		"practices/testing/limits.md":        []byte(ruleDocument("Limits.")),
	}
}

// loadFiles loads files in memory, as project builds load vendored snapshots.
func loadFiles(t *testing.T, files map[string][]byte, groups rules.GroupSelection, ruleIDs []string) (library.Catalog, error) {
	t.Helper()
	source, err := library.Snapshot{Files: files}.Source()
	if err != nil {
		t.Fatal(err)
	}
	return library.LoadSource(context.Background(), source, "team", groups, ruleIDs)
}

// TestLoadSource_IndividualRulesBringOnlyThemselvesAndTheirGroupMetadata keeps unselected rules and assets out.
func TestLoadSource_IndividualRulesBringOnlyThemselvesAndTheirGroupMetadata(t *testing.T) {
	catalog, err := loadFiles(t, selectionFiles(), rules.GroupSelection{Groups: []string{"practices/testing"}}, []string{"techs/go/errors"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"practices/testing/_group.yaml", "practices/testing/limits.md", "rule-library.yaml", "techs/go/_group.yaml", "techs/go/assets/errors/diagram.bin", "techs/go/errors.md"}
	if !reflect.DeepEqual(catalog.Paths(), want) {
		t.Fatalf("paths %v, want %v", catalog.Paths(), want)
	}
	if !reflect.DeepEqual(catalog.Rules, []string{"techs/go/errors"}) || len(catalog.Groups) != 2 || catalog.Groups[1].ID != "techs/go" || len(catalog.Groups[1].Rules) != 1 {
		t.Fatalf("unexpected catalog %+v", catalog)
	}
}

// TestLoadSource_IndividualRuleInASelectedGroupChangesNothing loads the whole group once.
func TestLoadSource_IndividualRuleInASelectedGroupChangesNothing(t *testing.T) {
	catalog, err := loadFiles(t, selectionFiles(), rules.GroupSelection{Pattern: "techs/*"}, []string{"techs/go/errors"})
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Groups) != 1 || len(catalog.Groups[0].Rules) != 2 || !reflect.DeepEqual(catalog.Rules, []string{"techs/go/errors"}) {
		t.Fatalf("unexpected catalog %+v", catalog)
	}
}

// TestLoadSource_RejectsAnIndividualRuleTheLibraryLacks names the missing rule, even in a selected group.
func TestLoadSource_RejectsAnIndividualRuleTheLibraryLacks(t *testing.T) {
	for name, groups := range map[string]rules.GroupSelection{"unselected group": {Groups: []string{}}, "selected group": {Groups: []string{"techs/go"}}} {
		t.Run(name, func(t *testing.T) {
			_, err := loadFiles(t, selectionFiles(), groups, []string{"techs/go/missing"})
			if err == nil || !strings.Contains(err.Error(), "techs/go/missing") {
				t.Fatalf("got %v", err)
			}
		})
	}
}

// TestLoadSource_SkipsInvalidUnselectedRules imports a valid rule from a group whose other rules are invalid.
func TestLoadSource_SkipsInvalidUnselectedRules(t *testing.T) {
	files := selectionFiles()
	files["techs/go/naming.md"] = []byte("no frontmatter")
	if _, err := loadFiles(t, files, rules.GroupSelection{Groups: []string{}}, []string{"techs/go/errors"}); err != nil {
		t.Fatal(err)
	}
	if _, err := loadFiles(t, files, rules.GroupSelection{Groups: []string{"techs/go"}}, nil); err == nil {
		t.Fatal("a selected group accepted an invalid rule")
	}
}

// TestSnapshot_StoresRulesFromOtherLibraryReleasesSeparately round-trips files through their stored paths.
func TestSnapshot_StoresRulesFromOtherLibraryReleasesSeparately(t *testing.T) {
	v1, v2 := rules.RuleVersion{Major: 1}, rules.RuleVersion{Major: 2}
	snapshot := library.Snapshot{Release: 3, Rules: map[string]library.ImportedRule{
		"techs/go/errors": {Version: &v1, Release: 1, Commit: strings.Repeat("1", 40)},
		"techs/go/naming": {Version: &v2, Release: 3, Commit: strings.Repeat("3", 40)},
	}}
	for file, want := range map[string]string{
		"techs/go/errors.md":                 "_releases/1/techs/go/errors.md",
		"techs/go/assets/errors/diagram.bin": "_releases/1/techs/go/assets/errors/diagram.bin",
		"techs/go/naming.md":                 "techs/go/naming.md",
		"techs/go/_group.yaml":               "techs/go/_group.yaml",
		"practices/testing/limits.md":        "practices/testing/limits.md",
	} {
		if got := snapshot.StoredPath(file); got != want {
			t.Errorf("StoredPath(%q) = %q, want %q", file, got, want)
		}
	}
	files := selectionFiles()
	delete(files, "practices/testing/limits.md")
	delete(files, "practices/testing/_group.yaml")
	stored := map[string][]byte{}
	for file, data := range files {
		stored[snapshot.StoredPath(file)] = data
	}
	snapshot.Files = stored
	source, err := snapshot.Source()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := library.LoadSource(context.Background(), source, "team", rules.GroupSelection{Groups: []string{"techs/go"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot.Store(catalog); !reflect.DeepEqual(got, stored) {
		t.Fatalf("stored %v, want %v", got, stored)
	}
}

// TestSnapshot_SourceRejectsAmbiguousStoredFiles refuses a file stored twice or outside a numbered directory.
func TestSnapshot_SourceRejectsAmbiguousStoredFiles(t *testing.T) {
	for name, files := range map[string]map[string][]byte{
		"stored twice":     {"techs/go/errors.md": {}, "_releases/1/techs/go/errors.md": {}},
		"unnumbered":       {"_releases/one/techs/go/errors.md": {}},
		"leading zero":     {"_releases/01/techs/go/errors.md": {}},
		"no path":          {"_releases/1": {}},
		"twice in release": {"_releases/1/techs/go/errors.md": {}, "_releases/2/techs/go/errors.md": {}},
	} {
		if _, err := (library.Snapshot{Files: files}).Source(); err == nil {
			t.Errorf("%s: accepted %v", name, files)
		}
	}
}
