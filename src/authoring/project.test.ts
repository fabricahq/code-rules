/** @fileoverview Exercises project onboarding through the real CLI with local files and real Git fixtures. */

import { beforeEach, afterEach, expect, test } from 'bun:test';
import { mkdir, readFile, writeFile, readdir, symlink } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import {
  createImportFixture,
  removeImportFixture,
  addLibrary,
  exampleFiles,
} from '../../tests/fixtures/import-fixture';
import type { ImportFixture } from '../../tests/fixtures/import-fixture';
import { readTree, treeDigest } from '../project-files/files';

let fixture: ImportFixture;
let root: string;
const cli = fileURLToPath(new URL('../cli.ts', import.meta.url));
const group = [
  '--name',
  'Testing',
  '--description',
  'Verify project behavior.',
  '--when-to-read',
  'Before changing any behavior.',
];
const rule = [
  '--title',
  'Bound attempts',
  '--when-to-read',
  'When changing retries.',
  '--impact',
  'HIGH',
  '--impact-description',
  'Avoid excessive requests.',
];
beforeEach(async () => {
  fixture = await createImportFixture();
  root = join(fixture.root, 'consumer');
  await mkdir(root);
});
afterEach(async () => {
  await removeImportFixture(fixture);
});
function run(args: string[], fault = false): ReturnType<typeof spawnSync> {
  const script = fault
    ? fileURLToPath(
        new URL('../../tests/fixtures/authoring-worker.ts', import.meta.url),
      )
    : cli;
  return spawnSync(process.execPath, [script, ...args], {
    cwd: root,
    env: fixture.env,
    encoding: 'utf8',
    timeout: 30000,
  });
}
function ok(args: string[]): string {
  const result = run(args);
  expect(result.stderr).toBe('');
  expect(result.status).toBe(0);
  return String(result.stdout);
}
async function identity(): Promise<string> {
  return treeDigest(await readTree(root));
}

test('empty repo to local rules to licensed library with explicit sync and offline check', async () => {
  ok(['init']);
  ok(['local', 'add', 'group', 'practices/testing', ...group]);
  const body = join(root, 'body.md');
  await writeFile(body, 'Use at most three attempts.\n');
  ok([
    'local',
    'add',
    'rule',
    'practices/testing/budget',
    ...rule,
    '--body-file',
    body,
  ]);
  ok(['build']);
  ok(['check']);
  const localBefore = treeDigest(
    await readTree(join(root, '.code-rules/local')),
  );
  const generatedBefore = treeDigest(
    await readTree(join(root, '.code-rules/generated')),
  );
  await addLibrary(fixture, 'team', {
    ...exampleFiles,
    'practices/logging/_group.json': JSON.stringify({
      name: 'Logging',
      description: 'Diagnostic logging.',
      whenToRead: ['When adding diagnostics.'],
    }),
  });
  ok([
    'add',
    'source',
    'team',
    '--repository',
    'https://github.com/fixture/team.git',
    '--ref',
    'v1',
    '--groups',
    '*',
  ]);
  expect(await readTree(join(root, '.code-rules/vendor'))).toBeNull();
  expect(treeDigest(await readTree(join(root, '.code-rules/generated')))).toBe(
    generatedBefore,
  );
  ok(['sync']);
  ok(['check']);
  expect(treeDigest(await readTree(join(root, '.code-rules/local')))).toBe(
    localBefore,
  );
  const provenance = JSON.parse(
    await readFile(join(root, '.code-rules/generated/provenance.json'), 'utf8'),
  );
  expect(provenance.rules.map((entry: { id: string }) => entry.id)).toContain(
    'local:practices/testing/budget',
  );
  expect(provenance.rules.map((entry: { id: string }) => entry.id)).toContain(
    'team:practices/testing/verify-retries',
  );
  expect(
    await readFile(
      join(root, '.code-rules/generated/libraries/team/licenses/LICENSE.md'),
      'utf8',
    ),
  ).toContain('CC0');
});

