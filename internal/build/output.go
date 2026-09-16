// Assemble generated guidance, untouched declared terms, and machine-readable provenance in memory.

package build

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
)

// Output owns generated-root-relative file contents, including byte-exact license and notice copies.
type Output struct {
	Files map[string][]byte `json:"files"`
}

// Options provides the declared tool version and per-index UTF-8 byte budget.
type Options struct {
	ToolVersion   string
	IndexMaxBytes int
}

// Prepare combines rendering and summary indexes with terms and provenance, returning no partial output.
// It accepts a Resolve result and performs no filesystem or network operations.
func Prepare(resolved Resolved, options Options) (Output, error) {
	if strings.TrimSpace(options.ToolVersion) == "" {
		return Output{}, invalid("toolVersion", "expected nonempty text")
	}
	rendered, err := RenderRules(resolved)
	if err != nil {
		return Output{}, err
	}
	indexes, err := RenderIndexes(resolved, options.IndexMaxBytes)
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
		for _, license := range source.Licenses {
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
		files["libraries/"+source.Name+"/README.md"] = []byte(libraryReadme(source))
	}
	files["rules/README.md"] = []byte("# Effective rules\n\nThese files contain the complete effective definitions after exclusions, replacements, and local additions. Start with [RULES.md](../RULES.md). See [provenance.json](../provenance.json) for origins. Edit source inputs and rebuild.\n")
	files["groups/README.md"] = []byte("# Rule groups\n\nStart with [RULES.md](../RULES.md), then open relevant group indexes and read each applicable rule in full. These pages contain summaries, not full guidance.\n")
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

// libraryReadme exposes source identity and generated terms without interpreting their legal meaning.
func libraryReadme(source Source) string {
	file := "libraries/" + source.Name + "/README.md"
	requested := source.Ref
	if source.Version != "" {
		requested = source.Version
	}
	sections := []string{"# " + escapeText(source.Name), "This folder retains declared library license and notice files.", "**Repository:** " + escapeText(source.Repository), "**Requested revision or version:** " + escapeText(requested), "**Resolved commit:** `" + source.Commit + "`"}
	if source.Tag != "" {
		sections = append(sections, "**Selected tag:** "+escapeText(source.Tag), "**Selected version:** "+escapeText(source.ResolvedVersion))
	}
	if len(source.Licenses) == 0 {
		sections = append(sections, "No library license declaration was supplied.")
	}
	for _, license := range source.Licenses {
		if license.SPDXExpression != nil {
			sections = append(sections, "**Declared license:** "+escapeText(*license.SPDXExpression))
		}
		for _, mapping := range licenseMappings(source.Name, license) {
			sections = append(sections, "- ["+escapeText(path.Base(mapping.Generated))+"]("+relativeURL(file, mapping.Generated)+")")
		}
	}
	sections = append(sections, "[provenance.json](../../provenance.json) records source revisions, origins, and original and generated term paths.", "Use [RULES.md](../../RULES.md) to find effective rules. These files are generated; edit source inputs and rebuild.")
	return strings.Join(sections, "\n\n") + "\n"
}

// provenanceLicense records original and generated term locations without copying their contents.
type provenanceLicense struct {
	SPDXExpression            *string  `json:"spdxExpression"`
	Files                     []string `json:"files"`
	AttributionFiles          []string `json:"attributionFiles"`
	GeneratedFiles            []string `json:"generatedFiles"`
	GeneratedAttributionFiles []string `json:"generatedAttributionFiles"`
}

// provenanceSource records supplied revision identity and actual group selection.
type provenanceSource struct {
	Name            string               `json:"name"`
	Repository      string               `json:"repository"`
	Ref             string               `json:"ref,omitempty"`
	Version         string               `json:"version,omitempty"`
	Tag             string               `json:"resolvedTag,omitempty"`
	ResolvedVersion string               `json:"resolvedVersion,omitempty"`
	Commit          string               `json:"resolvedCommit"`
	Groups          []string             `json:"groups"`
	Selection       rules.GroupSelection `json:"groupSelection"`
	Licenses        []provenanceLicense  `json:"licenses"`
}

// provenanceGroup retains all guidance plus the sources chosen for display.
type provenanceGroup struct {
	ID        string     `json:"id"`
	Guidance  []Guidance `json:"guidance"`
	Effective []string   `json:"effectiveGuidanceSources"`
}

// provenanceRule records effective origin, replacement history, and declared attribution.
type provenanceRule struct {
	ID           string              `json:"id"`
	Group        string              `json:"group"`
	Origin       Origin              `json:"origin"`
	Upstream     *Origin             `json:"upstream"`
	Reason       string              `json:"replacementReason,omitempty"`
	LicenseBasis string              `json:"licenseBasis"`
	Licenses     []provenanceLicense `json:"licenses"`
	Attribution  []rules.Attribution `json:"attribution"`
}

// termProvenance maps declared source terms to retained workspace and generated paths.
func termProvenance(source string, declarations []rules.LicenseDeclaration) []provenanceLicense {
	result := []provenanceLicense{}
	for _, license := range declarations {
		record := provenanceLicense{SPDXExpression: license.SPDXExpression, Files: []string{}, AttributionFiles: []string{}, GeneratedFiles: []string{}, GeneratedAttributionFiles: []string{}}
		for _, mapping := range licenseMappings(source, license) {
			original := "vendor/" + source + "/" + mapping.Source
			if mapping.Kind == "license" {
				record.Files = append(record.Files, original)
				record.GeneratedFiles = append(record.GeneratedFiles, mapping.Generated)
			} else {
				record.AttributionFiles = append(record.AttributionFiles, original)
				record.GeneratedAttributionFiles = append(record.GeneratedAttributionFiles, mapping.Generated)
			}
		}
		result = append(result, record)
	}
	return result
}

// renderProvenance serializes stable identities and policy outcomes, without redundant raw documents.
func renderProvenance(resolved Resolved, version string) ([]byte, error) {
	result := struct {
		ToolVersion string             `json:"toolVersion"`
		Sources     []provenanceSource `json:"sources"`
		Groups      []provenanceGroup  `json:"groups"`
		Rules       []provenanceRule   `json:"rules"`
	}{ToolVersion: version, Sources: []provenanceSource{}, Groups: []provenanceGroup{}, Rules: []provenanceRule{}}
	for _, source := range resolved.Sources {
		result.Sources = append(result.Sources, provenanceSource{Name: source.Name, Repository: source.Repository, Ref: source.Ref, Version: source.Version, Tag: source.Tag, ResolvedVersion: source.ResolvedVersion, Commit: source.Commit, Groups: source.Groups, Selection: source.Selection, Licenses: termProvenance(source.Name, source.Licenses)})
	}
	for _, group := range resolved.Groups {
		effective := []string{}
		for _, guidance := range EffectiveGuidance(group) {
			effective = append(effective, guidance.Source)
		}
		result.Groups = append(result.Groups, provenanceGroup{group.ID, group.Guidance, effective})
		for _, active := range group.Rules {
			basis := "undeclared"
			if len(active.Licenses) > 0 {
				basis = "library"
			}
			result.Rules = append(result.Rules, provenanceRule{ID: active.Rule.ID, Group: active.Rule.Group, Origin: active.Origin, Upstream: active.Upstream, Reason: active.Reason, LicenseBasis: basis, Licenses: termProvenance(active.Origin.Source, active.Licenses), Attribution: active.Rule.Attribution})
		}
	}
	slices.SortFunc(result.Sources, func(a, b provenanceSource) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(result.Groups, func(a, b provenanceGroup) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(result.Rules, func(a, b provenanceRule) int { return strings.Compare(a.ID, b.ID) })
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode provenance: %w", err)
	}
	return append(data, '\n'), nil
}
