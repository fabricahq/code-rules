/** @fileoverview Exercises sync, offline build, and check against real Git libraries and filesystem changes. */

import { afterEach, beforeEach, expect, test } from 'bun:test';
import {
  mkdir,
  writeFile,
  readFile,
  rm,
  readdir,
  symlink,
} from 'node:fs/promises';
import { join } from 'node:path';
import { hostname } from 'node:os';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import {
  createImportFixture,
  addLibrary,
  fixtureSource,
  fixtureGit,
  exampleFiles,
  removeImportFixture,
} from '../tests/fixtures/import-fixture';
import type { ImportFixture } from '../tests/fixtures/import-fixture';
import { readTree, treeDigest } from './project-files/files';
import { withWriter, applyChanges } from './project-files/apply';
import { checkProject, buildProject } from './project';
import { readProject, assertUnchanged } from './project-files/project';

let fixture: ImportFixture;
let root: string;
let library: { directory: string; commit: string };
const worker = fileURLToPath(
  new URL('../tests/fixtures/project-worker.ts', import.meta.url),
);

beforeEach(async () => {
  fixture = await createImportFixture();
  library = await addLibrary(fixture, 'team');
  root = join(fixture.root, 'project');
  await mkdir(root);
  await saveConfig({ team: fixtureSource('team') });
});
afterEach(async () => {
  await removeImportFixture(fixture);
});

async function saveConfig(sources: Record<string, unknown>): Promise<void> {
  await writeFile(
    join(root, 'config.json'),
    JSON.stringify({ schemaVersion: 1, sources }),
  );
}
function run(command: string, fault?: string): ReturnType<typeof spawnSync> {
  return spawnSync(
    process.execPath,
    [worker, command, '--config', join(root, 'config.json')],
    {
      env: { ...fixture.env, ...(fault ? { PROJECT_TEST_FAULT: fault } : {}) },
      encoding: 'utf8',
      timeout: 30000,
    },
  );
}
async function outputIdentity(): Promise<string[]> {
  return [
    treeDigest(await readTree(join(root, 'vendor'))),
    treeDigest(await readTree(join(root, 'generated'))),
  ];
}
function succeeded(result: ReturnType<typeof spawnSync>): void {
  expect(String(result.stderr)).toBe('');
  expect(result.status).toBe(0);
}

test('sync imports original bytes and licenses; repeated sync is unchanged and offline checks pass', async () => {
  succeeded(run('sync'));
  expect(await readFile(join(root, 'vendor/team/assets/special.pdf'))).toEqual(
    Buffer.from([0, 255, 128, 10]),
  );
  const license = exampleFiles['LICENSE.txt'];
  if (typeof license !== 'string')
    throw new Error('Fixture license must be text');
  expect(
    await readFile(
      join(root, 'generated/libraries/team/licenses/LICENSE.md'),
      'utf8',
    ),
  ).toBe(license);
  const before = await outputIdentity();
  const repeated = run('sync');
  succeeded(repeated);
  expect(JSON.parse(String(repeated.stdout))).toEqual({
    added: [],
    changed: [],
    removed: [],
  });
  expect(await outputIdentity()).toEqual(before);
  await rm(library.directory, { recursive: true });
  succeeded(run('build'));
  succeeded(run('check'));
});

test('check reports changed, extra, and missing output without changing any files', async () => {
  succeeded(run('sync'));
  await writeFile(join(root, 'generated/extra.md'), 'stale');
  await writeFile(join(root, 'generated/RULES.md'), 'changed');
  await rm(join(root, 'generated/provenance.json'));
  const before = await outputIdentity();
  const result = run('check');
  expect(result.status).toBe(1);
  const report = JSON.parse(String(result.stdout));
  expect(report.removed).toContain('extra.md');
  expect(report.changed).toContain('RULES.md');
  expect(report.added).toContain('provenance.json');
  expect(await outputIdentity()).toEqual(before);
  succeeded(run('build'));
  succeeded(run('check'));
});

