/** @fileoverview Renders active rules with source-aware links for individual generated files. */

import { posix } from 'node:path';
import { fromMarkdown } from 'mdast-util-from-markdown';
import { toMarkdown } from 'mdast-util-to-markdown';
import type { Root, RootContent } from 'mdast';
import type { ActiveRule, RuleOrigin } from './types';
import { invalid } from './validation';
import { repositoryFileUrl } from './repository';

/** Escape Markdown punctuation and flatten LF/CRLF line breaks so metadata renders as inline text. */
export function escapeText(value: string): string {
  return value
    .replace(/[\\`*_{}\[\]()<>#+|]/gu, '\\$&')
    .replace(/\r?\n/gu, ' ');
}

/** Percent-encode each path segment while preserving slash separators. */
function encodedPath(value: string): string {
  return value.split('/').map(encodeURIComponent).join('/');
}

/** Link to a known host's pinned source, otherwise to the retained definition relative to the generated file. */
function sourceLink(origin: RuleOrigin, outputPath: string): string {
  if (origin.repository !== null && origin.resolvedCommit !== null) {
    const remote = repositoryFileUrl(
      origin.repository,
      origin.resolvedCommit,
      origin.file,
      false,
    );
    if (remote !== null) return remote;
  }
  const root = origin.source === 'local' ? 'local' : `vendor/${origin.source}`;
  return workspaceLink(outputPath, `${root}/${origin.file}`);
}

/** Return a workspace-relative link from the actual generated file location. */
function workspaceLink(outputPath: string, target: string): string {
  return encodedPath(
    posix.relative(posix.dirname(`generated/${outputPath}`), target),
  );
}

/** Resolve relative references from the rule's source root, using vendored files or pinned remote URLs; missing local targets fail. */
function relocatedUrl(
  url: string,
  active: ActiveRule,
  image: boolean,
  outputPath: string,
): string {
  // External references and schemes retain their author's meaning.
  if (
    /^[a-z][a-z0-9+.-]*:/iu.test(url) ||
    url.startsWith('//') ||
    url.startsWith('#')
  )
    return url;
  const split = /^([^?#]*)([\s\S]*)$/u.exec(url);
  const pathname = split?.[1] ?? '';
  const suffix = split?.[2] ?? '';
  let decoded: string;
  try {
    decoded = decodeURIComponent(pathname);
  } catch {
    return invalid(active.rule.id, `invalid encoded link ${url}`);
  }
  if (/[\\\x00-\x1f]/u.test(decoded))
    return invalid(active.rule.id, `unsafe relative link ${url}`);
  const target =
    decoded === ''
      ? active.origin.file
      : posix.normalize(
          decoded.startsWith('/')
            ? decoded.slice(1)
            : posix.join(posix.dirname(active.origin.file), decoded),
        );
  if (target === '..' || target.startsWith('../'))
    return invalid(active.rule.id, `link escapes source root: ${url}`);
  if (active.sourceFiles.has(target)) {
    const base =
      active.origin.source === 'local'
        ? 'local'
        : `vendor/${active.origin.source}`;
    return `${workspaceLink(outputPath, `${base}/${target}`)}${suffix}`;
  }
  if (
    active.origin.repository !== null &&
    active.origin.resolvedCommit !== null
  ) {
    const remote = repositoryFileUrl(
      active.origin.repository,
      active.origin.resolvedCommit,
      target,
      image,
    );
    if (remote !== null) return `${remote}${suffix}`;
    return invalid(
      active.rule.id,
      `missing retained link destination: ${url}; include the target in the source snapshot or author an explicit URL for this host`,
    );
  }
  return invalid(active.rule.id, `missing local link destination: ${url}`);
}

/** A replacement span measured against the untouched source Markdown. */
type SourceEdit = {
  readonly start: number;
  readonly end: number;
  readonly value: string;
};
/** Markdown nodes whose URLs or reference labels need source-aware rewriting. */
type RewriteNode = Extract<
  RootContent,
  { type: 'link' | 'image' | 'definition' | 'linkReference' | 'imageReference' }
>;

/** Identify nodes requiring rewriting, including their nested content when serialized. */
function requiresRewrite(node: Root | RootContent): node is RewriteNode {
  return (
    node.type === 'link' ||
    node.type === 'image' ||
    node.type === 'definition' ||
    node.type === 'linkReference' ||
    node.type === 'imageReference'
  );
}

/** Mutate a parsed node's URL and reference identity; source Markdown is never changed here. */
function rewriteNode(
  node: RewriteNode,
  active: ActiveRule,
  referencePrefix: string,
  outputPath: string,
  standalonePath: string,
): void {
  if (
    node.type === 'link' ||
    node.type === 'image' ||
    node.type === 'definition'
  )
    node.url =
      node.url.startsWith('#') && outputPath !== standalonePath
        ? `${encodedPath(posix.relative(posix.dirname(outputPath), standalonePath))}${node.url}`
        : relocatedUrl(node.url, active, node.type === 'image', outputPath);
  if (
    node.type === 'definition' ||
    node.type === 'linkReference' ||
    node.type === 'imageReference'
  ) {
    node.identifier = referencePrefix + node.identifier;
    node.label = node.identifier;
    if (node.type !== 'definition') node.referenceType = 'full';
  }
}

/** Serialize one rewritten node with its children, omitting the serializer's trailing newline. */
function serializedNode(node: RewriteNode): string {
  const content: RootContent =
    node.type === 'definition' ? node : { type: 'paragraph', children: [node] };
  return toMarkdown({ type: 'root', children: [content] }).trimEnd();
}

/** Create an edit only when the parser supplied both offsets into the original source. */
function sourceEdit(node: RootContent, value: string): SourceEdit | undefined {
  const start = node.position?.start.offset;
  const end = node.position?.end.offset;
  return start === undefined || end === undefined
    ? undefined
    : { start, end, value };
}

/** Reject relative raw-HTML references, which cannot be safely relocated as Markdown nodes. */
function requireSupportedHtml(node: Root | RootContent, ruleId: string): void {
  if (
    node.type === 'html' &&
    /\b(?:href|src)\s*=\s*["'](?![a-z][a-z0-9+.-]*:|\/\/)/iu.test(node.value)
  )
    invalid(
      ruleId,
      'use Markdown links for relative references so Builds can relocate them',
    );
}

/** Rewrite children before parents and collect only outermost replacement spans; invalid references throw BuildError. */
function collectRewrites(
  tree: Root,
  active: ActiveRule,
  outputPath: string,
  standalonePath: string,
): Array<SourceEdit> {
  const edits: Array<SourceEdit> = [];
  // Keep reference identities stable if a consumer later combines individual rule files.
  const referencePrefix = `code-rules-${encodeURIComponent(active.rule.id)}-`;
  // Nest source sections below Guidance, retaining their relative hierarchy.
  const bodyHeadingDepth = outputPath === standalonePath ? 3 : 5;
  const headingOffset =
    bodyHeadingDepth -
    Math.min(
      bodyHeadingDepth,
      ...tree.children.flatMap((node) =>
        node.type === 'heading' ? [node.depth] : [],
      ),
    );
  /** Visit in postorder; an ancestor's edit includes rewritten children, so child edits must not overlap it. */
  function visit(node: Root | RootContent, ancestorOwnsEdit: boolean): void {
    const rewritesNode = requiresRewrite(node);
    const embedsHeading = node.type === 'heading' && headingOffset > 0;
    if ('children' in node)
      for (const child of node.children)
        visit(child, ancestorOwnsEdit || rewritesNode || embedsHeading);
    if (rewritesNode) {
      rewriteNode(node, active, referencePrefix, outputPath, standalonePath);
      if (!ancestorOwnsEdit) {
        const edit = sourceEdit(node, serializedNode(node));
        if (edit !== undefined) edits.push(edit);
      }
    }
    if (node.type === 'heading' && embedsHeading) {
      for (let shift = 0; shift < headingOffset && node.depth < 6; shift += 1)
        node.depth += 1;
      if (!ancestorOwnsEdit) {
        const edit = sourceEdit(
          node,
          toMarkdown({ type: 'root', children: [node] }).trimEnd(),
        );
        if (edit !== undefined) edits.push(edit);
      }
    }
    requireSupportedHtml(node, active.rule.id);
  }
  visit(tree, false);
  return edits;
}

/** Remove a first heading only when its plain text exactly matches the rule title. */
function redundantTitleEdit(tree: Root, title: string): SourceEdit | undefined {
  const first = tree.children[0];
  if (
    first?.type !== 'heading' ||
    !first.children.every((child) => child.type === 'text')
  )
    return undefined;
  const text = first.children
    .map((child) => (child.type === 'text' ? child.value : ''))
    .join('');
  return text === title ? sourceEdit(first, '') : undefined;
}

/** Apply non-overlapping source spans right to left, preserving all untouched text and trimming outer whitespace. */
function applySourceEdits(
  body: string,
  edits: ReadonlyArray<SourceEdit>,
): string {
  let result = body;
  for (const edit of [...edits].sort((a, b) => b.start - a.start))
    result = result.slice(0, edit.start) + edit.value + result.slice(edit.end);
  return result.trim();
}

/**
 * Return a body with relocated links, namespaced references, and a matching leading title removed.
 * Trims outer whitespace while preserving text outside the rewritten nodes.
 * Throws an invalid-input BuildError for unsafe/missing local targets or relative links in raw HTML.
 */
function ruleBody(
  active: ActiveRule,
  outputPath: string,
  standalonePath: string,
): string {
  const tree = fromMarkdown(active.rule.body);
  const titleEdit = redundantTitleEdit(tree, active.rule.title);
  // Remove the matching title before rewriting headings so source spans cannot overlap.
  if (titleEdit !== undefined) tree.children.shift();
  const edits = collectRewrites(tree, active, outputPath, standalonePath);
  if (titleEdit !== undefined) edits.push(titleEdit);
  return applySourceEdits(active.rule.body, edits);
}

/**
 * Return rule guidance followed by source, replacement, terms, and preserved metadata.
 * Relocates body links and throws an invalid-input BuildError for unsupported relative references.
 * When embedded at another path, nests headings and directs same-file fragments to the standalone definition to avoid cross-rule anchor collisions.
 */
export function renderRule(
  active: ActiveRule,
  outputPath: string,
  standalonePath = outputPath,
): string {
  const { rule, origin, upstream } = active;
  const titleHeading = outputPath === standalonePath ? '#' : '###';
  const sectionHeading = `${titleHeading}#`;
  // The outer fence must exceed every embedded backtick run so metadata cannot close it early.
  const metadataFence = '`'.repeat(
    Math.max(
      3,
      ...Array.from(
        rule.metadata.matchAll(/`+/gu),
        (match) => match[0].length + 1,
      ),
    ),
  );
  const lines = [
    `${titleHeading} ${escapeText(rule.title)}`,
    '',
    `Rule ID: \`${rule.id}\``,
    '',
    `**When to read:** ${escapeText(rule.whenToRead)}`,
    '',
    `**Impact:** ${escapeText(rule.impact)}`,
    '',
    `**Why it matters:** ${escapeText(rule.impactDescription)}`,
    '',
    `${sectionHeading} Guidance`,
    '',
    ruleBody(active, outputPath, standalonePath),
    '',
    `${sectionHeading} Source and attribution`,
    '',
    `**Rule source:** [Original rule](${sourceLink(origin, outputPath)})`,
  ];
  if (outputPath !== standalonePath)
    lines.push(
      '',
      `**Separate rule file:** [${escapeText(rule.title)}](${encodedPath(posix.relative(posix.dirname(outputPath), standalonePath))})`,
    );
  if (upstream !== null)
    lines.push(
      '',
      `[Replaces upstream definition](${sourceLink(upstream, outputPath)}). Reason: ${escapeText(active.reason ?? '')}`,
    );
  for (const attribution of rule.attribution) {
    lines.push(
      '',
      `**Attribution:** [${escapeText(attribution.description)}](<${attribution.url.replaceAll('>', '%3E').replaceAll('<', '%3C')}>)`,
    );
  }
  if (active.licenses.length) {
    lines.push(
      '',
      rule.licenses === null
        ? 'Library default license and notices:'
        : 'Rule-specific licenses and notices:',
    );
    const sourceRoot =
      origin.source === 'local' ? 'local' : `vendor/${origin.source}`;
    for (const license of active.licenses) {
      if (license.expression !== null)
        lines.push(
          '',
          `**Declared license:** ${escapeText(license.expression)}`,
          '',
        );
      for (const path of [...license.files, ...license.attributionFiles])
        lines.push(
          `- [${escapeText(path)}](${workspaceLink(outputPath, `${sourceRoot}/${path}`)})`,
        );
    }
  }
  lines.push(
    '',
    `${sectionHeading}# Source metadata`,
    '',
    `${metadataFence}yaml\n${rule.metadata}\n${metadataFence}`,
  );
  return lines.join('\n');
}
