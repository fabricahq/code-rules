// Package build resolves adopted rules and prepares generated output without filesystem writes.
package build

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// Library supplies a validated selected catalog and its caller-verified commit.
// Snapshot freshness and Git authenticity belong to the import/snapshot boundary.
type Library struct {
	Catalog library.Catalog
	Commit  string
	Tag     string
}

// Origin identifies an effective definition; local definitions have no repository or commit.
type Origin struct {
	Source     string `json:"source"`
	File       string `json:"file"`
	Repository string `json:"repository,omitempty"`
	Ref        string `json:"ref,omitempty"`
	Commit     string `json:"resolvedCommit,omitempty"`
}

// ActiveRule owns one parsed effective document. Upstream is non-nil only for replacements.
type ActiveRule struct {
	Rule     rules.Rule                `json:"rule"`
	Origin   Origin                    `json:"origin"`
	Upstream *Origin                   `json:"upstream"`
	Reason   string                    `json:"replacementReason,omitempty"`
	License  *rules.LicenseDeclaration `json:"license"`
}

// Guidance identifies the source of one complete group metadata definition.
type Guidance struct {
	Source   string              `json:"source"`
	Metadata rules.GroupMetadata `json:"metadata"`
}

// Group contains effective rules and guidance chosen by Resolve, plus original guidance for provenance.
type Group struct {
	ID                string       `json:"id"`
	EffectiveGuidance []Guidance   `json:"effectiveGuidance"`
	Guidance          []Guidance   `json:"guidance"`
	Rules             []ActiveRule `json:"rules"`
}

// Source records the adopted revision and complete retained inventory, including excluded rules.
// Files contains supporting bytes and inactive upstream documents; active rules own their original documents.
type Source struct {
	Name            string                    `json:"name"`
	Repository      string                    `json:"repository"`
	Ref             string                    `json:"ref,omitempty"`
	Version         string                    `json:"version,omitempty"`
	Tag             string                    `json:"resolvedTag,omitempty"`
	ResolvedVersion string                    `json:"resolvedVersion,omitempty"`
	Commit          string                    `json:"resolvedCommit"`
	Selection       rules.GroupSelection      `json:"groupSelection"`
	Groups          []string                  `json:"groups"`
	License         *rules.LicenseDeclaration `json:"license"`
	Paths           []string                  `json:"paths"`
	Files           map[string][]byte         `json:"retainedFiles"`
}

// Resolved owns effective rules; supporting bytes are shared read-only with the input catalogs.
// LocalPaths lists every supplied local path for link resolution, including metadata and attachments.
// LocalFiles holds supporting bytes; active rules own their documents.
type Resolved struct {
	Groups     []Group           `json:"groups"`
	Sources    []Source          `json:"sources"`
	LocalPaths []string          `json:"localPaths"`
	LocalFiles map[string][]byte `json:"localSupportingFiles"`
}

