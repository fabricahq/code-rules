/** @fileoverview Exercises semantic version imports through real Git tags, generated provenance, and offline rebuilds. */

import { afterEach, beforeEach, expect, test } from 'bun:test';
import { mkdir, readdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { buildRules } from '../builds';
import type { LibrarySnapshot } from '../builds';
import {
  addLibrary,
  createImportFixture,
  fixtureGit,
  fixtureSource,
  removeImportFixture,
  runImport,
} from '../../tests/fixtures/import-fixture';
import type { ImportFixture } from '../../tests/fixtures/import-fixture';

let fixture: ImportFixture;
beforeEach(async () => {
  fixture = await createImportFixture();
});
afterEach(async () => {
  expect(await readdir(join(fixture.root, 'scratch'))).toEqual([]);
  await removeImportFixture(fixture);
});

/** Declare a version constraint with the existing fixture's non-revision fields. */
function config(version: string): {
  schemaVersion: number;
  sources: Record<string, unknown>;
} {
  const { ref: _ref, ...source } = fixtureSource('one');
  return {
    schemaVersion: 1,
    sources: { one: { ...source, version } },
  };
}

/** Type the JSON protocol emitted by our own fixture worker for offline rebuild tests; error assertions precede every success-only access. */
function result(configuration: unknown) {
  return runImport(fixture, configuration, { build: true }) as {
    error?: string;
    message?: string;
    snapshots: Record<string, LibrarySnapshot>;
    generated: Record<string, string>;
  };
}

test.each([
  ['^1.2.0', 'v1.9.0'],
  ['~1.2.0', '1.2.4'],
  ['>=1.2.0 <1.3.0', '1.2.4'],
  ['1.2.x', '1.2.4'],
  ['1.2.3', 'v1.2.3'],
  ['^0.2.0', 'v0.2.9'],
  ['>=2.0.0-beta.1 <2.0.0', 'v2.0.0-beta.2'],
])('selects the highest matching tag for %s', async (range, tag) => {
  const library = await addLibrary(fixture, 'one');
  for (const name of [
    'v1.2.3',
    '1.2.4',
    'v1.9.0',
    'v2.0.0',
    'v2.0.0-beta.2',
    'v0.2.9',
    'v0.3.0',
    'v99',
    'release-99.0.0',
  ])
    fixtureGit(library.directory, ['tag', name]);
  const imported = result(config(range));
  expect(imported.error).toBeUndefined();
  expect(imported.snapshots.one).toMatchObject({
    version: range,
    resolvedVersion: tag.replace(/^v/u, ''),
    resolvedTag: tag,
    resolvedCommit: library.commit,
  });
  expect(imported.snapshots.one?.ref).toBeUndefined();
  expect(imported.generated['provenance.json']).toContain(
    `"version": "${range}"`,
  );
  expect(imported.generated['libraries/one/README.md']).toContain(
    `**Selected tag:** ${tag}`,
  );
});

test.each([
  { version: 'not-a-range' },
  { version: '>=1.0, <2.0' },
  { version: '' },
  { version: '^1.0.0', ref: 'v1' },
  { version: null },
  {},
])(
  'rejects invalid or conflicting revision declarations before Git: %j',
  (declaration) => {
    const { ref: _ref, ...source } = fixtureSource('one');
    expect(
      runImport(
        fixture,
        {
          schemaVersion: 1,
          sources: { one: { ...source, ...declaration } },
        },
        {},
        { ...fixture.env, PATH: '/nonexistent' },
      ),
    ).toMatchObject({ error: 'invalid-configuration' });
  },
);

test('reports no match rather than falling back to a prerelease, branch, or partial tag', async () => {
  const library = await addLibrary(fixture, 'one');
  for (const name of ['v1.2.0-beta.1', 'v1.2', 'release-1.2.0'])
    fixtureGit(library.directory, ['tag', name]);
  fixtureGit(library.directory, ['branch', 'v1.2.0']);
  expect(result(config('^1.2.0'))).toMatchObject({
    error: 'version-not-found',
  });
});

test('allows same-commit aliases, including annotated tags, with deterministic selection', async () => {
  const library = await addLibrary(fixture, 'one');
  fixtureGit(library.directory, [
    'tag',
    '-a',
    'v1.2.3',
    '-m',
    'Annotated release',
  ]);
  fixtureGit(library.directory, ['tag', '1.2.3']);
  expect(result(config('^1.0.0')).snapshots.one).toMatchObject({
    resolvedTag: '1.2.3',
    resolvedCommit: library.commit,
  });
  fixtureGit(library.directory, ['tag', '-d', '1.2.3']);
  expect(result(config('^1.0.0')).snapshots.one).toMatchObject({
    resolvedTag: 'v1.2.3',
    resolvedCommit: library.commit,
  });
});

test('rejects same-precedence tags pointing to different commits', async () => {
  const library = await addLibrary(fixture, 'one');
  fixtureGit(library.directory, ['tag', 'v1.2.3']);
  await writeFile(
    join(library.directory, 'LICENSE.txt'),
    'Updated fixture terms',
  );
  fixtureGit(library.directory, ['add', '.']);
  fixtureGit(library.directory, ['commit', '-m', 'Change fixture']);
  fixtureGit(library.directory, ['tag', '1.2.3+other']);
  expect(result(config('^1.2.0'))).toMatchObject({
    error: 'ambiguous-version',
  });
});

test('rejects version tags that do not resolve to commits', async () => {
  const library = await addLibrary(fixture, 'one');
  const blob = fixtureGit(library.directory, ['rev-parse', 'HEAD:LICENSE.txt']);
  fixtureGit(library.directory, ['tag', 'v1.2.0', blob]);
  expect(result(config('^1.0.0'))).toMatchObject({
    error: 'unsupported-content',
  });
});

test('keeps offline builds pinned while a new import discovers a later matching release', async () => {
  const library = await addLibrary(fixture, 'one');
  fixtureGit(library.directory, ['tag', 'v1.2.0']);
  const configuration = config('^1.2.0');
  const original = result(configuration);
  expect(original.error).toBeUndefined();
  await writeFile(join(library.directory, 'LICENSE.txt'), 'New release terms');
  fixtureGit(library.directory, ['add', '.']);
  fixtureGit(library.directory, ['commit', '-m', 'New release']);
  fixtureGit(library.directory, ['tag', 'v1.3.0']);
  expect(
    buildRules({
      configuration,
      snapshots: original.snapshots,
      localFiles: {},
      toolVersion: 'test',
    }).files,
  ).toEqual(original.generated);
  expect(result(configuration).snapshots.one).toMatchObject({
    resolvedTag: 'v1.3.0',
  });
  expect(() =>
    buildRules({
      configuration: config('^2.0.0'),
      snapshots: original.snapshots,
      localFiles: {},
      toolVersion: 'test',
    }),
  ).toThrow('differ from configuration');
  const snapshot = original.snapshots.one;
  if (snapshot === undefined)
    throw new Error('Expected the imported one snapshot.');
  for (const change of [
    { resolvedVersion: '9.0.0' },
    { resolvedTag: 'v1.9.0' },
    { resolvedVersion: '' },
    { version: '^2.0.0' },
  ]) {
    expect(() =>
      buildRules({
        configuration,
        snapshots: { one: { ...snapshot, ...change } },
        localFiles: {},
        toolVersion: 'test',
      }),
    ).toThrow();
  }
});

test('fails if the selected tag moves between discovery and fetch', async () => {
  const library = await addLibrary(fixture, 'one');
  fixtureGit(library.directory, ['tag', 'v1.2.0']);
  await writeFile(join(library.directory, 'LICENSE.txt'), 'Later terms');
  fixtureGit(library.directory, ['add', '.']);
  fixtureGit(library.directory, ['commit', '-m', 'Later commit']);
  const later = fixtureGit(library.directory, ['rev-parse', 'HEAD']);
  const realGit = Bun.which('git');
  if (!realGit) throw new Error('Git is required for this test.');
  const bin = join(fixture.root, 'bin');
  await mkdir(bin);
  await writeFile(
    join(bin, 'git'),
    `#!/bin/sh\n"$TEST_REAL_GIT" "$@"\nstatus=$?\ncase "$*" in\n  *'ls-remote --tags'*) GIT_CONFIG_COUNT=0 "$TEST_REAL_GIT" -C "$TEST_LIBRARY_ROOT" tag -f v1.2.0 "$TEST_LATER_COMMIT" >/dev/null ;;\nesac\nexit "$status"\n`,
    { mode: 0o755 },
  );
  expect(
    runImport(
      fixture,
      config('^1.0.0'),
      {},
      {
        ...fixture.env,
        PATH: `${bin}:${process.env.PATH}`,
        TEST_REAL_GIT: realGit,
        TEST_LIBRARY_ROOT: library.directory,
        TEST_LATER_COMMIT: later,
      },
    ),
  ).toMatchObject({ error: 'ref-changed' });
});
