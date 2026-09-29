// Verify persisted source snapshots before offline generation.

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

// sourceRecord is the version-1 record of one source's exact ref, resolved commit, selection,
// and file digests, without embedding file content.
type sourceRecord struct {
	FormatVersion int               `json:"formatVersion"`
	Repository    string            `json:"repository"`
	Ref           string            `json:"ref"`
	Commit        string            `json:"resolvedCommit"`
	Groups        []string          `json:"groups"`
	Selection     json.RawMessage   `json:"groupSelection,omitempty"`
	Files         map[string]string `json:"files"`
}

// encodeSnapshots prepares complete vendor bytes for parsed configuration, without writing files.
// Inputs stay unchanged; returned maps and byte slices belong to the caller. Errors return nil.
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
		record := sourceRecord{1, snapshot.Repository, snapshot.Ref, snapshot.Commit, snapshot.Groups, selection, map[string]string{}}
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
		if _, err := parseSourceRecord(encoded.Bytes(), source); err != nil {
			return nil, err
		}
		output[source.Name+"/_source.json"] = encoded.Bytes()
	}
	if err := rules.ValidatePaths(output, nil); err != nil {
		return nil, err
	}
	return output, nil
}

// decodeSnapshots verifies every original byte and source identity before returning owned snapshots.
// Unexpected source files and incomplete records fail closed. It performs no filesystem or Git access.
// Rule and manifest validation remains the catalog loader's responsibility after this integrity check.
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
		record, err := parseSourceRecord(data, source)
		if err != nil {
			return nil, err
		}
		expected[recordPath] = true
		files := map[string][]byte{}
		for _, file := range slices.Sorted(maps.Keys(record.Files)) {
			full := source.Name + "/" + file
			data, ok := vendor[full]
			if !ok || digest(data) != record.Files[file] {
				return nil, invalidSnapshot(full, "missing or modified imported content; run code-rules project sync")
			}
			files[file] = bytes.Clone(data)
			expected[full] = true
		}
		selection, err := rules.ParseGroupSelection(record.Selection, recordPath+".groupSelection")
		if err != nil {
			return nil, err
		}
		result[source.Name] = snapshot{Repository: record.Repository, Ref: record.Ref, Commit: record.Commit, Groups: record.Groups, Selection: selection, Files: files}
	}
	for _, file := range slices.Sorted(maps.Keys(vendor)) {
		if !expected[file] {
			return nil, invalidSnapshot(file, "unexpected imported file or removed source; run code-rules project sync")
		}
	}
	return result, nil
}

// parseSourceRecord validates exact fields, identity, selections, and the complete digest inventory.
func parseSourceRecord(data []byte, source rules.Source) (sourceRecord, error) {
	where := source.Name + "/_source.json"
	var fields map[string]json.RawMessage
	if !utf8.Valid(data) || json.Unmarshal(data, &fields) != nil || fields == nil {
		return sourceRecord{}, invalidSnapshot(where, "expected a UTF-8 source record object")
	}
	allowed := []string{"formatVersion", "repository", "ref", "resolvedCommit", "groups", "groupSelection", "files"}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if !slices.Contains(allowed, key) || bytes.Equal(bytes.TrimSpace(fields[key]), []byte("null")) {
			return sourceRecord{}, invalidSnapshot(where+"."+key, "unknown or null source record field")
		}
	}
	var record sourceRecord
	if json.Unmarshal(data, &record) != nil || record.FormatVersion != 1 || record.Files == nil {
		return sourceRecord{}, invalidSnapshot(where, "unsupported or incomplete source record")
	}
	if len(record.Selection) == 0 {
		record.Selection = fields["groups"]
	}
	groups, err := rules.ParseGroupSelection(fields["groups"], where+".groups")
	if err != nil {
		return sourceRecord{}, err
	}
	if groups.Pattern != "" {
		return sourceRecord{}, invalidSnapshot(where+".groups", "expected resolved group IDs, not a wildcard")
	}
	record.Groups = groups.Groups
	recorded, err := recordedSource(source.Name, fields["repository"], record.Ref, record.Selection, where)
	if err != nil {
		return sourceRecord{}, err
	}
	if err := matchSnapshotSource(source, recorded, record, where); err != nil {
		return sourceRecord{}, err
	}
	paths := map[string][]byte{}
	for file, hash := range record.Files {
		decoded, err := hex.DecodeString(hash)
		if err != nil || len(decoded) != sha256.Size || strings.ToLower(hash) != hash || file == "_source.json" {
			return sourceRecord{}, invalidSnapshot(where, "invalid or reserved file digest")
		}
		paths[file] = nil
	}
	if err := rules.ValidatePaths(paths, nil); err != nil {
		return sourceRecord{}, fmt.Errorf("%s: %w", where, err)
	}
	if _, ok := paths["rule-library.yaml"]; !ok {
		return sourceRecord{}, invalidSnapshot(where, "missing library manifest from inventory")
	}
	for _, group := range record.Groups {
		if _, ok := paths[group+"/_group.yaml"]; !ok {
			return sourceRecord{}, invalidSnapshot(where, "missing group metadata from inventory: "+group)
		}
	}
	return record, nil
}

