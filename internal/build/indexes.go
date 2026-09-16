// Render applicability indexes with complete entries and bounded UTF-8 pages.

package build

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
)

// IndexPages splits an index at entry boundaries; every returned file fits maxBytes.
// The original path is either the complete page or a complete directory of numbered parts.
func IndexPages(file, header string, entries []string, footer string, maxBytes int) (map[string]string, error) {
	if !fs.ValidPath(file) || !strings.HasSuffix(file, ".md") || strings.ContainsAny(file, "\\\x00") {
		return nil, invalid(file, "expected a contained Markdown output path")
	}
	if maxBytes <= 0 {
		return nil, invalid(file, "indexMaxBytes must be positive")
	}
	whole := indexDocument(header, entries, footer)
	if len(whole) <= maxBytes {
		return map[string]string{file: whole}, nil
	}
	output := map[string]string{}
	links := []string{}
	pending := []string{}
	part := 1
	// partHeader identifies the numbered page and provides a return path.
	partHeader := func() string {
		return fmt.Sprintf("%s\n\nPart %d. [All parts](%s).", header, part, encodedPath(path.Base(file)))
	}
	// finish records a complete page and its directory entry.
	finish := func() {
		partFile := fmt.Sprintf("%s.part-%d.md", strings.TrimSuffix(file, ".md"), part)
		output[partFile] = indexDocument(partHeader(), pending, footer)
		links = append(links, fmt.Sprintf("- [Part %d](%s)", part, encodedPath(path.Base(partFile))))
		part++
		pending = nil
	}
	for _, entry := range entries {
		proposed := append(slices.Clone(pending), entry)
		if len(indexDocument(partHeader(), proposed, footer)) > maxBytes && len(pending) > 0 {
			finish()
		}
		if len(indexDocument(partHeader(), []string{entry}, footer)) > maxBytes {
			return nil, invalid(file, "an index entry and its reading instructions exceed indexMaxBytes; shorten the metadata or increase the budget")
		}
		pending = append(pending, entry)
	}
	if len(pending) > 0 {
		finish()
	}
	directory := indexDocument(header, append([]string{"Read every numbered part to inspect this complete index. Rule bodies remain in their linked files."}, links...), footer)
	if len(directory) > maxBytes {
		return nil, invalid(file, "the complete index part directory exceeds indexMaxBytes; increase the budget")
	}
	output[file] = directory
	return output, nil
}

// indexDocument joins nonempty blocks with stable spacing and one final newline.
func indexDocument(header string, entries []string, footer string) string {
	blocks := []string{}
	for _, block := range append(append([]string{header}, entries...), footer) {
		if block != "" {
			blocks = append(blocks, block)
		}
	}
	return strings.Join(blocks, "\n\n") + "\n"
}

// RenderIndexes creates summary-only discovery pages for a Resolve result.
// It never truncates a rule or embeds its body; each summary links to the standalone rendered file.
func RenderIndexes(resolved Resolved, maxBytes int) (map[string]string, error) {
	output := map[string]string{}
	groupEntries := []string{}
	for _, group := range resolved.Groups {
		file := "groups/" + group.ID + ".md"
		name := groupTitle(group)
		cues := []string{}
		for _, guidance := range EffectiveGuidance(group) {
			cues = append(cues, "**"+escapeText(guidance.Source)+":** "+escapeText(guidance.Metadata.WhenToRead))
		}
		groupEntries = append(groupEntries, "### "+name+"\n\n"+strings.Join(cues, "\n\n")+"\n\n**Open group:** ["+name+"]("+encodedPath(file)+")")
		entries := []string{}
		for _, active := range group.Rules {
			r := active.Rule
			entries = append(entries, "### "+escapeText(r.Title)+"\n\nRule ID: `"+r.ID+"`\n\n**When to read:** "+escapeText(r.WhenToRead)+"\n\n**Impact:** "+escapeText(string(r.Impact))+"\n\n**Why it matters:** "+escapeText(r.ImpactDescription)+"\n\n**Read full rule:** ["+escapeText(r.Title)+"]("+relativeURL(file, RulePath(r))+")")
		}
		header := "# " + name + "\n\nGroup ID: `" + group.ID + "`\n\nThis page contains summaries only. Open and read the complete guidance of every applicable or plausibly applicable rule. Complete truncated reads before relying on a rule. Impact describes consequences, not applicability or finding severity."
		footer := "For other groups, open [RULES.md](../../RULES.md). Generated output: edit source rules or configuration and rebuild."
		pages, err := IndexPages(file, header, entries, footer, maxBytes)
		if err != nil {
			return nil, err
		}
		for path, text := range pages {
			output[path] = text
		}
	}
	header := "# Code Rules\n\nChoose technology and practice groups using their reading cues. Read full relevant or plausibly relevant rules before planning, implementation, validation, or diagnosis. Consider behavior as well as language. Revisit selection when scope changes and reload needed rules after compaction. Selection alone is not evidence of a violation."
	pages, err := IndexPages("RULES.md", header, groupEntries, "", maxBytes)
	if err != nil {
		return nil, err
	}
	for path, text := range pages {
		output[path] = text
	}
	return output, nil
}

// groupTitle joins distinct effective names in deterministic order.
func groupTitle(group Group) string {
	names := []string{}
	for _, guidance := range EffectiveGuidance(group) {
		names = append(names, guidance.Metadata.Name)
	}
	slices.Sort(names)
	names = slices.Compact(names)
	for i, name := range names {
		names[i] = escapeText(name)
	}
	return strings.Join(names, " / ")
}