// Resolve applies exclusions and replacements to parsed catalogs, then adds local rules.
// Configuration must come from ParseConfiguration. Every candidate document is validated
// before exceptions, and any error returns a zero result. Inputs are never modified.
func Resolve(config rules.Configuration, libraries map[string]Library, localFiles map[string][]byte) (Resolved, error) {
	result := Resolved{Groups: []Group{}, Sources: []Source{}, LocalPaths: slices.Sorted(maps.Keys(localFiles)), LocalFiles: map[string][]byte{}}
	groups := map[string]*Group{}
	declared := map[string]bool{}
	for _, source := range config.Sources {
		declared[source.Name] = true
	}
	for _, name := range slices.Sorted(maps.Keys(libraries)) {
		if !declared[name] {
			return Resolved{}, invalid(name, "library is not declared in configuration")
		}
	}
	localRules, err := parseLocal(localFiles, groups)
	if err != nil {
		return Resolved{}, err
	}
	if err := validateLocalLinks(localFiles, localRules); err != nil {
		return Resolved{}, err
	}
	for file, data := range localFiles {
		if _, ok := localRules[file]; !ok {
			result.LocalFiles[file] = data
		}
	}
	used := map[string]bool{}
	for _, source := range config.Sources {
		supplied, ok := libraries[source.Name]
		if !ok {
			return Resolved{}, invalid(source.Name, "missing library; load or sync the source")
		}
		if supplied.Catalog.Selection.Pattern != source.Groups.Pattern || !slices.Equal(supplied.Catalog.Selection.Groups, source.Groups.Groups) {
			return Resolved{}, invalid(source.Name, "loaded group selection differs from configuration; reload the source")
		}
		selectedVersion, err := selectedVersion(source, supplied)
		if err != nil {
			return Resolved{}, err
		}
		ref, err := rules.ParseGitRef(supplied.Commit, source.Name+".resolvedCommit")
		if err != nil || ref.Kind != "commit" {
			return Resolved{}, invalid(source.Name, "resolvedCommit must be a full commit SHA")
		}
		if source.ParsedRef != nil && source.ParsedRef.Kind == "commit" && source.ParsedRef.SHA != ref.SHA {
			return Resolved{}, invalid(source.Name, "resolved commit differs from configured commit")
		}
		candidates := map[string]rules.Rule{}
		ids := []string{}
		seenGroups := map[string]bool{}
		for _, group := range supplied.Catalog.Groups {
			if seenGroups[group.ID] {
				return Resolved{}, invalid(group.ID, "duplicate group")
			}
			seenGroups[group.ID] = true
			if err := rules.ValidateGroupID(group.ID, source.Name); err != nil {
				return Resolved{}, err
			}
			ids = append(ids, group.ID)
			target := ensureGroup(groups, group.ID)
			target.Guidance = append(target.Guidance, Guidance{source.Name, group.Metadata})
			for _, candidate := range group.Rules {
				parsed, err := rules.Parse(candidate.Document, candidate.Path, source.Name)
				if err != nil {
					return Resolved{}, err
				}
				if parsed.Group != group.ID {
					return Resolved{}, invalid(parsed.ID, "rule belongs to a different group")
				}
				key := strings.TrimSuffix(parsed.Path, ".md")
				if _, exists := candidates[key]; exists {
					return Resolved{}, invalid(parsed.ID, "duplicate rule")
				}
				candidates[key] = parsed
			}
		}
		slices.Sort(ids)
		if source.Groups.Pattern == "" && !slices.Equal(ids, source.Groups.Groups) {
			return Resolved{}, invalid(source.Name, "loaded groups differ from configured selection")
		}
		for _, id := range ids {
			if source.Groups.Pattern != "" && source.Groups.Pattern != "*" && !strings.HasPrefix(id, strings.TrimSuffix(source.Groups.Pattern, "*")) {
				return Resolved{}, invalid(source.Name, "loaded group outside configured selection")
			}
		}
		for _, target := range slices.Sorted(maps.Keys(source.Exclude)) {
			if _, ok := candidates[target]; !ok {
				return Resolved{}, invalid(source.Name+":"+target, "exception target is missing from selected groups")
			}
		}
		for _, target := range slices.Sorted(maps.Keys(source.Replace)) {
			if _, ok := candidates[target]; !ok {
				return Resolved{}, invalid(source.Name+":"+target, "replacement target is missing from selected groups")
			}
		}
		retained := maps.Clone(supplied.Catalog.SupportingFiles)
		if retained == nil {
			retained = map[string][]byte{}
		}
		for _, id := range slices.Sorted(maps.Keys(candidates)) {
			parsed := candidates[id]
			if _, excluded := source.Exclude[id]; excluded {
				retained[parsed.Path] = []byte(parsed.Document)
				continue
			}
			origin := Origin{Source: source.Name, File: parsed.Path, Repository: source.Repository, Ref: source.Ref, Commit: ref.SHA}
			if source.Version != "" {
				origin.Ref = supplied.Tag
			}
			active := ActiveRule{Rule: parsed, Origin: origin, License: supplied.Catalog.License}
			if replacement, ok := source.Replace[id]; ok {
				file := strings.TrimPrefix(replacement.File, "local/")
				replacementRule, ok := localRules[file]
				if !ok {
					return Resolved{}, invalid(replacement.File, "missing local replacement rule")
				}
				if used[file] {
					return Resolved{}, invalid(replacement.File, "replacement file is reused for multiple targets")
				}
				if replacementRule.Group != parsed.Group {
					return Resolved{}, invalid(replacement.File, "replacement must stay within the target group")
				}
				retained[parsed.Path] = []byte(parsed.Document)
				used[file] = true
				active = ActiveRule{Rule: replacementRule, Origin: Origin{Source: "local", File: file}, Upstream: &origin, Reason: replacement.Reason, License: nil}
			}
			group := ensureGroup(groups, parsed.Group)
			group.Rules = append(group.Rules, active)
		}
		result.Sources = append(result.Sources, Source{Name: source.Name, Repository: source.Repository, Ref: source.Ref, Version: source.Version, Tag: supplied.Tag, ResolvedVersion: selectedVersion, Commit: ref.SHA, Selection: source.Groups, Groups: ids, License: supplied.Catalog.License, Paths: supplied.Catalog.Paths(), Files: retained})
	}
	for _, file := range slices.Sorted(maps.Keys(localRules)) {
		if used[file] {
			continue
		}
		parsed := localRules[file]
		group, ok := groups[parsed.Group]
		if !ok || len(group.Guidance) == 0 {
			return Resolved{}, invalid(file, "local rule group has no metadata; add local group metadata or import this group")
		}
		group.Rules = append(group.Rules, ActiveRule{Rule: parsed, Origin: Origin{Source: "local", File: file}, License: nil})
	}
	for _, id := range slices.Sorted(maps.Keys(groups)) {
		group := groups[id]
		slices.SortFunc(group.Rules, func(a, b ActiveRule) int { return compareRuleIDs(a.Rule.ID, b.Rule.ID) })
		slices.SortFunc(group.Guidance, func(a, b Guidance) int { return strings.Compare(a.Source, b.Source) })
		group.EffectiveGuidance = resolveGuidance(group.Guidance)
		result.Groups = append(result.Groups, *group)
	}
	return result, nil
}

