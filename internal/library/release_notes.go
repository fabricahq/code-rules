// Render a library release's notes and tag message from its release record, without accessing Git or files.

package library

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/fabricahq/code-rules/internal/rules"
)

// majorChangesAdvice opens the major changes section of every library release's notes.
const majorChangesAdvice = "Code that complied with the previous rule version could fail\nthe new one, so review these before updating."

// releaseRules lists each rule a release record changed, added, or retired, in ID order.
func releaseRules(record rules.ReleaseRecord) []PendingRule {
	list := []PendingRule{}
	for id, change := range record.Changes {
		next := record.Rules[id]
		list = append(list, PendingRule{ID: id, Change: change.Change, CurrentVersion: change.From, NextVersion: &next})
	}
	for id, retired := range record.Retired {
		last := retired.LastVersion
		list = append(list, PendingRule{ID: id, Change: rules.ChangeRetired, CurrentVersion: &last, ReplacedBy: retired.ReplacedBy})
	}
	slices.SortFunc(list, func(a, b PendingRule) int { return strings.Compare(a.ID, b.ID) })
	return list
}

// renderReleaseNotes returns a library release's Markdown notes, without a trailing newline: a line counting
// the rule changes, a section for each kind of change that has entries, in the order major, minor, patch, new,
// and retired, and a collapsed table of every rule's version. Summaries keep one line per change note.
func renderReleaseNotes(record rules.ReleaseRecord) string {
	var out strings.Builder
	out.WriteString(countLine(record))
	sections := []struct {
		change  rules.Change
		heading string
	}{{rules.ChangeMajor, "Major changes"}, {rules.ChangeMinor, "Minor changes"}, {rules.ChangePatch, "Patch changes"}, {rules.ChangeNew, "New rules"}}
	for _, section := range sections {
		var items []string
		for _, id := range slices.Sorted(maps.Keys(record.Changes)) {
			change := record.Changes[id]
			if change.Change != section.change {
				continue
			}
			item := "- **" + id + "** "
			if change.From != nil {
				item += "`" + change.From.String() + "` → "
			}
			items = append(items, item+"`"+record.Rules[id].String()+"`"+summaryLines(change.Summary))
		}
		if len(items) == 0 {
			continue
		}
		out.WriteString("\n\n## " + section.heading + "\n\n")
		if section.change == rules.ChangeMajor {
			out.WriteString(majorChangesAdvice + "\n\n")
		}
		out.WriteString(strings.Join(items, "\n"))
	}
	if len(record.Retired) > 0 {
		out.WriteString("\n\n## Retired rules\n\n")
		var items []string
		for _, id := range slices.Sorted(maps.Keys(record.Retired)) {
			retired := record.Retired[id]
			item := "- **" + id + "**, last version `" + retired.LastVersion.String() + "`" + summaryLines(retired.Summary)
			if retired.ReplacedBy != "" {
				item += "\n  Replaced by **" + retired.ReplacedBy + "**."
			}
			items = append(items, item)
		}
		out.WriteString(strings.Join(items, "\n"))
	}
	if len(record.Rules) > 0 {
		out.WriteString("\n\n<details>\n<summary>All rule versions in this library release</summary>\n\n| Rule | Version |\n| --- | --- |\n")
		for _, id := range slices.Sorted(maps.Keys(record.Rules)) {
			out.WriteString("| " + id + " | " + record.Rules[id].String() + " |\n")
		}
		out.WriteString("\n</details>")
	}
	return out.String()
}

// countLine opens the notes, such as "Library release 4 changes 4 rules:" and a line with each kind's count.
// A library release that changes only library-wide files says so instead.
func countLine(record rules.ReleaseRecord) string {
	counts := map[rules.Change]int{rules.ChangeRetired: len(record.Retired)}
	for _, change := range record.Changes {
		counts[change.Change]++
	}
	total := len(record.Changes) + len(record.Retired)
	heading := "Library release " + strconv.Itoa(record.Release) + " changes "
	if total == 0 {
		return heading + "no rules, only library-wide files."
	}
	var kinds []string
	for _, change := range []rules.Change{rules.ChangeMajor, rules.ChangeMinor, rules.ChangePatch, rules.ChangeNew, rules.ChangeRetired} {
		if counts[change] > 0 {
			kinds = append(kinds, strconv.Itoa(counts[change])+" "+string(change))
		}
	}
	noun := " rules"
	if total == 1 {
		noun = " rule"
	}
	return heading + strconv.Itoa(total) + noun + ":\n" + series(kinds) + "."
}