test('offline build incorporates local replacements and preserves authored files and vendor bytes', async () => {
  succeeded(run('sync'));
  const path = 'practices/testing/custom.md';
  await mkdir(join(root, 'local/practices/testing'), { recursive: true });
  await writeFile(
    join(root, 'local', path),
    '---\ntitle: Custom\nwhenToRead: Changing retries.\nimpact: HIGH\nimpactDescription: avoid excess requests\n---\n\nUse three attempts.\n',
  );
  await saveConfig({
    team: {
      ...fixtureSource('team'),
      replace: {
        'practices/testing/verify-retries': {
          file: `local/${path}`,
          reason: 'Three attempts.',
        },
      },
    },
  });
  const before = treeDigest(await readTree(join(root, 'vendor')));
  succeeded(run('build'));
  const generatedRule = await readFile(
    join(root, 'generated/rules/local', path),
    'utf8',
  );
  expect(generatedRule).toContain('Use three attempts.');
  expect(generatedRule).toContain('Rule ID: `local:practices/testing/custom`');
  expect(generatedRule).not.toContain('Replaces upstream');
  expect(generatedRule).not.toContain('Three attempts.');
  await expect(
    readFile(
      join(root, 'generated/rules/team/practices/testing/verify-retries.md'),
    ),
  ).rejects.toThrow();
  const provenance = JSON.parse(
    await readFile(join(root, 'generated/provenance.json'), 'utf8'),
  );
  expect(provenance.rules).toEqual([
    expect.objectContaining({
      id: 'local:practices/testing/custom',
      origin: expect.objectContaining({ source: 'local', file: path }),
      upstream: expect.objectContaining({
        source: 'team',
        file: 'practices/testing/verify-retries.md',
      }),
      replacementReason: 'Three attempts.',
    }),
  ]);
  succeeded(run('check'));
  expect(treeDigest(await readTree(join(root, 'vendor')))).toBe(before);
  expect(await readFile(join(root, 'local', path), 'utf8')).toContain(
    'Use three attempts.',
  );
});

test('sync removes retired sources and their rules, assets, and license copies', async () => {
  await addLibrary(fixture, 'partner');
  await saveConfig({
    team: fixtureSource('team'),
    partner: fixtureSource('partner'),
  });
  succeeded(run('sync'));
  await saveConfig({ team: fixtureSource('team') });
  succeeded(run('sync'));
  expect(await readdir(join(root, 'vendor'))).toEqual(['team']);
  expect(await readdir(join(root, 'generated/libraries'))).toEqual(['team']);
  expect(new Set(await readdir(join(root, 'generated/rules')))).toEqual(
    new Set(['team', 'README.md']),
  );
});

test('version constraints persist selected tags and commits and refresh only on sync', async () => {
  fixtureGit(library.directory, ['tag', 'v1.2.0']);
  const source = fixtureSource('team');
  delete source.ref;
  await saveConfig({ team: { ...source, version: '^1.0.0' } });
  succeeded(run('sync'));
  const record = JSON.parse(
    await readFile(join(root, 'vendor/team/_source.json'), 'utf8'),
  );
  expect(record.resolvedTag).toBe('v1.2.0');
  expect(record.resolvedCommit).toBe(library.commit);
  expect(record.files['LICENSE.txt']).toMatch(/^[a-f0-9]{64}$/u);
  await writeFile(join(library.directory, 'NOTICE.txt'), 'New notice');
  fixtureGit(library.directory, ['add', '.']);
  fixtureGit(library.directory, ['commit', '-m', 'New release']);
  fixtureGit(library.directory, ['tag', 'v1.3.0']);
  const before = await outputIdentity();
  succeeded(run('build'));
  expect(await outputIdentity()).toEqual(before);
  succeeded(run('sync'));
  expect(
    JSON.parse(await readFile(join(root, 'vendor/team/_source.json'), 'utf8'))
      .resolvedTag,
  ).toBe('v1.3.0');
});

test.each(['missing', 'changed', 'extra'])(
  'offline operations reject %s imported bytes',
  async (kind) => {
    succeeded(run('sync'));
    if (kind === 'missing') await rm(join(root, 'vendor/team/NOTICE.txt'));
    if (kind === 'changed')
      await writeFile(join(root, 'vendor/team/assets/special.pdf'), 'tampered');
    if (kind === 'extra')
      await writeFile(join(root, 'vendor/team/extra.txt'), 'unexpected');
    const before = await outputIdentity();
    expect(run('build').status).toBe(1);
    expect(run('check').status).toBe(1);
    expect(await outputIdentity()).toEqual(before);
    succeeded(run('sync'));
  },
);

test('failed import or generation preserves both previous directories', async () => {
  succeeded(run('sync'));
  const before = await outputIdentity();
  await saveConfig({
    team: fixtureSource('team'),
    bad: fixtureSource('missing'),
  });
  expect(run('sync').status).toBe(1);
  expect(await outputIdentity()).toEqual(before);
  await saveConfig({
    team: {
      ...fixtureSource('team'),
      replace: {
        'practices/testing/verify-retries': {
          file: 'local/missing.md',
          reason: 'test',
        },
      },
    },
  });
  expect(run('sync').status).toBe(1);
  expect(await outputIdentity()).toEqual(before);
});

