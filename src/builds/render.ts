/** @fileoverview Renders resolved rules as applicability indexes, individual effective definitions, and provenance. */

import { posix } from 'node:path';
import type { ActiveRule, BuildOutput, Group, ResolvedRules } from './types';
import { compare, invalid } from './validation';
import { escapeText, renderRule } from './markdown';
import { indexPages } from './index-pages';

/** Map source-qualified identity to a portable path without using the replacement's local filename. */
function rulePath(active: ActiveRule): string {
  return `rules/${active.rule.id.replace(':', '/')}.md`;
}

/** Describe all sources contributing selection guidance to a group without merging their policies. */
function groupEntry(group: Group): string {
  const sections = [`### [${group.id}](${group.id}.md)`];
  for (const guidance of group.guidance) {
    sections.push(
      `**${guidance.source}: ${escapeText(guidance.metadata.name)}**`,
      escapeText(guidance.metadata.description),
      '**When to read this group:**',
      ...guidance.metadata.whenToRead.map(
        (reason) => `- ${escapeText(reason)}`,
      ),
    );
  }
  sections.push(
    "**Next:** Open this group's index to select relevant rules and read their full guidance.",
  );
  return sections.join('\n\n');
}

/** Give agents the group-selection procedure without including full rule bodies. */
function indexHeader(): string {
  return [
    '# Code Rules',
    'This project uses [Fabrica Code Rules](https://github.com/fabricahq/code-rules) to declare its adopted engineering practices.',
    'Before planning or writing code, use the descriptions under **Technology and practice group indexes** below to choose which indexes to open. Consider the intended behavior as well as the technology; testing guidance can apply even when no test files have changed.',
    'Each group index lists every active rule with its when-to-read guidance and a link to the full effective definition. Exclusions and replacements are already applied.',
    'Read the full text of every applicable or plausibly applicable rule before relying on it. Complete truncated reads. Revisit selection when scope changes and reload needed rules after compaction.',
    'During validation or diagnosis, independently select relevant rules from the task, code, and surrounding contracts. Cite rule IDs and concrete evidence for findings; selection alone is not evidence of a violation.',
    'These files are generated. Edit source rules or configuration and rebuild to change them.',
    '[Source versions and rule origins](provenance.json).',
    '## Technology and practice group indexes',
  ].join('\n\n');
}

/** Render a group's reading instructions and links back to project-wide selection guidance. */
function groupHeader(group: Group): string {
  const title = [
    ...new Set(group.guidance.map(({ metadata }) => metadata.name)),
  ]
    .sort(compare)
    .map(escapeText)
    .join(' / ');
  return [
    `# ${title}`,
    `Group ID: \`${group.id}\``,
    'This generated index lists the active rules in this group. Read each relevant or plausibly relevant full rule before implementation, validation, or diagnosis. The description helps selection; the full rule defines its obligation and exceptions.',
    'For other technology and practice groups, read [RULES.md](../RULES.md). See [provenance.json](../provenance.json) for origins. Edit source rules and rebuild to change this index.',
  ].join('\n\n');
}

/** Render selection metadata and a relative link to this rule's effective definition. */
function ruleEntry(active: ActiveRule, indexPath: string): string {
  const link = posix.relative(posix.dirname(indexPath), rulePath(active));
  return [
    `## [${escapeText(active.rule.title)}](${link})`,
    `Rule ID: \`${active.rule.id}\``,
    `Impact: ${escapeText(active.rule.impact)}`,
    `When to read: ${escapeText(active.rule.whenToRead)}`,
  ].join('\n\n');
}

/** Serialize source revisions and active rule origins in stable order, deriving rules from their groups. */
function renderProvenance(
  resolved: ResolvedRules,
  toolVersion: string,
): string {
  const provenance = {
    toolVersion,
    sources: resolved.sources,
    rules: resolved.groups
      .flatMap((group) => group.rules)
      .sort((a, b) => compare(a.rule.id, b.rule.id))
      .map((active) => ({
        id: active.rule.id,
        group: active.rule.group,
        origin: active.origin,
        upstream: active.upstream,
        replacementReason: active.reason,
      })),
  };
  return `${JSON.stringify(provenance, null, 2)}\n`;
}

/** Return path-sorted generated files; invalid metadata, index budgets, or Markdown references throw BuildError. */
export function renderGeneratedFiles(
  resolved: ResolvedRules,
  toolVersion: string,
  indexMaxBytes: number,
): BuildOutput {
  const output = new Map(
    indexPages(
      'RULES.md',
      indexHeader(),
      resolved.groups.map(groupEntry),
      indexMaxBytes,
    ),
  );
  for (const group of resolved.groups) {
    const path = `${group.id}.md`;
    const rules = [...group.rules].sort((a, b) =>
      compare(a.rule.id, b.rule.id),
    );
    const entries = rules.length
      ? rules.map((active) => ruleEntry(active, path))
      : ['No active rules in this group.'];
    for (const [indexPath, content] of indexPages(
      path,
      groupHeader(group),
      entries,
      indexMaxBytes,
    ))
      output.set(indexPath, content);
    for (const active of rules) {
      const path = rulePath(active);
      output.set(path, `${renderRule(active, path)}\n`);
    }
  }
  output.set('provenance.json', renderProvenance(resolved, toolVersion));
  for (const path of output.keys()) {
    for (
      let parent = posix.dirname(path);
      parent !== '.';
      parent = posix.dirname(parent)
    ) {
      if (output.has(parent))
        invalid(path, `generated path conflicts with file ${parent}`);
    }
  }
  return {
    files: Object.fromEntries(
      [...output.entries()].sort(([a], [b]) => compare(a, b)),
    ),
  };
}
