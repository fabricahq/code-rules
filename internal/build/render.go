// Render resolved rules into standalone generated Markdown files.

package build

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
)

// RulePath maps one source-qualified identity to its standalone generated path.
func RulePath(rule rules.Rule) string {
	return "rules/" + strings.Replace(rule.ID, ":", "/", 1) + ".md"
}

// RenderRules renders a Resolve result without modifying inputs or writing files.
// Each rule gets one standalone document; combined group delivery is a separate capability.
func RenderRules(resolved Resolved) (map[string]string, error) {
	output := map[string]string{}
	sources := map[string]Source{}
	for _, source := range resolved.Sources {
		sources[source.Name] = source
	}
	for _, group := range resolved.Groups {
		for _, active := range group.Rules {
			paths := resolved.LocalPaths
			if active.Origin.Source != "local" {
				paths = sources[active.Origin.Source].Paths
			}
			file := RulePath(active.Rule)
			text, err := renderRule(active, paths, file)
			if err != nil {
				return nil, err
			}
			if _, exists := output[file]; exists {
				return nil, invalid(file, "duplicate output rule")
			}
			output[file] = text
		}
	}
	if err := validateOutputPaths(output); err != nil {
		return nil, err
	}
	return output, nil
}

// renderRule wraps rewritten guidance in applicability, origin, and original frontmatter sections.
func renderRule(active ActiveRule, paths []string, outputPath string) (string, error) {
	split, err := rules.SplitDocument(active.Rule.Document, active.Rule.ID)
	if err != nil {
		return "", err
	}
	body, err := renderBody(split.Body, active, paths, outputPath)
	if err != nil {
		return "", err
	}
	source, err := sourceLink(active.Origin, outputPath)
	if err != nil {
		return "", err
	}
	r := active.Rule
	lines := []string{"# " + escapeText(r.Title), "", "Rule ID: `" + r.ID + "`", "", "**When to read:** " + escapeText(r.WhenToRead), "", "**Impact:** " + escapeText(string(r.Impact)), "", "**Why it matters:** " + escapeText(r.ImpactDescription), "", "## Guidance", "", body, "", "## Source and attribution", "", "**Rule source:** [Original rule](" + source + ")"}
	for _, attribution := range r.Attribution {
		lines = append(lines, "", "**Attribution:** ["+escapeText(attribution.Description)+"](<"+strings.NewReplacer("<", "%3C", ">", "%3E").Replace(attribution.URL)+">)")
	}
	if license := active.License; license != nil {
		if license.SPDXExpression != nil {
			lines = append(lines, "", "**Declared license:** "+escapeText(*license.SPDXExpression))
		}
		for _, mapping := range licenseMappings(active.Origin.Source, license) {
			lines = append(lines, "", "- ["+escapeText(path.Base(mapping.Generated))+"]("+relativeURL(outputPath, mapping.Generated)+")")
		}
	}
	fence := "```"
	for _, run := range strings.FieldsFunc(split.Frontmatter, func(r rune) bool { return r != '`' }) {
		if len(run) >= len(fence) {
			fence = strings.Repeat("`", len(run)+1)
		}
	}
	lines = append(lines, "", "### Source metadata", "", fence+"yaml\n"+split.Frontmatter+"\n"+fence)
	return strings.Join(lines, "\n") + "\n", nil
}

// escapeText keeps metadata on one line and escapes Markdown punctuation.
func escapeText(value string) string {
	var result strings.Builder
	for _, r := range strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", " "), "\n", " ") {
		if strings.ContainsRune("\\`*_{}[]()<>#+|", r) {
			result.WriteByte('\\')
		}
		result.WriteRune(r)
	}
	return result.String()
}

// encodedPath escapes URL segments without turning directory separators into data.
func encodedPath(file string) string {
	parts := strings.Split(file, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

// relativeURL derives a portable Markdown link between generated-root-relative paths.
func relativeURL(from, to string) string {
	relative, _ := filepath.Rel(filepath.FromSlash(path.Dir(from)), filepath.FromSlash(to))
	return encodedPath(filepath.ToSlash(relative))
}

// workspaceLink links from generated output back to retained local or vendor input.
func workspaceLink(outputPath, source, file string) string {
	root := "vendor/" + source
	if source == "local" {
		root = "local"
	}
	return relativeURL("generated/"+outputPath, root+"/"+file)
}

// sourceLink prefers a recognized repository's immutable commit URL and falls back to retained source.
func sourceLink(origin Origin, outputPath string) (string, error) {
	if origin.Repository != "" {
		raw, _ := json.Marshal(origin.Repository)
		repository, err := rules.ParseRepository(raw, origin.Source)
		if err != nil {
			return "", err
		}
		if link, ok := repository.FileURL(origin.Commit, origin.File, false); ok {
			return link, nil
		}
	}
	return workspaceLink(outputPath, origin.Source, origin.File), nil
}

// relocatedURL validates ownership and links local destinations to terms or retained files.
func relocatedURL(destination string, active ActiveRule, paths []string, outputPath string) (string, error) {
	if strings.HasPrefix(destination, "#") {
		return destination, nil
	}
	target, suffix, local, err := rules.RelativeTarget(destination, active.Origin.File)
	if err != nil {
		return "", err
	}
	if !local {
		return destination, nil
	}
	if err := rules.RequireAllowedTarget(active.Origin.File, target, rules.LicensePaths(active.License)); err != nil {
		return "", err
	}
	if license := active.License; license != nil {
		for _, mapping := range licenseMappings(active.Origin.Source, license) {
			if mapping.Source == target {
				return relativeURL(outputPath, mapping.Generated) + suffix, nil
			}
		}
	}
	if slices.Contains(paths, target) {
		return workspaceLink(outputPath, active.Origin.Source, target) + suffix, nil
	}
	if rules.AssetDirectory(target) != "" {
		return "", invalid(active.Rule.ID, "missing asset link destination: "+destination)
	}
	return "", invalid(active.Rule.ID, "missing retained link destination: "+destination)
}

// licenseMapping identifies a source term file and its fixed generated destination.
type licenseMapping struct{ Source, Generated, Kind string }

// licenseMappings assigns stable license and notice names while preserving declaration order.
func licenseMappings(source string, license *rules.LicenseDeclaration) []licenseMapping {
	root := "libraries/" + source + "/licenses"
	result := []licenseMapping{}
	for _, file := range license.Files {
		result = append(result, licenseMapping{file, root + "/LICENSE.md", "license"})
	}
	for i, file := range license.AttributionFiles {
		result = append(result, licenseMapping{file, fmt.Sprintf("%s/notices/%03d.md", root, i+1), "notice"})
	}
	return result
}