test('repeat init preserves configuration, local README, and authored rules byte for byte', async () => {
  ok(['init']);
  await writeFile(
    join(root, '.code-rules/local/README.md'),
    '# Our local rules\r\n',
  );
  ok(['local', 'add', 'group', 'practices/testing', ...group]);
  const before = await identity();
  expect(JSON.parse(ok(['init'])).files).toEqual([]);
  expect(await identity()).toBe(before);
});

test('creates a draft and missing group together without inventing rule guidance', async () => {
  ok(['init']);
  const output = JSON.parse(
    ok([
      'local',
      'add',
      'rule',
      'techs/go/errors',
      ...rule,
      '--create-group',
      '--group-name',
      'Go',
      '--group-description',
      'Go programming.',
      '--group-when-to-read',
      'When writing Go.',
    ]),
  );
  const text = await readFile(
    join(root, '.code-rules/local/techs/go/errors.md'),
    'utf8',
  );
  expect(text).toContain('### Implementation');
  expect(text).toContain('### Validation');
  expect(text).toContain('**Incorrect (counterexample):**');
  expect(text).toContain('Repeat the application block');
  expect(output.next).toContain('Complete the draft');
});

test('adds a local rule to a verified imported group without creating local metadata', async () => {
  await addLibrary(fixture, 'team', {
    ...exampleFiles,
    'practices/logging/_group.json': JSON.stringify({
      name: 'Logging',
      description: 'Diagnostic logging.',
      whenToRead: ['When adding diagnostics.'],
    }),
  });
  ok(['init']);
  ok([
    'add',
    'source',
    'team',
    '--repository',
    'https://github.com/fixture/team.git',
    '--ref',
    'v1',
    '--groups',
    'practices/*',
  ]);
  ok(['sync']);
  ok(['local', 'add', 'rule', 'practices/testing/budget', ...rule]);
  expect(
    await readdir(join(root, '.code-rules/local/practices/testing')),
  ).toEqual(['budget.md']);
});

test('custom config locations support initialization, groups, rules, and source addition', async () => {
  const config = ['--config', 'metadata/nested/rules.json'];
  ok(['init', ...config]);
  ok(['local', 'add', 'group', 'techs/go', ...group, ...config]);
  ok(['local', 'add', 'rule', 'techs/go/errors', ...rule, ...config]);
  ok([
    'add',
    'source',
    'team',
    '--repository',
    'https://gitlab.com/example/rules.git',
    '--version',
    '^1.2.0',
    '--groups',
    'techs/*',
    ...config,
  ]);
  expect(await readTree(join(root, '.code-rules'))).toBeNull();
  const saved = JSON.parse(
    await readFile(join(root, 'metadata/nested/rules.json'), 'utf8'),
  );
  expect(saved.sources.team.version).toBe('^1.2.0');
});

test.each([
  ['local', 'add', 'group', 'practices/testing'],
  ['local', 'add', 'rule', 'practices/testing/budget', ...rule],
  ['local', 'add', 'group', 'practices/assets', ...group],
  ['local', 'add', 'group', 'practices/../elsewhere', ...group],
  [
    'local',
    'add',
    'rule',
    'practices/testing/budget',
    ...rule,
    '--impact',
    'LOW',
  ],
  [
    'add',
    'source',
    'team',
    '--repository',
    'https://github.com/example/rules.git',
    '--ref',
    'v1',
    '--version',
    '^1.0.0',
    '--groups',
    '*',
  ],
  [
    'add',
    'source',
    'team',
    '--repository',
    'https://github.com/example/rules.git',
    '--ref',
    'v1',
    '--groups',
    '*',
    '--groups',
    'techs/go',
  ],
  ['init', '--unknown'],
])(
  'invalid or incomplete invocation leaves authored files unchanged: %j',
  async (...args) => {
    ok(['init']);
    const before = await identity();
    const result = run(args);
    expect(result.status).not.toBe(0);
    expect(await identity()).toBe(before);
  },
);

test('existing rule prevents partial missing-group creation', async () => {
  ok(['init']);
  await mkdir(join(root, '.code-rules/local/techs/go'), { recursive: true });
  await writeFile(
    join(root, '.code-rules/local/techs/go/errors.md'),
    'Existing author work',
  );
  const before = await identity();
  expect(
    run([
      'local',
      'add',
      'rule',
      'techs/go/errors',
      ...rule,
      '--create-group',
      '--group-name',
      'Go',
      '--group-description',
      'Go code.',
      '--group-when-to-read',
      'Writing Go.',
    ]).status,
  ).not.toBe(0);
  expect(await identity()).toBe(before);
});

