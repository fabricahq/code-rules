/** @fileoverview Tests public Imports behavior against real Git repositories and the offline Builds boundary. */

import { afterEach, beforeEach, expect, test } from 'bun:test';
import { readdir, writeFile, symlink, mkdir } from 'node:fs/promises';
import { join } from 'node:path';
import {
  addLibrary,
  createImportFixture,
  exampleFiles,
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

/** Build a valid project configuration with caller-selected fixture sources. */
function config(sources: Record<string, unknown>): unknown {
  return { schemaVersion: 1, sources, localGroups: [] };
}

/** Assert an error through the public child-process boundary, without relying on internal helpers. */
function failure(configuration: unknown, code: string): void {
  expect(runImport(fixture, configuration)).toMatchObject({ error: code });
}

test('imports two libraries, preserves bytes and origins, and builds one group', async () => {
  const one = await addLibrary(fixture, 'one');
  const two = await addLibrary(fixture, 'two');
  const result = runImport(
    fixture,
    config({ fabrica: fixtureSource('one'), acme: fixtureSource('two') }),
    { build: true },
  );
  expect(result).toMatchObject({
    snapshots: {
      fabrica: { resolvedCommit: one.commit, ref: 'v1' },
      acme: { resolvedCommit: two.commit },
    },
    files: {
      fabrica: {
        'images/flow.png': Buffer.from(
          exampleFiles['images/flow.png'] ?? '',
        ).toString('base64'),
        'LICENSE.txt': Buffer.from(exampleFiles['LICENSE.txt'] ?? '').toString(
          'base64',
        ),
        'terms/extra.txt': Buffer.from(
          'Additional original fixture terms.',
        ).toString('base64'),
      },
    },
  });
  const serialized = JSON.stringify(result);
  expect(serialized).toContain('fabrica:practices/testing/verify-retries');
  expect(serialized).toContain('acme:practices/testing/verify-retries');
  expect(serialized).toContain('../../vendor/fabrica/images/flow.png');
  expect(serialized).toContain('../../vendor/fabrica/terms/special.pdf');
  expect(serialized).not.toContain('logging/example`');
  expect(serialized).not.toContain('unselected.txt');
  expect(
    runImport(
      fixture,
      config({ acme: fixtureSource('two'), fabrica: fixtureSource('one') }),
      { build: true },
    ),
  ).toEqual(result);
});

test('excludes only one origin while keeping both upstream source files', async () => {
  await addLibrary(fixture, 'one');
  await addLibrary(fixture, 'two');
  const result = runImport(
    fixture,
    config({
      fabrica: {
        ...fixtureSource('one'),
        exclude: { 'practices/testing/verify-retries': 'Covered by Acme.' },
      },
      acme: fixtureSource('two'),
    }),
    { build: true },
  );
  expect(result).toMatchObject({
    files: {
      fabrica: { 'practices/testing/verify-retries.md': expect.any(String) },
    },
    generated: {
      'practices/testing.md': expect.not.stringContaining('Rule ID: `fabrica:'),
    },
  });
});

test('retains a local replacement under the upstream identity', async () => {
  await addLibrary(fixture, 'one');
  const replacement =
    String(exampleFiles['practices/testing/verify-retries.md']).split(
      '\n\n',
    )[0] + '\n\nUse three attempts.';
  const result = runImport(
    fixture,
    config({
      fabrica: {
        ...fixtureSource('one'),
        replace: {
          'practices/testing/verify-retries': {
            file: 'local/practices/testing/bounded.md',
            reason: 'Project retry budget.',
          },
        },
      },
    }),
    {
      build: true,
      localFiles: { 'practices/testing/bounded.md': replacement },
    },
  );
  expect(result).toMatchObject({
    generated: {
      'practices/testing.md': expect.stringContaining('Use three attempts.'),
    },
  });
});

test('accepts empty sources without Git, and reports invalid configuration before fetching', () => {
  expect(
    runImport(
      fixture,
      config({}),
      {},
      { ...fixture.env, PATH: '/nonexistent' },
    ),
  ).toMatchObject({ snapshots: {}, files: {} });
  failure({}, 'invalid-configuration');
});

test('reports unavailable Git and caller cancellation', () => {
  expect(
    runImport(
      fixture,
      config({ one: fixtureSource('one') }),
      {},
      { ...fixture.env, PATH: '/nonexistent' },
    ),
  ).toMatchObject({ error: 'git-unavailable' });
  expect(
    runImport(fixture, config({ one: fixtureSource('one') }), { cancel: true }),
  ).toMatchObject({ error: 'cancelled' });
});

test('imports a full uppercase SHA and preserves its configured spelling', async () => {
  const { commit } = await addLibrary(fixture, 'one');
  expect(
    runImport(
      fixture,
      config({ one: fixtureSource('one', commit.toUpperCase()) }),
    ),
  ).toMatchObject({
    snapshots: { one: { ref: commit.toUpperCase(), resolvedCommit: commit } },
  });
});

test('resolves annotated and nested tags and ignores a same-name branch', async () => {
  const { directory, commit } = await addLibrary(fixture, 'one');
  fixtureGit(directory, ['tag', '-a', 'annotated', '-m', 'Annotated']);
  fixtureGit(directory, ['tag', '-a', 'nested', 'annotated', '-m', 'Nested']);
  fixtureGit(directory, ['branch', 'nested']);
  for (const ref of ['annotated', 'refs/tags/nested'])
    expect(
      runImport(fixture, config({ one: fixtureSource('one', ref) })),
    ).toMatchObject({ snapshots: { one: { ref, resolvedCommit: commit } } });
});

test('fetches a moved tag again while commit imports retain the old content', async () => {
  const { directory, commit } = await addLibrary(fixture, 'one');
  await writeFile(join(directory, 'LICENSE.txt'), 'Changed terms');
  fixtureGit(directory, ['add', '.']);
  fixtureGit(directory, ['commit', '-m', 'Change terms']);
  fixtureGit(directory, ['tag', '-f', 'v1']);
  const next = fixtureGit(directory, ['rev-parse', 'HEAD']);
  expect(
    runImport(fixture, config({ one: fixtureSource('one') })),
  ).toMatchObject({ snapshots: { one: { resolvedCommit: next } } });
  expect(
    runImport(fixture, config({ one: fixtureSource('one', commit) })),
  ).toMatchObject({ snapshots: { one: { resolvedCommit: commit } } });
});

test('fails on branch-only names, missing tags, and refused commits without fallback', async () => {
  const { directory } = await addLibrary(fixture, 'one');
  fixtureGit(directory, ['branch', 'branch-only']);
  for (const ref of ['branch-only', 'missing', 'a'.repeat(40)])
    failure(config({ one: fixtureSource('one', ref) }), 'ref-not-found');
});

test('rejects tags pointing to blobs or trees', async () => {
  const { directory } = await addLibrary(fixture, 'one');
  fixtureGit(directory, [
    'tag',
    'blob',
    fixtureGit(directory, ['rev-parse', 'HEAD:LICENSE.txt']),
  ]);
  fixtureGit(directory, [
    'tag',
    'tree',
    fixtureGit(directory, ['rev-parse', 'HEAD^{tree}']),
  ]);
  for (const ref of ['blob', 'tree'])
    failure(config({ one: fixtureSource('one', ref) }), 'unsupported-content');
});

test.each([
  'release//v1',
  'release/.hidden',
  'release/foo.lock/v1',
  'bad\u007f',
  'refs/heads/main',
])('rejects invalid ref %s before Git', (ref) => {
  failure(config({ one: fixtureSource('one', ref) }), 'invalid-configuration');
});

test('accepts explicitly qualified numeric tags', async () => {
  const { directory, commit } = await addLibrary(fixture, 'one');
  fixtureGit(directory, ['tag', '2026']);
  expect(
    runImport(fixture, config({ one: fixtureSource('one', 'refs/tags/2026') })),
  ).toMatchObject({ snapshots: { one: { resolvedCommit: commit } } });
});

test('reports missing repositories without claiming whether access was denied', () => {
  failure(config({ one: fixtureSource('absent') }), 'not-found-or-no-access');
});

test('returns no partial result when a later source fails', async () => {
  await addLibrary(fixture, 'one');
  expect(
    runImport(
      fixture,
      config({ a: fixtureSource('one'), z: fixtureSource('absent') }),
    ),
  ).toMatchObject({ error: 'not-found-or-no-access' });
});

test.each([
  'rule-library.json',
  'practices/testing/_group.json',
  'LICENSE.txt',
  'terms/special.pdf',
])('rejects a missing required file: %s', async (path) => {
  const files = Object.fromEntries(
    Object.entries(exampleFiles).filter(([name]) => name !== path),
  );
  await addLibrary(fixture, 'one', files);
  failure(config({ one: fixtureSource('one') }), 'invalid-library');
});

test.each([
  'rule-library.json',
  'practices/testing/_group.json',
  'practices/testing/verify-retries.md',
  'notes/explanation.md',
])('rejects invalid UTF-8 in interpreted text: %s', async (path) => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    [path]: new Uint8Array([255]),
  });
  failure(config({ one: fixtureSource('one') }), 'invalid-library');
});

