/** @fileoverview Validates Builds input shapes, rule metadata, and paths before aggregation. */

import { parseDocument } from 'yaml';
import type {
  Configuration,
  FileContents,
  GroupMetadata,
  Replacement,
  Rule,
  Source,
} from './types';

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

function knownFields(
  value: Record<string, unknown>,
  allowed: ReadonlyArray<string>,
  location: string,
): void {
  for (const name of Object.keys(value)) {
    if (!allowed.includes(name)) invalid(location, `unknown field ${name}`);
  }
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

/**
 * Return validated configuration with sources, groups, and exception keys sorted.
 * Throws an invalid-input BuildError for unsupported fields, malformed values, or conflicting declarations.
 * Rejects abbreviated-commit-shaped refs unless explicitly qualified as refs/tags/<name>.
 * This offline syntax check does not resolve refs or establish their existence in Git.
 */
export function configuration(input: unknown): Configuration {
  const config = object(input, 'configuration');
  knownFields(
    config,
    ['schemaVersion', 'sources', 'localGroups'],
    'configuration',
  );
  if (field(config, 'schemaVersion') !== 1)
    return invalid('schemaVersion', 'only version 1 is supported');
  const repositories = new Set<string>();
  const sources: Array<Source> = Object.entries(
    object(field(config, 'sources'), 'sources'),
  )
    .sort(([a], [b]) => compare(a, b))
    .map(([name, raw]) => {
      const where = `sources.${name}`;
      if (!/^[a-z][a-z0-9-]*$/u.test(name) || name === 'local')
        return invalid(where, 'invalid or reserved source name');
      const source = object(raw, where);
      knownFields(
        source,
        ['repository', 'ref', 'groups', 'exclude', 'replace'],
        where,
      );
      const repository = nonempty(
        field(source, 'repository'),
        `${where}.repository`,
      );
      if (
        !/^[A-Za-z0-9][A-Za-z0-9-]*\/[A-Za-z0-9_.-]+$/u.test(repository) ||
        /\/(\.|\.\.)$/u.test(repository)
      )
        return invalid(where, 'repository must use owner/name form');
      if (repositories.has(repository.toLowerCase()))
        return invalid(
          where,
          `repository ${repository} is declared more than once`,
        );
      repositories.add(repository.toLowerCase());
      const ref = nonempty(field(source, 'ref'), `${where}.ref`);
      if (
        /\s|[~^:?*\[\\]|\.\.|@\{|\/$|\.$|\.lock$|^refs\/(?!tags\/)/u.test(
          ref,
        ) ||
        ref === '@' ||
        ref.startsWith('-') ||
        ref.endsWith('/') ||
        ref === 'refs/tags/'
      )
        return invalid(
          `${where}.ref`,
          'expected a full commit SHA or exact tag name',
        );
      if (/^[a-f0-9]{4,39}$/iu.test(ref))
        return invalid(
          `${where}.ref`,
          'abbreviated commits are unsupported; use a full SHA or refs/tags/<name>',
        );
      const groups = strings(field(source, 'groups'), `${where}.groups`)
        .map((id) => groupId(id, where))
        .sort(compare);
      const exclude = new Map(
        Object.entries(object(field(source, 'exclude'), `${where}.exclude`))
          .sort(([a], [b]) => compare(a, b))
          .map(([id, reason]) => [
            id,
            nonempty(reason, `${where}.exclude.${id}`),
          ]),
      );
      const replace = new Map<string, Replacement>(
        Object.entries(object(field(source, 'replace'), `${where}.replace`))
          .sort(([a], [b]) => compare(a, b))
          .map(([id, rawReplacement]) => {
            const replacement = object(
              rawReplacement,
              `${where}.replace.${id}`,
            );
            knownFields(
              replacement,
              ['file', 'reason'],
              `${where}.replace.${id}`,
            );
            const path = relativePath(
              nonempty(
                field(replacement, 'file'),
                `${where}.replace.${id}.file`,
              ),
              where,
            );
            if (!path.startsWith('local/'))
              return invalid(where, 'replacement files must be under local/');
            return [
              id,
              {
                file: path,
                reason: nonempty(
                  field(replacement, 'reason'),
                  `${where}.replace.${id}.reason`,
                ),
              },
            ];
          }),
      );
      for (const id of [...exclude.keys(), ...replace.keys()]) {
        ruleGroup(`${id}.md`, where);
        if (exclude.has(id) && replace.has(id))
          return invalid(
            `${where}:${id}`,
            'rule is both excluded and replaced',
          );
      }
      return { name, repository, ref, groups, exclude, replace };
    });
  const localGroups = strings(field(config, 'localGroups'), 'localGroups')
    .map((id) => groupId(id, 'localGroups'))
    .sort(compare);
  for (const id of localGroups) {
    if (sources.some((source) => source.groups.includes(id)))
      return invalid(
        id,
        'an imported group cannot also be declared in localGroups',
      );
  }
  return { sources, localGroups };
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

/**
 * Parse a rule into its source-qualified identity, metadata text, and Markdown body.
 * Preserves extra frontmatter fields and body whitespace; requires supported impact and non-blank content.
 * Throws an invalid-input BuildError for invalid paths/frontmatter, duplicate YAML keys, or aliases.
 */
export function rule(text: string, path: string, source: string): Rule {
  const location = `${source}:${path}`;
  const group = ruleGroup(path, location);
  const match = /^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)([\s\S]*)$/u.exec(text);
  const metadata = match?.[1];
  const body = match?.[2];
  if (metadata === undefined || body === undefined)
    return invalid(location, 'expected YAML frontmatter followed by Markdown');
  const parsed = parseDocument(metadata, { uniqueKeys: true });
  if (parsed.errors.length)
    return invalid(location, `invalid YAML: ${parsed.errors[0]?.message}`);
  let raw: unknown;
  try {
    raw = parsed.toJS({ maxAliasCount: 0 });
  } catch {
    return invalid(location, 'YAML aliases are unsupported');
  }
  const data = object(raw, location);
  const title = nonempty(field(data, 'title'), `${location}.title`);
  const impact = nonempty(field(data, 'impact'), `${location}.impact`);
  if (
    ![
      'CRITICAL',
      'HIGH',
      'MEDIUM-HIGH',
      'MEDIUM',
      'LOW-MEDIUM',
      'LOW',
    ].includes(impact)
  )
    return invalid(location, `unknown impact ${impact}`);
  nonempty(field(data, 'impactDescription'), `${location}.impactDescription`);
  const tags = field(data, 'tags');
  if (typeof tags === 'string') nonempty(tags, `${location}.tags`);
  else strings(tags, `${location}.tags`);
  nonempty(body, `${location}.body`);
  return {
    id: `${source}:${path.slice(0, -3)}`,
    group,
    path,
    title,
    metadata,
    body,
  };
}
