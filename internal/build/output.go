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
		files["libraries/"+source.Name+"/README.md"] = []byte(libraryReadme(source))
	}
	files["rules/README.md"] = []byte("# Resolved rules\n\nThese files contain the complete resolved definitions after exclusions, replacements, and local additions. Start with [RULES.md](../RULES.md). See [provenance.json](../provenance.json) for origins. Edit source inputs and rebuild.\n")
	files["groups/README.md"] = []byte("# Rule groups\n\nStart with [RULES.md](../RULES.md), then open relevant group indexes and read each applicable rule in full. Each group page provides reading instructions and either complete rules or summaries with explicit links to the full definitions.\n")
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
func libraryReadme(source resolvedSource) string {
	file := "libraries/" + source.Name + "/README.md"
	requested := source.Ref
	if source.Version != "" {
		requested = source.Version
	}
	sections := []string{"# " + escapeText(source.Name), "This folder retains declared library license and notice files.", "**Repository:** " + escapeText(source.Repository), "**Requested revision or version:** " + escapeText(requested), "**Resolved commit:** `" + source.Commit + "`"}
	if source.Tag != "" {
		sections = append(sections, "**Selected tag:** "+escapeText(source.Tag), "**Selected version:** "+escapeText(source.ResolvedVersion))
	}
	if source.License == nil {
		sections = append(sections, "No library license declaration was supplied.")
	}
	if license := source.License; license != nil {
		if license.SPDXExpression != nil {
			sections = append(sections, "**Declared license:** "+escapeText(*license.SPDXExpression))
		}
		for _, mapping := range licenseMappings(source.Name, license) {
			sections = append(sections, "- ["+escapeText(path.Base(mapping.Generated))+"]("+relativeURL(file, mapping.Generated)+")")
		}
	}
	sections = append(sections, "[provenance.json](../../provenance.json) records source revisions, origins, and original and generated term paths.", "Use [RULES.md](../../RULES.md) to find resolved rules. These files are generated; edit source inputs and rebuild.")
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
	LicenseFiles    []string             `json:"licenseFiles"`
	License         *provenanceLicense   `json:"license"`
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
		ToolVersion string             `json:"toolVersion"`
		Sources     []provenanceSource `json:"sources"`
		Groups      []provenanceGroup  `json:"groups"`
		Rules       []provenanceRule   `json:"rules"`
	}{ToolVersion: version, Sources: []provenanceSource{}, Groups: []provenanceGroup{}, Rules: []provenanceRule{}}
	for _, source := range resolved.Sources {
		result.Sources = append(result.Sources, provenanceSource{Name: source.Name, Repository: source.Repository, Ref: source.Ref, Version: source.Version, Tag: source.Tag, ResolvedVersion: source.ResolvedVersion, Commit: source.Commit, Groups: source.Groups, Selection: source.Selection, LicenseFiles: rules.LicensePaths(source.License), License: termProvenance(source.Name, "", source.License)})
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
			result.Rules = append(result.Rules, provenanceRule{ID: active.Rule.ID, Group: active.Rule.Group, Origin: *originProvenance(&active.Origin), Upstream: originProvenance(active.Upstream), Reason: nullableText(active.Reason), LicenseBasis: basis, License: termProvenance(active.Origin.Source, "vendor/"+active.Origin.Source+"/", active.License), Attribution: active.Rule.Attribution})
		}
	}
	slices.SortFunc(result.Sources, func(a, b provenanceSource) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(result.Groups, func(a, b provenanceGroup) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(result.Rules, func(a, b provenanceRule) int { return strings.Compare(a.ID, b.ID) })
	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	encoder.SetIndent("", "  ")
	// This is a standalone JSON file; preserve readable constraints such as ">= 1.0.0".
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(result); err != nil {
		return nil, fmt.Errorf("encode provenance: %w", err)
	}
	return data.Bytes(), nil
}

// provenanceOrigin preserves explicit nulls for unavailable repository identity fields.
type provenanceOrigin struct {
	Source     string  `json:"source"`
	File       string  `json:"file"`
	Repository *string `json:"repository"`
	Ref        *string `json:"ref"`
	Commit     *string `json:"resolvedCommit"`
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
	return &provenanceOrigin{origin.Source, origin.File, nullableText(origin.Repository), nullableText(origin.Ref), nullableText(origin.Commit)}
}