test('rejects invalid library metadata', async () => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    'rule-library.json': '{',
  });
  failure(config({ one: fixtureSource('one') }), 'invalid-library');
});

test('treats underscore Markdown as an attachment, not a rule', async () => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    'practices/testing/_README.md': 'An introduction.',
  });
  expect(
    runImport(fixture, config({ one: fixtureSource('one') }), { build: true }),
  ).toMatchObject({
    generated: { 'practices/testing.md': expect.any(String) },
  });
});

test('rejects symlinks reached through an attachment link', async () => {
  const { directory } = await addLibrary(fixture, 'one');
  await symlink('../LICENSE.txt', join(directory, 'notes/linked.txt'));
  await writeFile(
    join(directory, 'notes/explanation.md'),
    '[Link](linked.txt)',
  );
  fixtureGit(directory, ['add', '.']);
  fixtureGit(directory, ['commit', '-m', 'Add symlink']);
  fixtureGit(directory, ['tag', '-f', 'v1']);
  failure(config({ one: fixtureSource('one') }), 'unsupported-content');
});

test('rejects submodule entries in selected groups without initializing them', async () => {
  const { directory, commit } = await addLibrary(fixture, 'one');
  fixtureGit(directory, [
    'update-index',
    '--add',
    '--cacheinfo',
    `160000,${commit},practices/testing/module`,
  ]);
  fixtureGit(directory, ['commit', '-m', 'Add gitlink']);
  fixtureGit(directory, ['tag', '-f', 'v1']);
  failure(config({ one: fixtureSource('one') }), 'unsupported-content');
});