// recordedSource validates the identity a record repeats from configuration: its repository address,
// exact ref, and requested group selection.
func recordedSource(name string, repository json.RawMessage, ref string, selection json.RawMessage, where string) (rules.Source, error) {
	if _, err := rules.ParseRepository(repository, where+".repository"); err != nil {
		return rules.Source{}, err
	}
	parsedRef, err := rules.ParseGitRef(ref, where+".ref")
	if err != nil {
		return rules.Source{}, err
	}
	groups, err := rules.ParseGroupSelection(selection, where+".groupSelection")
	if err != nil {
		return rules.Source{}, err
	}
	var address string
	if err := json.Unmarshal(repository, &address); err != nil {
		return rules.Source{}, fmt.Errorf("%s: decode validated repository: %v", where, err)
	}
	return rules.Source{Name: name, Repository: address, Ref: ref, ParsedRef: &parsedRef, Groups: groups}, nil
}

// matchSnapshotSource rejects stale selectors and inconsistent resolved revisions without fetching Git.
// A format 1 record can't hold individually selected rules, so configuration that selects any never matches.
func matchSnapshotSource(want, got rules.Source, record sourceRecord, where string) error {
	if want.Repository != got.Repository || want.Ref != got.Ref || len(want.Rules) > 0 || want.Groups.Pattern != got.Groups.Pattern || !slices.Equal(want.Groups.Groups, got.Groups.Groups) {
		return invalidSnapshot(where, "source identity or group selection changed; run code-rules project sync")
	}
	commit, err := rules.ParseGitRef(record.Commit, where+".resolvedCommit")
	if err != nil || commit.Kind != rules.GitRefCommit {
		return invalidSnapshot(where, "resolvedCommit must be a full commit SHA")
	}
	if got.ParsedRef.Kind == rules.GitRefCommit && got.ParsedRef.SHA != commit.SHA {
		return invalidSnapshot(where, "resolved commit differs from requested commit")
	}
	if got.Groups.Pattern == "" && !slices.Equal(record.Groups, got.Groups.Groups) {
		return invalidSnapshot(where, "resolved groups differ from the explicit selection")
	}
	for _, group := range record.Groups {
		if got.Groups.Pattern != "" && got.Groups.Pattern != "*" && !strings.HasPrefix(group, strings.TrimSuffix(got.Groups.Pattern, "*")) {
			return invalidSnapshot(where, "resolved group is outside the requested selection")
		}
	}
	return nil
}

// digest hashes exact bytes, preserving binary data and line endings.
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// invalidSnapshot reports caller-visible validation failures without logging source contents.
func invalidSnapshot(location, problem string) error {
	return &rules.ValidationError{Location: location, Problem: problem}
}
