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

/** Keep the discovery hierarchy separate from individual effective definitions. */
function groupPath(group: Group): string {
  return `groups/${group.id}.md`;
}

/** Keep all contributing group names visible in a stable, escaped heading. */
function groupTitle(group: Group): string {
  return [...new Set(group.guidance.map(({ metadata }) => metadata.name))]
    .sort(compare)
    .map(escapeText)
    .join(' / ');
}

/** Describe all sources contributing selection guidance to a group without merging their policies. */
function groupEntry(group: Group): string {
  const sections = [`### ${groupTitle(group)}`];
  for (const guidance of group.guidance) {
    if (group.guidance.length > 1)
      sections.push(
        `**${guidance.source}: ${escapeText(guidance.metadata.name)}**`,
      );
    sections.push(
      '**When to read:**',
      ...guidance.metadata.whenToRead.map(
        (reason) => `- ${escapeText(reason)}`,
      ),
    );
  }
  sections.push(`**Open group:** [${groupTitle(group)}](${groupPath(group)})`);
  return sections.join('\n\n');
}

/** Give agents the group-selection procedure without including full rule bodies. */
function indexHeader(): string {
  return [
    '# Code Rules',
    'This project uses [Fabrica Code Rules](https://github.com/fabricahq/code-rules) to declare its adopted engineering practices.',
    'Before planning or writing code, use the descriptions under **Technology and practice group indexes** below to choose which indexes to open. Consider the intended behavior as well as the technology; testing guidance can apply even when no test files have changed.',
    'Each group page includes full rules or summaries with explicit reading links. Exclusions and replacements are already applied.',
    'Read the full text of every applicable or plausibly applicable rule before relying on it. Complete truncated reads. Revisit selection when scope changes and reload needed rules after compaction.',
    'During validation or diagnosis, independently select relevant rules from the task, code, and surrounding contracts. Cite rule IDs and concrete evidence for findings; selection alone is not evidence of a violation.',
    '## Technology and practice group indexes',
    'Open the relevant group indexes below, then select applicable rules and read their full guidance.',
  ].join('\n\n');
}

/** Keep impact interpretation consistent across both group delivery modes. */
const impactGuidance =
  'Use “When to read” to select rules. Read and follow every applicable rule, regardless of impact. Impact describes the consequence the rule addresses; it does not determine applicability, override exceptions, or set a review finding’s severity. Assess findings from concrete evidence and consequences.';

const groupFooter =
  'For other technology and practice groups, open [RULES.md](../../RULES.md). See [provenance.json](../../provenance.json) for origins. These files are generated. Edit source rules or configuration and rebuild to change them.';

/** Identify the delivery mode and give agents an explicit reading procedure. */
function groupHeader(group: Group, mode: 'inline' | 'summaries'): string {
  return [
    `# ${groupTitle(group)}`,
    `Group ID: \`${group.id}\``,
    '## How to use this group',
    mode === 'inline'
      ? 'Full rules are included below. Read each relevant or plausibly relevant rule completely before planning, implementation, validation, or diagnosis. Separate rule files remain available for direct references.'
      : 'This file contains summaries only. Follow the reading instructions below to load the full rules.',
    '1. Compare each “When to read” cue with your intended task or the behavior you are reviewing.',
    mode === 'inline'
      ? '2. Read the complete guidance and exceptions for every relevant or plausibly relevant rule below. Complete truncated reads.'
      : '2. For every relevant or plausibly relevant rule, open its “Read full rule” link and read the complete file. Complete truncated reads.',
    '3. Apply the full rule’s guidance and exceptions. Selection alone is insufficient evidence for a review finding.',
    impactGuidance,
  ].join('\n\n');
}

/** Render selection metadata with a separate, explicit reading action. */
function ruleEntry(active: ActiveRule, indexPath: string): string {
  const link = posix.relative(posix.dirname(indexPath), rulePath(active));
  return [
    `## ${escapeText(active.rule.title)}`,
    `Rule ID: \`${active.rule.id}\``,
    `**When to read:** ${escapeText(active.rule.whenToRead)}`,
    `**Impact:** ${escapeText(active.rule.impact)}`,
    `**Why it matters:** ${escapeText(active.rule.impactDescription)}`,
    `**Read full rule:** [${escapeText(active.rule.title)}](${link})`,
  ].join('\n\n');
}

/** Inline only complete groups that fit both budgets; otherwise paginate summaries without truncating rules. */
function groupPages(
  group: Group,
  budgets: {
    readonly indexMaxBytes: number;
    readonly groupInlineMaxBytes: number;
  },
): ReadonlyMap<string, string> {
  const path = groupPath(group);
  const rules = [...group.rules].sort((a, b) => compare(a.rule.id, b.rule.id));
  if (budgets.groupInlineMaxBytes > 0 && rules.length > 0) {
    const inline = inlineGroupPage(
      group,
      rules,
      Math.min(budgets.groupInlineMaxBytes, budgets.indexMaxBytes),
    );
    if (inline !== null) return new Map([[path, inline]]);
  }
  return indexPages({
    path,
    header: groupHeader(group, 'summaries'),
    entries: rules.length
      ? rules.map((active) => ruleEntry(active, path))
      : ['No active rules in this group.'],
    maxBytes: budgets.indexMaxBytes,
    footer: groupFooter,
  });
}

/** Stop considering inline delivery as soon as the complete page cannot fit; full standalone rendering still validates every rule. */
function inlineGroupPage(
  group: Group,
  rules: ReadonlyArray<ActiveRule>,
  maxBytes: number,
): string | null {
  const sections = [groupHeader(group, 'inline')];
  let bytes = Buffer.byteLength(`${sections[0]}\n\n${groupFooter}\n`, 'utf8');
  for (const active of rules) {
    const section = renderRule(active, groupPath(group), rulePath(active));
    bytes += Buffer.byteLength(section, 'utf8') + 2;
    if (bytes > maxBytes) return null;
    sections.push(section);
  }
  return [...sections, groupFooter].join('\n\n') + '\n';
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
  budgets: {
    readonly indexMaxBytes: number;
    readonly groupInlineMaxBytes: number;
  },
): BuildOutput {
  const output = new Map(
    indexPages({
      path: 'RULES.md',
      header: indexHeader(),
      entries: resolved.groups.map(groupEntry),
      maxBytes: budgets.indexMaxBytes,
      footer:
        'These files are generated. Edit source rules or configuration and rebuild to change them. [Source versions and rule origins](provenance.json).',
    }),
  );
  for (const group of resolved.groups) {
    for (const [path, content] of groupPages(group, budgets))
      output.set(path, content);
    for (const active of group.rules) {
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
