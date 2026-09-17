// Package project verifies persisted source snapshots and coordinates project-owned files.
package project

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/fabricahq/code-rules/internal/rules"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// Snapshot owns original library bytes and the identity recorded when they were imported.
// A verified digest establishes integrity against the record, not authenticity of its origin.
// Files excludes _source.json. Binary content remains bytes; required text is checked by the loader.
type Snapshot struct {
	Repository string `json:"repository"`
	Ref        string `json:"ref,omitempty"`
	Version    string `json:"version,omitempty"`
	// Tag and ResolvedVersion are present only for a version constraint, not an exact ref.
	Tag             string `json:"resolvedTag,omitempty"`
	ResolvedVersion string `json:"resolvedVersion,omitempty"`
	Commit          string `json:"resolvedCommit"`
	// Groups lists the resolved IDs; Selection retains the requested IDs or wildcard.
	Groups    []string             `json:"groups"`
	Selection rules.GroupSelection `json:"groupSelection"`
	Files     map[string][]byte    `json:"files"`
}

// sourceRecord preserves the version-1 TypeScript record layout without embedding file content.
type sourceRecord struct {
	FormatVersion   int               `json:"formatVersion"`
	Repository      string            `json:"repository"`
	Ref             string            `json:"ref,omitempty"`
	Version         string            `json:"version,omitempty"`
	Tag             string            `json:"resolvedTag,omitempty"`
	ResolvedVersion string            `json:"resolvedVersion,omitempty"`
	Commit          string            `json:"resolvedCommit"`
	Groups          []string          `json:"groups"`
	Selection       json.RawMessage   `json:"groupSelection,omitempty"`
	Files           map[string]string `json:"files"`
}

