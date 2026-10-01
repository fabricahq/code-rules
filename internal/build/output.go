// Assemble generated guidance, untouched declared terms, and machine-readable provenance in memory.

package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/rules"
)

// prepare combines rendering and group discovery pages with terms and provenance, returning no partial output.
// It accepts a resolve result and performs no filesystem or network operations.
func prepare(resolved resolution, options Options) (Output, error) {
	if err := rules.ValidateToolVersion(options.ToolVersion); err != nil {
		return Output{}, err
	}
	rendered, err := renderRules(resolved)
	if err != nil {
		return Output{}, err
	}
	inlineMaxBytes := 8 * 1024
	if options.GroupInlineMaxBytes != nil {
		inlineMaxBytes = *options.GroupInlineMaxBytes
	}
	indexes, err := renderIndexes(resolved, options.IndexMaxLines, inlineMaxBytes)
	if err != nil {
		return Output{}, err
	}
	files := map[string][]byte{}
	for file, text := range rendered {
		files[file] = []byte(text)
	}
	for file, text := range indexes {
		if _, ok := files[file]; ok {
			return Output{}, invalid(file, "duplicate generated path")
		}
		files[file] = []byte(text)
	}
	for _, source := range resolved.Sources {
		if license := source.License; license != nil {
			for _, mapping := range licenseMappings(source.Name, license) {
				data, ok := source.Files[mapping.Source]
				if !ok {
					return Output{}, invalid(source.Name+":"+mapping.Source, "missing declared license or notice bytes")
				}
				if previous, ok := files[mapping.Generated]; ok && !bytes.Equal(previous, data) {
					return Output{}, invalid(mapping.Generated, "conflicting declared terms")
				}
				files[mapping.Generated] = bytes.Clone(data)
			}
		}
		files["libraries/"+source.Name+"/README.md"] = []byte(generatedMarkdown(libraryReadme(source)))
	}
	files["rules/README.md"] = []byte(generatedMarkdown("# Resolved rules\n\nThese files contain the complete resolved definitions after exclusions, replacements, and local additions. Start with [RULES.md](../RULES.md). See [provenance.json](../provenance.json) for origins. Edit source inputs and rebuild.\n"))
	files["groups/README.md"] = []byte(generatedMarkdown("# Rule groups\n\nStart with [RULES.md](../RULES.md), then open relevant group indexes and read each applicable rule in full. Each group page provides reading instructions and either complete rules or summaries with explicit links to the full definitions.\n"))
	provenance, err := renderProvenance(resolved, options.ToolVersion)
	if err != nil {
		return Output{}, err
	}
	files["provenance.json"] = provenance
	if err := validateOutputPaths(files); err != nil {
		return Output{}, err
	}
	return Output{Files: files}, nil
}

// libraryReadme summarizes the imported library release, each imported rule's version, the source's pins, and
// generated terms, without interpreting their legal meaning. It says when the source imports unreleased changes.
func libraryReadme(source resolvedSource) string {
	file := "libraries/" + source.Name + "/README.md"
	sections := []string{"# " + escapeText(source.Name)}
	// Only a library that declares license or notice files has copies here to describe.
	if source.License != nil && len(licenseMappings(source.Name, source.License)) > 0 {
		sections = append(sections, "This folder retains byte-for-byte copies of declared library license and notice files. Do not edit these copies; change the upstream library and run `code-rules project sync`.")
	}
	sections = append(sections, "**Repository:** "+escapeText(source.Repository))
	if source.Release != 0 {
		sections = append(sections, fmt.Sprintf("**Library release:** release/%d", source.Release))
	}
	if source.Ref != "" {
		sections = append(sections, "**Requested revision:** "+escapeText(source.Ref))
	}
	sections = append(sections, "**Resolved commit:** `"+source.Commit+"`")
	if source.Ref != "" && source.Release == 0 && slices.ContainsFunc(slices.Collect(maps.Values(source.Versions)), func(rule library.ImportedRule) bool { return rule.Version == nil }) {
		sections = append(sections, "**Imported from unreleased changes.** This source's ref isn't a library release, so the source doesn't follow rule versions: rules with unreleased changes have no version to cite.")
	}
	sections = append(sections, "## Rule versions", ruleVersionTable(source))
	if len(source.Pins) > 0 {
		sections = append(sections, "## Pins", "`code-rules project update` keeps these rules at their pinned versions.", pinList(source.Pins))
	}
	sections = append(sections, "## License terms")
	if source.License == nil {
		sections = append(sections, "No library license declaration was supplied.")
	}
	if license := source.License; license != nil {
		if license.SPDXExpression != nil {
			sections = append(sections, "**Declared license:** "+escapeText(*license.SPDXExpression))
		}
		for _, mapping := range licenseMappings(source.Name, license) {
			sections = append(sections, "- ["+escapeText(path.Base(mapping.Generated))+"]("+rules.RelativeLink(file, mapping.Generated)+")")
		}
	}
	sections = append(sections, "[provenance.json](../../provenance.json) records source revisions, origins, and original and generated term paths.", "Use [RULES.md](../../RULES.md) to find resolved rules. These files are generated; edit source inputs and rebuild.")
	return strings.Join(sections, "\n\n") + "\n"
}

