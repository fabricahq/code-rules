/** @fileoverview Splits generated indexes at entry boundaries while preserving a complete, bounded part directory. */

import { posix } from 'node:path';
import { invalid } from './validation';

/** Render blocks with stable spacing and a final newline. */
function document(
  header: string,
  entries: ReadonlyArray<string>,
  footer: string,
): string {
  return [header, ...entries, footer].filter(Boolean).join('\n\n') + '\n';
}

/**
 * Return one index or a directory plus numbered sibling parts, each within the UTF-8 byte budget.
 * Never splits entries; throws BuildError if an entry, header, or complete part directory cannot fit.
 */
export function indexPages({
  path,
  header,
  entries,
  maxBytes,
  footer,
}: {
  readonly path: string;
  readonly header: string;
  readonly entries: ReadonlyArray<string>;
  readonly maxBytes: number;
  readonly footer: string;
}): ReadonlyMap<string, string> {
  const whole = document(header, entries, footer);
  if (Buffer.byteLength(whole, 'utf8') <= maxBytes)
    return new Map([[path, whole]]);

  const pages = new Map<string, string>();
  const partLinks: Array<string> = [];
  let pending: Array<string> = [];
  let part = 1;
  const partHeader = (): string =>
    `${header}\n\nPart ${part}. [All parts](${posix.basename(path)}).`;
  const fits = (items: ReadonlyArray<string>): boolean =>
    Buffer.byteLength(document(partHeader(), items, footer), 'utf8') <=
    maxBytes;
  const finishPart = (): void => {
    const partPath = `${path.slice(0, -3)}.part-${part}.md`;
    pages.set(partPath, document(partHeader(), pending, footer));
    partLinks.push(`- [Part ${part}](${posix.basename(partPath)})`);
    part += 1;
    pending = [];
  };

  for (const entry of entries) {
    if (!fits([...pending, entry]) && pending.length) finishPart();
    if (!fits([entry]))
      invalid(
        path,
        'an index entry and its reading instructions exceed indexMaxBytes; shorten the metadata or increase the budget',
      );
    pending.push(entry);
  }
  if (pending.length) finishPart();
  const directory = document(
    header,
    [
      'Read every numbered part to inspect this complete index. Rule bodies remain in their linked files.',
      ...partLinks,
    ],
    footer,
  );
  if (Buffer.byteLength(directory, 'utf8') > maxBytes)
    invalid(
      path,
      'the complete index part directory exceeds indexMaxBytes; increase the budget',
    );
  pages.set(path, directory);
  return pages;
}
