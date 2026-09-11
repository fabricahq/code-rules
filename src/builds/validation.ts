/** @fileoverview Provides shared input guards, path checks, and group metadata parsing for Builds. */

import type { FileContents, GroupMetadata } from './types';

/** Identifies invalid input or a snapshot requiring sync; the message includes the failing location. */
export class BuildError extends Error {
  constructor(
    readonly code: 'invalid-input' | 'needs-sync',
    message: string,
  ) {
    super(message);
    this.name = 'BuildError';
  }
}

/** Throw an invalid-input BuildError with the location prefixed to the supplied message. */
export function invalid(location: string, message: string): never {
  throw new BuildError('invalid-input', `${location}: ${message}`);
}

/** Copy own enumerable properties from a non-null, non-array object, or throw an invalid-input BuildError. */
export function object(
  value: unknown,
  location: string,
): Record<string, unknown> {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return invalid(location, 'expected an object');
  }
  return Object.fromEntries(Object.entries(value));
}

/** Return an own property value, or undefined when the property is absent or inherited. */
export function field(value: Record<string, unknown>, name: string): unknown {
  return Object.hasOwn(value, name) ? value[name] : undefined;
}

/** Return non-blank text unchanged, including surrounding whitespace; throw an invalid-input BuildError otherwise. */
export function nonempty(value: unknown, location: string): string {
  if (typeof value !== 'string' || value.trim() === '')
    return invalid(location, 'expected nonempty text');
  return value;
}

/** Return unique, non-blank strings in input order; throw an invalid-input BuildError for invalid or duplicate entries. */
export function strings(value: unknown, location: string): Array<string> {
  if (!Array.isArray(value))
    return invalid(location, 'expected an array of strings');
  const entries: Array<unknown> = value;
  const result = entries.map((item, index) =>
    nonempty(item, `${location}[${index}]`),
  );
  if (new Set(result).size !== result.length)
    return invalid(location, 'duplicate entries');
  return result;
}

/** Return a contained relative path unchanged; reject absolute paths, empty/dot segments, backslashes, and control characters with BuildError. */
export function relativePath(value: string, location: string): string {
  if (
    value.startsWith('/') ||
    /[\\\x00-\x1f\x7f]/u.test(value) ||
    value.split('/').some((part) => !part || part === '.' || part === '..') ||
    /^[A-Za-z]:/u.test(value)
  ) {
    return invalid(
      location,
      `expected a contained relative path, got ${JSON.stringify(value)}`,
    );
  }
  return value;
}

/** Return a techs/<slug> or practices/<slug> ID unchanged, or throw an invalid-input BuildError. */
export function groupId(value: string, location: string): string {
  if (!/^(techs|practices)\/[a-z][a-z0-9-]*$/u.test(value))
    return invalid(location, `invalid group ID ${JSON.stringify(value)}`);
  return value;
}

/** Return the group ID from a contained Markdown rule path, or throw an invalid-input BuildError. */
export function ruleGroup(path: string, location: string): string {
  relativePath(path, location);
  const parts = path.split('/');
  if (
    parts.length < 3 ||
    !path.endsWith('.md') ||
    parts.slice(2).some((part) => !/^[a-z0-9][a-z0-9.-]*$/u.test(part))
  ) {
    return invalid(location, `invalid rule path ${JSON.stringify(path)}`);
  }
  return groupId(parts.slice(0, 2).join('/'), location);
}

/** Parse JSON as unknown, wrapping syntax failures in an invalid-input BuildError at the supplied location. */
export function json(text: string, location: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return invalid(location, 'invalid JSON');
  }
}

/** Return a path-sorted map of text files; reject invalid paths or non-string contents with BuildError. */
export function files(
  input: FileContents,
  location: string,
): ReadonlyMap<string, string> {
  const entries = Object.entries(object(input, location));
  return new Map(
    entries
      .sort(([a], [b]) => compare(a, b))
      .map(([path, content]) => {
        relativePath(path, location);
        if (typeof content !== 'string')
          return invalid(`${location}/${path}`, 'expected UTF-8 text');
        return [path, content];
      }),
  );
}

/** Return the file text, including empty text; throw an invalid-input BuildError when the path is absent. */
export function requiredFile(
  input: ReadonlyMap<string, string>,
  path: string,
  location: string,
): string {
  const text = input.get(path);
  if (text === undefined)
    return invalid(`${location}/${path}`, 'missing required file');
  return text;
}

/** Compare strings case-sensitively by UTF-16 code units, returning -1, 0, or 1 without locale collation. */
export function compare(a: string, b: string): number {
  return a < b ? -1 : a > b ? 1 : 0;
}

/** Parse group selection guidance, preserving when-to-read order; throw BuildError for invalid JSON or required fields. */
export function groupMetadata(text: string, location: string): GroupMetadata {
  const data = object(json(text, location), location);
  return {
    name: nonempty(field(data, 'name'), `${location}.name`),
    description: nonempty(
      field(data, 'description'),
      `${location}.description`,
    ),
    whenToRead: strings(field(data, 'whenToRead'), `${location}.whenToRead`),
  };
}
