// Show how a catalog reads a library's release tags, their records, and a rule file at a tagged commit.

package libraryformat_test

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/fabricahq/code-rules/libraryformat"
)

// Example reads the annotated tags a catalog found in a library's repository, such as with go-git: it skips tags
// that aren't library releases, reads each release's notes and record, then reads a rule file from the tagged
// commit.
func Example() {
	tags := []struct{ name, message string }{
		{"v0.1.0", "An unrelated tag."},
		{"release/1", `Publishes 1 rule.
---
formatVersion: 1
release: 1
rules:
  techs/go/handle-errors: 1.0.0
changes:
  techs/go/handle-errors:
    change: new
    summaries:
      - Add the rule.
`},
		{"release/2", `Updates 1 rule.
---
formatVersion: 1
release: 2
rules:
  techs/go/handle-errors: 1.1.0
changes:
  techs/go/handle-errors:
    change: minor
    from: 1.0.0
    summaries:
      - Add an example for wrapped errors.
`},
	}
	for _, tag := range tags {
		number, err := libraryformat.ParseReleaseTag(tag.name)
		if errors.Is(err, libraryformat.ErrNotReleaseTag) {
			fmt.Println("skip", tag.name)
			continue
		}
		notes, record, err := libraryformat.ParseReleaseMessage(tag.name, []byte(tag.message))
		var unsupported *libraryformat.UnsupportedReleaseRecordError
		if errors.As(err, &unsupported) {
			fmt.Println(tag.name, "needs a newer libraryformat for record format", unsupported.FormatVersion)
			continue
		}
		if err != nil {
			fmt.Println("invalid release:", err)
			continue
		}
		fmt.Printf("library release %d: %s\n", number, notes)
		for _, id := range slices.Sorted(maps.Keys(record.Changes)) {
			change := record.Changes[id]
			fmt.Printf("  %s %s %s: %s\n", id, change.Change, record.Rules[id], change.Summaries[0])
		}
	}

	// The rule's file at the tagged commit, read with go-git.
	file := `---
title: Handle errors
impact: HIGH
impactDescription: Unhandled errors hide failures.
whenToRead: Before writing Go code that returns errors.
---

Check every error a function returns.
`
	rule, err := libraryformat.ParseRule(file, "techs/go/handle-errors.md", "fabrica")
	if err != nil {
		fmt.Println("invalid rule:", err)
		return
	}
	fmt.Println(rule.ID, rule.Group, rule.Impact, rule.Title)

	// Output:
	// skip v0.1.0
	// library release 1: Publishes 1 rule.
	//   techs/go/handle-errors new 1.0.0: Add the rule.
	// library release 2: Updates 1 rule.
	//   techs/go/handle-errors minor 1.1.0: Add an example for wrapped errors.
	// fabrica:techs/go/handle-errors techs/go HIGH Handle errors
}