test('failed publication rolls back new group and rule files', async () => {
  ok(['init']);
  const before = await identity();
  const result = run(
    [
      'local',
      'add',
      'rule',
      'techs/go/errors',
      ...rule,
      '--create-group',
      '--group-name',
      'Go',
      '--group-description',
      'Go code.',
      '--group-when-to-read',
      'Writing Go.',
    ],
    true,
  );
  expect(result.status).toBe(1);
  expect(String(result.stderr)).toContain('Simulated publication failure');
  expect(await identity()).toBe(before);
});

test('rejects linked local directories without writing through them', async () => {
  ok(['init']);
  await mkdir(join(root, 'outside'));
  await symlink(join(root, 'outside'), join(root, '.code-rules/local/techs'));
  const before = await identity().catch(() => 'linked');
  expect(run(['local', 'add', 'group', 'techs/go', ...group]).status).toBe(1);
  expect(await readdir(join(root, 'outside'))).toEqual([]);
  expect(await identity().catch(() => 'linked')).toBe(before);
});

test('source addition preserves existing exceptions and rejects alias or repository duplication', async () => {
  ok(['init']);
  ok([
    'add',
    'source',
    'team',
    '--repository',
    'https://github.com/example/team.git',
    '--ref',
    'v1',
    '--groups',
    'techs/go',
    '--groups',
    'practices/testing',
  ]);
  const path = join(root, '.code-rules/config.json');
  const config = JSON.parse(await readFile(path, 'utf8'));
  config.sources.team.exclude = { 'techs/go/old': 'Project exception.' };
  await writeFile(path, JSON.stringify(config));
  ok([
    'add',
    'source',
    'partner',
    '--repository',
    'https://github.com/example/partner.git',
    '--version',
    '^2.0.0',
    '--groups',
    '*',
  ]);
  expect(JSON.parse(await readFile(path, 'utf8')).sources.team).toEqual(
    config.sources.team,
  );
  const before = await identity();
  for (const alias of ['partner', 'another']) {
    expect(
      run([
        'add',
        'source',
        alias,
        '--repository',
        'https://github.com/example/partner.git',
        '--ref',
        'v1',
        '--groups',
        '*',
      ]).status,
    ).toBe(1);
    expect(await identity()).toBe(before);
  }
});

test('distributed draft matches the canonical documentation template', async () => {
  const doc = await readFile(
    new URL(
      '../../docs/src/content/docs/reference/rule-authoring.md',
      import.meta.url,
    ),
    'utf8',
  );
  const template = await readFile(
    new URL('./rule-template.md', import.meta.url),
    'utf8',
  );
  const documented = doc.split('````md\n')[1]?.split('\n````')[0];
  if (documented === undefined) throw new Error('Missing canonical template.');
  expect(template.trim()).toBe(documented);
});

test('rejects case-colliding group paths before writing', async () => {
  ok(['init']);
  await mkdir(join(root, '.code-rules/local/Techs'));
  const before = await identity();
  expect(run(['local', 'add', 'group', 'techs/go', ...group]).status).toBe(1);
  expect(await identity()).toBe(before);
});

test('invalid impact and non-UTF-8 body input leave local files unchanged', async () => {
  ok(['init']);
  ok(['local', 'add', 'group', 'practices/testing', ...group]);
  const body = join(root, 'invalid-body.md');
  await writeFile(body, Buffer.from([255]));
  const before = await identity();
  const badImpact = rule.map((value) => (value === 'HIGH' ? 'URGENT' : value));
  expect(
    run(['local', 'add', 'rule', 'practices/testing/budget', ...badImpact])
      .status,
  ).toBe(1);
  expect(
    run([
      'local',
      'add',
      'rule',
      'practices/testing/budget',
      ...rule,
      '--body-file',
      body,
    ]).status,
  ).toBe(1);
  expect(await identity()).toBe(before);
});
