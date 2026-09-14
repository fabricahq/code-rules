/** @fileoverview Tests public Imports behavior against real Git repositories and the offline Builds boundary. */

import { afterEach, beforeEach, expect, test } from 'bun:test';
import { readdir, writeFile, symlink, rm } from 'node:fs/promises';
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
        'assets/images/flow.png': Buffer.from(
          exampleFiles['assets/images/flow.png'] ?? '',
        ).toString('base64'),
        'LICENSE.txt': Buffer.from(exampleFiles['LICENSE.txt'] ?? '').toString(
          'base64',
        ),
        'assets/extra.txt': Buffer.from(
          'Additional original fixture terms.',
        ).toString('base64'),
      },
    },
  });
  const serialized = JSON.stringify(result);
  expect(serialized).toContain('fabrica:practices/testing/verify-retries');
  expect(serialized).toContain('acme:practices/testing/verify-retries');
  expect(serialized).toContain(
    '../../../vendor/fabrica/assets/images/flow.png',
  );
  expect(serialized).toContain('../../../vendor/fabrica/assets/special.pdf');
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
      'groups/practices/testing.md':
        expect.not.stringContaining('Rule ID: `fabrica:'),
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
      'groups/practices/testing.md': expect.stringContaining(
        'Use three attempts.',
      ),
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
  'NOTICE.txt',
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
  'assets/notes/explanation.md',
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

test('rejects supporting Markdown outside assets directories', async () => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    'practices/testing/_README.md': 'An introduction.',
  });
  failure(config({ one: fixtureSource('one') }), 'invalid-library');
});

test('rejects symlinks reached through an attachment link', async () => {
  const { directory } = await addLibrary(fixture, 'one');
  await symlink('../LICENSE.txt', join(directory, 'assets/notes/linked.txt'));
  await writeFile(
    join(directory, 'assets/notes/explanation.md'),
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
    'assets/images/flow.png':
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
    'assets/images/flow.png': new Uint8Array(8 * 1024 * 1024 + 1),
  });
  failure(config({ one: fixtureSource('one') }), 'limit-exceeded');
});

test('preserves encoded link targets and rejects links escaping the library', async () => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    'assets/notes/explanation.md': '[Space](space%20name.txt)',
    'assets/notes/space name.txt': 'Retain me.',
  });
  expect(
    JSON.stringify(runImport(fixture, config({ one: fixtureSource('one') }))),
  ).toContain('assets/notes/space name.txt');
  await addLibrary(fixture, 'two', {
    ...exampleFiles,
    'assets/notes/explanation.md': '[Escape](../../outside)',
  });
  failure(config({ two: fixtureSource('two') }), 'invalid-library');
});

test('preserves original bytes without running configured checkout filters', async () => {
  const path = 'practices/testing/verify-retries.md';
  const { directory } = await addLibrary(fixture, 'one', {
    ...exampleFiles,
    '.gitattributes': '*.md filter=fixture\n',
  });
  const marker = join(fixture.root, 'FILTER-RAN');
  const script = join(fixture.root, 'smudge.sh');
  await writeFile(script, `#!/bin/sh\nprintf ran > '${marker}'\ncat\n`);
  const command = `sh '${script}'`;
  await rm(join(directory, path));
  fixtureGit(directory, [
    '-c',
    `filter.fixture.smudge=${command}`,
    'checkout-index',
    '--force',
    '--all',
  ]);
  expect(await readdir(fixture.root)).toContain('FILTER-RAN');
  await rm(marker);
  const result = runImport(
    fixture,
    config({ one: fixtureSource('one') }),
    {},
    {
      ...fixture.env,
      GIT_CONFIG_COUNT: '3',
      GIT_CONFIG_KEY_2: 'filter.fixture.smudge',
      GIT_CONFIG_VALUE_2: command,
    },
  );
  expect(result).toMatchObject({
    files: {
      one: {
        [path]: Buffer.from(exampleFiles[path]!).toString('base64'),
      },
    },
  });
  expect(await readdir(fixture.root)).not.toContain('FILTER-RAN');
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
    ).replace(
      '../../assets/images/flow.png',
      '/%2Fassets/images/flow.png?size=1#flow',
    ),
  });
  expect(
    runImport(fixture, config({ one: fixtureSource('one') }), { build: true }),
  ).toMatchObject({
    generated: {
      'groups/practices/testing.md': expect.stringContaining(
        '../../../vendor/one/assets/images/flow.png?size=1#flow',
      ),
    },
  });
});

