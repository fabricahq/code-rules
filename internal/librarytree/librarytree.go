// Package librarytree describes the structure of files and directories in a Code Rules library, and the group and
// rule IDs derived from positions in that tree: which paths are rules, their assets, group READMEs, or
// library-wide files. It checks syntax only, accesses no files, and never cleans or normalizes paths, which use
// forward slashes on every OS.
package librarytree

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/decode"
)

var (
	groupPattern    = regexp.MustCompile(`^(techs|practices)/[a-z][a-z0-9-]*$`)
	rulePartPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)
)

// ValidateGroupID accepts a technology or practice group ID, except the reserved
// assets name. It checks syntax only; it does not establish that the group exists.
func ValidateGroupID(value, location string) error {
	if !groupPattern.MatchString(value) {
		return invalid(location, "invalid group ID "+decode.Quote(value)+": expected format "+groupPattern.String())
	}
	if strings.HasSuffix(value, "/assets") {
		return invalid(location, "invalid group ID "+decode.Quote(value)+`: "assets" is a reserved group name`)
	}
	return nil
}

// GroupFromPath returns the owning group of a contained Markdown rule path.
// Containment errors take precedence over rule syntax, then group syntax errors.
func GroupFromPath(path, location string) (string, error) {
	if !decode.Contained(path) {
		return "", invalid(location, "expected a contained relative path, got "+decode.Quote(path)+`: use forward slashes; no leading slash, drive prefix, empty segments, "." or ".." segments, or control characters`)
	}
	parts := strings.Split(path, "/")
	if len(parts) < 3 || !strings.HasSuffix(path, ".md") {
		return "", invalid(location, "invalid rule path "+decode.Quote(path)+": expected a .md file beneath a group directory")
	}
	for i, part := range parts {
		if i > 0 && part == "assets" {
			return "", invalid(location, "invalid rule path "+decode.Quote(path)+`: "assets" is reserved for supporting files, not rules`)
		}
		if i >= 2 && !rulePartPattern.MatchString(part) {
			return "", invalid(location, "invalid rule path "+decode.Quote(path)+": rule file and subdirectory names must match "+rulePartPattern.String())
		}
	}
	group := strings.Join(parts[:2], "/")
	if err := ValidateGroupID(group, location); err != nil {
		return "", fmt.Errorf("rule path %s: %w", decode.Quote(path), err)
	}
	return group, nil
}

// ValidateRuleID accepts a library rule ID: a rule's contained path without its final .md, such as
// practices/testing/verify-retry-limits. It accepts exactly the IDs of valid rule paths, so the ID of a rule file
// named example.md.md is example.md. It checks syntax only; it does not establish that the rule exists.
func ValidateRuleID(id, location string) error {
	_, err := GroupFromPath(id+".md", location)
	return err
}

// VersionedRule returns the ID of the rule whose versions cover file: the rule's own Markdown file, or a file in
// its asset directory, assets/<rule-name>/ beside it. It reports false for library-wide files, such as group
// metadata, group READMEs, and the library-root assets/ directory, and for paths that can't be rule content.
func VersionedRule(file string) (string, bool) {
	parts := strings.Split(file, "/")
	if len(parts) < 3 || (parts[0] != "techs" && parts[0] != "practices") {
		return "", false
	}
	for i := 2; i < len(parts); i++ {
		if parts[i] == "assets" {
			if i+2 >= len(parts) {
				return "", false
			}
			return strings.Join(parts[:i], "/") + "/" + parts[i+1], true
		}
	}
	if IsGroupReadme(file) {
		return "", false
	}
	if _, err := GroupFromPath(file, file); err != nil {
		return "", false
	}
	return strings.TrimSuffix(file, ".md"), true
}

// IsRuleContent reports whether path is part of some rule's version: a rule's Markdown file, or a file in an asset
// directory inside a technology or practice group, since every such directory belongs to a rule. Group metadata
// and the library-root assets/ directory are library-wide instead; group READMEs are authoring notes, neither
// rule content nor library-wide. Library license declarations reject declared license and notice files that are
// rule content, so every path is one or the other, never both.
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

// IsGroupReadme recognizes only the orientation file directly inside a valid group, outside rule and asset paths.
func IsGroupReadme(file string) bool {
	group, found := strings.CutSuffix(file, "/README.md")
	return found && ValidateGroupID(group, file) == nil
}
