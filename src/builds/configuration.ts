/** @fileoverview Interprets project configuration and source-scoped exception policies in deterministic validation order. */

import type { ProjectConfig, RuleReplacement, LibrarySource } from './types';
import {
  compare,
  field,
  groupId,
  invalid,
  nonempty,
  object,
  relativePath,
  ruleGroup,
  strings,
} from './validation';

/** Reject unknown own fields before interpreting a configuration object. */
function knownFields(
  value: Record<string, unknown>,
  allowed: ReadonlyArray<string>,
  location: string,
): void {
  for (const name of Object.keys(value)) {
    if (!allowed.includes(name)) invalid(location, `unknown field ${name}`);
  }
}

/** Validate and register a repository before checking the source's later fields; duplicate names are case-insensitive. */
function sourceRepository(
  value: unknown,
  where: string,
  repositories: Set<string>,
): string {
  const repository = nonempty(value, `${where}.repository`);
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
  return repository;
}

/** Validate offline ref syntax without resolving it; reject abbreviated-commit-shaped bare tags. */
function sourceRef(value: unknown, where: string): string {
  const ref = nonempty(value, `${where}.ref`);
  if (
    /\s|[~^:?*\[\\]|\.\.|@\{|\/$|\.$|\.lock$|^refs\/(?!tags\/)/u.test(ref) ||
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
  return ref;
}

/** Read nonempty exclusion reasons in sorted rule-ID order. */
function exclusionDecisions(
  value: unknown,
  where: string,
): ReadonlyMap<string, string> {
  const exclude = new Map(
    Object.entries(object(value, `${where}.exclude`))
      .sort(([a], [b]) => compare(a, b))
      .map(([id, reason]) => [id, nonempty(reason, `${where}.exclude.${id}`)]),
  );
  return exclude;
}

/** Validate one replacement's local file and reason without resolving the referenced file. */
function replacementDeclaration(
  rawReplacement: unknown,
  id: string,
  where: string,
): RuleReplacement {
  const replacement = object(rawReplacement, `${where}.replace.${id}`);
  knownFields(replacement, ['file', 'reason'], `${where}.replace.${id}`);
  const location = `${where}.replace.${id}.file`;
  const text = nonempty(field(replacement, 'file'), location);
  const path = relativePath(text, location);
  if (!path.startsWith('local/'))
    return invalid(location, 'replacement files must be under local/');
  return {
    file: path,
    reason: nonempty(
      field(replacement, 'reason'),
      `${where}.replace.${id}.reason`,
    ),
  };
}

/** Read replacement declarations in sorted target-ID order. */
function replacementDecisions(
  value: unknown,
  where: string,
): ReadonlyMap<string, RuleReplacement> {
  const replace = new Map<string, RuleReplacement>(
    Object.entries(object(value, `${where}.replace`))
      .sort(([a], [b]) => compare(a, b))
      .map(([id, rawReplacement]) => [
        id,
        replacementDeclaration(rawReplacement, id, where),
      ]),
  );
  return replace;
}

/** Validate distinct group IDs at their original array indices before sorting for resolution. */
function selectedGroups(
  value: unknown,
  location: string,
): ReadonlyArray<string> {
  return strings(value, location)
    .map((id, index) => groupId(id, `${location}[${index}]`))
    .sort(compare);
}

/** Parse a source's fields in declaration-policy order, then reject contradictory exceptions. */
function sourceConfiguration(
  name: string,
  raw: unknown,
  repositories: Set<string>,
): LibrarySource {
  const where = `sources.${name}`;
  if (!/^[a-z][a-z0-9-]*$/u.test(name) || name === 'local')
    return invalid(where, 'invalid or reserved source name');
  const source = object(raw, where);
  knownFields(
    source,
    ['repository', 'ref', 'groups', 'exclude', 'replace'],
    where,
  );
  const repository = sourceRepository(
    field(source, 'repository'),
    where,
    repositories,
  );
  const ref = sourceRef(field(source, 'ref'), where);
  const groups = selectedGroups(field(source, 'groups'), `${where}.groups`);
  const exclude = exclusionDecisions(field(source, 'exclude'), where);
  const replace = replacementDecisions(field(source, 'replace'), where);
  for (const id of [...exclude.keys(), ...replace.keys()]) {
    const decision = exclude.has(id) ? 'exclude' : 'replace';
    ruleGroup(`${id}.md`, `${where}.${decision}.${id}`);
    if (exclude.has(id) && replace.has(id))
      return invalid(`${where}:${id}`, 'rule is both excluded and replaced');
  }
  return { name, repository, ref, groups, exclude, replace };
}

/**
 * Return validated configuration with sources, groups, and exception keys sorted.
 * Throws an invalid-input BuildError for unsupported fields, malformed values, or conflicting declarations.
 * Rejects abbreviated-commit-shaped refs unless explicitly qualified as refs/tags/<name>.
 * This offline syntax check does not resolve refs or establish their existence in Git.
 */
export function configuration(input: unknown): ProjectConfig {
  const config = object(input, 'configuration');
  knownFields(
    config,
    ['schemaVersion', 'sources', 'localGroups'],
    'configuration',
  );
  if (field(config, 'schemaVersion') !== 1)
    return invalid('schemaVersion', 'only version 1 is supported');
  const repositories = new Set<string>();
  const sources: Array<LibrarySource> = Object.entries(
    object(field(config, 'sources'), 'sources'),
  )
    .sort(([a], [b]) => compare(a, b))
    .map(([name, raw]) => sourceConfiguration(name, raw, repositories));
  const localGroups = selectedGroups(
    field(config, 'localGroups'),
    'localGroups',
  );
  for (const id of localGroups) {
    if (sources.some((source) => source.groups.includes(id)))
      return invalid(
        id,
        'an imported group cannot also be declared in localGroups',
      );
  }
  return { sources, localGroups };
}
