// Render applicability indexes with complete entries and bounded UTF-8 pages.

package build

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode/utf8"
)

// IndexPages splits an index at entry boundaries; every returned file fits maxBytes.
// The original path is either the complete page or a complete directory of numbered parts.
func IndexPages(file, header string, entries []string, footer string, maxBytes int) (map[string]string, error) {
	if !strings.HasSuffix(file, ".md") {
		return nil, invalid(file, "expected a contained Markdown output path")
	}
	if err := validateOutputPaths(map[string]string{file: ""}); err != nil {
		return nil, err
	}
	if maxBytes <= 0 {
		return nil, invalid(file, "indexMaxBytes must be positive")
	}
	for _, value := range append([]string{file, header, footer}, entries...) {
		if !utf8.ValidString(value) {
			return nil, invalid(file, "index paths and content must be valid UTF-8")
		}
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
	overhead := len(indexDocument(partHeader(), nil, footer))
	pageBytes := overhead
	// finish records a complete page and its directory entry.
	finish := func() {
		partFile := fmt.Sprintf("%s.part-%d.md", strings.TrimSuffix(file, ".md"), part)
		output[partFile] = indexDocument(partHeader(), pending, footer)
		links = append(links, fmt.Sprintf("- [Part %d](%s)", part, encodedPath(path.Base(partFile))))
		part++
		pending = nil
		overhead = len(indexDocument(partHeader(), nil, footer))
		pageBytes = overhead
	}
	for _, entry := range entries {
		entryBytes := 0
		if entry != "" {
			// A part header is always nonempty, so each nonempty entry adds one separator.
			entryBytes = len(entry) + 2
		}
		if pageBytes+entryBytes > maxBytes && len(pending) > 0 {
			finish()
		}
		if overhead+entryBytes > maxBytes {
			return nil, invalid(file, "an index entry and its reading instructions exceed indexMaxBytes; shorten the metadata or increase the budget")
		}
		pending = append(pending, entry)
		pageBytes += entryBytes
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
		cues := groupReadingGuidance(group)
		groupEntries = append(groupEntries, "### "+name+"\n\n"+cues+"\n\n**Open group:** ["+name+"]("+encodedPath(file)+")")
		entries := []string{}
		for _, active := range group.Rules {
			r := active.Rule
			entries = append(entries, "### "+escapeText(r.Title)+"\n\nRule ID: `"+r.ID+"`\n\n**When to read:** "+escapeText(r.WhenToRead)+"\n\n**Impact:** "+escapeText(string(r.Impact))+"\n\n**Why it matters:** "+escapeText(r.ImpactDescription)+"\n\n**Read full rule:** ["+escapeText(r.Title)+"]("+relativeURL(file, RulePath(r))+")")
		}
		if len(entries) == 0 {
			entries = append(entries, "No active rules in this group.")
		}
		header := "# " + name + "\n\nGroup ID: `" + group.ID + "`\n\n" + cues + "\n\nThis page contains summaries only. Open each applicable rule’s full file. " + indexReadingInstructions + " Impact describes consequences, not applicability or finding severity."
		footer := "For other groups, open [RULES.md](../../RULES.md). Generated output: edit source rules or configuration and rebuild."
		pages, err := IndexPages(file, header, entries, footer, maxBytes)
		if err != nil {
			return nil, err
		}
		for path, text := range pages {
			output[path] = text
		}
	}
	header := "# Code Rules\n\nChoose technology and practice groups using their reading cues. " + indexReadingInstructions
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
	for _, guidance := range group.EffectiveGuidance {
		names = append(names, guidance.Metadata.Name)
	}
	slices.Sort(names)
	names = slices.Compact(names)
	for i, name := range names {
		names[i] = escapeText(name)
	}
	return strings.Join(names, " / ")
}

// Shared reading instructions keep entry indexes and directly opened group pages self-contained.
const indexReadingInstructions = "Read full relevant or plausibly relevant rules before planning, implementation, validation, or diagnosis. Complete truncated reads before relying on a rule. Consider behavior as well as language. Revisit selection when scope changes and reload needed rules after compaction. Selection alone is not evidence of a violation."

// groupReadingGuidance repeats the resolved group cues, labeling sources only when multiple definitions apply.
func groupReadingGuidance(group Group) string {
	if len(group.EffectiveGuidance) == 1 {
		return "**When to read this group:** " + escapeText(group.EffectiveGuidance[0].Metadata.WhenToRead)
	}
	cues := []string{"**When to read this group:**"}
	for _, guidance := range group.EffectiveGuidance {
		cues = append(cues, "**"+escapeText(guidance.Source)+": "+escapeText(guidance.Metadata.Name)+":** "+escapeText(guidance.Metadata.WhenToRead))
	}
	return strings.Join(cues, "\n\n")
}
