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

	"github.com/fabricahq/code-rules/libraryformat"
)

// The notes write each paragraph and each list item on one line, because GitHub renders a GitHub Release page's
// body with hard line breaks, so a line break inside a sentence would show on the page.

// majorChangesAdvice opens the major changes section of every library release's notes.
const majorChangesAdvice = "Code that complied with the previous rule version could fail the new one, so review these before updating."

// sharedFilesSentence follows the rule sections of a library release that also changes library-wide files,
// except the first, which adds every file because nothing existed before it. The notes never list those files
// by path; the release record does.
const sharedFilesSentence = "This library release also updates shared files, such as group descriptions or shared assets."

// releaseRules lists each rule a release record changed, added, or retired, in ID order.
func releaseRules(record libraryformat.ReleaseRecord) []PendingRule {
	list := []PendingRule{}
	for id, change := range record.Changes {
		next := record.Rules[id]
		list = append(list, PendingRule{ID: id, Change: change.Change, From: change.From, To: &next, Summaries: slices.Clone(change.Summaries)})
	}
	for id, retired := range record.Retired {
		last := retired.LastVersion
		list = append(list, PendingRule{ID: id, Change: libraryformat.ChangeRetired, LastVersion: &last, ReplacedBy: retired.ReplacedBy, Summaries: slices.Clone(retired.Summaries)})
	}
	slices.SortFunc(list, func(a, b PendingRule) int { return strings.Compare(a.ID, b.ID) })
	return list
}

