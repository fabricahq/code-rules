/** @fileoverview Exercises library authoring and checking through the CLI, including importing authored content into a project. */

import { beforeEach, afterEach, test, expect } from 'bun:test';
import { mkdir, writeFile, readFile, readdir, symlink } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import {
  createImportFixture,
  removeImportFixture,
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
  'Check behavior.',
  '--when-to-read',
  'Before changing behavior.',
];
const rule = [
  '--title',
  'Bound retries',
  '--when-to-read',
  'Before designing or reviewing retries.',
  '--impact',
  'HIGH',
  '--impact-description',
  'Prevent excessive requests.',
];
beforeEach(async () => {
  fixture = await createImportFixture();
  root = join(fixture.root, 'team.git');
  await mkdir(root);
});
afterEach(async () => {
  await removeImportFixture(fixture);
});
function run(args: string[], cwd = root): ReturnType<typeof spawnSync> {
  return spawnSync(process.execPath, [cli, ...args], {
    cwd,
    env: fixture.env,
    encoding: 'utf8',
    timeout: 30000,
  });
}
function ok(args: string[], cwd = root): string {
  const result = run(args, cwd);
  expect(result.stderr).toBe('');
  expect(result.status).toBe(0);
  return String(result.stdout);
}
function git(args: string[]): void {
  const result = spawnSync('git', args, {
    cwd: root,
    env: fixture.env,
    encoding: 'utf8',
  });
  expect(result.status).toBe(0);
}
async function completed(): Promise<void> {
  ok(['library', 'init']);
  ok(['library', 'add', 'group', 'practices/testing', ...group]);
  await writeFile(
    join(fixture.root, 'body.md'),
    'Allow at most three attempts.\n',
  );
  ok([
    'library',
    'add',
    'rule',
    'practices/testing/budget',
    ...rule,
    '--body-file',
    join(fixture.root, 'body.md'),
  ]);
}

test('authors a licensed library, imports exact terms and rules, and checks offline', async () => {
  const license = 'Original test library terms.\r\n';
  const notice = 'Original author notice.\r\n';
  await writeFile(join(fixture.root, 'license.txt'), license);
  await writeFile(join(fixture.root, 'notice.txt'), notice);
  ok([
    'library',
    'init',
    '--spdx',
    'LicenseRef-Test',
    '--license-file',
    join(fixture.root, 'license.txt'),
    '--notice-file',
    join(fixture.root, 'notice.txt'),
  ]);
  ok(['library', 'add', 'group', 'practices/testing', ...group]);
  await writeFile(
    join(fixture.root, 'body.md'),
    'Bound retries. [Cases](assets/budget/cases.md)\n',
  );
  ok([
    'library',
    'add',
    'rule',
    'practices/testing/budget',
    ...rule,
    '--body-file',
    join(fixture.root, 'body.md'),
  ]);
  await mkdir(join(root, 'practices/testing/assets/budget'), {
    recursive: true,
  });
  await writeFile(
    join(root, 'practices/testing/assets/budget/cases.md'),
    'Check exhausted retries.\n',
  );
  const before = treeDigest(await readTree(root));
  expect(JSON.parse(ok(['library', 'check']))).toEqual({
    groups: 1,
    rules: 1,
    warnings: [],
  });
  expect(treeDigest(await readTree(root))).toBe(before);
  git(['init', '--quiet']);
  git(['add', '.']);
  git([
    '-c',
    'user.name=Test',
    '-c',
    'user.email=test@example.com',
    'commit',
    '-qm',
    'Library',
  ]);
  git(['tag', 'v1.0.0']);
  const consumer = join(fixture.root, 'consumer');
  await mkdir(consumer);
  ok(['init'], consumer);
  ok(
    [
      'add',
      'source',
      'team',
      '--repository',
      'https://github.com/fixture/team.git',
      '--version',
      '^1.0.0',
      '--groups',
      '*',
    ],
    consumer,
  );
  ok(['sync'], consumer);
  ok(['check'], consumer);
  expect(
    await readFile(
      join(
        consumer,
        '.code-rules/generated/libraries/team/licenses/LICENSE.md',
      ),
      'utf8',
    ),
  ).toBe(license);
  expect(
    await readFile(
      join(
        consumer,
        '.code-rules/generated/libraries/team/licenses/notices/001.md',
      ),
      'utf8',
    ),
  ).toBe(notice);
  expect(
    await readFile(
      join(
        consumer,
        '.code-rules/generated/rules/team/practices/testing/budget.md',
      ),
      'utf8',
    ),
  ).toContain('Bound retries.');
});

test('empty initialized library and empty group are valid; repeated init preserves authored README', async () => {
  ok(['library', 'init']);
  expect(JSON.parse(ok(['library', 'check']))).toMatchObject({
    groups: 0,
    rules: 0,
    warnings: [expect.stringContaining('undeclared')],
  });
  await writeFile(join(root, 'README.md'), 'My library.\n');
  ok(['library', 'init']);
  expect(await readFile(join(root, 'README.md'), 'utf8')).toBe('My library.\n');
  ok(['library', 'add', 'group', 'practices/testing', ...group]);
  expect(JSON.parse(ok(['library', 'check']))).toMatchObject({
    groups: 1,
    rules: 0,
  });
});