// ensureGroup returns a group accumulator with explicit empty collections.
func ensureGroup(groups map[string]*Group, id string) *Group {
	if group, ok := groups[id]; ok {
		return group
	}
	group := &Group{ID: id, Guidance: []Guidance{}, Rules: []ActiveRule{}}
	groups[id] = group
	return group
}

// parseLocal validates every local definition before any replacement or exclusion can hide errors.
func parseLocal(files map[string][]byte, groups map[string]*Group) (map[string]rules.Rule, error) {
	result := map[string]rules.Rule{}
	for _, file := range slices.Sorted(maps.Keys(files)) {
		if !fs.ValidPath(file) || file == "." || strings.ContainsAny(file, "\\:") || strings.ContainsFunc(file, func(r rune) bool { return r < 32 || r == 127 }) {
			return nil, invalid(file, "expected a contained portable local path")
		}
		if file == "README.md" || rules.IsGroupReadme(file) {
			continue
		}
		if slices.Contains(strings.Split(file, "/"), "assets") && !(strings.Count(file, "/") == 2 && path.Base(file) == "_group.json" && (strings.HasPrefix(file, "techs/") || strings.HasPrefix(file, "practices/"))) {
			continue
		}
		if path.Base(file) == "_group.json" {
			id := path.Dir(file)
			if err := rules.ValidateGroupID(id, "local/"+file); err != nil {
				return nil, err
			}
			metadata, err := rules.ParseGroupMetadata(json.RawMessage(files[file]), "local/"+file)
			if err != nil {
				return nil, err
			}
			ensureGroup(groups, id).Guidance = append(ensureGroup(groups, id).Guidance, Guidance{Source: "local", Metadata: metadata})
			continue
		}
		if !strings.HasSuffix(file, ".md") || strings.HasPrefix(path.Base(file), "_") {
			continue
		}
		parsed, err := rules.Parse(string(files[file]), file, "local")
		if err != nil {
			return nil, err
		}
		result[file] = parsed
	}
	return result, nil
}