test('rejects Git LFS pointers rather than vendoring missing content', async () => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    'images/flow.png':
      'version https://git-lfs.github.com/spec/v1\noid sha256:abc\nsize 10\n',
  });
  failure(config({ one: fixtureSource('one') }), 'unsupported-content');
});

test('rejects retained paths with case collisions in parent directories', async () => {
  const { directory } = await addLibrary(fixture, 'one');
  const blob = fixtureGit(directory, ['rev-parse', 'HEAD:LICENSE.txt']);
  fixtureGit(directory, [
    'update-index',
    '--add',
    '--cacheinfo',
    `100644,${blob},practices/testing/Docs/a.txt`,
  ]);
  fixtureGit(directory, [
    'update-index',
    '--add',
    '--cacheinfo',
    `100644,${blob},practices/testing/docs/b.txt`,
  ]);
  fixtureGit(directory, ['commit', '-m', 'Add ambiguous paths']);
  fixtureGit(directory, ['tag', '-f', 'v1']);
  failure(config({ one: fixtureSource('one') }), 'unsupported-content');
});

test('rejects files exceeding the size ceiling before retaining them', async () => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    'images/flow.png': new Uint8Array(8 * 1024 * 1024 + 1),
  });
  failure(config({ one: fixtureSource('one') }), 'limit-exceeded');
});