test.each(['vendor', 'generated', 'local'])(
  'rejects a symlinked %s directory without changing its target',
  async (name) => {
    const outside = join(fixture.root, 'outside');
    await mkdir(outside);
    await writeFile(join(outside, 'keep.txt'), 'preserve');
    await symlink(outside, join(root, name));
    expect(run('sync').status).toBe(1);
    expect(await readFile(join(outside, 'keep.txt'), 'utf8')).toBe('preserve');
  },
);

test('failed second directory rename restores the previous complete result', async () => {
  succeeded(run('sync'));
  const before = await outputIdentity();
  await saveConfig({});
  expect(run('sync', 'rename-error').status).toBe(1);
  expect(await outputIdentity()).toEqual(before);
  await saveConfig({ team: fixtureSource('team') });
  succeeded(run('check'));
});

test.each([false, true])(
  'next writer recovers changed output after a killed apply (previous output: %s)',
  async (existing) => {
    if (existing) succeeded(run('sync'));
    const before = await outputIdentity();
    await writeFile(
      join(library.directory, 'NOTICE.txt'),
      'Changed before interruption',
    );
    fixtureGit(library.directory, ['add', '.']);
    fixtureGit(library.directory, ['commit', '-m', 'Changed notice']);
    fixtureGit(library.directory, ['tag', '-f', 'v1']);
    expect(run('sync', 'crash').signal).toBe('SIGKILL');
    expect(await outputIdentity()).not.toEqual(before);
    expect(run('check').status).toBe(1);
    await rm(library.directory, { recursive: true });
    expect(run('sync').status).toBe(1);
    expect(await outputIdentity()).toEqual(before);
    expect(await readdir(root)).not.toContain('.code-rules-transaction');
  },
);

test('active writers and read-only checks reject concurrent operations', async () => {
  await withWriter(root, async () => {
    expect(run('sync').status).toBe(1);
    await expect(
      checkProject({ configPath: join(root, 'config.json') }),
    ).rejects.toThrow('write is active');
  });
});

test('cancellation leaves existing output untouched', async () => {
  succeeded(run('sync'));
  const before = await outputIdentity();
  await expect(
    buildProject({
      configPath: join(root, 'config.json'),
      signal: AbortSignal.abort(),
    }),
  ).rejects.toThrow('cancelled');
  expect(await outputIdentity()).toEqual(before);
});

test('concurrent-change rejection discards staging before any replacement', async () => {
  succeeded(run('sync'));
  const before = await outputIdentity();
  const baseline = await readProject(root, join(root, 'config.json'));
  await withWriter(root, async () => {
    await expect(
      applyChanges(
        root,
        { generated: new Map([['new.md', Buffer.from('new')]]) },
        async () => {
          await writeFile(
            join(root, 'config.json'),
            Buffer.concat([baseline.configBytes, Buffer.from('\n')]),
          );
          await assertUnchanged(root, join(root, 'config.json'), baseline, {});
        },
      ),
    ).rejects.toThrow('changed during the operation');
  });
  expect(await outputIdentity()).toEqual(before);
});

test('recovery preserves edits made after interruption and retains the previous backup', async () => {
  succeeded(run('sync'));
  expect(run('sync', 'crash').signal).toBe('SIGKILL');
  await writeFile(
    join(root, 'vendor/team/NOTICE.txt'),
    'User edit after interruption',
  );
  expect(run('sync').status).toBe(1);
  expect(await readFile(join(root, 'vendor/team/NOTICE.txt'), 'utf8')).toBe(
    'User edit after interruption',
  );
  expect(
    await readFile(
      join(root, '.code-rules-transaction/old-vendor/team/NOTICE.txt'),
      'utf8',
    ),
  ).toBe('Original fixture attribution.\r\n');
});

test('local-only projects need no imported libraries and preserve local files', async () => {
  await writeFile(
    join(root, 'config.json'),
    JSON.stringify({
      schemaVersion: 1,
      sources: {},
    }),
  );
  await mkdir(join(root, 'local/practices/testing'), { recursive: true });
  const group = exampleFiles['practices/testing/_group.json'];
  if (group === undefined) throw new Error('Missing fixture group');
  await writeFile(join(root, 'local/practices/testing/_group.json'), group);
  await writeFile(
    join(root, 'local/practices/testing/example.md'),
    '---\ntitle: Example\nwhenToRead: Changing behavior.\nimpact: HIGH\nimpactDescription: avoid mistakes\n---\n\nVerify behavior.\n',
  );
  succeeded(run('sync'));
  succeeded(run('check'));
  expect(await readdir(join(root, 'vendor'))).toEqual([]);
  expect(
    await readFile(
      join(root, 'generated/rules/local/practices/testing/example.md'),
      'utf8',
    ),
  ).toContain('Verify behavior.');
});

