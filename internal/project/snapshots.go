// Encode, read, and verify persisted source snapshots, the lockfile that sync restores and offline builds verify.

package project

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/fabricahq/code-rules/coderules"
	"github.com/fabricahq/code-rules/internal/errs"
	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/librarytree"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/fabricahq/code-rules/internal/skills"
)

// snapshot is the library-owned value persisted by project snapshot encoding.
type snapshot = library.Snapshot

// sourceRecordFormat is the only supported _source.json format.
const sourceRecordFormat = 3

// The kinds of source a record names: a library, whose repository has a library manifest at the recorded commit, or
// another repository, which supplies only skills.
const (
	libraryKind    = "library"
	nonLibraryKind = "non-library"
)

// sourceRecord is the format 3 record in vendor/<source>/_source.json: the source's kind and configuration when it
// was recorded, the revision that supplied its library-wide files and every skill, each imported rule's version, each
// installed skill's level, and file digests and modes. A record of a source that isn't a library has no release, and
// empty group and rule fields.
type sourceRecord struct {
	FormatVersion int `json:"formatVersion"`
	// Checksum is the SHA-256 of the record's other fields, so offline checks can tell a record changed outside
	// sync, such as by hand or in a merge, from one sync wrote.
	Checksum      string                 `json:"checksum,omitempty"`
	Kind          string                 `json:"kind"`
	Repository    string                 `json:"repository"`
	Ref           string                 `json:"ref,omitempty"`
	Release       int                    `json:"release,omitempty"`
	Commit        string                 `json:"resolvedCommit"`
	Selection     json.RawMessage        `json:"groupSelection"`
	RuleSelection []string               `json:"ruleSelection,omitempty"`
	Groups        []string               `json:"groups"`
	Rules         map[string]recordRule  `json:"rules"`
	RetiredRules  []string               `json:"retiredRules"`
	SkillInclude  []string               `json:"skillInclude"`
	Skills        map[string]recordSkill `json:"skills"`
	Files         map[string]string      `json:"files"`
	Executable    []string               `json:"executable"`
}

// recordSkill is one installed skill's record. Level is null for a skill from a source that isn't a library.
type recordSkill struct {
	Level *string `json:"level"`
}

// recordRule is one imported rule's version record; Version and Release are null for an unreleased rule.
type recordRule struct {
	Version *coderules.RuleVersion `json:"version"`
	Release *int                   `json:"release"`
	Commit  string                 `json:"commit"`
	// LegacyLayout is library.ImportedRule's.
	LegacyLayout bool `json:"legacyLayout,omitempty"`
}

// unsupportedRecord explains how to replace a source record this version can't read.
const unsupportedRecord = "unsupported source record; delete .code-rules/vendor/ and run code-rules project sync to import it again, which imports unpinned rules at their newest versions"

// reimport is the next step for a source record sync can't read: importing the source again, as if it were new.
func reimport(name string) string {
	return "delete .code-rules/vendor/" + name + "/ and run code-rules project sync to import the source again, which imports its unpinned rules at their newest versions"
}

// changedOutsideSync refuses the source record of the source name, whose checksum shows it was changed outside sync,
// for offline commands and sync alike, since nothing in it can be trusted, and says how to recover: restore a record
// sync wrote, or import the source again.
func changedOutsideSync(name string) error {
	record := ".code-rules/vendor/" + name + "/_source.json"
	return invalidSnapshot("vendor/"+name+"/_source.json", "was changed outside code-rules project sync, such as by hand or in a merge, and only a record sync wrote can be trusted; restore one with git checkout -- "+record+", or, during a merge conflict, take one side with git checkout --ours or --theirs -- "+record+", then run code-rules project sync, which applies any configuration changes; or delete .code-rules/vendor/"+name+"/ and run code-rules project sync to import the source again, which imports its unpinned rules at their newest versions")
}

// isChangedOutsideSync reports whether err is changedOutsideSync's refusal of the source name's record.
func isChangedOutsideSync(err error, name string) bool {
	var invalid errs.ValidationError
	refusal := changedOutsideSync(name).(errs.ValidationError)
	return errors.As(err, &invalid) && invalid.ValidationLocation() == refusal.ValidationLocation() && invalid.ValidationProblem() == refusal.ValidationProblem()
}

// recordChecksum returns the SHA-256 of record's fields other than its checksum, encoded as sync writes them.
func recordChecksum(record sourceRecord) string {
	record.Checksum = ""
	encoded, err := json.Marshal(record)
	if err != nil {
		return ""
	}
	return digest(encoded)
}

