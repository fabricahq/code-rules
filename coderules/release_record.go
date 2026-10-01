// Parse the release record in a library release tag's message without accessing Git.

package coderules

import (
	"encoding/json"
	"maps"
	"slices"
	"strconv"

	"go.yaml.in/yaml/v4"

	"github.com/fabricahq/code-rules/internal/authored"
	"github.com/fabricahq/code-rules/internal/librarypath"
)

// ReleaseRecord is the permanent record of one library release, read from its release/<number> tag's message. Rule
// IDs are library rule IDs, such as practices/testing/verify-retries. A parsed record's maps and LibraryFiles are
// never nil.
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
	// Summaries holds one summary per change note that named the rule, in note order; it is never empty, and each
	// is one line of text without control characters.
	Summaries []string `json:"summaries"`
}

// RetiredRule is a rule a library release retired. ReplacedBy is empty when nothing replaces it. Summaries is
// as in RecordedChange.
type RetiredRule struct {
	LastVersion RuleVersion `json:"lastVersion"`
	ReplacedBy  string      `json:"replacedBy,omitempty"`
	Summaries   []string    `json:"summaries"`
}

// ReleaseRecordFormat is the newest release record format this version of Code Rules writes and reads, recorded in
// each record's formatVersion. It changes only for incompatible changes: readers ignore fields they don't know, so new
// optional fields keep the format.
const ReleaseRecordFormat = 1

// UnsupportedReleaseRecordError reports a release record whose formatVersion is above ReleaseRecordFormat: a newer
// Code Rules published it, and only a version of Code Rules, and of this package, that supports its format can read
// it. Check for it with errors.As.
type UnsupportedReleaseRecordError struct {
	// Location is the record's formatVersion field, such as release/4.formatVersion.
	Location string
	// FormatVersion is the record's format, above ReleaseRecordFormat.
	FormatVersion int
}

// Error names the record's format and asks the reader to upgrade Code Rules.
func (e *UnsupportedReleaseRecordError) Error() string {
	return e.Location + ": the release record uses format " + strconv.Itoa(e.FormatVersion) + ", but this version of Code Rules reads only format " + strconv.Itoa(ReleaseRecordFormat) + "; upgrade Code Rules to read this library release"
}

// Limits on a release record's collections bound the work of reading a record from a library nobody vetted. A
// library holds at most 10,000 files, so a library release can't list more rules, changes, or retirements, and its
// libraryFiles can name at most every file of the previous library release and of the new one.
const (
	maxRecordedRules = 10_000
	maxLibraryFiles  = 20_000
)

// requireAtMost fails when a collection at location holds more than limit entries.
func requireAtMost(count, limit int, location string) error {
	if count > limit {
		return authored.Invalid(location, "expected at most "+thousands(limit)+" entries")
	}
	return nil
}

// thousands writes a positive count with comma separators, such as 10,000.
func thousands(count int) string {
	text := strconv.Itoa(count)
	for i := len(text) - 3; i > 0; i -= 3 {
		text = text[:i] + "," + text[i:]
	}
	return text
}

