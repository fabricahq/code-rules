import { posix } from 'node:path';
import { fromMarkdown } from 'mdast-util-from-markdown';
import { toMarkdown } from 'mdast-util-to-markdown';
import type { Root, RootContent } from 'mdast';
import type { ActiveRule, Origin } from './types';
import { invalid } from './validation';

export function escapeText(value: string): string {
  return value
    .replace(/[\\`*_{}\[\]()<>#+|]/gu, '\\$&')
    .replace(/\r?\n/gu, ' ');
}

export function encodedPath(value: string): string {
  return value.split('/').map(encodeURIComponent).join('/');
}

export function sourceLink(origin: Origin): string {
  if (origin.repository !== null && origin.resolvedCommit !== null) {
    return `https://github.com/${origin.repository}/blob/${origin.resolvedCommit}/${encodedPath(origin.file)}`;
  }
  return `../../local/${encodedPath(origin.file)}`;
}

function relocatedUrl(url: string, active: ActiveRule, image: boolean): string {
  // External references and schemes retain their author's meaning.
  if (/^[a-z][a-z0-9+.-]*:/iu.test(url) || url.startsWith('//')) return url;
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
        ? '../../local'
        : `../../vendor/${active.origin.source}`;
    return `${base}/${encodedPath(target)}${suffix}`;
  }
  if (
    active.origin.repository !== null &&
    active.origin.resolvedCommit !== null
  ) {
    const base = image
      ? `https://raw.githubusercontent.com/${active.origin.repository}/${active.origin.resolvedCommit}`
      : `https://github.com/${active.origin.repository}/blob/${active.origin.resolvedCommit}`;
    return `${base}/${encodedPath(target)}${suffix}`;
  }
  return invalid(active.rule.id, `missing local link destination: ${url}`);
}

/** Change link destinations by position so unrelated Markdown stays byte-for-byte intact. */
export function ruleBody(active: ActiveRule): string {
  const tree = fromMarkdown(active.rule.body);
  const edits: Array<{ start: number; end: number; value: string }> = [];
  const referencePrefix = `code-rules-${encodeURIComponent(active.rule.id)}-`;
  function visit(node: Root | RootContent, captured = false): void {
    const changed =
      node.type === 'link' ||
      node.type === 'image' ||
      node.type === 'definition' ||
      node.type === 'linkReference' ||
      node.type === 'imageReference';
    if ('children' in node)
      for (const child of node.children) visit(child, captured || changed);
    if (
      node.type === 'link' ||
      node.type === 'image' ||
      node.type === 'definition'
    ) {
      node.url = relocatedUrl(node.url, active, node.type === 'image');
    }
    if (
      node.type === 'definition' ||
      node.type === 'linkReference' ||
      node.type === 'imageReference'
    ) {
      node.identifier = referencePrefix + node.identifier;
      node.label = node.identifier;
      if (node.type !== 'definition') node.referenceType = 'full';
    }
    if (changed && !captured) {
      const start = node.position?.start.offset;
      const end = node.position?.end.offset;
      if (start !== undefined && end !== undefined) {
        // Capture a whole link so nested images are rewritten without overlapping edits.
        if (
          node.type === 'link' ||
          node.type === 'image' ||
          node.type === 'definition' ||
          node.type === 'linkReference' ||
          node.type === 'imageReference'
        ) {
          const content: RootContent =
            node.type === 'definition'
              ? node
              : { type: 'paragraph', children: [node] };
          edits.push({
            start,
            end,
            value: toMarkdown({ type: 'root', children: [content] }).trimEnd(),
          });
        }
      }
    }
    if (
      node.type === 'html' &&
      /\b(?:href|src)\s*=\s*["'](?![a-z][a-z0-9+.-]*:|\/\/)/iu.test(node.value)
    ) {
      invalid(
        active.rule.id,
        'use Markdown links for relative references so Builds can relocate them',
      );
    }
  }
  visit(tree);
  const first = tree.children[0];
  if (
    first?.type === 'heading' &&
    first.children.every((child) => child.type === 'text') &&
    first.children
      .map((child) => (child.type === 'text' ? child.value : ''))
      .join('') === active.rule.title
  ) {
    const start = first.position?.start.offset;
    const end = first.position?.end.offset;
    if (start !== undefined && end !== undefined)
      edits.push({ start, end, value: '' });
  }
  let result = active.rule.body;
  for (const edit of edits.sort((a, b) => b.start - a.start)) {
    result = result.slice(0, edit.start) + edit.value + result.slice(edit.end);
  }
  return result.trim();
}

export function renderRule(active: ActiveRule): string {
  const { rule, origin, upstream } = active;
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
    `## ${escapeText(rule.title)}`,
    '',
    `Rule ID: \`${rule.id}\``,
    '',
    `[Active definition](${sourceLink(origin)})`,
  ];
  if (upstream !== null)
    lines.push(
      '',
      `[Replaces upstream definition](${sourceLink(upstream)}). Reason: ${escapeText(active.reason ?? '')}`,
    );
  if (active.licenseFiles.length) {
    lines.push(
      '',
      'Library default license and notices:',
      ...active.licenseFiles.map(
        (path) =>
          `- [${escapeText(path)}](../../vendor/${origin.source}/${encodedPath(path)})`,
      ),
    );
  } else
    lines.push(
      '',
      origin.source === 'local'
        ? 'Project-authored definition; retain any terms and attribution stated below.'
        : 'Library license: unspecified. Retain any rule-specific terms and attribution stated below.',
    );
  lines.push(
    '',
    `${metadataFence}yaml\n${rule.metadata}\n${metadataFence}`,
    '',
    ruleBody(active),
  );
  return lines.join('\n');
}