// validateLocalLinks checks local rules and Markdown attachments against the shared destination policy.
// It checks retained attachments even if no active rule links to them; target existence belongs to rendering.
func validateLocalLinks(files map[string][]byte, localRules map[string]rules.Rule) error {
	for _, file := range slices.Sorted(maps.Keys(files)) {
		_, rule := localRules[file]
		if !rule && (rules.AssetDirectory(file) == "" || !strings.HasSuffix(file, ".md")) {
			continue
		}
		targets, err := rules.MarkdownTargets(string(files[file]), file)
		if err != nil {
			return fmt.Errorf("local/%s: %w", file, err)
		}
		for _, target := range targets {
			if err := rules.RequireAllowedTarget(file, target, nil); err != nil {
				return fmt.Errorf("local/%s: %w", file, err)
			}
		}
	}
	return nil
}

// invalid identifies input relationships that prevent an effective rule set from being produced.
func invalid(location, problem string) error {
	return &rules.ValidationError{Location: location, Problem: problem}
}

// resolveGuidance chooses the complete local definition when present, otherwise retains all imported definitions.
func resolveGuidance(definitions []Guidance) []Guidance {
	for _, guidance := range definitions {
		if guidance.Source == "local" {
			return []Guidance{guidance}
		}
	}
	return slices.Clone(definitions)
}

// selectedVersion verifies a supplied release tag satisfies its requested constraint and returns its normalized version.
func selectedVersion(source rules.Source, supplied Library) (string, error) {
	if source.Version == "" {
		if supplied.Tag != "" {
			return "", invalid(source.Name, "supply a selected tag only for version-based sources")
		}
		return "", nil
	}
	version, err := rules.TagVersion(supplied.Tag, source.Name+".resolvedTag")
	if err != nil {
		return "", err
	}
	constraint, err := rules.ParseVersionConstraint(source.Version, source.Name+".version")
	if err != nil {
		return "", err
	}
	matches, err := constraint.Matches(supplied.Tag, source.Name+".resolvedTag")
	if err != nil {
		return "", err
	}
	if !matches {
		return "", invalid(source.Name, "selected release does not satisfy the configured version constraint")
	}
	return version, nil
}

// compareRuleIDs compares digit runs numerically and other bytes lexically, without integer overflow.
// Numerically equivalent spellings use the full ID as a deterministic tie-breaker.
func compareRuleIDs(a, b string) int {
	left, right := a, b
	for len(left) > 0 && len(right) > 0 {
		if left[0] >= '0' && left[0] <= '9' && right[0] >= '0' && right[0] <= '9' {
			i, j := digitRunEnd(left), digitRunEnd(right)
			x, y := strings.TrimLeft(left[:i], "0"), strings.TrimLeft(right[:j], "0")
			if len(x) < len(y) {
				return -1
			}
			if len(x) > len(y) {
				return 1
			}
			if order := strings.Compare(x, y); order != 0 {
				return order
			}
			left, right = left[i:], right[j:]
			continue
		}
		if left[0] < right[0] {
			return -1
		}
		if left[0] > right[0] {
			return 1
		}
		left, right = left[1:], right[1:]
	}
	if order := strings.Compare(left, right); order != 0 {
		return order
	}
	return strings.Compare(a, b)
}

// digitRunEnd finds the end of a leading ASCII decimal run in a rule ID.
func digitRunEnd(text string) int {
	i := 0
	for i < len(text) && text[i] >= '0' && text[i] <= '9' {
		i++
	}
	return i
}