test('source records reject traversal digests without reading outside the library', async () => {
  succeeded(run('sync'));
  const path = join(root, 'vendor/team/_source.json');
  const record = JSON.parse(await readFile(path, 'utf8'));
  record.files['../../config.json'] = '0'.repeat(64);
  await writeFile(path, JSON.stringify(record));
  const before = await outputIdentity();
  expect(run('build').status).toBe(1);
  expect(await outputIdentity()).toEqual(before);
});

test('rejects an individual symlink and a case-colliding output before modifying live files', async () => {
  succeeded(run('sync'));
  await symlink(join(root, 'config.json'), join(root, 'generated/link.json'));
  expect(run('build').status).toBe(1);
  await rm(join(root, 'generated/link.json'));
  const before = await outputIdentity();
  await withWriter(root, async () => {
    await expect(
      applyChanges(
        root,
        {
          generated: new Map([
            ['A/one.md', Buffer.from('one')],
            ['a/two.md', Buffer.from('two')],
          ]),
        },
        async () => {},
      ),
    ).rejects.toThrow('collide');
  });
  expect(await outputIdentity()).toEqual(before);
});

test('local binary assets retain links without becoming text or adopted rules', async () => {
  succeeded(run('sync'));
  await mkdir(join(root, 'local/practices/testing/assets/local-example'), {
    recursive: true,
  });
  await writeFile(
    join(root, 'local/practices/testing/local-example.md'),
    '---\ntitle: Local example\nwhenToRead: Changing retries.\nimpact: HIGH\nimpactDescription: avoids mistakes\n---\n\nRead [sample](assets/local-example/sample.bin).\n',
  );
  const path = join(
    root,
    'local/practices/testing/assets/local-example/sample.bin',
  );
  await writeFile(path, new Uint8Array([0, 255, 128]));
  succeeded(run('build'));
  expect(
    await readFile(
      join(root, 'generated/rules/local/practices/testing/local-example.md'),
      'utf8',
    ),
  ).toContain('local/practices/testing/assets/local-example/sample.bin');
  expect(await readFile(path)).toEqual(Buffer.from([0, 255, 128]));
  succeeded(run('check'));
});

test('invalid UTF-8 local rules fail rather than silently disappearing', async () => {
  succeeded(run('sync'));
  await mkdir(join(root, 'local/practices/testing'), { recursive: true });
  await writeFile(
    join(root, 'local/practices/testing/invalid.md'),
    new Uint8Array([255]),
  );
  const before = await outputIdentity();
  expect(run('build').status).toBe(1);
  expect(await outputIdentity()).toEqual(before);
});

test('an interrupted staging-only operation can be discarded without touching live output', async () => {
  succeeded(run('sync'));
  const before = await outputIdentity();
  await mkdir(join(root, '.code-rules-transaction/new-vendor'), {
    recursive: true,
  });
  await writeFile(
    join(root, '.code-rules-transaction/new-vendor/partial'),
    'unfinished',
  );
  expect(run('check').status).toBe(1);
  succeeded(run('build'));
  expect(await outputIdentity()).toEqual(before);
});

test('unsupported snapshot versions fail offline while sync can refresh the record', async () => {
  succeeded(run('sync'));
  const path = join(root, 'vendor/team/_source.json');
  const record = JSON.parse(await readFile(path, 'utf8'));
  record.formatVersion = 99;
  await writeFile(path, JSON.stringify(record));
  expect(run('build').status).toBe(1);
  succeeded(run('sync'));
  succeeded(run('check'));
});

test('interrupted cleanup keeps committed output and never restores the old revision', async () => {
  succeeded(run('sync'));
  await writeFile(
    join(library.directory, 'NOTICE.txt'),
    'Committed new notice',
  );
  fixtureGit(library.directory, ['add', '.']);
  fixtureGit(library.directory, ['commit', '-m', 'New notice']);
  fixtureGit(library.directory, ['tag', '-f', 'v1']);
  expect(run('sync', 'cleanup-crash').signal).toBe('SIGKILL');
  const committed = await outputIdentity();
  await rm(library.directory, { recursive: true });
  expect(run('sync').status).toBe(1);
  expect(await outputIdentity()).toEqual(committed);
  succeeded(run('check'));
});