// ruleVersionTable lists each imported rule of source with its version, the library release that published it, and
// whether agents read it: an excluded rule, which the source still imports so updates can report its changes, is
// marked excluded, or replaced by its local rule, and a retired rule the source still imports is marked retired,
// with the ref or pin that keeps it.
func ruleVersionTable(source resolvedSource) string {
	versions, exclude := source.Versions, source.Exclude
	if len(versions) == 0 {
		return "This source imports no rules."
	}
	rows := []string{"| Rule | Version | Library release | Status |", "| --- | --- | --- | --- |"}
	for _, id := range slices.Sorted(maps.Keys(versions)) {
		rule := versions[id]
		status := "Active"
		pin, pinned := source.Pins[id]
		switch exclusion, excluded := exclude[id]; {
		case excluded && exclusion.ReplacedBy != "":
			status = "Replaced by `" + exclusion.ReplacedBy + "`"
		case excluded:
			status = "Excluded"
		case slices.Contains(source.Retired, id) && source.Ref != "":
			status = "Retired upstream; kept by ref " + source.Ref
		case slices.Contains(source.Retired, id) && pinned:
			status = "Retired, pinned at " + pin.Version.String()
		case slices.Contains(source.Retired, id):
			status = "Retired; the next update drops it"
		}
		if rule.Version == nil {
			rows = append(rows, "| `"+id+"` | No version | Unreleased | "+status+" |")
			continue
		}
		rows = append(rows, fmt.Sprintf("| `%s` | %s | release/%d | %s |", id, rule.Version, rule.Release, status))
	}
	return strings.Join(rows, "\n")
}

// pinList lists each pinned rule, in ID order, with its pinned version and the reason the configuration gives.
func pinList(pins map[string]rules.Pin) string {
	items := []string{}
	for _, id := range slices.Sorted(maps.Keys(pins)) {
		items = append(items, fmt.Sprintf("- `%s`: %s. Reason: %s", id, pins[id].Version, escapeText(pins[id].Reason)))
	}
	return strings.Join(items, "\n")
}

// provenanceLicense records original and generated term locations without copying their contents.
type provenanceLicense struct {
	SPDXExpression            *string  `json:"spdxExpression"`
	Files                     []string `json:"files"`
	AttributionFiles          []string `json:"attributionFiles"`
	GeneratedFiles            []string `json:"generatedFiles"`
	GeneratedAttributionFiles []string `json:"generatedAttributionFiles"`
}

// provenanceSource records the requested versions, the library release that supplied group metadata and terms,
// and the requested and actual selection. Pins, ref, release, and ruleSelection are omitted when empty, as in
// the source's _source.json.
type provenanceSource struct {
	Name          string               `json:"name"`
	Repository    string               `json:"repository"`
	Pins          map[string]rules.Pin `json:"pins,omitempty"`
	Ref           string               `json:"ref,omitempty"`
	Release       int                  `json:"release,omitempty"`
	Commit        string               `json:"resolvedCommit"`
	Groups        []string             `json:"groups"`
	Selection     rules.GroupSelection `json:"groupSelection"`
	RuleSelection []string             `json:"ruleSelection,omitempty"`
	LicenseFiles  []string             `json:"licenseFiles"`
	License       *provenanceLicense   `json:"license"`
}

// provenanceGroup retains all guidance plus the sources chosen for display.
type provenanceGroup struct {
	ID        string          `json:"id"`
	Guidance  []groupGuidance `json:"guidance"`
	Effective []string        `json:"effectiveGuidanceSources"`
}

// provenanceRule records effective origin, replacement history, and declared attribution.
type provenanceRule struct {
	ID           string              `json:"id"`
	Group        string              `json:"group"`
	Origin       provenanceOrigin    `json:"origin"`
	Upstream     *provenanceOrigin   `json:"upstream"`
	Reason       *string             `json:"replacementReason"`
	BasedOn      *rules.RuleVersion  `json:"basedOn"`
	LicenseBasis string              `json:"licenseBasis"`
	License      *provenanceLicense  `json:"license"`
	Attribution  []rules.Attribution `json:"attribution"`
}

