// Render applicability indexes with complete entries and bounded Markdown pages.

package build

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode/utf8"
)

// DefaultIndexMaxLines keeps ordinary summaries together before pagination.
const DefaultIndexMaxLines = 750

// IndexPages counts Markdown source lines, including blank lines and reading instructions.
// LF and CRLF both end one line; visual wrapping does not add lines.
// IndexPages splits an index at entry boundaries; every returned file fits maxLines.
// The original path is either the complete page or a complete directory of numbered parts.
func IndexPages(file, header string, entries []string, footer string, maxLines int) (map[string]string, error) {
	if !strings.HasSuffix(file, ".md") {
		return nil, invalid(file, "expected a contained Markdown output path")
	}
	if err := validateOutputPaths(map[string]string{file: ""}); err != nil {
		return nil, err
	}
	if maxLines <= 0 {
		return nil, invalid(file, "indexMaxLines must be positive")
	}
	for _, value := range append([]string{file, header, footer}, entries...) {
		if !utf8.ValidString(value) {
			return nil, invalid(file, "index paths and content must be valid UTF-8")
		}
	}
	whole := indexDocument(header, entries, footer)
	if strings.Count(whole, "\n") <= maxLines {
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
	overhead := strings.Count(indexDocument(partHeader(), nil, footer), "\n")
	pageLines := overhead
	// finish records a complete page and its directory entry.
	finish := func() {
		partFile := fmt.Sprintf("%s.part-%d.md", strings.TrimSuffix(file, ".md"), part)
		output[partFile] = indexDocument(partHeader(), pending, footer)
		links = append(links, fmt.Sprintf("- [Part %d](%s)", part, encodedPath(path.Base(partFile))))
		part++
		pending = nil
		overhead = strings.Count(indexDocument(partHeader(), nil, footer), "\n")
		pageLines = overhead
	}
	for _, entry := range entries {
		entryLines := 0
		if entry != "" {
			// A part header is always nonempty, so each nonempty entry adds one separator.
			entryLines = strings.Count(entry, "\n") + 2
		}
		if pageLines+entryLines > maxLines && len(pending) > 0 {
			finish()
		}
		if overhead+entryLines > maxLines {
			return nil, invalid(file, "an index entry and its reading instructions exceed indexMaxLines; shorten the metadata or increase the budget")
		}
		pending = append(pending, entry)
		pageLines += entryLines
	}
	if len(pending) > 0 {
		finish()
	}
	directory := indexDocument(header, append([]string{"Read every numbered part to inspect this complete index. Rule bodies remain in their linked files."}, links...), footer)
	if strings.Count(directory, "\n") > maxLines {
		return nil, invalid(file, "the complete index part directory exceeds indexMaxLines; increase the budget")
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
func RenderIndexes(resolved Resolved, maxLines int) (map[string]string, error) {
	return renderIndexes(resolved, maxLines, 0)
}

// renderIndexes selects complete inline groups when both budgets permit, otherwise paginates summaries.
func renderIndexes(resolved Resolved, maxLines, inlineMaxBytes int) (map[string]string, error) {
	if maxLines <= 0 {
		return nil, invalid("indexMaxLines", "indexMaxLines must be positive")
	}
	if inlineMaxBytes < 0 {
		return nil, invalid("groupInlineMaxBytes", "groupInlineMaxBytes must be nonnegative")
	}
	paths := renderSourcePaths(resolved)
	output := map[string]string{}
	groupEntries := []string{}
	for _, group := range resolved.Groups {
		file := "groups/" + group.ID + ".md"
		name := groupTitle(group)
		cues := groupReadingGuidance(group)
		groupEntries = append(groupEntries, "### "+name+"\n\n"+cues+"\n\n**Open group:** ["+name+"]("+encodedPath(file)+")")
		footer := "For other technology and practice groups, open [RULES.md](../../RULES.md). These files are generated. Edit source rules or configuration and rebuild to change them."
		if inlineMaxBytes > 0 && len(group.Rules) > 0 {
			page, fits, err := inlineGroupPage(group, paths, file, footer, inlineMaxBytes)
			if err != nil {
				return nil, err
			}
			if fits && strings.Count(page, "\n") <= maxLines {
				output[file] = page
				continue
			}
		}
		entries := []string{}
		for _, active := range group.Rules {
			r := active.Rule
			entries = append(entries, "### "+escapeText(r.Title)+"\n\nRule ID: `"+r.ID+"`\n\n**When to read:** "+escapeText(r.WhenToRead)+"\n\n**Impact:** "+escapeText(string(r.Impact))+"\n\n**Why it matters:** "+escapeText(r.ImpactDescription)+"\n\n**Read full rule:** ["+escapeText(r.Title)+"]("+relativeURL(file, RulePath(r))+")")
		}
		if len(entries) == 0 {
			entries = append(entries, "No active rules in this group.")
		}
		header := groupIndexHeader(group.ID, name, cues, false)
		pages, err := IndexPages(file, header, entries, footer, maxLines)
		if err != nil {
			return nil, err
		}
		for path, text := range pages {
			output[path] = text
		}
	}
	header := indexHeader()
	pages, err := IndexPages("RULES.md", header, groupEntries, "These files are generated. Edit source rules or configuration and rebuild to change them.", maxLines)
	if err != nil {
		return nil, err
	}
	for path, text := range pages {
		output[path] = text
	}
	if err := validateOutputPaths(output); err != nil {
		return nil, err
	}
	return output, nil
}

// inlineGroupPage returns a whole group or a size miss, without truncation or partial output.
func inlineGroupPage(group Group, paths map[string][]string, file, footer string, maxBytes int) (string, bool, error) {
	header := groupIndexHeader(group.ID, groupTitle(group), groupReadingGuidance(group), true)
	bytes := len(indexDocument(header, nil, footer))
	entries := []string{}
	for _, active := range group.Rules {
		section, err := renderRule(active, paths[active.Origin.Source], file)
		if err != nil {
			return "", false, err
		}
		section = strings.TrimSuffix(section, "\n")
		bytes += len(section) + 2
		if bytes > maxBytes {
			return "", false, nil
		}
		entries = append(entries, section)
	}
	return indexDocument(header, entries, footer), true, nil
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

// Shared instructions keep complete reading and evidence requirements on every entry page.
const fullReadingInstructions = "Read the full text of every applicable or plausibly applicable rule before relying on it. Complete truncated reads. Revisit selection when scope changes and reload needed rules after compaction."
const validationInstructions = "During validation or diagnosis, independently select relevant rules from the task, code, and surrounding contracts. Cite rule IDs and concrete evidence for findings; selection alone is not evidence of a violation."

// indexHeader preserves the reference CLI's selection procedure for summary-only delivery.
func indexHeader() string {
	return strings.Join([]string{
		"# Code Rules",
		"This project uses [Fabrica Code Rules](https://github.com/fabricahq/code-rules) to declare its adopted engineering practices.",
		"Before planning or writing code, use the descriptions under **Technology and practice group indexes** below to choose which indexes to open. Consider the intended behavior as well as the technology; testing guidance can apply even when no test files have changed.",
		"Each group page includes full rules or summaries with explicit reading links. Exclusions and replacements are already applied.",
		fullReadingInstructions,
		validationInstructions,
		"## Technology and practice group indexes",
		"Open the relevant group indexes below, then select applicable rules and read their full guidance.",
	}, "\n\n")
}

// groupIndexHeader combines resolved selection cues with the reference CLI's numbered reading procedure.
func groupIndexHeader(id, name, cues string, inline bool) string {
	mode := "This file contains summaries only. Follow the reading instructions below to load the full rules."
	read := "2. For every relevant or plausibly relevant rule, open its “Read full rule” link and read the complete file. Complete truncated reads."
	if inline {
		mode = "Full rules are included below. Read each relevant or plausibly relevant rule completely before planning, implementation, validation, or diagnosis. Separate rule files remain available for direct references."
		read = "2. Read every relevant or plausibly relevant rule below completely, including its guidance and exceptions. Complete truncated reads."
	}
	return strings.Join([]string{
		"# " + name,
		"Group ID: `" + id + "`",
		cues,
		"## How to use this group",
		mode,
		"1. Compare each “When to read” cue with your intended task or the behavior you are reviewing.",
		read,
		"3. Apply the full rule’s guidance and exceptions. When present, use Implementation guidance when planning or changing code, and Validation guidance when reviewing, testing, or diagnosing behavior. Use both when your task includes both activities. These sections support the rule’s guidance; they do not replace it. Selection alone is insufficient evidence for a review finding.",
		"Use “When to read” to select rules. Read and follow every applicable rule, regardless of impact. Impact describes the consequence the rule addresses; it does not determine applicability, override exceptions, or set a review finding’s severity. Assess findings from concrete evidence and consequences.",
		fullReadingInstructions,
		validationInstructions,
		"## Rules",
	}, "\n\n")
}

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