test('CLI defaults to .code-rules while explicit config still supports another location', async () => {
  const directory = join(root, '.code-rules');
  await mkdir(directory);
  await writeFile(
    join(directory, 'config.json'),
    JSON.stringify({ schemaVersion: 1, sources: {} }),
  );
  const cli = fileURLToPath(new URL('./cli.ts', import.meta.url));
  for (const command of ['sync', 'build', 'check']) {
    succeeded(
      spawnSync(process.execPath, [cli, command], {
        cwd: root,
        env: fixture.env,
        encoding: 'utf8',
        timeout: 30000,
      }),
    );
  }
  expect(
    await readFile(join(directory, 'generated/RULES.md'), 'utf8'),
  ).toContain('# Code Rules');
  await expect(readFile(join(root, 'generated/RULES.md'))).rejects.toThrow();
  succeeded(run('sync'));
  expect(await readFile(join(root, 'generated/RULES.md'), 'utf8')).toContain(
    '# Code Rules',
  );
});

test('offline commands accept snapshots with omitted group selection', async () => {
  succeeded(run('sync'));
  const path = join(root, 'vendor/team/_source.json');
  const record = JSON.parse(await readFile(path, 'utf8'));
  delete record.groupSelection;
  await writeFile(path, JSON.stringify(record));
  const before = await outputIdentity();
  await rm(library.directory, { recursive: true });
  succeeded(run('build'));
  succeeded(run('check'));
  expect(await outputIdentity()).toEqual(before);
});

test('a writer without permission to probe the lock owner reports busy and preserves files', async () => {
  succeeded(run('sync'));
  const before = await outputIdentity();
  const lock = join(root, '.code-rules-lock');
  await mkdir(lock);
  const owner = JSON.stringify({
    host: hostname(),
    pid: process.pid,
    token: 'live-owner',
  });
  await writeFile(join(lock, 'owner.json'), owner);
  const result = run('build', 'probe-permission');
  expect(result.status).toBe(1);
  expect(String(result.stderr)).toContain(
    'Another writer is using this project.',
  );
  expect(await readFile(join(lock, 'owner.json'), 'utf8')).toBe(owner);
  expect(await readdir(lock)).toEqual(['owner.json']);
  expect(await outputIdentity()).toEqual(before);
});

test('local groups survive adding and removing a library without changing local files', async () => {
  const group = 'practices/testing';
  await writeFile(
    join(root, 'config.json'),
    JSON.stringify({ schemaVersion: 1, sources: {} }),
  );
  await mkdir(join(root, 'local', group), { recursive: true });
  await writeFile(
    join(root, 'local', group, '_group.json'),
    JSON.stringify({
      name: 'Project testing',
      description: 'Test project contracts.',
      whenToRead: ['Before changing any project behavior.'],
    }),
  );
  await writeFile(
    join(root, 'local', group, 'budget.md'),
    '---\ntitle: Bound attempts\nwhenToRead: When changing retries.\nimpact: HIGH\nimpactDescription: Avoid excess requests.\n---\n\nUse three attempts.\n',
  );
  await writeFile(
    join(root, 'local/README.md'),
    '# Local rules\n\nAuthor project rules here.',
  );
  const before = treeDigest(await readTree(join(root, 'local')));
  succeeded(run('build'));
  await writeFile(
    join(root, 'config.json'),
    JSON.stringify({
      schemaVersion: 1,
      sources: { team: fixtureSource('team') },
    }),
  );
  succeeded(run('sync'));
  const index = await readFile(join(root, 'generated/RULES.md'), 'utf8');
  expect(index).toContain('Project testing');
  expect(index).toContain('Before changing any project behavior.');
  expect(index).not.toContain('Changing behavior.');
  const provenance = JSON.parse(
    await readFile(join(root, 'generated/provenance.json'), 'utf8'),
  );
  expect(
    provenance.rules.map((rule: { id: string }) => rule.id).sort(),
  ).toEqual([
    'local:practices/testing/budget',
    'team:practices/testing/verify-retries',
  ]);
  expect(
    provenance.groups[0].guidance.map(
      (entry: { source: string }) => entry.source,
    ),
  ).toEqual(['team', 'local']);
  expect(provenance.groups[0].effectiveGuidanceSources).toEqual(['local']);
  await writeFile(
    join(root, 'config.json'),
    JSON.stringify({ schemaVersion: 1, sources: {} }),
  );
  succeeded(run('sync'));
  succeeded(run('check'));
  expect(treeDigest(await readTree(join(root, 'local')))).toBe(before);
  const resolved = await readTree(join(root, 'generated/rules'));
  if (resolved === null) throw new Error('Expected generated rules directory');
  expect([...resolved.files.keys()].sort()).toEqual([
    'README.md',
    'local/practices/testing/budget.md',
  ]);
});