// renderReleaseNotes returns a library release's Markdown notes, without a trailing newline: a line counting
// the rule changes, a section for each kind of change that has entries, in the order new, major, minor, patch,
// and retired, so readers meet the rules they'd adopt before the changes to rules they have, a sentence noting shared files when a library release after the first also lists library-wide
// files, and a collapsed table of every rule's version. Each rule is a list item with its summaries, one per
// change note, as nested items, except in the first library release, whose rules all have the same placeholder
// summary, firstReleaseSummary. Every paragraph and list item is one line. Library-wide files are never listed.
func renderReleaseNotes(record libraryformat.ReleaseRecord) string {
	var out strings.Builder
	out.WriteString(countLine(record))
	sections := []struct {
		change  libraryformat.Change
		heading string
	}{{libraryformat.ChangeNew, "New rules"}, {libraryformat.ChangeMajor, "Major changes"}, {libraryformat.ChangeMinor, "Minor changes"}, {libraryformat.ChangePatch, "Patch changes"}}
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
			item += "`" + record.Rules[id].String() + "`"
			if record.Release > 1 {
				item += summaryLines(change.Summaries)
			}
			items = append(items, item)
		}
		if len(items) == 0 {
			continue
		}
		out.WriteString("\n\n## " + section.heading + "\n\n")
		if section.change == libraryformat.ChangeMajor {
			out.WriteString(majorChangesAdvice + "\n\n")
		}
		out.WriteString(strings.Join(items, "\n"))
	}
	if len(record.Retired) > 0 {
		out.WriteString("\n\n## Retired rules\n\n")
		var items []string
		for _, id := range slices.Sorted(maps.Keys(record.Retired)) {
			retired := record.Retired[id]
			item := "- **" + id + "**, last version `" + retired.LastVersion.String() + "`"
			if retired.ReplacedBy != "" {
				item += ", replaced by **" + retired.ReplacedBy + "**"
			}
			items = append(items, item+summaryLines(retired.Summaries))
		}
		out.WriteString(strings.Join(items, "\n"))
	}
	if record.Release > 1 && len(record.LibraryFiles) > 0 && len(record.Changes)+len(record.Retired) > 0 {
		out.WriteString("\n\n" + sharedFilesSentence)
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

// countLine opens the notes, such as "Library release 4 changes 4 rules: 3 new and 1 major.", on one line, counting
// each kind of change in the order of the sections. A
// library release that changes no rules says it updates shared files instead, and the first library release says
// how many rules it publishes, since they are all new.
func countLine(record libraryformat.ReleaseRecord) string {
	if record.Release == 1 {
		switch count := len(record.Changes); count {
		case 0:
			return "Library release 1 publishes no rules, only shared files, such as group descriptions or shared assets."
		case 1:
			return "Library release 1 publishes 1 rule."
		default:
			return "Library release 1 publishes " + strconv.Itoa(count) + " rules."
		}
	}
	counts := map[libraryformat.Change]int{libraryformat.ChangeRetired: len(record.Retired)}
	for _, change := range record.Changes {
		counts[change.Change]++
	}
	total := len(record.Changes) + len(record.Retired)
	heading := "Library release " + strconv.Itoa(record.Release) + " changes "
	if total == 0 {
		return heading + "no rules. It updates shared files, such as group descriptions or shared assets."
	}
	var kinds []string
	for _, change := range []libraryformat.Change{libraryformat.ChangeNew, libraryformat.ChangeMajor, libraryformat.ChangeMinor, libraryformat.ChangePatch, libraryformat.ChangeRetired} {
		if counts[change] > 0 {
			kinds = append(kinds, strconv.Itoa(counts[change])+" "+string(change))
		}
	}
	noun := " rules"
	if total == 1 {
		noun = " rule"
	}
	return heading + strconv.Itoa(total) + noun + ": " + series(kinds) + "."
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

// summaryLines writes each summary as a nested list item of its rule's item.
func summaryLines(summaries []string) string {
	var out strings.Builder
	for _, summary := range summaries {
		out.WriteString("\n  - " + summary)
	}
	return out.String()
}

// releaseMessage returns a library release tag's message: the notes, a line containing only ---, and the YAML
// release record. It fails unless the message parses back to exactly these notes and this record, so a tag
// never carries a record that projects would read differently or reject.
func releaseMessage(notes string, record libraryformat.ReleaseRecord) ([]byte, error) {
	encoded, err := encodeReleaseRecord(record)
	if err != nil {
		return nil, err
	}
	message := notes + "\n---\n" + string(encoded)
	tag := "release/" + strconv.Itoa(record.Release)
	parsedNotes, parsed, err := libraryformat.ParseReleaseMessage(tag, []byte(message))
	if err != nil {
		return nil, fmt.Errorf("render the release record for %s: %w", tag, err)
	}
	if parsedNotes != notes || !reflect.DeepEqual(parsed, normalizedRecord(record)) {
		return nil, failure("invalid-release-record", "the release record for "+tag+" doesn't read back as written", nil)
	}
	return []byte(message), nil
}

// normalizedRecord is how the parser returns record: absent sections become empty ones.
func normalizedRecord(record libraryformat.ReleaseRecord) libraryformat.ReleaseRecord {
	if record.Rules == nil {
		record.Rules = map[string]libraryformat.RuleVersion{}
	}
	if record.Changes == nil {
		record.Changes = map[string]libraryformat.RecordedChange{}
	}
	if record.Retired == nil {
		record.Retired = map[string]libraryformat.RetiredRule{}
	}
	if record.LibraryFiles == nil {
		record.LibraryFiles = []string{}
	}
	return record
}

// encodeReleaseRecord writes the record's YAML, starting with its formatVersion, with sorted keys, leaving out
// empty changes, retired, and libraryFiles sections. rules is always present, since the parser requires it.
func encodeReleaseRecord(record libraryformat.ReleaseRecord) ([]byte, error) {
	text := func(value string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value} }
	mapping := func(pairs ...*yaml.Node) *yaml.Node { return &yaml.Node{Kind: yaml.MappingNode, Content: pairs} }
	versions := mapping()
	for _, id := range slices.Sorted(maps.Keys(record.Rules)) {
		versions.Content = append(versions.Content, text(id), text(record.Rules[id].String()))
	}
	number := func(value int) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(value)}
	}
	list := func(values []string) *yaml.Node {
		sequence := &yaml.Node{Kind: yaml.SequenceNode}
		for _, value := range values {
			sequence.Content = append(sequence.Content, text(value))
		}
		return sequence
	}
	document := mapping(text("formatVersion"), number(libraryformat.ReleaseRecordFormat), text("release"), number(record.Release), text("rules"), versions)
	if len(record.Changes) > 0 {
		changes := mapping()
		for _, id := range slices.Sorted(maps.Keys(record.Changes)) {
			change := record.Changes[id]
			entry := mapping(text("change"), text(string(change.Change)))
			if change.From != nil {
				entry.Content = append(entry.Content, text("from"), text(change.From.String()))
			}
			entry.Content = append(entry.Content, text("summaries"), list(change.Summaries))
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
			entry.Content = append(entry.Content, text("summaries"), list(rule.Summaries))
			retired.Content = append(retired.Content, text(id), entry)
		}
		document.Content = append(document.Content, text("retired"), retired)
	}
	if len(record.LibraryFiles) > 0 {
		document.Content = append(document.Content, text("libraryFiles"), list(record.LibraryFiles))
	}
	// A width of -1 keeps each summary whole.
	return yaml.Dump(document, yaml.WithV3Defaults(), yaml.WithIndent(2), yaml.WithLineWidth(-1))
}
