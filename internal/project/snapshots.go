// Encode, read, and verify persisted source snapshots, the lockfile that sync restores and offline builds verify.

package project

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// snapshot is the library-owned value persisted by project snapshot encoding.
type snapshot = library.Snapshot

// sourceRecordFormat is the only supported _source.json format.
const sourceRecordFormat = 2

// sourceRecord is the format 2 record in vendor/<source>/_source.json: the source's configuration when it was
// recorded, the revision that supplied its library-wide files, each imported rule's version, and file digests.
type sourceRecord struct {
	FormatVersion int                          `json:"formatVersion"`
	Repository    string                       `json:"repository"`
	Pins          map[string]rules.RuleVersion `json:"pins,omitempty"`
	Ref           string                       `json:"ref,omitempty"`
	Release       int                          `json:"release,omitempty"`
	Commit        string                       `json:"resolvedCommit"`
	Selection     json.RawMessage              `json:"groupSelection"`
	RuleSelection []string                     `json:"ruleSelection,omitempty"`
	Groups        []string                     `json:"groups"`
	Rules         map[string]recordRule        `json:"rules"`
	RetiredRules  []string                     `json:"retiredRules"`
	Files         map[string]string            `json:"files"`
}

// recordRule is one imported rule's version record; Version and Release are null for an unreleased rule.
type recordRule struct {
	Version *rules.RuleVersion `json:"version"`
	Release *int               `json:"release"`
	Commit  string             `json:"commit"`
}

// unsupportedRecord explains how to replace a source record this version can't read.
const unsupportedRecord = "unsupported source record; delete .code-rules/vendor/ and run code-rules project sync to import it again"

// encodeSnapshots prepares complete vendor bytes for parsed configuration, without writing files. Each snapshot
// must satisfy the same checks offline builds apply. Inputs stay unchanged; returned maps and byte slices belong
// to the caller. Errors return nil.
func encodeSnapshots(config rules.Configuration, snapshots map[string]snapshot) (map[string][]byte, error) {
	output := map[string][]byte{}
	if len(snapshots) != len(config.Sources) {
		return nil, invalidSnapshot("vendor", "source inventory differs from configuration; run code-rules project sync")
	}
	for _, source := range config.Sources {
		snapshot, ok := snapshots[source.Name]
		if !ok {
			return nil, invalidSnapshot(source.Name, "missing source snapshot; run code-rules project sync")
		}
		selection, err := json.Marshal(snapshot.Selection)
		if err != nil {
			return nil, fmt.Errorf("encode selection for %s: %v", source.Name, err)
		}
		record := sourceRecord{FormatVersion: sourceRecordFormat, Repository: snapshot.Repository, Ref: snapshot.Ref.String(), Release: snapshot.Release, Commit: snapshot.Commit, Selection: selection, RuleSelection: snapshot.RuleSelection, Groups: snapshot.Groups, Rules: map[string]recordRule{}, Files: map[string]string{}}
		if len(snapshot.Pins) > 0 {
			record.Pins = snapshot.Pins
		}
		record.RetiredRules = snapshot.RetiredRules
		if record.RetiredRules == nil {
			record.RetiredRules = []string{}
		}
		for id, rule := range snapshot.Rules {
			entry := recordRule{Version: rule.Version, Commit: rule.Commit}
			if rule.Version != nil {
				entry.Release = &rule.Release
			}
			record.Rules[id] = entry
		}
		if err := rules.ValidatePaths(snapshot.Files, nil); err != nil {
			return nil, err
		}
		for _, file := range slices.Sorted(maps.Keys(snapshot.Files)) {
			if file == "_source.json" {
				return nil, invalidSnapshot(source.Name, "_source.json is reserved for the source record")
			}
			record.Files[file] = digest(snapshot.Files[file])
			output[source.Name+"/"+file] = bytes.Clone(snapshot.Files[file])
		}
		var encoded bytes.Buffer
		encoder := json.NewEncoder(&encoded)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(record); err != nil {
			return nil, fmt.Errorf("encode snapshot %s: %v", source.Name, err)
		}
		// Validate the same persisted representation the reader will consume.
		parsed, err := parseSourceRecord(encoded.Bytes(), source.Name)
		if err != nil {
			return nil, err
		}
		if err := matchSnapshotSource(source, parsed); err != nil {
			return nil, err
		}
		output[source.Name+"/_source.json"] = encoded.Bytes()
	}
	if err := rules.ValidatePaths(output, nil); err != nil {
		return nil, err
	}
	return output, nil
}

