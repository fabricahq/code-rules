/** @fileoverview Verifies a packed CLI installed in an isolated prefix, without importing application source or using checkout dependencies at runtime. */

import { beforeAll, afterAll, beforeEach, test, expect } from 'bun:test';
import { spawnSync } from 'node:child_process';
import { mkdir, readFile, writeFile, readdir } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import {
  createImportFixture,
  removeImportFixture,
  addLibrary,
  exampleFiles,
  fixtureGit,
} from '../fixtures/import-fixture';
import type { ImportFixture } from '../fixtures/import-fixture';
import { readTree, treeDigest } from '../../src/project-files/files';
import metadata from '../../package.json';

let fixture: ImportFixture;
let binary: string;
let prefix: string;
let tarball: string;
let project: string;
let sequence = 0;
const repo = resolve(import.meta.dir, '../..');

function command(
  executable: string,
  args: string[],
  cwd = repo,
  env = process.env,
): ReturnType<typeof spawnSync> {
  return spawnSync(executable, args, {
    cwd,
    env,
    encoding: 'utf8',
    timeout: 120000,
  });
}
function succeeds(result: ReturnType<typeof spawnSync>): string {
  if (result.status !== 0)
    throw new Error(
      `Command failed (${result.status}): ${result.stderr}\n${result.stdout}`,
    );
  return String(result.stdout);
}
function cli(args: string[], cwd = project): string {
  return succeeds(command(binary, args, cwd, fixture.env));
}
function install(): void {
  succeeds(
    command('npm', [
      'install',
      '--global',
      '--prefix',
      prefix,
      '--cache',
      join(fixture.root, 'npm-cache'),
      '--ignore-scripts',
      '--no-audit',
      '--no-fund',
      tarball,
    ]),
  );
}

beforeAll(async () => {
  fixture = await createImportFixture();
  prefix = join(fixture.root, 'installation');
  if (process.env.CODE_RULES_TARBALL)
    tarball = resolve(process.env.CODE_RULES_TARBALL);
  else {
    succeeds(command('bun', ['run', 'build']));
    succeeds(
      command('npm', [
        'pack',
        '--ignore-scripts',
        '--pack-destination',
        fixture.root,
      ]),
    );
    const name = (await readdir(fixture.root)).find((name) =>
      name.endsWith('.tgz'),
    );
    if (!name) throw new Error('npm pack did not produce a tarball.');
    tarball = join(fixture.root, name);
  }
  install();
  binary = join(prefix, 'bin/code-rules');
}, 120000);
afterAll(async () => {
  if (fixture) await removeImportFixture(fixture);
});
beforeEach(async () => {
  project = join(fixture.root, `project-${++sequence}`);
  await mkdir(project);
});

const group = [
  '--name',
  'Testing',
  '--description',
  'Verify behavior.',
  '--when-to-read',
  'Before changing behavior.',
];
const rule = [
  '--title',
  'Limit attempts',
  '--when-to-read',
  'Before changing retries.',
  '--impact',
  'HIGH',
  '--impact-description',
  'Avoid excess requests.',
];

async function local(): Promise<void> {
  cli(['init']);
  cli(['local', 'add', 'group', 'practices/testing', ...group]);
  await writeFile(join(project, 'body.md'), 'Allow three attempts.\n');
  cli([
    'local',
    'add',
    'rule',
    'practices/testing/budget',
    ...rule,
    '--body-file',
    join(project, 'body.md'),
  ]);
  cli(['build']);
  cli(['check']);
}

test('installed executable reports its version and help without creating project files', async () => {
  expect(cli(['--version']).trim()).toBe(metadata.version);
  expect(cli(['library', 'init', '--help'])).toContain('library check');
  expect(await readdir(project)).toEqual([]);
});

test('installed package includes the canonical draft and runs the complete library-authoring workflow', async () => {
  cli(['library', 'init']);
  cli(['library', 'add', 'group', 'practices/testing', ...group]);
  cli(['library', 'add', 'rule', 'practices/testing/budget', ...rule]);
  const path = join(project, 'practices/testing/budget.md');
  expect(await readFile(path, 'utf8')).toContain('code-rules:draft');
  expect(command(binary, ['library', 'check'], project).status).toBe(1);
  const draft = await readFile(path, 'utf8');
  const end = draft.indexOf('\n---\n', 4) + 5;
  await writeFile(
    path,
    draft.slice(0, end) + '\nAllow at most three attempts.\n',
  );
  expect(JSON.parse(cli(['library', 'check']))).toMatchObject({
    groups: 1,
    rules: 1,
  });
});