// series joins items as English prose: "a", "a and b", or "a, b, and c".
func series(items []string) string {
	switch len(items) {
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	}
	return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
}

// summaryLines indents each line of a summary as a continuation of its list item.
func summaryLines(summary string) string {
	var out strings.Builder
	for line := range strings.SplitSeq(summary, "\n") {
		out.WriteString("\n  " + line)
	}
	return out.String()
}

// releaseMessage returns a library release tag's message: the notes, a line containing only ---, and the YAML
// release record. It fails unless the message parses back to exactly these notes and this record, so a tag
// never carries a record that projects would read differently or reject.
func releaseMessage(notes string, record rules.ReleaseRecord) ([]byte, error) {
	encoded, err := encodeReleaseRecord(record)
	if err != nil {
		return nil, err
	}
	message := notes + "\n---\n" + string(encoded)
	tag := "release/" + strconv.Itoa(record.Release)
	parsedNotes, parsed, err := rules.ParseReleaseMessage(tag, []byte(message))
	if err != nil {
		return nil, fmt.Errorf("render the release record for %s: %w", tag, err)
	}
	if parsedNotes != notes || !reflect.DeepEqual(parsed, normalizedRecord(record)) {
		return nil, failure("invalid-release-record", "the release record for "+tag+" doesn't read back as written", nil)
	}
	return []byte(message), nil
}

// normalizedRecord is how the parser returns record: absent sections become empty ones.
func normalizedRecord(record rules.ReleaseRecord) rules.ReleaseRecord {
	if record.Rules == nil {
		record.Rules = map[string]rules.RuleVersion{}
	}
	if record.Changes == nil {
		record.Changes = map[string]rules.RecordedChange{}
	}
	if record.Retired == nil {
		record.Retired = map[string]rules.RetiredRule{}
	}
	if record.LibraryFiles == nil {
		record.LibraryFiles = []string{}
	}
	return record
}

// encodeReleaseRecord writes the record's YAML with sorted keys, leaving out empty changes, retired, and
// libraryFiles sections. rules is always present, since the parser requires it.
func encodeReleaseRecord(record rules.ReleaseRecord) ([]byte, error) {
	text := func(value string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value} }
	mapping := func(pairs ...*yaml.Node) *yaml.Node { return &yaml.Node{Kind: yaml.MappingNode, Content: pairs} }
	versions := mapping()
	for _, id := range slices.Sorted(maps.Keys(record.Rules)) {
		versions.Content = append(versions.Content, text(id), text(record.Rules[id].String()))
	}
	document := mapping(text("release"), &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(record.Release)}, text("rules"), versions)
	if len(record.Changes) > 0 {
		changes := mapping()
		for _, id := range slices.Sorted(maps.Keys(record.Changes)) {
			change := record.Changes[id]
			entry := mapping(text("change"), text(string(change.Change)))
			if change.From != nil {
				entry.Content = append(entry.Content, text("from"), text(change.From.String()))
			}
			entry.Content = append(entry.Content, text("summary"), text(change.Summary))
			changes.Content = append(changes.Content, text(id), entry)
		}
		document.Content = append(document.Content, text("changes"), changes)
	}
	if len(record.Retired) > 0 {
		retired := mapping()
		for _, id := range slices.Sorted(maps.Keys(record.Retired)) {
			rule := record.Retired[id]
			entry := mapping(text("lastVersion"), text(rule.LastVersion.String()))
			if rule.ReplacedBy != "" {
				entry.Content = append(entry.Content, text("replacedBy"), text(rule.ReplacedBy))
			}
			entry.Content = append(entry.Content, text("summary"), text(rule.Summary))
			retired.Content = append(retired.Content, text(id), entry)
		}
		document.Content = append(document.Content, text("retired"), retired)
	}
	if len(record.LibraryFiles) > 0 {
		files := &yaml.Node{Kind: yaml.SequenceNode}
		for _, name := range record.LibraryFiles {
			files.Content = append(files.Content, text(name))
		}
		document.Content = append(document.Content, text("libraryFiles"), files)
	}
	// A width of -1 keeps each summary line whole.
	return yaml.Dump(document, yaml.WithV3Defaults(), yaml.WithIndent(2), yaml.WithLineWidth(-1))
}
