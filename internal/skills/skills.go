// Package skills reads what installing an Agent Skills folder needs: skill IDs, install names, the SKILL.md fields
// that generated guidance shows, and the marker line every installed copy of SKILL.md carries. It does no file, Git,
// or network access, and reports invalid input as its own *ValidationError at the location the caller names.
package skills

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/fabricahq/syllabus/internal/decode"
)

// File is the file whose presence makes a folder a skill folder.
const File = "SKILL.md"

// Wildcard is the skills.include selector that includes every skill a source offers.
const Wildcard = "*"

// Recommended is the skills.include selector that includes the skills a library recommends.
const Recommended = "recommended"

// The levels a library gives its skills. A project that uses the library installs its required skills whatever it
// includes, its recommended skills when it includes Recommended, and an offered skill when it names it or includes
// Wildcard.
const (
	LevelRequired    = "required"
	LevelRecommended = "recommended"
	LevelOffered     = "offered"
)

// Levels lists the levels from the strongest to the weakest.
var Levels = []string{LevelRequired, LevelRecommended, LevelOffered}

// Directory is the folder, in a repository's Syllabus directory, that holds the repository's authored skills,
// and, in a library's vendor snapshot, the skills the library publishes.
const Directory = "skills"

// ValidName reports whether name follows the Agent Skills name grammar: 1 to 64 lowercase letters, digits, and
// hyphens, without a hyphen at either end or two in a row.
func ValidName(name string) bool {
	if name == "" || len(name) > 64 || strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") || strings.Contains(name, "--") {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// ValidateID checks that id names a skill folder by its path in the repository that holds it, such as
// skills/build-feature: a relative path without empty, ".", or ".." segments, backslashes, control characters, a
// wildcard, or a trailing slash. It doesn't check that the folder exists.
func ValidateID(id, location string) error {
	if !decode.Contained(id) || strings.Contains(id, "*") || len(strings.Split(id, "/")) > 63 {
		return invalid(location, "expected a skill ID, the skill folder's path in its repository, such as skills/build-feature, got "+decode.Quote(id))
	}
	return nil
}

// AuthoredID returns the ID of the skill a repository authors, or a library publishes, in the folder named name of
// its skills/ folder.
func AuthoredID(name string) string { return Directory + "/" + name }

// Metadata is what generated guidance shows of a skill's SKILL.md frontmatter.
type Metadata struct {
	// Name is the frontmatter's name, or empty when the file has no frontmatter or no string name.
	Name string
	// Description is the frontmatter's description, or empty.
	Description string
	// Type is metadata.syllabus-skill-type as written, or empty.
	Type string
	// RulesUsed and SkillsUsed are the lines of metadata.syllabus-rules-used and metadata.syllabus-skills-used, trimmed of
	// surrounding whitespace, without blank lines, in the order written; nil when the key is absent. Their IDs are
	// unchecked: Check validates them.
	RulesUsed, SkillsUsed []string
	// ReservedKeys lists, sorted, the metadata keys that start with syllabus- but aren't one of the three Syllabus
	// defines, which only Check rejects; nil when there are none.
	ReservedKeys []string
	// Frontmatter reports whether the file starts with a frontmatter block.
	Frontmatter bool
}

// ReadMetadata reads the frontmatter of a SKILL.md file. A file without frontmatter, such as one from a repository
// that isn't a library, has empty metadata. It fails when the frontmatter isn't one closed YAML mapping, or a field
// it reads isn't a string; it checks nothing else, so a field it doesn't read can hold anything, and Check
// validates what it reads.
func ReadMetadata(data []byte, location string) (Metadata, error) {
	header, _, found, err := split(data, location)
	if err != nil || !found {
		return Metadata{}, err
	}
	_, encoded, err := decode.YAML(header, location)
	if err != nil {
		return Metadata{}, err
	}
	var fields struct {
		Name        *json.RawMessage `json:"name"`
		Description *json.RawMessage `json:"description"`
		Metadata    *json.RawMessage `json:"metadata"`
	}
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return Metadata{}, invalid(location, "expected frontmatter that is a YAML mapping")
	}
	result := Metadata{Frontmatter: true}
	for _, field := range []struct {
		raw   *json.RawMessage
		key   string
		value *string
	}{{fields.Name, "name", &result.Name}, {fields.Description, "description", &result.Description}} {
		if field.raw != nil && json.Unmarshal(*field.raw, field.value) != nil {
			return Metadata{}, invalid(location+"."+field.key, "expected a string; quote the value")
		}
	}
	if fields.Metadata != nil {
		var metadata map[string]json.RawMessage
		if json.Unmarshal(*fields.Metadata, &metadata) != nil {
			return Metadata{}, invalid(location+".metadata", "expected a mapping of string keys to string values")
		}
		if raw, ok := metadata[TypeKey]; ok && json.Unmarshal(raw, &result.Type) != nil {
			return Metadata{}, invalid(location+".metadata."+TypeKey, "expected a string, such as playbook or review")
		}
		for _, field := range []struct {
			key   string
			lines *[]string
		}{{RulesUsedKey, &result.RulesUsed}, {SkillsUsedKey, &result.SkillsUsed}} {
			raw, ok := metadata[field.key]
			if !ok {
				continue
			}
			var text string
			if json.Unmarshal(raw, &text) != nil {
				return Metadata{}, invalid(location+".metadata."+field.key, "expected a string with one ID per line; write it as a YAML block string, starting with |")
			}
			*field.lines = idLines(text)
		}
		for _, key := range slices.Sorted(maps.Keys(metadata)) {
			if strings.HasPrefix(key, "syllabus-") && !slices.Contains([]string{TypeKey, RulesUsedKey, SkillsUsedKey}, key) {
				result.ReservedKeys = append(result.ReservedKeys, key)
			}
		}
	}
	return result, nil
}

