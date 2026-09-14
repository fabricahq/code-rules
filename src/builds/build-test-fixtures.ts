/** @fileoverview Supplies original in-memory fixtures and output readers shared by public Builds behavior tests. */

import { posix } from 'node:path';
import { buildRules } from './index';
import type { BuildInput, FileContents, LibrarySnapshot } from './index';

/** Synthetic pinned revision shared by the original test libraries. */
export const commit = 'a'.repeat(40);

/** Testing group shared by the baseline library and local fixtures. */
export const group = 'practices/testing';

/** Library-relative rule identity used by baseline exclusions and replacements. */
export const ruleId = `${group}/verify-retries`;

/** Valid testing-group metadata with behavior-based reading guidance. */
export const metadata = JSON.stringify({
  name: 'Testing',
  description: 'Check externally visible behavior.',
  whenToRead: ['Changing behavior, including production code.'],
});

/** Create a complete Markdown rule with an optional body for public Builds tests. */
export function ruleText(
  title: string,
  body = 'Verify the retry limit before shipping.',
): string {
  return `---\ntitle: ${title}\nwhenToRead: When planning, implementing, or reviewing retries.\nimpact: HIGH\nimpactDescription: Prevent unbounded retries.\ntags: testing, retries\n---\n\n## ${title}\n\n${body}\n`;
}

/** Declare one imported testing group with source-scoped exceptions. */
export function source(
  repository: string,
  exclude: Record<string, string> = {},
  replace: Record<string, { file: string; reason: string }> = {},
): object {
  return { repository, ref: 'v1.0.0', groups: [group], exclude, replace };
}

/** Create a pinned library snapshot with optional files overriding the defaults. */
export function snapshot(
  repository: string,
  title: string,
  extra: FileContents = {},
): LibrarySnapshot {
  return {
    repository,
    ref: 'v1.0.0',
    resolvedCommit: commit,
    groups: [group],
    files: {
      'rule-library.json': '{"formatVersion":1}',
      [`${group}/_group.json`]: metadata,
      [`${ruleId}.md`]: ruleText(title),
      ...extra,
    },
  };
}

/** Create two independent libraries sharing a testing group and rule path. */
export function input(): BuildInput {
  return {
    configuration: {
      schemaVersion: 1,
      sources: {
        fabrica: source('fabrica/rules'),
        acme: source('acme/rules'),
      },
      localGroups: [],
    },
    snapshots: {
      fabrica: snapshot('fabrica/rules', 'Fabrica retries'),
      acme: snapshot('acme/rules', 'Acme retries'),
    },
    localFiles: {},
    toolVersion: '0.0.0-test',
  };
}

/** Read one generated file through Builds, failing when the expected path is absent. */
export function generated(build: BuildInput, path: string): string {
  const content = buildRules(build).files[path];
  if (content === undefined) throw new Error(`Expected generated ${path}`);
  return content;
}

/** Create a project with one local-only testing group and supplied definitions. */
export function localInput(localFiles: FileContents): BuildInput {
  return {
    configuration: { schemaVersion: 1, sources: {}, localGroups: [group] },
    snapshots: {},
    localFiles: { [`${group}/_group.json`]: metadata, ...localFiles },
    toolVersion: 'test',
  };
}

/** Freeze nested fixture data so accidental mutations fail at the public Builds boundary. */
export function freezeInput(value: unknown): void {
  if (value === null || typeof value !== 'object') return;
  for (const child of Object.values(value)) freezeInput(child);
  Object.freeze(value);
}

/** Follow the Markdown links emitted by an index, keeping destinations relative to generated/. */
export function indexedPaths(path: string, text: string): Array<string> {
  return Array.from(text.matchAll(/\]\(([^)]+\.md)\)/gu), (match) =>
    posix.normalize(posix.join(posix.dirname(path), match[1] ?? '')),
  );
}
