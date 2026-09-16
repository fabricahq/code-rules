// Package build resolves adopted rules and prepares generated output without filesystem writes.
package build

import (
	"encoding/json"
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
	Rule     rules.Rule                 `json:"rule"`
	Origin   Origin                     `json:"origin"`
	Upstream *Origin                    `json:"upstream"`
	Reason   string                     `json:"replacementReason,omitempty"`
	Licenses []rules.LicenseDeclaration `json:"licenses"`
}

// Guidance retains every source's group metadata; local guidance takes precedence for display.
type Guidance struct {
	Source   string              `json:"source"`
	Metadata rules.GroupMetadata `json:"metadata"`
}

// Group contains ID-sorted effective rules and source-labeled reading guidance.
type Group struct {
	ID       string       `json:"id"`
	Guidance []Guidance   `json:"guidance"`
	Rules    []ActiveRule `json:"rules"`
}

// Source records the adopted revision and complete retained inventory, including excluded rules.
// Files contains supporting bytes only; active rules own their original documents.
type Source struct {
	Name       string                     `json:"name"`
	Repository string                     `json:"repository"`
	Ref        string                     `json:"ref,omitempty"`
	Version    string                     `json:"version,omitempty"`
	Commit     string                     `json:"resolvedCommit"`
	Selection  rules.GroupSelection       `json:"groupSelection"`
	Groups     []string                   `json:"groups"`
	Licenses   []rules.LicenseDeclaration `json:"licenses"`
	Paths      []string                   `json:"paths"`
	Files      map[string][]byte          `json:"supportingFiles"`
}

// Resolved owns effective rules; supporting bytes are shared read-only with the input catalogs.
// LocalPaths includes replacement and additional rule paths without duplicating their documents.
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
		for _, id := range slices.Sorted(maps.Keys(candidates)) {
			parsed := candidates[id]
			if _, excluded := source.Exclude[id]; excluded {
				continue
			}
			origin := Origin{Source: source.Name, File: parsed.Path, Repository: source.Repository, Ref: source.Ref, Commit: ref.SHA}
			active := ActiveRule{Rule: parsed, Origin: origin, Licenses: supplied.Catalog.Licenses}
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
				used[file] = true
				active = ActiveRule{Rule: replacementRule, Origin: Origin{Source: "local", File: file}, Upstream: &origin, Reason: replacement.Reason, Licenses: []rules.LicenseDeclaration{}}
			}
			group := ensureGroup(groups, parsed.Group)
			group.Rules = append(group.Rules, active)
		}
		result.Sources = append(result.Sources, Source{Name: source.Name, Repository: source.Repository, Ref: source.Ref, Version: source.Version, Commit: ref.SHA, Selection: source.Groups, Groups: ids, Licenses: supplied.Catalog.Licenses, Paths: supplied.Catalog.Paths(), Files: supplied.Catalog.SupportingFiles})
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
		group.Rules = append(group.Rules, ActiveRule{Rule: parsed, Origin: Origin{Source: "local", File: file}, Licenses: []rules.LicenseDeclaration{}})
	}
	for _, id := range slices.Sorted(maps.Keys(groups)) {
		group := groups[id]
		slices.SortFunc(group.Rules, func(a, b ActiveRule) int { return strings.Compare(a.Rule.ID, b.Rule.ID) })
		slices.SortFunc(group.Guidance, func(a, b Guidance) int { return strings.Compare(a.Source, b.Source) })
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
		if file == "README.md" {
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

// invalid identifies input relationships that prevent an effective rule set from being produced.
func invalid(location, problem string) error {
	return &rules.ValidationError{Location: location, Problem: problem}
}

// EffectiveGuidance returns local metadata when present, otherwise all imported descriptions.
func EffectiveGuidance(group Group) []Guidance {
	for _, guidance := range group.Guidance {
		if guidance.Source == "local" {
			return []Guidance{guidance}
		}
	}
	return slices.Clone(group.Guidance)
}