// idLines returns the lines of text, trimmed of surrounding whitespace, without blank lines.
func idLines(text string) []string {
	lines := []string{}
	for line := range strings.Lines(text) {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// InstallName returns the name a skill installs under, which names its installed folders: its frontmatter name, or,
// for a SKILL.md without one, which only a repository that isn't a library can supply, the skill folder's name. It
// fails when that name doesn't follow the Agent Skills name grammar, since agents find skills by it.
func InstallName(id string, metadata Metadata, location string) (string, error) {
	name := metadata.Name
	if name == "" {
		name = id[strings.LastIndex(id, "/")+1:]
	}
	if !ValidName(name) {
		return "", invalid(location, "the skill's name "+decode.Quote(name)+" doesn't follow the Agent Skills name grammar: 1 to 64 lowercase letters, digits, and hyphens, without a hyphen at either end or two in a row. Agents find skills by name, so Syllabus can't install it")
	}
	return name, nil
}

// Mark returns the bytes of an installed copy of a SKILL.md file: data, with marker on its own line right after the
// frontmatter's closing line, or as the first line when the file has no frontmatter, since agents read the
// frontmatter from the first byte. data must be a SKILL.md file ReadMetadata accepted.
func Mark(data []byte, marker string) []byte {
	_, body, found, _ := split(data, "")
	prefix := data[:len(data)-len(body)]
	line := []byte(marker + "\n")
	if !found {
		prefix, body = nil, data
	} else if !bytes.HasSuffix(prefix, []byte("\n")) {
		// A closing line at the end of the file has no line break to end it.
		line = append([]byte("\n"), line...)
	}
	return bytes.Join([][]byte{prefix, line, body}, nil)
}

// split separates a frontmatter block that starts at the first byte, "---" on its own line, from the rest of data.
// header is the YAML between the delimiter lines, and body everything after the closing line. found is false, with
// body data, when data doesn't start with a frontmatter block. A block without a closing line fails.
func split(data []byte, location string) (header, body []byte, found bool, err error) {
	rest, ok := bytes.CutPrefix(data, []byte("---\n"))
	if !ok {
		if rest, ok = bytes.CutPrefix(data, []byte("---\r\n")); !ok {
			return nil, data, false, nil
		}
	}
	for offset := 0; offset <= len(rest); {
		end := bytes.IndexByte(rest[offset:], '\n')
		line := rest[offset:]
		if end >= 0 {
			line = rest[offset : offset+end]
		}
		if string(bytes.TrimSuffix(line, []byte("\r"))) == "---" {
			next := len(rest)
			if end >= 0 {
				next = offset + end + 1
			}
			return rest[:offset], rest[next:], true, nil
		}
		if end < 0 {
			break
		}
		offset += end + 1
	}
	return nil, nil, false, invalid(location, "the frontmatter that starts on the first line has no closing --- line")
}