// encodeSnapshots prepares complete vendor bytes for parsed configuration, without writing files. Each source's
// record lists its files by library path, and each file is stored where librarytree.SnapshotPath places it, so the
// snapshot mirrors the library's Code Rules directory. Each snapshot must satisfy the same checks offline builds
// apply. Inputs stay unchanged; returned maps and byte slices belong to the caller. Errors return nil.
func encodeSnapshots(config rules.Configuration, snapshots map[string]snapshot) (map[string][]byte, error) {
	output := map[string][]byte{}
	if len(snapshots) != len(config.Sources) {
		return nil, invalidSnapshot("vendor", "source inventory differs from configuration; run code-rules project sync")
	}
	for _, source := range config.Sources {
		snapshot, ok := snapshots[source.Name]
		if !ok {
			return nil, invalidSnapshot("vendor/"+source.Name, "missing source snapshot; run code-rules project sync")
		}
		selection, err := json.Marshal(snapshot.Selection)
		if err != nil {
			return nil, fmt.Errorf("encode selection for %s: %v", source.Name, err)
		}
		kind := libraryKind
		if snapshot.NonLibrary {
			kind = nonLibraryKind
		}
		record := sourceRecord{FormatVersion: sourceRecordFormat, Kind: kind, Repository: snapshot.Repository, Ref: snapshot.Ref.String(), Release: snapshot.Release, Commit: snapshot.Commit, Selection: selection, RuleSelection: snapshot.RuleSelection, Groups: nonNil(snapshot.Groups), Rules: map[string]recordRule{}, Files: map[string]string{}}
		record.RetiredRules = nonNil(snapshot.RetiredRules)
		record.SkillInclude = nonNil(snapshot.SkillSelection)
		record.Executable = nonNil(snapshot.Executable)
		record.Skills = map[string]recordSkill{}
		for id, skill := range snapshot.Skills {
			entry := recordSkill{}
			if skill.Level != "" {
				entry.Level = &skill.Level
			}
			record.Skills[id] = entry
		}
		for id, rule := range snapshot.Rules {
			entry := recordRule{Version: rule.Version, Commit: rule.Commit, LegacyLayout: rule.LegacyLayout}
			if rule.Version != nil {
				entry.Release = &rule.Release
			}
			record.Rules[id] = entry
		}
		if err := rules.ValidatePaths(snapshot.Files, nil); err != nil {
			return nil, err
		}
		placed := map[string]string{}
		for _, file := range slices.Sorted(maps.Keys(snapshot.Files)) {
			if file == "_source.json" {
				return nil, invalidSnapshot("vendor/"+source.Name, "_source.json is reserved for the source record")
			}
			stored := source.Name + "/" + librarytree.SnapshotPath(file)
			if other, taken := placed[stored]; taken {
				return nil, invalidSnapshot("vendor/"+stored, "the library files "+other+" and "+file+" would both be stored here")
			}
			placed[stored] = file
			record.Files[file] = digest(snapshot.Files[file])
			output[stored] = bytes.Clone(snapshot.Files[file])
		}
		record.Checksum = recordChecksum(record)
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

// nonNil returns list, or an empty list for nil, so a record encodes an empty list rather than null.
func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

// vendorExecutable returns, sorted, the executable files of the snapshots, by path in vendor/.
func vendorExecutable(snapshots map[string]snapshot) []string {
	executable := []string{}
	for name, snapshot := range snapshots {
		for _, file := range snapshot.Executable {
			executable = append(executable, name+"/"+file)
		}
	}
	slices.Sort(executable)
	return executable
}

// verifyExecutable fails, asking for sync, when the executable files in vendor, the vendor tree, differ from those
// the decoded snapshots record: a file's mode is part of its content.
func verifyExecutable(snapshots map[string]snapshot, vendor *filetxn.Tree) error {
	actual := []string{}
	if vendor != nil {
		actual = nonNil(vendor.Executable)
	}
	if expected := vendorExecutable(snapshots); !slices.Equal(actual, expected) {
		for _, file := range slices.Concat(actual, expected) {
			if !slices.Contains(actual, file) || !slices.Contains(expected, file) {
				return invalidSnapshot("vendor/"+file, "its executable bit differs from what the source record lists; run code-rules project sync")
			}
		}
	}
	return nil
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
			return nil, invalidSnapshot("vendor/"+recordPath, "missing source record; run code-rules project sync")
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
			full := source.Name + "/" + librarytree.SnapshotPath(file)
			data, ok := vendor[full]
			if !ok || digest(data) != record.digests[file] {
				return nil, invalidSnapshot("vendor/"+full, "missing or modified imported content; run code-rules project sync")
			}
			record.Files[file] = bytes.Clone(data)
			expected[full] = true
		}
		result[source.Name] = record.Snapshot
	}
	for _, file := range slices.Sorted(maps.Keys(vendor)) {
		if !expected[file] {
			return nil, invalidSnapshot("vendor/"+file, "unexpected imported file or removed source; run code-rules project sync")
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

// parseSourceRecord is the one way to read a source record from disk. It validates a format 3 record's exact fields,
// value syntax, and complete digest inventory, and refuses, with changedOutsideSync, a record whose checksum doesn't
// match its fields, or that has none, so no caller can use a record changed outside sync. Relationships to
// configuration are matchSnapshotSource's. A record it can't read fails validation with advice to import the source
// again, and so does an older format, such as 2; a newer one fails with code unsupported-source-record, asking to
// upgrade Code Rules.
func parseSourceRecord(data []byte, name string) (parsedRecord, error) {
	record, err := readSourceRecord(data, name)
	var invalid errs.ValidationError
	if errors.As(err, &invalid) && err == error(invalid) && !strings.Contains(invalid.ValidationProblem(), "code-rules project sync") {
		return parsedRecord{}, invalidSnapshot(invalid.ValidationLocation(), invalid.ValidationProblem()+"; "+reimport(name))
	}
	return record, err
}

// readSourceRecord is parseSourceRecord without the next step its validation errors share.
func readSourceRecord(data []byte, name string) (parsedRecord, error) {
	where := "vendor/" + name + "/_source.json"
	if err := rules.RequireResolvedMerge(data, where, "take one side, which sync wrote, such as with git checkout --ours or --theirs -- .code-rules/vendor/"+name+"/_source.json, and run code-rules project sync"); err != nil {
		return parsedRecord{}, err
	}
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
	allowed := []string{"formatVersion", "checksum", "kind", "repository", "ref", "release", "resolvedCommit", "groupSelection", "ruleSelection", "groups", "rules", "retiredRules", "skillInclude", "skills", "files", "executable"}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if !slices.Contains(allowed, key) || bytes.Equal(bytes.TrimSpace(fields[key]), []byte("null")) {
			return parsedRecord{}, invalidSnapshot(where+"."+key, "unknown or null source record field")
		}
	}
	for _, key := range []string{"kind", "repository", "resolvedCommit", "groupSelection", "groups", "rules", "retiredRules", "skillInclude", "skills", "files", "executable"} {
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
	result := parsedRecord{Snapshot: library.Snapshot{Repository: record.Repository, Release: record.Release, Commit: record.Commit, RuleSelection: []string{}, Rules: map[string]library.ImportedRule{}, NonLibrary: record.Kind == nonLibraryKind, Skills: map[string]library.ImportedSkill{}}, digests: record.Files}
	if record.Checksum != recordChecksum(record) {
		return parsedRecord{}, changedOutsideSync(name)
	}
	if record.Kind != libraryKind && record.Kind != nonLibraryKind {
		return parsedRecord{}, invalidSnapshot(where+".kind", "expected library or non-library")
	}
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
		if err := librarytree.ValidateRuleID(id, location); err != nil {
			return parsedRecord{}, err
		}
		if (rule.Version == nil) != (rule.Release == nil) || (rule.Release != nil && *rule.Release < 1) || !fullCommit(rule.Commit) {
			return parsedRecord{}, invalidSnapshot(location, "expected a version, the library release that published it, and its commit, or a null version and release")
		}
		imported := library.ImportedRule{Version: rule.Version, Commit: rule.Commit, LegacyLayout: rule.LegacyLayout}
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
	if err := readRecordSkills(record, where, paths, &result.Snapshot); err != nil {
		return parsedRecord{}, err
	}
	if result.NonLibrary {
		if result.Release != 0 || len(result.Groups) > 0 || len(result.Rules) > 0 || len(result.RetiredRules) > 0 || !result.Selection.IsEmpty() || len(result.RuleSelection) > 0 {
			return parsedRecord{}, invalidSnapshot(where, "a record of a source that isn't a library has no library release, groups, or rules")
		}
		for file := range paths {
			if !result.IsSkillFile(file) {
				return parsedRecord{}, invalidSnapshot(where+".files", file+" belongs to no recorded skill")
			}
		}
		return result, nil
	}
	if _, ok := paths[librarytree.Manifest]; !ok {
		if _, legacy := paths[librarytree.LegacyManifest]; legacy {
			return parsedRecord{}, invalidSnapshot(where, "records a library in the layout from before this version of Code Rules, with rule-library.yaml, which this version can't read. Once the library has moved its files into .code-rules/ and published a library release, "+reimport(name))
		}
		return parsedRecord{}, invalidSnapshot(where, "missing library manifest from inventory")
	}
	for _, group := range result.Groups {
		if _, ok := paths[group+"/_group.yaml"]; !ok {
			return parsedRecord{}, invalidSnapshot(where, "missing group metadata from inventory: "+group)
		}
	}
	return result, nil
}

// readRecordSkills validates the record's skill selection, skills, and executable files against its file inventory
// paths, and stores them in snapshot. A library's skills have the IDs of its skills/ folder and the level offered;
// another repository's have no level. Each skill needs its SKILL.md.
func readRecordSkills(record sourceRecord, where string, paths map[string][]byte, snapshot *library.Snapshot) error {
	snapshot.SkillSelection = []string{}
	for i, id := range record.SkillInclude {
		if id != skills.Wildcard {
			if err := skills.ValidateID(id, where+".skillInclude"); err != nil {
				return err
			}
		}
		if i > 0 && record.SkillInclude[i-1] >= id {
			return invalidSnapshot(where+".skillInclude", "expected distinct selectors in sorted order")
		}
		snapshot.SkillSelection = append(snapshot.SkillSelection, id)
	}
	for _, id := range slices.Sorted(maps.Keys(record.Skills)) {
		location := where + ".skills." + id
		if err := skills.ValidateID(id, location); err != nil {
			return err
		}
		level := record.Skills[id].Level
		name, authored := strings.CutPrefix(id, skills.Directory+"/")
		if snapshot.NonLibrary && level != nil || !snapshot.NonLibrary && (level == nil || *level != "offered" || !authored || !skills.ValidName(name)) {
			return invalidSnapshot(location, "expected a library's skill, skills/<name>, with level offered, or a skill from another repository with a null level")
		}
		for other := range record.Skills {
			if strings.HasPrefix(id, other+"/") {
				return invalidSnapshot(location, "a skill's folder can't be inside another skill's folder, "+other)
			}
		}
		if _, ok := paths[id+"/"+skills.File]; !ok {
			return invalidSnapshot(location, "missing the skill's SKILL.md from inventory")
		}
		imported := library.ImportedSkill{}
		if level != nil {
			imported.Level = *level
		}
		snapshot.Skills[id] = imported
	}
	snapshot.Executable = []string{}
	for i, file := range record.Executable {
		if _, ok := paths[file]; !ok || i > 0 && record.Executable[i-1] >= file || !snapshot.IsSkillFile(file) {
			return invalidSnapshot(where+".executable", "expected distinct skill files from the inventory in sorted order")
		}
		snapshot.Executable = append(snapshot.Executable, file)
	}
	return nil
}

// matchSnapshotSource checks a record against its source's configuration without fetching Git, as documented in
// "What offline checks can verify": the recorded identity and selections, pinned versions, and ref.
func matchSnapshotSource(source rules.Source, record parsedRecord) error {
	where := "vendor/" + source.Name + "/_source.json"
	if source.Repository != record.Repository || !source.Ref.Equal(record.Ref) || !source.Groups.Equal(record.Selection) || !slices.Equal(source.Rules, record.RuleSelection) || !slices.Equal(source.Skills, record.SkillSelection) {
		return invalidSnapshot(where, "source identity or selection changed; run code-rules project sync")
	}
	if !slices.Equal(source.Skills, []string{skills.Wildcard}) && !slices.Equal(source.Skills, slices.Sorted(maps.Keys(record.Skills))) {
		return invalidSnapshot(where, "recorded skills differ from sources."+source.Name+".skills.include; run code-rules project sync")
	}
	if record.NonLibrary && source.Ref.IsZero() {
		return invalidSnapshot(where, "a source that isn't a library needs a ref; run code-rules project sync")
	}
	for _, id := range slices.Sorted(maps.Keys(source.Pins)) {
		pin := source.Pins[id]
		// A pin of a rule the library retired keeps nothing, so any version passes; sync warns about it.
		rule, imported := record.Rules[id]
		if imported && (rule.Version == nil || *rule.Version != pin.Version) || !imported && !slices.Contains(record.RetiredRules, id) {
			return invalidSnapshot(where, "sources."+source.Name+".rules.pins."+id+" names a version the snapshot doesn't import; run code-rules project sync")
		}
	}
	// An exclusion matching neither an imported rule nor a retired one could be a typo that leaves the intended rule
	// active; only the library, which sync reads, can tell.
	for _, id := range slices.Sorted(maps.Keys(source.Exclude)) {
		matches := func(rule string) bool { return rules.SelectorMatches(id, rule) }
		if !slices.ContainsFunc(slices.Collect(maps.Keys(record.Rules)), matches) && !slices.ContainsFunc(record.RetiredRules, matches) {
			return invalidSnapshot("sources."+source.Name+".rules.exclude."+id, "names no rule the library supplies to this source; run code-rules project sync to check it against the library")
		}
	}
	for _, group := range source.Groups.Groups {
		if !slices.Contains(record.Groups, group) {
			return invalidSnapshot(where, "resolved groups differ from the explicit selection")
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
	where := "vendor/" + source.Name + "/_source.json"
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
	number, err := coderules.ParseReleaseTag(strings.TrimPrefix(ref.Canonical(), "refs/tags/"))
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
	return &ValidationError{Location: location, Problem: problem}
}
