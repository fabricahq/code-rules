/** @fileoverview Compares Go release selection with the pinned importer using a real local Git repository. */
import { deepStrictEqual, equal } from 'node:assert';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { selectVersion } from '../../src/imports/version-selection';
import { ImportError } from '../../src/imports/errors';

const candidate = process.argv[2];
if (!candidate) throw new Error('Expected Go adapter path');
const root = await mkdtemp(join(tmpdir(), 'rules-selection-'));
/** Run bounded Git commands only in the test-owned repository. */
function git(...args: string[]): string {
  const result = Bun.spawnSync(
    [
      'git',
      '-c',
      'core.hooksPath=/dev/null',
      '-c',
      'user.name=Fixture',
      '-c',
      'user.email=fixture@example.invalid',
      '-c',
      'commit.gpgsign=false',
      ...args,
    ],
    { cwd: root, stdout: 'pipe', stderr: 'pipe', timeout: 10_000 },
  );
  if (result.exitCode !== 0) throw new Error(result.stderr.toString());
  return result.stdout.toString().trim();
}
/** Feed the real advertisement to native Go and read its serialized result. */
function native(constraint: string): Record<string, unknown> {
  const input = { advertisement: git('ls-remote', '--tags', root), constraint };
  const result = Bun.spawnSync([resolve(candidate!)], {
    stdin: Buffer.from(
      JSON.stringify({
        operation: 'selectVersion',
        input,
        location: 'selection',
      }) + '\n',
    ),
    stdout: 'pipe',
    stderr: 'pipe',
    timeout: 10_000,
  });
  equal(result.exitCode, 0, result.stderr.toString());
  return JSON.parse(result.stdout.toString());
}
try {
  git('init', '--initial-branch=main');
  git('config', 'protocol.file.allow', 'always');
  git('commit', '--allow-empty', '-m', 'first');
  git('tag', 'v1.2.3');
  git('tag', '-a', '1.2.3', '-m', 'annotated alias');
  const constraint = '>= 1.0.0';
  const reference = await selectVersion(
    root,
    root,
    constraint,
    AbortSignal.timeout(10_000),
  );
  equal(reference.tag, '1.2.3');
  equal(reference.object, git('rev-parse', 'refs/tags/1.2.3'));
  deepStrictEqual(native(constraint), { ok: true, value: reference });
  git('commit', '--allow-empty', '-m', 'second');
  git('tag', 'v1.2.3+other');
  try {
    await selectVersion(root, root, constraint, AbortSignal.timeout(10_000));
    throw new Error('Reference accepted ambiguous aliases');
  } catch (error) {
    if (!(error instanceof ImportError) || error.code !== 'ambiguous-version')
      throw error;
  }
  const result = native(constraint);
  equal(result.ok, false);
  equal((result.error as { code: string }).code, 'ambiguous-version');
  console.log(
    'PASS: real Git annotated aliases and ambiguous-version selection',
  );
} finally {
  await rm(root, { recursive: true, force: true });
}
