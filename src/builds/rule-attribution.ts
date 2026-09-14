/** @fileoverview Parses external-source attribution without treating citations as license grants. */

import type { Attribution } from './types';
import { field, invalid, nonempty, object } from './validation';

/** Read explicit external-source citations for adaptations; these are declarations, not authenticated origins or license grants. */
export function ruleAttribution(
  value: unknown,
  location: string,
): ReadonlyArray<Attribution> {
  if (value === undefined) return [];
  if (!Array.isArray(value))
    return invalid(location, 'expected an attribution array');
  const entries: ReadonlyArray<unknown> = value;
  return entries.map((entry, index) => {
    const where = `${location}[${index}]`;
    const data = object(entry, where);
    const url = nonempty(field(data, 'url'), `${where}.url`);
    let parsed: URL;
    try {
      parsed = new URL(url);
    } catch {
      return invalid(`${where}.url`, 'expected an absolute HTTP(S) URL');
    }
    if (
      !['http:', 'https:'].includes(parsed.protocol) ||
      parsed.username ||
      parsed.password
    )
      return invalid(
        `${where}.url`,
        'expected an absolute HTTP(S) URL without credentials',
      );
    return {
      url: parsed.href,
      description: nonempty(field(data, 'description'), `${where}.description`),
    };
  });
}
