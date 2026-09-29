// Parse the release record in a library release tag's message without accessing Git.

package rules

import (
	"bytes"
	"encoding/json"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// ReleaseRecord is the permanent record of one library release, read from its release/<number> tag.
type ReleaseRecord struct {
	// Release is the library release number, starting at 1.
	Release int `json:"release"`
	// Rules maps every rule current after this library release to its version.
	Rules map[string]RuleVersion `json:"rules"`
	// Changes lists the rules this library release added or published new versions of; each is also in Rules.
	Changes map[string]RecordedChange `json:"changes"`
	// Retired lists the rules this library release retired; none of them is in Rules.
	Retired map[string]RetiredRule `json:"retired"`
	// LibraryFiles lists changed library-wide files, such as group metadata and shared assets, in authored order.
	LibraryFiles []string `json:"libraryFiles"`
}

// RecordedChange is one rule's change in a library release. From is nil exactly when Change is new.
type RecordedChange struct {
	Change Change       `json:"change"`
	From   *RuleVersion `json:"from,omitempty"`
	// Summary has one line per change note that named the rule.
	Summary string `json:"summary"`
}

// RetiredRule is a rule a library release retired. ReplacedBy is empty when nothing replaces it.
type RetiredRule struct {
	LastVersion RuleVersion `json:"lastVersion"`
	ReplacedBy  string      `json:"replacedBy,omitempty"`
	Summary     string      `json:"summary"`
}

// releaseSeparator divides a release tag's Markdown notes from its YAML record.
const releaseSeparator = "---"

var releaseTagPattern = regexp.MustCompile(`^release/([1-9][0-9]{0,8})$`)

// ParseReleaseTag returns the library release number of a tag named release/<number>, such as release/4.
// It rejects other tag names, including leading zeros, so each library release has exactly one tag name.
func ParseReleaseTag(name string) (int, error) {
	parts := releaseTagPattern.FindStringSubmatch(name)
	if parts == nil {
		return 0, invalid(name, "expected a library release tag named release/<number>, such as release/4")
	}
	number, _ := strconv.Atoi(parts[1])
	return number, nil
}

// ParseReleaseMessage parses the message of the library release tag named tag. It splits the message at its last
// line containing only ---, returning the Markdown release notes before it and the parsed record after it; the
// notes may contain --- lines themselves. The record's release number must match the tag's.
func ParseReleaseMessage(tag string, message []byte) (string, ReleaseRecord, error) {
	number, err := ParseReleaseTag(tag)
	if err != nil {
		return "", ReleaseRecord{}, err
	}
	lines := bytes.Split(message, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		if string(bytes.TrimRight(lines[i], "\r")) != releaseSeparator {
			continue
		}
		notes := string(bytes.TrimRight(bytes.Join(lines[:i], []byte("\n")), "\r\n"))
		record, err := ParseReleaseRecord(bytes.Join(lines[i+1:], []byte("\n")), tag)
		if err != nil {
			return "", ReleaseRecord{}, err
		}
		if record.Release != number {
			return "", ReleaseRecord{}, invalid(tag+".release", "the record is for library release "+strconv.Itoa(record.Release)+", but its tag is "+tag)
		}
		return notes, record, nil
	}
	return "", ReleaseRecord{}, invalid(tag, "expected release notes, a line containing only ---, and a release record")
}

// ParseReleaseRecord validates a release record's YAML, including that each change leads to the version in rules.
func ParseReleaseRecord(input []byte, location string) (ReleaseRecord, error) {
	_, data, err := authoredYAML(input, location)
	if err != nil {
		return ReleaseRecord{}, err
	}
	fields, err := jsonObject(data, location)
	if err != nil {
		return ReleaseRecord{}, err
	}
	if err := knownJSONFields(fields, []string{"release", "rules", "changes", "retired", "libraryFiles"}, location); err != nil {
		return ReleaseRecord{}, err
	}
	record := ReleaseRecord{Changes: map[string]RecordedChange{}, Retired: map[string]RetiredRule{}, LibraryFiles: []string{}}
	if record.Release, err = releaseNumber(fields["release"], location+".release"); err != nil {
		return ReleaseRecord{}, err
	}
	if record.Rules, err = recordedVersions(fields["rules"], location+".rules"); err != nil {
		return ReleaseRecord{}, err
	}
	if raw, ok := fields["changes"]; ok {
		if record.Changes, err = recordedChanges(raw, record.Rules, location+".changes"); err != nil {
			return ReleaseRecord{}, err
		}
	}
	if raw, ok := fields["retired"]; ok {
		if record.Retired, err = retiredRules(raw, record.Rules, location+".retired"); err != nil {
			return ReleaseRecord{}, err
		}
	}
	if raw, ok := fields["libraryFiles"]; ok {
		if record.LibraryFiles, err = libraryFiles(raw, location+".libraryFiles"); err != nil {
			return ReleaseRecord{}, err
		}
	}
	if record.Release == 1 {
		if err := requireFirstRelease(record, location); err != nil {
			return ReleaseRecord{}, err
		}
	}
	return record, nil
}

// requireFirstRelease requires the first library release to publish every rule as new, so each rule's history
// starts with a changes entry for 1.0.0, and to retire nothing, since no rule had a version before it.
func requireFirstRelease(record ReleaseRecord, location string) error {
	if len(record.Retired) > 0 {
		return invalid(location+".retired", "the first library release can't retire rules")
	}
	for _, id := range slices.Sorted(maps.Keys(record.Rules)) {
		if change, ok := record.Changes[id]; !ok || change.Change != ChangeNew {
			return invalid(location+".changes."+id, "the first library release must list every rule as new")
		}
	}
	return nil
}

// releaseNumber accepts a positive integer below one billion.
func releaseNumber(input json.RawMessage, location string) (int, error) {
	number, err := strconv.Atoi(string(input))
	if err != nil || number < 1 || number >= 1_000_000_000 {
		return 0, invalid(location, "expected a library release number, starting at 1")
	}
	return number, nil
}

// recordedVersions reads the required map of every current rule to its version; it may be empty.
func recordedVersions(input json.RawMessage, location string) (map[string]RuleVersion, error) {
	entries, err := jsonObject(input, location)
	if err != nil {
		return nil, err
	}
	versions := map[string]RuleVersion{}
	for _, id := range slices.Sorted(maps.Keys(entries)) {
		if err := ValidateRuleID(id, location+"."+id); err != nil {
			return nil, err
		}
		version, err := versionField(entries[id], location+"."+id)
		if err != nil {
			return nil, err
		}
		versions[id] = version
	}
	return versions, nil
}

// recordedChanges requires each changed rule's version in rules to follow from its change and previous version.
func recordedChanges(input json.RawMessage, versions map[string]RuleVersion, location string) (map[string]RecordedChange, error) {
	entries, err := jsonObject(input, location)
	if err != nil {
		return nil, err
	}
	changes := map[string]RecordedChange{}
	for _, id := range slices.Sorted(maps.Keys(entries)) {
		entryLocation := location + "." + id
		if err := ValidateRuleID(id, entryLocation); err != nil {
			return nil, err
		}
		fields, err := jsonObject(entries[id], entryLocation)
		if err != nil {
			return nil, err
		}
		if err := knownJSONFields(fields, []string{"change", "from", "summary"}, entryLocation); err != nil {
			return nil, err
		}
		var change RecordedChange
		var name string
		if json.Unmarshal(fields["change"], &name) != nil {
			return nil, invalid(entryLocation+".change", "expected new, major, minor, or patch")
		}
		change.Change = Change(name)
		if change.Summary, err = summaryText(fields["summary"], entryLocation+".summary"); err != nil {
			return nil, err
		}
		current, ok := versions[id]
		if !ok {
			return nil, invalid(entryLocation, "changed rule is missing from rules")
		}
		raw, hasFrom := fields["from"]
		switch change.Change {
		case ChangeNew:
			if hasFrom {
				return nil, invalid(entryLocation+".from", "a new rule has no previous version")
			}
			if current != FirstRuleVersion {
				return nil, invalid(entryLocation, "a new rule must be version 1.0.0 in rules, got "+current.String())
			}
		case ChangeMajor, ChangeMinor, ChangePatch:
			if !hasFrom {
				return nil, invalid(entryLocation+".from", "expected the rule's previous version")
			}
			from, err := versionField(raw, entryLocation+".from")
			if err != nil {
				return nil, err
			}
			next, err := from.Next(change.Change)
			if err != nil {
				return nil, invalid(entryLocation, err.Error())
			}
			if next != current {
				return nil, invalid(entryLocation, "a "+name+" change from "+from.String()+" leads to "+next.String()+", but rules records "+current.String())
			}
			change.From = &from
		default:
			return nil, invalid(entryLocation+".change", "unknown change "+quote(name)+"; expected new, major, minor, or patch")
		}
		changes[id] = change
	}
	return changes, nil
}

// retiredRules requires each retired rule to be absent from rules and any replacement to be present.
func retiredRules(input json.RawMessage, versions map[string]RuleVersion, location string) (map[string]RetiredRule, error) {
	entries, err := jsonObject(input, location)
	if err != nil {
		return nil, err
	}
	retired := map[string]RetiredRule{}
	for _, id := range slices.Sorted(maps.Keys(entries)) {
		entryLocation := location + "." + id
		if err := ValidateRuleID(id, entryLocation); err != nil {
			return nil, err
		}
		if _, current := versions[id]; current {
			return nil, invalid(entryLocation, "a retired rule can't also be in rules")
		}
		fields, err := jsonObject(entries[id], entryLocation)
		if err != nil {
			return nil, err
		}
		if err := knownJSONFields(fields, []string{"lastVersion", "replacedBy", "summary"}, entryLocation); err != nil {
			return nil, err
		}
		var rule RetiredRule
		if rule.LastVersion, err = versionField(fields["lastVersion"], entryLocation+".lastVersion"); err != nil {
			return nil, err
		}
		if rule.Summary, err = summaryText(fields["summary"], entryLocation+".summary"); err != nil {
			return nil, err
		}
		if raw, ok := fields["replacedBy"]; ok {
			if json.Unmarshal(raw, &rule.ReplacedBy) != nil {
				return nil, invalid(entryLocation+".replacedBy", "expected a rule ID")
			}
			if _, current := versions[rule.ReplacedBy]; !current {
				return nil, invalid(entryLocation+".replacedBy", "replacement "+quote(rule.ReplacedBy)+" is missing from rules")
			}
		}
		retired[id] = rule
	}
	return retired, nil
}

// libraryFiles reads contained relative paths, keeping their order and rejecting duplicates and files that
// belong to a rule's version.
func libraryFiles(input json.RawMessage, location string) ([]string, error) {
	var items []json.RawMessage
	if json.Unmarshal(input, &items) != nil || items == nil {
		return nil, invalid(location, "expected a list of file paths")
	}
	paths := []string{}
	for i, item := range items {
		path, err := jsonPath(item, location+"["+strconv.Itoa(i)+"]")
		if err != nil {
			return nil, err
		}
		if slices.Contains(paths, path) {
			return nil, invalid(location+"["+strconv.Itoa(i)+"]", "duplicate path "+quote(path))
		}
		if IsRuleContent(path) {
			return nil, invalid(location+"["+strconv.Itoa(i)+"]", quote(path)+" belongs to a rule's version, not the library-wide files")
		}
		paths = append(paths, path)
	}
	return paths, nil
}

// IsRuleContent reports whether path is part of some rule's version: a rule's Markdown file, or a file in an asset
// directory inside a technology or practice group, since every such directory belongs to a rule. Group metadata,
// group READMEs, and the library-root assets/ directory are library-wide.
func IsRuleContent(path string) bool {
	parts := strings.Split(path, "/")
	if parts[0] != "techs" && parts[0] != "practices" {
		return false
	}
	if slices.Contains(parts[1:], "assets") {
		return true
	}
	if _, err := GroupFromPath(path, path); err == nil {
		return !IsGroupReadme(path)
	}
	return false
}

// versionField reads a rule version from YAML text. 1.3.0 is text in YAML; a number such as 1.0 is rejected.
func versionField(input json.RawMessage, location string) (RuleVersion, error) {
	var text string
	if json.Unmarshal(input, &text) != nil {
		return RuleVersion{}, invalid(location, "expected a rule version, such as 1.3.0")
	}
	return ParseRuleVersion(text, location)
}