test('creates a missing group with a rule and requires draft completion', async () => {
  ok(['library', 'init']);
  ok([
    'library',
    'add',
    'rule',
    'practices/testing/budget',
    ...rule,
    '--create-group',
    '--group-name',
    'Testing',
    '--group-description',
    'Check behavior.',
    '--group-when-to-read',
    'Before behavior changes.',
  ]);
  const result = run(['library', 'check']);
  expect(result.status).toBe(1);
  expect(result.stderr).toContain('complete the draft');
});

test.each([
  ['unknown SPDX', ['--spdx', 'NotAnSPDX', '--license-file', 'terms.txt']],
  ['missing license input', ['--spdx', 'MIT']],
  ['notice without license', ['--notice-file', 'terms.txt']],
] as const)('rejects %s before publishing', async (_label, flags) => {
  await writeFile(join(root, 'terms.txt'), 'Terms.');
  expect(run(['library', 'init', ...flags]).status).not.toBe(0);
  expect(await readdir(root)).toEqual(['terms.txt']);
});

test('preserves existing license file on a collision', async () => {
  await writeFile(join(root, 'LICENSE.md'), 'Existing terms.');
  await writeFile(join(fixture.root, 'terms.txt'), 'New terms.');
  expect(
    run([
      'library',
      'init',
      '--spdx',
      'MIT',
      '--license-file',
      join(fixture.root, 'terms.txt'),
    ]).status,
  ).toBe(1);
  expect(await readdir(root)).toEqual(['LICENSE.md']);
  expect(await readFile(join(root, 'LICENSE.md'), 'utf8')).toBe(
    'Existing terms.',
  );
});

test.each([
  [
    'orphan rule',
    'practices/missing/rule.md',
    '---\ntitle: Rule\nwhenToRead: When coding.\nimpact: HIGH\nimpactDescription: Avoid failure.\n---\nDo the thing.',
  ],
  ['unsupported group content', 'practices/testing/extra.txt', 'Unsupported'],
  [
    'missing asset',
    'practices/testing/missing.md',
    '---\ntitle: Rule\nwhenToRead: When coding.\nimpact: HIGH\nimpactDescription: Avoid failure.\n---\n[Missing](assets/missing/file.md)',
  ],
  ['orphan asset', 'practices/testing/assets/orphan/file.txt', 'No owner'],
] as const)('rejects %s without writing files', async (_label, path, text) => {
  await completed();
  await mkdir(join(root, path, '..'), { recursive: true });
  await writeFile(join(root, path), text);
  const before = treeDigest(await readTree(root));
  expect(run(['library', 'check']).status).toBe(1);
  expect(treeDigest(await readTree(root))).toBe(before);
});

test('ignores unrelated repository files but rejects a linked declared license parent', async () => {
  await completed();
  await mkdir(join(root, 'node_modules'));
  await symlink('/dev/null', join(root, 'node_modules/unrelated'));
  ok(['library', 'check']);
  await symlink(fixture.root, join(root, 'legal'));
  await writeFile(join(fixture.root, 'license.md'), 'External');
  await writeFile(
    join(root, 'rule-library.json'),
    JSON.stringify({
      formatVersion: 1,
      license: { spdxExpression: 'MIT', file: 'legal/license.md', notices: [] },
    }),
  );
  expect(run(['library', 'check']).stderr).toContain('without symlinks');
});

test('uses an explicit library directory and rejects project configuration options', async () => {
  ok(['library', 'init', '--directory', root], fixture.root);
  expect(run(['library', 'check', '--config', 'unused']).status).toBe(2);
  expect(
    run(['library', 'add', 'group', '../escape', ...group]).status,
  ).not.toBe(0);
});

test.each(['init', 'group', 'rule'] as const)(
  'cancellation during library %s leaves authored files unchanged',
  async (kind) => {
    if (kind !== 'init') ok(['library', 'init']);
    if (kind === 'rule')
      ok(['library', 'add', 'group', 'practices/testing', ...group]);
    const before = treeDigest(await readTree(root));
    const worker = fileURLToPath(
      new URL('../../tests/fixtures/authoring-race-worker.ts', import.meta.url),
    );
    const args =
      kind === 'init'
        ? ['library', 'init']
        : kind === 'group'
          ? ['library', 'add', 'group', 'practices/testing', ...group]
          : ['library', 'add', 'rule', 'practices/testing/budget', ...rule];
    const result = spawnSync(process.execPath, [worker, ...args], {
      cwd: root,
      env: { ...fixture.env, AUTHORING_FAULT: 'cancel' },
      encoding: 'utf8',
      timeout: 30000,
    });
    expect(result.status).toBe(1);
    expect(result.stderr).toContain('cancelled');
    expect(treeDigest(await readTree(root))).toBe(before);
  },
);
