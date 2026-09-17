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
	// Navigation is one source line regardless of page count or which adjacent links exist.
	// Reserve its top and bottom spacing before partitioning; render once the total is known.
	navigation := indexPageNavigation(file, 1, 1)
	overhead := strings.Count(indexDocument(navigation+"\n\n"+header, nil, indexPageFooter(footer, navigation)), "\n")
	parts := [][]string{}
	pending := []string{}
	pageLines := overhead
	for _, entry := range entries {
		entryLines := 0
		if entry != "" {
			entryLines = strings.Count(entry, "\n") + 2
		}
		if pageLines+entryLines > maxLines && len(pending) > 0 {
			parts = append(parts, pending)
			pending = nil
			pageLines = overhead
		}
		if overhead+entryLines > maxLines {
			return nil, invalid(file, "an index entry and its reading instructions exceed indexMaxLines; shorten the metadata or increase the budget")
		}
		pending = append(pending, entry)
		pageLines += entryLines
	}
	if len(pending) > 0 {
		parts = append(parts, pending)
	}
	output := map[string]string{}
	links := []string{}
	for i, entries := range parts {
		page := i + 1
		partFile := indexPartPath(file, page)
		navigation := indexPageNavigation(file, page, len(parts))
		output[partFile] = indexDocument(navigation+"\n\n"+header, entries, indexPageFooter(footer, navigation))
		links = append(links, fmt.Sprintf("- [Page %d of %d](%s)", page, len(parts), encodedPath(path.Base(partFile))))
	}
	directory := indexDocument(header, append([]string{"Read every numbered page to inspect this complete index. Rule bodies remain in their linked files."}, links...), footer)
	if strings.Count(directory, "\n") > maxLines {
		return nil, invalid(file, "the complete index part directory exceeds indexMaxLines; increase the budget")
	}
	output[file] = directory
	return output, nil
}

// indexPartPath names a numbered sibling of the index directory.
func indexPartPath(file string, page int) string {
	return fmt.Sprintf("%s.part-%d.md", strings.TrimSuffix(file, ".md"), page)
}

// indexPageNavigation identifies the page and links directly to its neighbors and directory.
func indexPageNavigation(file string, page, total int) string {
	links := []string{fmt.Sprintf("**Page %d of %d**", page, total), "[All pages](" + encodedPath(path.Base(file)) + ")"}
	if page > 1 {
		links = append(links, "[Previous page]("+encodedPath(path.Base(indexPartPath(file, page-1)))+")")
	}
	if page < total {
		links = append(links, "[Next page]("+encodedPath(path.Base(indexPartPath(file, page+1)))+")")
	}
	return strings.Join(links, " | ")
}

// indexPageFooter places navigation last without introducing blank lines for an absent notice.
func indexPageFooter(footer, navigation string) string {
	if footer == "" {
		return navigation
	}
	return footer + "\n\n" + navigation
}

// indexDocument joins content, separates the footer with a thematic break, and adds one final newline.
func indexDocument(header string, entries []string, footer string) string {
	blocks := []string{}
	for _, block := range append([]string{header}, entries...) {
		if block != "" {
			blocks = append(blocks, block)
		}
	}
	if footer != "" {
		blocks = append(blocks, "---", footer)
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
	header := indexHeader(resolved.Groups)
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

// Root-index instructions require complete reading and evidence when selecting groups.
const fullReadingInstructions = "Read the full text of every applicable or plausibly applicable rule before relying on it. Complete truncated reads. Revisit selection when scope changes and reload needed rules after compaction."
const validationInstructions = "During validation or diagnosis, independently select relevant rules from the task, code, and surrounding contracts. Cite rule IDs and concrete evidence for findings; selection alone is not evidence of a violation."

// indexHeader explains empty selections or guides agents to the resolved groups and rules.
func indexHeader(groups []Group) string {
	blocks := []string{
		"# Code Rules",
		"This project uses [Fabrica Code Rules](https://github.com/fabricahq/code-rules) to declare its adopted engineering practices.",
	}
	hasRules := false
	for _, group := range groups {
		if len(group.Rules) > 0 {
			hasRules = true
			break
		}
	}
	if !hasRules {
		blocks = append(blocks, "**No active rules are selected for this project.**", "There is no Code Rules guidance to load. Continue using the project’s other instructions.")
		if len(groups) > 0 {
			blocks = append(blocks, "## Technology and practice group indexes", "The selected groups below contain no active rules.")
		}
		return strings.Join(blocks, "\n\n")
	}
	return strings.Join(append(blocks,
		"Before planning or writing code, use the descriptions under **Technology and practice group indexes** below to choose which indexes to open. Consider the intended behavior as well as the technology; testing guidance can apply even when no test files have changed.",
		"Each group page includes full rules or summaries with explicit reading links. Exclusions and replacements are already applied.",
		fullReadingInstructions,
		validationInstructions,
		"## Technology and practice group indexes",
		"Open the relevant group indexes below, then select applicable rules and read their full guidance.",
	), "\n\n")
}

// groupIndexHeader combines resolved selection cues with the reading procedure for full rules or summaries.
func groupIndexHeader(id, name, cues string, inline bool) string {
	mode := "This file contains summaries only. Follow the reading instructions below to load the full rules."
	read := "2. **Read the rule completely.** Open its “Read full rule” link and read the guidance and exceptions before relying on it. Complete truncated reads. Revisit selection when scope changes and reload needed rules after compaction."
	if inline {
		mode = "Full rules are included below. Separate rule files remain available for direct references."
		read = "2. **Read the rule completely.** Read the guidance and exceptions before relying on a rule. Complete truncated reads. Revisit selection when scope changes and reload needed rules after compaction."
	}
	return strings.Join([]string{
		"# " + name,
		"**Group ID:** `" + id + "`",
		cues,
		"## How to use this group",
		mode,
		"1. **Select relevant rules.** Before planning, implementation, validation, or diagnosis, compare each “When to read” cue with your task, the code’s behavior, and surrounding contracts. Include plausibly relevant rules.",
		read,
		"3. **Apply the appropriate guidance.** Use Implementation guidance when planning or changing code, and Validation guidance when reviewing, testing, or diagnosing behavior. Use both when needed; these sections supplement the rule’s main guidance.",
		"4. **Ground findings in evidence.** During validation or diagnosis, independently select applicable rules and cite their IDs with concrete evidence. Selecting a rule does not establish a violation.",
		"**Impact describes consequences, not applicability or finding severity.** Follow every applicable rule regardless of impact, respect its exceptions, and assess findings using the actual evidence and consequences.",
		"## Rules",
	}, "\n\n")
}

// groupReadingGuidance repeats resolved descriptions and reading cues, labeling multiple definitions by source.
func groupReadingGuidance(group Group) string {
	blocks := []string{}
	for _, guidance := range group.EffectiveGuidance {
		metadata := guidance.Metadata
		text := "**Description:** " + escapeText(metadata.Description) + "\n\n**When to read this group:** " + escapeText(metadata.WhenToRead)
		if len(group.EffectiveGuidance) > 1 {
			text = "**" + escapeText(guidance.Source) + ": " + escapeText(metadata.Name) + ":**\n\n" + text
		}
		blocks = append(blocks, text)
	}
	return strings.Join(blocks, "\n\n")
}
