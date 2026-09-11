/** @fileoverview Resolves library-relative Markdown references consistently for importing and rendering. */

import { posix } from 'node:path';
import { fromMarkdown } from 'mdast-util-from-markdown';
import type { Root, RootContent } from 'mdast';
import { invalid } from './validation';

/** A contained destination with the author's query and fragment retained verbatim. */
export type RelativeTarget = {
  readonly target: string;
  readonly suffix: string;
};

/** Resolve a local reference within its source root; external URLs return null and unsafe paths fail. */
export function relativeTarget(
  url: string,
  file: string,
  location: string,
): RelativeTarget | null {
  if (/^[a-z][a-z0-9+.-]*:/iu.test(url) || url.startsWith('//')) return null;
  const split = /^([^?#]*)([\s\S]*)$/u.exec(url);
  let decoded: string;
  try {
    decoded = decodeURIComponent(split?.[1] ?? '');
  } catch {
    return invalid(location, `invalid encoded link ${url}`);
  }
  if (/[\\\x00-\x1f]/u.test(decoded))
    return invalid(location, `unsafe relative link ${url}`);
  const target =
    decoded === ''
      ? file
      : posix.normalize(
          decoded.startsWith('/')
            ? decoded.slice(1)
            : posix.join(posix.dirname(file), decoded),
        );
  if (target === '..' || target.startsWith('../'))
    return invalid(location, `link escapes source root: ${url}`);
  return { target, suffix: split?.[2] ?? '' };
}

/** Collect standard local Markdown destinations without interpreting code fences or remote URLs. */
export function markdownTargets(
  text: string,
  file: string,
): ReadonlyArray<string> {
  const targets = new Set<string>();
  /** Visit URL-bearing nodes, including definitions used by reference-style links. */
  function visit(node: Root | RootContent): void {
    if (
      node.type === 'link' ||
      node.type === 'image' ||
      node.type === 'definition'
    ) {
      const link = relativeTarget(node.url, file, file);
      if (link !== null) targets.add(link.target);
    }
    if ('children' in node) for (const child of node.children) visit(child);
  }
  // Frontmatter is metadata, not a Markdown paragraph containing dependencies.
  const body = text.replace(/^\uFEFF?---\r?\n[\s\S]*?\r?\n---(?:\r?\n|$)/u, '');
  visit(fromMarkdown(body));
  return [...targets].sort();
}
