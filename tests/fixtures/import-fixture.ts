/** @fileoverview Creates original Git libraries and runs Imports in isolated child environments for tests and manual inspection. */

import { mkdtemp, mkdir, writeFile, rm } from 'node:fs/promises';
import { spawnSync } from 'node:child_process';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

/** Original rule corpus shared by the integration tests and manual scenario. */
export const exampleFiles: Readonly<Record<string, string | Uint8Array>> = {
  'rule-library.json': JSON.stringify({
    formatVersion: 1,
    license: {
      spdxExpression: 'CC0-1.0',
      file: 'LICENSE.txt',
      notices: ['NOTICE.txt'],
    },
  }),
  'LICENSE.txt':
    'Original fixture content, released under CC0 for testing.\r\n',
  'NOTICE.txt': 'Original fixture attribution.\r\n',
  'assets/special.pdf': new Uint8Array([0, 255, 128, 10]),
  'practices/testing/_group.json': JSON.stringify({
    name: 'Testing',
    description: 'Check behavior.',
    whenToRead: ['Changing behavior.'],
  }),
  'practices/testing/verify-retries.md':
    '---\ntitle: Verify retries\nwhenToRead: When planning, changing, or reviewing retry behavior.\nimpact: HIGH\nimpactDescription: prevents excess requests\ntags: testing, retry\n---\n\nCheck retry limits.\n\n![Flow](../../assets/images/flow.png)\n[Attachment](../../assets/special.pdf)\n[More](../../assets/notes/explanation.md)\n',
  'assets/images/flow.png': new Uint8Array([137, 80, 78, 71, 255, 0, 10]),
  'assets/notes/explanation.md':
    '[Terms](../extra.txt)\n[Cycle](explanation.md)',
  'assets/extra.txt': 'Additional original fixture terms.',
  'practices/logging/example.md':
    '---\ntitle: Logging\nwhenToRead: When logging.\nimpact: HIGH\nimpactDescription: Keep diagnostics.\n---\nLog failures.\n',
  'unselected.txt': 'Do not import this file.',
};

/** Temporary repositories plus isolated Git URL routing; cleanup belongs to the caller. */
export type ImportFixture = {
  readonly root: string;
  readonly env: NodeJS.ProcessEnv;
};

/** Run fixture Git commands without invoking repository hooks or inheriting author configuration. */
export function fixtureGit(
  directory: string,
  args: ReadonlyArray<string>,
): string {
  const result = spawnSync(
    'git',
    [
      '-c',
      'core.hooksPath=/dev/null',
      '-c',
      'core.precomposeUnicode=false',
      '-c',
      'user.name=Code Rules Tests',
      '-c',
      'user.email=tests@example.invalid',
      ...args,
    ],
    {
      cwd: directory,
      encoding: 'utf8',
      env: {
        ...process.env,
        GIT_CONFIG_NOSYSTEM: '1',
        GIT_CONFIG_GLOBAL: '/dev/null',
      },
    },
  );
  if (result.status !== 0)
    throw new Error(`Fixture Git failed: ${result.stderr}`);
  return result.stdout.trim();
}

/** Create URL routing only in the child environment, leaving the user's Git configuration untouched. */
export async function createImportFixture(): Promise<ImportFixture> {
  const root = await mkdtemp(join(tmpdir(), 'code-rules-fixture-'));
  const scratch = join(root, 'scratch');
  await mkdir(scratch);
  return {
    root,
    env: {
      ...process.env,
      TMPDIR: scratch,
      GIT_CONFIG_NOSYSTEM: '1',
      GIT_CONFIG_GLOBAL: '/dev/null',
      GIT_CONFIG_COUNT: '2',
      GIT_CONFIG_KEY_0: `url.file://${root}/.insteadOf`,
      GIT_CONFIG_VALUE_0: 'https://github.com/fixture/',
      GIT_CONFIG_KEY_1: 'protocol.file.allow',
      GIT_CONFIG_VALUE_1: 'always',
    },
  };
}

/** Commit a library with a v1 tag and return its directory and exact commit. */
export async function addLibrary(
  fixture: ImportFixture,
  name: string,
  files: Readonly<Record<string, string | Uint8Array>> = exampleFiles,
): Promise<{ readonly directory: string; readonly commit: string }> {
  const directory = join(fixture.root, `${name}.git`);
  await mkdir(directory);
  fixtureGit(directory, ['init', '--quiet', '--template=']);
  for (const [path, content] of Object.entries(files)) {
    await mkdir(dirname(join(directory, path)), { recursive: true });
    await writeFile(join(directory, path), content);
  }
  fixtureGit(directory, ['add', '.']);
  fixtureGit(directory, [
    'commit',
    '--quiet',
    '-m',
    'Original library fixture',
  ]);
  fixtureGit(directory, ['tag', 'v1']);
  return { directory, commit: fixtureGit(directory, ['rev-parse', 'HEAD']) };
}

/** Return a complete source selection for a fixture library. */
export function fixtureSource(
  name: string,
  ref = 'v1',
): Record<string, unknown> {
  return {
    repository: `https://github.com/fixture/${name}.git`,
    ref,
    groups: ['practices/testing'],
    exclude: {},
    replace: {},
  };
}

/** Execute the public Imports API in a separate process with fixture-only transport routing. */
export function runImport(
  fixture: ImportFixture,
  configuration: unknown,
  options: Record<string, unknown> = {},
  env: NodeJS.ProcessEnv = fixture.env,
): unknown {
  const worker = fileURLToPath(new URL('./import-worker.ts', import.meta.url));
  const result = spawnSync(
    process.execPath,
    [worker, JSON.stringify({ configuration, ...options })],
    { env, encoding: 'utf8', maxBuffer: 100 * 1024 * 1024, timeout: 20_000 },
  );
  if (result.status !== 0)
    throw new Error(`Import worker failed: ${result.stderr}`);
  return JSON.parse(result.stdout);
}

/** Remove all fixture repositories and any scratch files after a scenario. */
export async function removeImportFixture(
  fixture: ImportFixture,
): Promise<void> {
  await rm(fixture.root, { recursive: true, force: true });
}