// EncodeSnapshots prepares complete vendor bytes for parsed configuration, without writing files.
// Inputs stay unchanged; returned maps and byte slices belong to the caller. Errors return nil.
func EncodeSnapshots(config rules.Configuration, snapshots map[string]Snapshot) (map[string][]byte, error) {
	output := map[string][]byte{}
	if len(snapshots) != len(config.Sources) {
		return nil, invalidSnapshot("vendor", "source inventory differs from configuration; run sync")
	}
	for _, source := range config.Sources {
		snapshot, ok := snapshots[source.Name]
		if !ok {
			return nil, invalidSnapshot(source.Name, "missing source snapshot; run sync")
		}
		selection, err := json.Marshal(snapshot.Selection)
		if err != nil {
			return nil, fmt.Errorf("encode selection for %s: %v", source.Name, err)
		}
		record := sourceRecord{1, snapshot.Repository, snapshot.Ref, snapshot.Version, snapshot.Tag, snapshot.ResolvedVersion, snapshot.Commit, snapshot.Groups, selection, map[string]string{}}
		if err := validateFilePaths(snapshot.Files); err != nil {
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
	if err := validateFilePaths(output); err != nil {
		return nil, err
	}
	return output, nil
}

// DecodeSnapshots verifies every original byte and source identity before returning owned snapshots.
// Unexpected source files and incomplete records fail closed. It performs no filesystem or Git access.
// Rule and manifest validation remains the catalog loader's responsibility after this integrity check.
func DecodeSnapshots(config rules.Configuration, vendor map[string][]byte) (map[string]Snapshot, error) {
	if err := validateFilePaths(vendor); err != nil {
		return nil, err
	}
	result := map[string]Snapshot{}
	expected := map[string]bool{}
	for _, source := range config.Sources {
		recordPath := source.Name + "/_source.json"
		data, ok := vendor[recordPath]
		if !ok {
			return nil, invalidSnapshot(recordPath, "missing source record; run sync")
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
				return nil, invalidSnapshot(full, "missing or modified imported content; run sync")
			}
			files[file] = bytes.Clone(data)
			expected[full] = true
		}
		selection, err := rules.ParseGroupSelection(record.Selection, recordPath+".groupSelection")
		if err != nil {
			return nil, err
		}
		result[source.Name] = Snapshot{record.Repository, record.Ref, record.Version, record.Tag, record.ResolvedVersion, record.Commit, record.Groups, selection, files}
	}
	for _, file := range slices.Sorted(maps.Keys(vendor)) {
		if !expected[file] {
			return nil, invalidSnapshot(file, "unexpected imported file or removed source; run sync")
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
	allowed := []string{"formatVersion", "repository", "ref", "version", "resolvedTag", "resolvedVersion", "resolvedCommit", "groups", "groupSelection", "files"}
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
	declaration := map[string]any{"repository": fields["repository"], "groups": record.Selection, "exclude": map[string]string{}, "replace": map[string]string{}}
	if value, ok := fields["ref"]; ok {
		declaration["ref"] = value
	}
	if value, ok := fields["version"]; ok {
		declaration["version"] = value
	}
	input, err := json.Marshal(map[string]any{"schemaVersion": 1, "sources": map[string]any{source.Name: declaration}})
	if err != nil {
		return sourceRecord{}, fmt.Errorf("encode source record declaration: %v", err)
	}
	parsed, err := rules.ParseConfiguration(input)
	if err != nil {
		return sourceRecord{}, fmt.Errorf("%s: %w", where, err)
	}
	recorded := parsed.Sources[0]
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
	if err := validateFilePaths(paths); err != nil {
		return sourceRecord{}, fmt.Errorf("%s: %w", where, err)
	}
	if _, ok := paths["rule-library.json"]; !ok {
		return sourceRecord{}, invalidSnapshot(where, "missing library manifest from inventory")
	}
	for _, group := range record.Groups {
		if _, ok := paths[group+"/_group.json"]; !ok {
			return sourceRecord{}, invalidSnapshot(where, "missing group metadata from inventory: "+group)
		}
	}
	return record, nil
}

// matchSnapshotSource rejects stale selectors and inconsistent resolved revisions without fetching Git.
func matchSnapshotSource(want, got rules.Source, record sourceRecord, where string) error {
	if want.Repository != got.Repository || want.Ref != got.Ref || want.Version != got.Version || want.Groups.Pattern != got.Groups.Pattern || !slices.Equal(want.Groups.Groups, got.Groups.Groups) {
		return invalidSnapshot(where, "source identity or group selection changed; run sync")
	}
	commit, err := rules.ParseGitRef(record.Commit, where+".resolvedCommit")
	if err != nil || commit.Kind != rules.GitRefCommit {
		return invalidSnapshot(where, "resolvedCommit must be a full commit SHA")
	}
	if got.ParsedRef != nil && got.ParsedRef.Kind == rules.GitRefCommit && got.ParsedRef.SHA != commit.SHA {
		return invalidSnapshot(where, "resolved commit differs from requested commit")
	}
	if got.Version != "" {
		version, err := rules.TagVersion(record.Tag, where+".resolvedTag")
		if err != nil {
			return err
		}
		constraint, err := rules.ParseVersionConstraint(got.Version, where+".version")
		if err != nil {
			return err
		}
		matches, err := constraint.Matches(record.Tag, where+".resolvedTag")
		if err != nil {
			return err
		}
		if !matches || version != record.ResolvedVersion {
			return invalidSnapshot(where, "resolved release does not satisfy the recorded version constraint")
		}
	} else if record.Tag != "" || record.ResolvedVersion != "" {
		return invalidSnapshot(where, "release fields require a version constraint")
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

// validateFilePaths enforces contained portable names and rejects file/directory collisions before use.
func validateFilePaths(files map[string][]byte) error {
	return validatePaths(files, nil)
}

// validatePaths checks portable spelling across both files and empty directories.
func validatePaths(files map[string][]byte, directories []string) error {
	names := maps.Clone(files)
	if names == nil {
		names = map[string][]byte{}
	}
	for _, dir := range directories {
		if _, ok := files[dir]; ok {
			return invalidSnapshot(dir, "file also used as directory")
		}
		names[dir] = nil
	}
	spellings := map[string]string{}
	for _, file := range slices.Sorted(maps.Keys(names)) {
		if file == "." || !fs.ValidPath(file) || strings.ContainsAny(file, "\\:\x00") || !utf8.ValidString(file) {
			return invalidSnapshot(file, "expected a contained relative file path")
		}
		parts := strings.Split(file, "/")
		for i, part := range parts {
			if strings.ContainsFunc(part, func(r rune) bool { return r < 32 || r == 127 }) {
				return invalidSnapshot(file, "control characters are unsupported in paths")
			}
			prefix := strings.Join(parts[:i+1], "/")
			folded := cases.Fold().String(norm.NFC.String(prefix))
			if previous, ok := spellings[folded]; ok && previous != prefix {
				return invalidSnapshot(file, "portable path collision with "+previous)
			}
			spellings[folded] = prefix
			if i < len(parts)-1 {
				if _, ok := files[prefix]; ok {
					return invalidSnapshot(file, "file also used as directory: "+prefix)
				}
			}
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