// decodeSnapshots verifies every original byte, each record, and its agreement with configuration before
// returning owned snapshots. Unexpected source files and incomplete records fail closed. It performs no
// filesystem or Git access. Rule and manifest validation remains the catalog loader's responsibility after
// this integrity check.
func decodeSnapshots(config rules.Configuration, vendor map[string][]byte) (map[string]snapshot, error) {
	if err := rules.ValidatePaths(vendor, nil); err != nil {
		return nil, err
	}
	result := map[string]snapshot{}
	expected := map[string]bool{}
	for _, source := range config.Sources {
		recordPath := source.Name + "/_source.json"
		data, ok := vendor[recordPath]
		if !ok {
			return nil, invalidSnapshot(recordPath, "missing source record; run code-rules project sync")
		}
		record, err := parseSourceRecord(data, source.Name)
		if err != nil {
			return nil, err
		}
		if err := matchSnapshotSource(source, record); err != nil {
			return nil, err
		}
		expected[recordPath] = true
		record.Files = map[string][]byte{}
		for _, file := range slices.Sorted(maps.Keys(record.digests)) {
			full := source.Name + "/" + file
			data, ok := vendor[full]
			if !ok || digest(data) != record.digests[file] {
				return nil, invalidSnapshot(full, "missing or modified imported content; run code-rules project sync")
			}
			record.Files[file] = bytes.Clone(data)
			expected[full] = true
		}
		result[source.Name] = record.Snapshot
	}
	for _, file := range slices.Sorted(maps.Keys(vendor)) {
		if !expected[file] {
			return nil, invalidSnapshot(file, "unexpected imported file or removed source; run code-rules project sync")
		}
	}
	return result, nil
}

// recordedSnapshots reads the snapshot record of each configured source that has one, without files or checking
// it against configuration, so sync can restore its versions. A source without a record is absent from the
// result. An unsupported or invalid record fails, explaining how to import the source again.
func recordedSnapshots(config rules.Configuration, vendor map[string][]byte) (map[string]library.Snapshot, error) {
	result := map[string]library.Snapshot{}
	for _, source := range config.Sources {
		data, ok := vendor[source.Name+"/_source.json"]
		if !ok {
			continue
		}
		record, err := parseSourceRecord(data, source.Name)
		if err != nil {
			return nil, err
		}
		result[source.Name] = record.Snapshot
	}
	return result, nil
}

// parsedRecord is a validated source record: its snapshot, without files, and its file digests by stored path.
type parsedRecord struct {
	library.Snapshot
	digests map[string]string
}