test('rejects encoded absolute paths that traverse beyond the library root', async () => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    'assets/notes/explanation.md': '[Escape](/%2F../outside.txt)',
  });
  failure(config({ one: fixtureSource('one') }), 'invalid-library');
});

test.each([
  'https://gitlab.com/team/nested/one.git',
  'ssh://git@git.example.org:2222/srv/one.git',
  'git@git.example.org:team/one.git',
  'git@git.example.org:/srv/one.git',
])(
  'imports the exact configured Git address %s and retains its provenance',
  async (repository) => {
    const { directory, commit } = await addLibrary(fixture, 'one');
    const result = runImport(
      fixture,
      config({ one: { ...fixtureSource('one'), repository } }),
      { build: true },
      {
        ...fixture.env,
        GIT_CONFIG_KEY_0: `url.file://${directory}.insteadOf`,
        GIT_CONFIG_VALUE_0: repository,
      },
    );
    expect(result).toMatchObject({
      snapshots: { one: { repository, resolvedCommit: commit } },
    });
    expect(JSON.stringify(result)).toContain(repository);
  },
);

test('rejects a rewrite to a local transport without an explicit Git allowance', async () => {
  await addLibrary(fixture, 'one');
  expect(
    runImport(
      fixture,
      config({ one: fixtureSource('one') }),
      {},
      {
        ...fixture.env,
        GIT_CONFIG_COUNT: '1',
      },
    ),
  ).toMatchObject({ error: 'not-found-or-no-access' });
});

test.each([
  'fixture/one',
  'file:///tmp/library',
  'git::https://example.org/rules.git',
  'https://user:secret@example.org/rules.git',
])('rejects unsupported repository %s before invoking Git', (repository) => {
  expect(
    runImport(
      fixture,
      config({ one: { ...fixtureSource('one'), repository } }),
      {},
      {
        ...fixture.env,
        PATH: '/nonexistent',
      },
    ),
  ).toMatchObject({ error: 'invalid-configuration' });
});

test.each([
  {
    selector: '*',
    groups: ['practices/logging', 'practices/testing', 'techs/go'],
  },
  {
    selector: 'practices/*',
    groups: ['practices/logging', 'practices/testing'],
  },
  { selector: 'techs/*', groups: ['techs/go'] },
])(
  'imports complete wildcard snapshots for $selector and builds their active rules',
  async ({ selector, groups }) => {
    await addLibrary(fixture, 'one', {
      ...exampleFiles,
      'practices/logging/_group.json':
        exampleFiles['practices/testing/_group.json']!,
      'practices/logging/example.md':
        '---\ntitle: Log failures\nwhenToRead: When handling errors.\nimpact: HIGH\nimpactDescription: Failures stay visible.\n---\n\nRecord the failure.',
      'techs/go/_group.json': exampleFiles['practices/testing/_group.json']!,
    });
    const result = runImport(
      fixture,
      config({ one: { ...fixtureSource('one'), groups: selector } }),
      { build: true },
    );
    expect(result).toMatchObject({
      snapshots: { one: { groupSelection: selector, groups } },
      generated: Object.fromEntries(
        groups.map((group) => [`groups/${group}.md`, expect.any(String)]),
      ),
    });
    expect(result).not.toHaveProperty('error');
  },
);

test('copies declared UTF-8 license and notice text into generated output unchanged', async () => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    'LICENSE.txt': '\ufeffOriginal terms\r\n',
  });
  const result = runImport(fixture, config({ one: fixtureSource('one') }), {
    build: true,
  });
  expect(result).toMatchObject({
    generated: {
      'libraries/one/licenses/LICENSE.md': '\ufeffOriginal terms\r\n',
      'libraries/one/licenses/notices/001.md': exampleFiles['NOTICE.txt'],
    },
  });
});

test('rejects binary declared license text instead of corrupting generated terms', async () => {
  await addLibrary(fixture, 'one', {
    ...exampleFiles,
    'LICENSE.txt': new Uint8Array([255, 128]),
  });
  failure(config({ one: fixtureSource('one') }), 'invalid-library');
});