test('preserves encoded link targets and rejects links escaping the library', async () => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    'notes/explanation.md': '[Space](space%20name.txt)',
    'notes/space name.txt': 'Retain me.',
  });
  expect(
    JSON.stringify(runImport(fixture, config({ one: fixtureSource('one') }))),
  ).toContain('notes/space name.txt');
  await addLibrary(fixture, 'two', {
    ...exampleFiles,
    'notes/explanation.md': '[Escape](../../outside)',
  });
  failure(config({ two: fixtureSource('two') }), 'invalid-library');
});

test('does not execute library hooks or checkout filters', async () => {
  const { directory } = await addLibrary(fixture, 'one');
  await mkdir(join(directory, '.git/hooks'), { recursive: true });
  await writeFile(
    join(directory, '.git/hooks/post-checkout'),
    '#!/bin/sh\ntouch HOOK-RAN\n',
    { mode: 0o755 },
  );
  expect(
    runImport(fixture, config({ one: fixtureSource('one') })),
  ).toMatchObject({ snapshots: { one: { ref: 'v1' } } });
  expect(await readdir(directory)).not.toContain('HOOK-RAN');
});

test('does not inherit another checkout object database', async () => {
  await addLibrary(fixture, 'one');
  expect(
    runImport(
      fixture,
      config({ one: fixtureSource('one') }),
      {},
      {
        ...fixture.env,
        GIT_DIR: '/nonexistent',
        GIT_WORK_TREE: '/nonexistent',
      },
    ),
  ).toMatchObject({ snapshots: { one: { ref: 'v1' } } });
});

test.each([
  ['sigma-σ.txt', 'sigma-ς.txt'],
  ['sharp-ß.txt', 'sharp-SS.txt'],
  ['sharp-ẞ.txt', 'sharp-SS.txt'],
  ['café.txt', 'cafe\u0301.txt'],
])('rejects colliding Unicode filenames %s and %s', async (first, second) => {
  const { directory } = await addLibrary(fixture, 'one');
  const blob = fixtureGit(directory, ['rev-parse', 'HEAD:LICENSE.txt']);
  for (const name of [first, second])
    fixtureGit(directory, [
      'update-index',
      '--add',
      '--cacheinfo',
      `100644,${blob},practices/testing/${name}`,
    ]);
  fixtureGit(directory, ['commit', '-m', 'Add Unicode collision']);
  fixtureGit(directory, ['tag', '-f', 'v1']);
  failure(config({ one: fixtureSource('one') }), 'unsupported-content');
});

test('preserves the primary import error when cleanup also fails', () => {
  expect(
    runImport(fixture, config({ one: fixtureSource('absent') }), {
      cleanupFailure: true,
    }),
  ).toMatchObject({
    error: 'not-found-or-no-access',
    cause: expect.stringContaining('temporary import storage'),
  });
});

test('reports cleanup failure when the import itself succeeds', async () => {
  await addLibrary(fixture, 'one');
  expect(
    runImport(fixture, config({ one: fixtureSource('one') }), {
      cleanupFailure: true,
    }),
  ).toMatchObject({
    error: 'io-error',
    message: expect.stringContaining('temporary import storage'),
  });
});

test('retains root-relative attachments with encoded leading slashes', async () => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    'practices/testing/verify-retries.md': String(
      exampleFiles['practices/testing/verify-retries.md'],
    ).replace('../../images/flow.png', '/%2Fimages/flow.png?size=1#flow'),
  });
  expect(
    runImport(fixture, config({ one: fixtureSource('one') }), { build: true }),
  ).toMatchObject({
    generated: {
      'practices/testing.md': expect.stringContaining(
        '../../vendor/one/images/flow.png?size=1#flow',
      ),
    },
  });
});

test('rejects encoded absolute paths that traverse beyond the library root', async () => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    'notes/explanation.md': '[Escape](/%2F../outside.txt)',
  });
  failure(config({ one: fixtureSource('one') }), 'invalid-library');
});