// termProvenance maps declared source terms to retained workspace and generated paths.
func termProvenance(source, originalPrefix string, license *rules.LicenseDeclaration) *provenanceLicense {
	if license == nil {
		return nil
	}
	record := &provenanceLicense{SPDXExpression: license.SPDXExpression, Files: []string{}, AttributionFiles: []string{}, GeneratedFiles: []string{}, GeneratedAttributionFiles: []string{}}
	for _, mapping := range licenseMappings(source, license) {
		original := originalPrefix + mapping.Source
		if mapping.Kind == "license" {
			record.Files = append(record.Files, original)
			record.GeneratedFiles = append(record.GeneratedFiles, mapping.Generated)
		} else {
			record.AttributionFiles = append(record.AttributionFiles, original)
			record.GeneratedAttributionFiles = append(record.GeneratedAttributionFiles, mapping.Generated)
		}
	}
	return record
}

// renderProvenance serializes stable identities and policy outcomes, without redundant raw documents.
func renderProvenance(resolved resolution, version string) ([]byte, error) {
	result := struct {
		GeneratedNotice string             `json:"generatedNotice"`
		ToolVersion     string             `json:"toolVersion"`
		Sources         []provenanceSource `json:"sources"`
		Groups          []provenanceGroup  `json:"groups"`
		Rules           []provenanceRule   `json:"rules"`
	}{GeneratedNotice: generatedNotice + " " + strings.ReplaceAll(regenerationNotice, "`", ""), ToolVersion: version, Sources: []provenanceSource{}, Groups: []provenanceGroup{}, Rules: []provenanceRule{}}
	for _, source := range resolved.Sources {
		record := provenanceSource{Name: source.Name, Repository: source.Repository, Ref: source.Ref, Release: source.Release, Commit: source.Commit, Groups: source.Groups, Selection: source.Selection, RuleSelection: source.Rules, LicenseFiles: rules.LicensePaths(source.License), License: termProvenance(source.Name, "", source.License)}
		if len(source.Pins) > 0 {
			record.Pins = source.Pins
		}
		result.Sources = append(result.Sources, record)
	}
	for _, group := range resolved.Groups {
		effective := []string{}
		for _, guidance := range group.EffectiveGuidance {
			effective = append(effective, guidance.Source)
		}
		guidance := slices.Clone(group.Guidance)
		// Keep source guidance in its established order, with local guidance last.
		slices.SortStableFunc(guidance, func(a, b groupGuidance) int {
			if a.Source == "local" && b.Source != "local" {
				return 1
			}
			if a.Source != "local" && b.Source == "local" {
				return -1
			}
			return 0
		})
		result.Groups = append(result.Groups, provenanceGroup{group.ID, guidance, effective})
		for _, active := range group.Rules {
			basis := "undeclared"
			if active.License != nil {
				basis = "library"
			}
			result.Rules = append(result.Rules, provenanceRule{ID: active.Rule.ID, Group: active.Rule.Group, Origin: *originProvenance(&active.Origin), Upstream: originProvenance(active.Upstream), Reason: nullableText(active.Reason), BasedOn: active.BasedOn, LicenseBasis: basis, License: termProvenance(active.Origin.Source, "vendor/"+active.Origin.Source+"/", active.License), Attribution: active.Rule.Attribution})
		}
	}
	slices.SortFunc(result.Sources, func(a, b provenanceSource) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(result.Groups, func(a, b provenanceGroup) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(result.Rules, func(a, b provenanceRule) int { return strings.Compare(a.ID, b.ID) })
	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	encoder.SetIndent("", "  ")
	// This is a standalone JSON file; keep characters such as <, >, and & readable.
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(result); err != nil {
		return nil, fmt.Errorf("encode provenance: %w", err)
	}
	return data.Bytes(), nil
}

// provenanceOrigin preserves explicit nulls for unavailable repository identity and version fields. Commit is
// the commit that supplied the rule's files; Version and Release are null for local rules and for imported
// files that aren't a published version.
type provenanceOrigin struct {
	Source     string             `json:"source"`
	File       string             `json:"file"`
	Repository *string            `json:"repository"`
	Ref        *string            `json:"ref"`
	Commit     *string            `json:"resolvedCommit"`
	Version    *rules.RuleVersion `json:"version"`
	Release    *int               `json:"release"`
}

// nullableText represents absent optional provenance text as JSON null.
func nullableText(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// originProvenance converts internal identity values to the public nullable provenance shape.
func originProvenance(origin *ruleOrigin) *provenanceOrigin {
	if origin == nil {
		return nil
	}
	result := &provenanceOrigin{origin.Source, origin.File, nullableText(origin.Repository), nullableText(origin.Ref), nullableText(origin.Commit), origin.Version, nil}
	if origin.Version != nil {
		release := origin.Release
		result.Release = &release
	}
	return result
}
