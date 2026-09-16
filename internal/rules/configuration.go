// Parse project source declarations without resolving Git refs or reading files.

package rules

import (
	"encoding/json"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// Configuration contains sources in alias order; an empty list is a local-only project.
type Configuration struct {
	Sources []Source `json:"sources"`
}

// Source declares one remote library and its selection and exception policy.
// Exactly one of Ref and Version is populated. ParsedRef is nil for version selections.
type Source struct {
	Name       string                 `json:"name"`
	Repository string                 `json:"repository"`
	Ref        string                 `json:"ref,omitempty"`
	Version    string                 `json:"version,omitempty"`
	ParsedRef  *GitRef                `json:"parsedRef,omitempty"`
	Groups     GroupSelection         `json:"groups"`
	Exclude    map[string]string      `json:"exclude"`
	Replace    map[string]Replacement `json:"replace"`
}

// Replacement refers to a local file and preserves the authored reason.
type Replacement struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
}

var sourceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ParseConfiguration validates schema version 1, rejecting unknown fields and
// contradictory source declarations. Errors return no partial configuration.
// Strings retain authored whitespace; group IDs and source aliases are sorted.
func ParseConfiguration(input json.RawMessage) (Configuration, error) {
	fields, err := jsonObject(input, "configuration")
	if err != nil {
		return Configuration{}, err
	}
	if _, ok := fields["localGroups"]; ok {
		return Configuration{}, invalid("localGroups", "remove localGroups; local groups are discovered from local/<group>/_group.json")
	}
	if err := knownJSONFields(fields, []string{"schemaVersion", "sources"}, "configuration"); err != nil {
		return Configuration{}, err
	}
	var schema float64
	if json.Unmarshal(fields["schemaVersion"], &schema) != nil || schema != 1 {
		return Configuration{}, invalid("schemaVersion", "only version 1 is supported")
	}
	sources, err := jsonObject(fields["sources"], "sources")
	if err != nil {
		return Configuration{}, err
	}
	result := Configuration{Sources: make([]Source, 0, len(sources))}
	repositories := make(map[string]bool)
	for _, name := range slices.Sorted(maps.Keys(sources)) {
		source, err := parseSource(name, sources[name], repositories)
		if err != nil {
			return Configuration{}, err
		}
		result.Sources = append(result.Sources, source)
	}
	return result, nil
}

// parseSource validates fields in dependency order before checking exception conflicts.
func parseSource(name string, input json.RawMessage, repositories map[string]bool) (Source, error) {
	where := "sources." + name
	if !sourceNamePattern.MatchString(name) || name == "local" {
		return Source{}, invalid(where, "invalid or reserved source name")
	}
	fields, err := jsonObject(input, where)
	if err != nil {
		return Source{}, err
	}
	if err := knownJSONFields(fields, []string{"repository", "ref", "version", "groups", "exclude", "replace"}, where); err != nil {
		return Source{}, err
	}
	repository, err := jsonText(fields["repository"], where+".repository")
	if err != nil {
		return Source{}, err
	}
	address, err := ParseRepository(fields["repository"], where+".repository")
	if err != nil {
		return Source{}, err
	}
	if repositories[address.Identity] {
		return Source{}, invalid(where, "repository "+repository+" is declared more than once")
	}
	repositories[address.Identity] = true
	_, hasRef := fields["ref"]
	_, hasVersion := fields["version"]
	if hasRef == hasVersion {
		return Source{}, invalid(where, "specify exactly one of ref or version")
	}
	result := Source{Name: name, Repository: repository}
	if hasVersion {
		text, err := jsonText(fields["version"], where+".version")
		if err != nil {
			return Source{}, err
		}
		constraint, err := ParseVersionConstraint(text, where+".version")
		if err != nil {
			return Source{}, err
		}
		result.Version = constraint.String()
	} else {
		text, err := jsonText(fields["ref"], where+".ref")
		if err != nil {
			return Source{}, err
		}
		ref, err := ParseGitRef(text, where+".ref")
		if err != nil {
			return Source{}, err
		}
		result.Ref = text
		result.ParsedRef = &ref
	}
	result.Groups, err = ParseGroupSelection(fields["groups"], where+".groups")
	if err != nil {
		return Source{}, err
	}
	excludes, err := jsonObject(fields["exclude"], where+".exclude")
	if err != nil {
		return Source{}, err
	}
	result.Exclude = make(map[string]string, len(excludes))
	for _, id := range slices.Sorted(maps.Keys(excludes)) {
		text, err := jsonText(excludes[id], where+".exclude."+id)
		if err != nil {
			return Source{}, err
		}
		result.Exclude[id] = text
	}
	replacements, err := jsonObject(fields["replace"], where+".replace")
	if err != nil {
		return Source{}, err
	}
	result.Replace = make(map[string]Replacement, len(replacements))
	for _, id := range slices.Sorted(maps.Keys(replacements)) {
		replacement, err := parseReplacement(replacements[id], where+".replace."+id)
		if err != nil {
			return Source{}, err
		}
		result.Replace[id] = replacement
	}
	for _, id := range append(slices.Sorted(maps.Keys(result.Exclude)), slices.Sorted(maps.Keys(result.Replace))...) {
		decision := "replace"
		if _, ok := result.Exclude[id]; ok {
			decision = "exclude"
		}
		if _, err := GroupFromPath(id+".md", where+"."+decision+"."+id); err != nil {
			return Source{}, err
		}
		_, excluded := result.Exclude[id]
		_, replaced := result.Replace[id]
		if excluded && replaced {
			return Source{}, invalid(where+":"+id, "rule is both excluded and replaced")
		}
	}
	return result, nil
}

// parseReplacement checks path containment and local ownership without opening the file.
func parseReplacement(input json.RawMessage, location string) (Replacement, error) {
	fields, err := jsonObject(input, location)
	if err != nil {
		return Replacement{}, err
	}
	if err := knownJSONFields(fields, []string{"file", "reason"}, location); err != nil {
		return Replacement{}, err
	}
	file, err := jsonPath(fields["file"], location+".file")
	if err != nil {
		return Replacement{}, err
	}
	if !strings.HasPrefix(file, "local/") {
		return Replacement{}, invalid(location+".file", "replacement files must be under local/")
	}
	reason, err := jsonText(fields["reason"], location+".reason")
	if err != nil {
		return Replacement{}, err
	}
	return Replacement{File: file, Reason: reason}, nil
}