// parseSourceRecord validates a format 2 record's exact fields, value syntax, and complete digest inventory.
// Relationships to configuration are matchSnapshotSource's. An older format, such as 1, fails validation with advice
// to import the source again; a newer one fails with code unsupported-source-record, asking to upgrade Code Rules.
func parseSourceRecord(data []byte, name string) (parsedRecord, error) {
	where := name + "/_source.json"
	var fields map[string]json.RawMessage
	if !utf8.Valid(data) || json.Unmarshal(data, &fields) != nil || fields == nil {
		return parsedRecord{}, invalidSnapshot(where, "expected a UTF-8 source record object")
	}
	var format int
	if json.Unmarshal(fields["formatVersion"], &format) == nil && format > sourceRecordFormat {
		// Deleting vendor/ and syncing would rewrite the project in an older format, so this isn't the advice.
		return parsedRecord{}, failure("unsupported-source-record", fmt.Sprintf("%s was written by a newer version of Code Rules, in source record format %d, but this version reads only format %d; upgrade Code Rules to use this project", where, format, sourceRecordFormat), nil)
	}
	if format != sourceRecordFormat {
		return parsedRecord{}, invalidSnapshot(where, unsupportedRecord)
	}
	allowed := []string{"formatVersion", "repository", "pins", "ref", "release", "resolvedCommit", "groupSelection", "ruleSelection", "groups", "rules", "retiredRules", "files"}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if !slices.Contains(allowed, key) || bytes.Equal(bytes.TrimSpace(fields[key]), []byte("null")) {
			return parsedRecord{}, invalidSnapshot(where+"."+key, "unknown or null source record field")
		}
	}
	for _, key := range []string{"repository", "resolvedCommit", "groupSelection", "groups", "rules", "retiredRules", "files"} {
		if _, ok := fields[key]; !ok {
			return parsedRecord{}, invalidSnapshot(where+"."+key, "missing source record field")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record sourceRecord
	if decoder.Decode(&record) != nil {
		return parsedRecord{}, invalidSnapshot(where, "invalid source record field values")
	}
	result := parsedRecord{Snapshot: library.Snapshot{Repository: record.Repository, Pins: map[string]rules.RuleVersion{}, Release: record.Release, Commit: record.Commit, RuleSelection: []string{}, Rules: map[string]library.ImportedRule{}}, digests: record.Files}
	if _, err := rules.ParseRepository(fields["repository"], where+".repository"); err != nil {
		return parsedRecord{}, err
	}
	if record.Ref != "" {
		var err error
		if result.Ref, err = rules.ParseGitRef(record.Ref, where+".ref"); err != nil {
			return parsedRecord{}, err
		}
	}
	if raw, ok := fields["release"]; ok && record.Release < 1 {
		return parsedRecord{}, invalidSnapshot(where+".release", "expected a library release number, got "+string(raw))
	}
	if !fullCommit(record.Commit) {
		return parsedRecord{}, invalidSnapshot(where+".resolvedCommit", "expected a full commit SHA")
	}
	for id, version := range record.Pins {
		if err := rules.ValidateRuleID(id, where+".pins."+id); err != nil {
			return parsedRecord{}, err
		}
		result.Pins[id] = version
	}
	var err error
	if result.Selection, err = rules.ParseGroupSelection(fields["groupSelection"], where+".groupSelection"); err != nil {
		return parsedRecord{}, err
	}
	if result.RetiredRules, err = rules.ParseRuleList(fields["retiredRules"], where+".retiredRules"); err != nil {
		return parsedRecord{}, err
	}
	if raw, ok := fields["ruleSelection"]; ok {
		if result.RuleSelection, err = rules.ParseRuleList(raw, where+".ruleSelection"); err != nil {
			return parsedRecord{}, err
		}
	}
	groups, err := rules.ParseGroupSelection(fields["groups"], where+".groups")
	if err != nil {
		return parsedRecord{}, err
	}
	if groups.Pattern != "" {
		return parsedRecord{}, invalidSnapshot(where+".groups", "expected resolved group IDs, not a wildcard")
	}
	result.Groups = groups.Groups
	for id, rule := range record.Rules {
		location := where + ".rules." + id
		if err := rules.ValidateRuleID(id, location); err != nil {
			return parsedRecord{}, err
		}
		if (rule.Version == nil) != (rule.Release == nil) || (rule.Release != nil && *rule.Release < 1) || !fullCommit(rule.Commit) {
			return parsedRecord{}, invalidSnapshot(location, "expected a version, the library release that published it, and its commit, or a null version and release")
		}
		imported := library.ImportedRule{Version: rule.Version, Commit: rule.Commit}
		if rule.Release != nil {
			imported.Release = *rule.Release
		}
		result.Rules[id] = imported
	}
	paths := map[string][]byte{}
	for file, hash := range record.Files {
		decoded, err := hex.DecodeString(hash)
		if err != nil || len(decoded) != sha256.Size || strings.ToLower(hash) != hash || file == "_source.json" {
			return parsedRecord{}, invalidSnapshot(where, "invalid or reserved file digest")
		}
		paths[file] = nil
	}
	if err := rules.ValidatePaths(paths, nil); err != nil {
		return parsedRecord{}, fmt.Errorf("%s: %w", where, err)
	}
	if _, ok := paths["rule-library.yaml"]; !ok {
		return parsedRecord{}, invalidSnapshot(where, "missing library manifest from inventory")
	}
	for _, group := range result.Groups {
		if _, ok := paths[group+"/_group.yaml"]; !ok {
			return parsedRecord{}, invalidSnapshot(where, "missing group metadata from inventory: "+group)
		}
	}
	return result, nil
}

// matchSnapshotSource checks a record against its source's configuration without fetching Git, as documented in
// "What offline checks can verify": the recorded identity and selections, pinned versions, and ref.
func matchSnapshotSource(source rules.Source, record parsedRecord) error {
	where := source.Name + "/_source.json"
	if source.Repository != record.Repository || !source.Ref.Equal(record.Ref) || source.Groups.Pattern != record.Selection.Pattern || !slices.Equal(source.Groups.Groups, record.Selection.Groups) || !slices.Equal(source.Rules, record.RuleSelection) {
		return invalidSnapshot(where, "source identity or selection changed; run code-rules project sync")
	}
	for _, id := range slices.Sorted(maps.Keys(source.Pins)) {
		pin := source.Pins[id]
		rule, imported := record.Rules[id]
		recordedPin, recorded := record.Pins[id]
		if imported && (rule.Version == nil || *rule.Version != pin.Version) || !imported && (!recorded || recordedPin != pin.Version) {
			return invalidSnapshot(where, "sources."+source.Name+".pins."+id+" names a version the snapshot doesn't import; run code-rules project sync")
		}
	}
	// An exclusion naming neither an imported rule nor a retired one could be a typo that leaves the intended rule
	// active; only the library, which sync reads, can tell.
	for _, id := range slices.Sorted(maps.Keys(source.Exclude)) {
		if _, imported := record.Rules[id]; !imported && !slices.Contains(record.RetiredRules, id) {
			return invalidSnapshot("sources."+source.Name+".exclude."+id, "names no rule the library supplies to this source; run code-rules project sync to check it against the library")
		}
	}
	if source.Groups.Pattern == "" && !slices.Equal(record.Groups, source.Groups.Groups) {
		return invalidSnapshot(where, "resolved groups differ from the explicit selection")
	}
	for _, group := range record.Groups {
		if !source.Groups.Includes(group) {
			return invalidSnapshot(where, "resolved group is outside the requested selection")
		}
	}
	if err := matchRevision(source, record); err != nil {
		return err
	}
	newest := 0
	commits := map[int]string{}
	for _, id := range slices.Sorted(maps.Keys(record.Rules)) {
		rule := record.Rules[id]
		if rule.Version == nil && record.Release != 0 {
			return invalidSnapshot(where+".rules."+id, "only a ref that isn't a library release can import a rule without a version")
		}
		if (rule.Version == nil || rule.Release == record.Release) && rule.Commit != record.Commit {
			return invalidSnapshot(where+".rules."+id, "the rule's commit differs from the resolved commit of its library release")
		}
		if commit, seen := commits[rule.Release]; rule.Version != nil && seen && commit != rule.Commit {
			return invalidSnapshot(where+".rules."+id, "rules from one library release record different commits")
		}
		commits[rule.Release] = rule.Commit
		newest = max(newest, rule.Release)
	}
	if source.Ref.IsZero() && record.Release < newest {
		return invalidSnapshot(where+".release", "expected the library release that supplies the shared files to be at least as new as every imported rule version's library release")
	}
	return nil
}

// matchRevision checks the record's library release and commit against how the source chooses versions.
func matchRevision(source rules.Source, record parsedRecord) error {
	where := source.Name + "/_source.json"
	ref := source.Ref
	if ref.IsZero() {
		if record.Release == 0 {
			return invalidSnapshot(where+".release", "a source without ref records the library release that supplied its files")
		}
		return nil
	}
	if ref.Kind() == rules.GitRefCommit {
		if record.Commit != ref.Canonical() {
			return invalidSnapshot(where, "resolved commit differs from requested commit")
		}
	}
	number, err := rules.ParseReleaseTag(strings.TrimPrefix(ref.Canonical(), "refs/tags/"))
	if ref.Kind() != rules.GitRefTag || err != nil {
		if record.Release != 0 {
			return invalidSnapshot(where+".release", "a ref that isn't a library release tag records no library release")
		}
		return nil
	}
	if record.Release != number {
		return invalidSnapshot(where+".release", "expected the library release that sources."+source.Name+".ref names")
	}
	for _, id := range slices.Sorted(maps.Keys(record.Rules)) {
		if record.Rules[id].Release > number {
			return invalidSnapshot(where+".rules."+id, "a rule imported from "+source.Ref.String()+" can't come from a later library release")
		}
	}
	return nil
}

// fullCommit accepts a lowercase 40-digit commit SHA.
func fullCommit(text string) bool {
	ref, err := rules.ParseGitRef(text, "commit")
	return err == nil && ref.Kind() == rules.GitRefCommit && ref.Canonical() == text
}

// digest hashes exact bytes, preserving binary data and line endings.
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// invalidSnapshot reports caller-visible validation failures without logging source contents.
func invalidSnapshot(location, problem string) error {
	return &rules.ValidationError{Location: location, Problem: problem}
}