test('installed CLI generates local-only rules and embeds its real version in provenance', async () => {
  await local();
  const provenance = JSON.parse(
    await readFile(
      join(project, '.code-rules/generated/provenance.json'),
      'utf8',
    ),
  );
  expect(provenance.toolVersion).toBe(metadata.version);
  expect(provenance.rules.map((rule: { id: string }) => rule.id)).toEqual([
    'local:practices/testing/budget',
  ]);
  expect(JSON.parse(cli(['build']))).toEqual({
    added: [],
    changed: [],
    removed: [],
  });
});

test('installed CLI imports licensed versioned sources, resolves exceptions, and then works offline', async () => {
  await local();
  const team = await addLibrary(fixture, 'team', exampleFiles);
  fixtureGit(team.directory, ['tag', 'v1.0.0']);
  await addLibrary(fixture, 'partner', exampleFiles);
  cli([
    'add',
    'source',
    'team',
    '--repository',
    'https://github.com/fixture/team.git',
    '--version',
    '^1.0.0',
    '--groups',
    'practices/testing',
  ]);
  cli([
    'add',
    'source',
    'partner',
    '--repository',
    'https://github.com/fixture/partner.git',
    '--ref',
    'v1',
    '--groups',
    'practices/testing',
  ]);
  cli(['sync']);
  const configPath = join(project, '.code-rules/config.json');
  const config = JSON.parse(await readFile(configPath, 'utf8'));
  config.sources.team.replace['practices/testing/verify-retries'] = {
    file: 'local/practices/testing/budget.md',
    reason: 'Our API permits three attempts.',
  };
  config.sources.partner.exclude['practices/testing/verify-retries'] =
    'Use our project rule.';
  await writeFile(configPath, JSON.stringify(config, null, 2));
  cli(['build']);
  cli(['check']);
  const provenance = JSON.parse(
    await readFile(
      join(project, '.code-rules/generated/provenance.json'),
      'utf8',
    ),
  );
  expect(provenance.rules.map((rule: { id: string }) => rule.id)).toEqual([
    'local:practices/testing/budget',
  ]);
  const expectedLicense = exampleFiles['LICENSE.txt'];
  if (typeof expectedLicense !== 'string')
    throw new Error('Expected textual fixture license.');
  expect(
    await readFile(
      join(project, '.code-rules/generated/libraries/team/licenses/LICENSE.md'),
      'utf8',
    ),
  ).toBe(expectedLicense);
  const offline = {
    ...fixture.env,
    GIT_CONFIG_VALUE_0: 'https://unused.invalid/',
  };
  succeeds(command(binary, ['check'], project, offline));
});

test('stale output exits nonzero without writing, while build repairs it', async () => {
  await local();
  const path = join(project, '.code-rules/generated/RULES.md');
  await writeFile(path, 'Stale output.');
  expect(command(binary, ['check'], project).status).toBe(1);
  expect(await readFile(path, 'utf8')).toBe('Stale output.');
  cli(['build']);
  cli(['check']);
});

test('reinstalling the packed executable preserves a committed project and its generated files', async () => {
  await local();
  succeeds(command('git', ['init', '--quiet'], project, fixture.env));
  succeeds(command('git', ['add', '.'], project, fixture.env));
  succeeds(
    command(
      'git',
      [
        '-c',
        'user.name=Test',
        '-c',
        'user.email=test@example.com',
        'commit',
        '-qm',
        'Adopt local rules',
      ],
      project,
      fixture.env,
    ),
  );
  const before = treeDigest(await readTree(join(project, '.code-rules')));
  install();
  cli(['check']);
  expect(treeDigest(await readTree(join(project, '.code-rules')))).toBe(before);
  expect(
    succeeds(command('git', ['status', '--porcelain'], project, fixture.env)),
  ).toBe('');
});