// ParseReleaseRecord parses a release record: the YAML after the last --- line of a library release tag's message.
// location names the record in errors, such as its tag name. It reads formatVersion before anything else and
// fails with *UnsupportedReleaseRecordError when it's above ReleaseRecordFormat, so it never misreads a newer
// format. It ignores fields it doesn't know, at any level and whatever their values, so later formats can add
// fields that older readers skip; the record must still be one YAML document without anchors, aliases, explicit
// tags, or duplicate keys. It validates every field it knows, including that each change leads to the version in
// rules and that the first library release lists every rule as new, and it refuses more than 10,000 entries in
// rules, changes, or retired, and more than 20,000 in libraryFiles.
func ParseReleaseRecord(input []byte, location string) (ReleaseRecord, error) {
	document, err := authored.Document(input, location)
	if err != nil {
		return ReleaseRecord{}, err
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return ReleaseRecord{}, authored.Invalid(location, "expected an object")
	}
	// The format comes first: a later format may hold content this version can't interpret at all.
	var format json.RawMessage
	if node := field(root, "formatVersion"); node != nil {
		// A value that isn't a plain scalar leaves format empty, which recordFormat reports as invalid.
		format, _ = authored.JSON(node, location+".formatVersion")
	}
	if err := recordFormat(format, location+".formatVersion"); err != nil {
		return ReleaseRecord{}, err
	}
	fields, err := knownFields(root, location, map[string][]string{"release": nil, "rules": nil, "libraryFiles": nil, "changes": {"change", "from", "summaries"}, "retired": {"lastVersion", "replacedBy", "summaries"}})
	if err != nil {
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

// field returns the value of mapping's field name, or nil when it has none.
func field(mapping *yaml.Node, name string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if key := mapping.Content[i]; key.Tag == "!!str" && key.Value == name {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// knownFields converts to JSON, strictly, the fields of mapping that known names, in document order, and ignores
// every other field, whatever its value. A known field with entry field names holds a mapping from IDs to entries,
// whose named fields are converted the same way; an entry, or a known field, of any other shape converts whole,
// so its validator reports the wrong type.
func knownFields(mapping *yaml.Node, location string, known map[string][]string) (map[string]json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i], mapping.Content[i+1]
		entryFields, isKnown := known[key.Value]
		if key.Tag != "!!str" || !isKnown {
			continue
		}
		where := location + "." + key.Value
		var err error
		if entryFields == nil || value.Kind != yaml.MappingNode {
			fields[key.Value], err = authored.JSON(value, where)
		} else {
			fields[key.Value], err = knownEntries(value, where, entryFields)
		}
		if err != nil {
			return nil, err
		}
	}
	return fields, nil
}

// knownEntries converts a mapping from IDs to entries to a JSON object, keeping only each entry's named fields.
func knownEntries(mapping *yaml.Node, location string, names []string) (json.RawMessage, error) {
	entries := map[string]json.RawMessage{}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i], mapping.Content[i+1]
		if key.Tag != "!!str" {
			return nil, authored.Invalid(location, "mapping keys must be strings; quote wildcard selectors")
		}
		where := location + "." + key.Value
		var err error
		if value.Kind != yaml.MappingNode {
			entries[key.Value], err = authored.JSON(value, where)
		} else {
			known := map[string][]string{}
			for _, name := range names {
				known[name] = nil
			}
			var fields map[string]json.RawMessage
			if fields, err = knownFields(value, where, known); err == nil {
				entries[key.Value], err = json.Marshal(fields)
			}
		}
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(entries)
}

// requireFirstRelease requires the first library release to publish every rule as new, so each rule's history
// starts with a changes entry for 1.0.0, and to retire nothing, since no rule had a version before it.
func requireFirstRelease(record ReleaseRecord, location string) error {
	if len(record.Retired) > 0 {
		return authored.Invalid(location+".retired", "the first library release can't retire rules")
	}
	for _, id := range slices.Sorted(maps.Keys(record.Rules)) {
		if change, ok := record.Changes[id]; !ok || change.Change != ChangeNew {
			return authored.Invalid(location+".changes."+id, "the first library release must list every rule as new")
		}
	}
	return nil
}

// recordFormat accepts ReleaseRecordFormat, reporting a larger whole number as a format this version can't read.
func recordFormat(input json.RawMessage, location string) error {
	format, err := strconv.Atoi(string(input))
	switch {
	case err != nil || format < 1:
		return authored.Invalid(location, "expected the release record format, a whole number such as 1")
	case format > ReleaseRecordFormat:
		return &UnsupportedReleaseRecordError{Location: location, FormatVersion: format}
	}
	return nil
}

// releaseNumber accepts a positive integer below one billion.
func releaseNumber(input json.RawMessage, location string) (int, error) {
	number, err := strconv.Atoi(string(input))
	if err != nil || number < 1 || number >= 1_000_000_000 {
		return 0, authored.Invalid(location, "expected a library release number, starting at 1")
	}
	return number, nil
}

// recordedVersions reads the required map of every current rule to its version; it may be empty.
func recordedVersions(input json.RawMessage, location string) (map[string]RuleVersion, error) {
	entries, err := authored.Object(input, location)
	if err != nil {
		return nil, err
	}
	if err := requireAtMost(len(entries), maxRecordedRules, location); err != nil {
		return nil, err
	}
	versions := map[string]RuleVersion{}
	for _, id := range slices.Sorted(maps.Keys(entries)) {
		if err := librarypath.ValidateRuleID(id, location+"."+id); err != nil {
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
	entries, err := authored.Object(input, location)
	if err != nil {
		return nil, err
	}
	if err := requireAtMost(len(entries), maxRecordedRules, location); err != nil {
		return nil, err
	}
	changes := map[string]RecordedChange{}
	for _, id := range slices.Sorted(maps.Keys(entries)) {
		entryLocation := location + "." + id
		if err := librarypath.ValidateRuleID(id, entryLocation); err != nil {
			return nil, err
		}
		fields, err := authored.Object(entries[id], entryLocation)
		if err != nil {
			return nil, err
		}
		var change RecordedChange
		var name string
		if json.Unmarshal(fields["change"], &name) != nil {
			return nil, authored.Invalid(entryLocation+".change", "expected new, major, minor, or patch")
		}
		change.Change = Change(name)
		if change.Summaries, err = recordSummaries(fields["summaries"], entryLocation+".summaries"); err != nil {
			return nil, err
		}
		current, ok := versions[id]
		if !ok {
			return nil, authored.Invalid(entryLocation, "changed rule is missing from rules")
		}
		raw, hasFrom := fields["from"]
		switch change.Change {
		case ChangeNew:
			if hasFrom {
				return nil, authored.Invalid(entryLocation+".from", "a new rule has no previous version")
			}
			if current != FirstRuleVersion {
				return nil, authored.Invalid(entryLocation, "a new rule must be version 1.0.0 in rules, got "+current.String())
			}
		case ChangeMajor, ChangeMinor, ChangePatch:
			if !hasFrom {
				return nil, authored.Invalid(entryLocation+".from", "expected the rule's previous version")
			}
			from, err := versionField(raw, entryLocation+".from")
			if err != nil {
				return nil, err
			}
			next, err := from.Next(change.Change)
			if err != nil {
				return nil, authored.Invalid(entryLocation, err.Error())
			}
			if next != current {
				return nil, authored.Invalid(entryLocation, "a "+name+" change from "+from.String()+" leads to "+next.String()+", but rules records "+current.String())
			}
			change.From = &from
		default:
			return nil, authored.Invalid(entryLocation+".change", "unknown change "+authored.Quote(name)+"; expected new, major, minor, or patch")
		}
		changes[id] = change
	}
	return changes, nil
}

// retiredRules requires each retired rule to be absent from rules and any replacement to be present.
func retiredRules(input json.RawMessage, versions map[string]RuleVersion, location string) (map[string]RetiredRule, error) {
	entries, err := authored.Object(input, location)
	if err != nil {
		return nil, err
	}
	if err := requireAtMost(len(entries), maxRecordedRules, location); err != nil {
		return nil, err
	}
	retired := map[string]RetiredRule{}
	for _, id := range slices.Sorted(maps.Keys(entries)) {
		entryLocation := location + "." + id
		if err := librarypath.ValidateRuleID(id, entryLocation); err != nil {
			return nil, err
		}
		if _, current := versions[id]; current {
			return nil, authored.Invalid(entryLocation, "a retired rule can't also be in rules")
		}
		fields, err := authored.Object(entries[id], entryLocation)
		if err != nil {
			return nil, err
		}
		var rule RetiredRule
		if rule.LastVersion, err = versionField(fields["lastVersion"], entryLocation+".lastVersion"); err != nil {
			return nil, err
		}
		if rule.Summaries, err = recordSummaries(fields["summaries"], entryLocation+".summaries"); err != nil {
			return nil, err
		}
		if raw, ok := fields["replacedBy"]; ok {
			if json.Unmarshal(raw, &rule.ReplacedBy) != nil {
				return nil, authored.Invalid(entryLocation+".replacedBy", "expected a rule ID")
			}
			if _, current := versions[rule.ReplacedBy]; !current {
				return nil, authored.Invalid(entryLocation+".replacedBy", "replacement "+authored.Quote(rule.ReplacedBy)+" is missing from rules")
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
		return nil, authored.Invalid(location, "expected a list of file paths")
	}
	if err := requireAtMost(len(items), maxLibraryFiles, location); err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(items))
	seen := make(map[string]bool, len(items))
	for i, item := range items {
		path, err := authored.Path(item, location+"["+strconv.Itoa(i)+"]")
		if err != nil {
			return nil, err
		}
		if seen[path] {
			return nil, authored.Invalid(location+"["+strconv.Itoa(i)+"]", "duplicate path "+authored.Quote(path))
		}
		if librarypath.IsRuleContent(path) {
			return nil, authored.Invalid(location+"["+strconv.Itoa(i)+"]", authored.Quote(path)+" belongs to a rule's version, not the library-wide files")
		}
		seen[path] = true
		paths = append(paths, path)
	}
	return paths, nil
}

// versionField reads a rule version from YAML text. 1.3.0 is text in YAML; a number such as 1.0 is rejected.
func versionField(input json.RawMessage, location string) (RuleVersion, error) {
	var text string
	if json.Unmarshal(input, &text) != nil {
		return RuleVersion{}, authored.Invalid(location, "expected a rule version, such as 1.3.0")
	}
	return ParseRuleVersion(text, location)
}

// recordSummaries reads a release record's non-empty list of summaries, one per change note, in order. Each follows
// a change note's summary rules.
func recordSummaries(input json.RawMessage, location string) ([]string, error) {
	var items []json.RawMessage
	if json.Unmarshal(input, &items) != nil || len(items) == 0 {
		return nil, authored.Invalid(location, "expected a list with one summary per change note")
	}
	summaries := make([]string, len(items))
	for i, item := range items {
		summary, err := authored.Line(item, location+"["+strconv.Itoa(i)+"]")
		if err != nil {
			return nil, err
		}
		summaries[i] = summary
	}
	return summaries, nil
}
