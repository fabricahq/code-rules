/** @fileoverview Proves the migration sensor accepts the reference and rejects incorrect executables, output values, and filesystem changes. */

import { test, expect, beforeAll, afterAll } from 'bun:test';
import { spawnSync } from 'node:child_process';
import { mkdtemp, writeFile, chmod, rm, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import {
  compareExecutions,
  compareTrees,
  execute,
} from '../tests/migration/observe';
import type { Execution } from '../tests/migration/observe';
import contracts from '../migration/contracts.json';
import { scenarios } from '../tests/migration/scenarios';

const baseline = contracts.referenceRevision;
const executable = resolve('dist/cli.js');
let root: string;
const retained: Array<string> = [];

beforeAll(async () => {
  root = await mkdtemp(join(tmpdir(), 'code-rules-sensor-test-'));
  const result = spawnSync('bun', ['run', 'build'], { encoding: 'utf8' });
  expect(result.status, result.stderr).toBe(0);
});
afterAll(async () => {
  for (const path of retained) await rm(path, { recursive: true, force: true });
  if (root) await rm(root, { recursive: true, force: true });
});

async function run(
  candidate: string,
  name: string,
): Promise<{
  readonly status: number | null;
  readonly report: Record<string, unknown>;
}> {
  const reportPath = join(root, `${name}.json`);
  const result = spawnSync(
    'bun',
    [
      'tests/migration/compare.ts',
      '--reference',
      executable,
      '--candidate',
      candidate,
      '--reference-revision',
      baseline,
      '--report',
      reportPath,
    ],
    { encoding: 'utf8', timeout: 90_000 },
  );
  expect(result.error).toBeUndefined();
  expect(result.status, result.stderr).not.toBe(2);
  const report: unknown = JSON.parse(await readFile(reportPath, 'utf8'));
  if (
    typeof report !== 'object' ||
    report === null ||
    !('evidenceRoot' in report) ||
    typeof report.evidenceRoot !== 'string'
  )
    throw new Error('Missing evidence root');
  retained.push(report.evidenceRoot);
  return { status: result.status, report: { ...report } };
}

async function mutant(name: string, mutation: string): Promise<string> {
  const path = join(root, `${name}.cjs`);
  await writeFile(
    path,
    `#!/usr/bin/env node\nconst {spawnSync}=require('node:child_process');\nconst r=spawnSync(process.execPath,[${JSON.stringify(executable)},...process.argv.slice(2)],{encoding:'utf8',env:process.env});\n${mutation}\nprocess.stdout.write(r.stdout);process.stderr.write(r.stderr);process.exit(r.status ?? 99);\n`,
  );
  await chmod(path, 0o755);
  return path;
}

test('should pass real reference comparisons and independent assertions in separate workspaces', async () => {
  const result = await run(executable, 'reference-self');
  expect(result.status, JSON.stringify(result.report.results)).toBe(0);
  expect(result.report.passed).toBe(true);
}, 90_000);

test('should reject a candidate that reports the wrong version output', async () => {
  const candidate = await mutant(
    'wrong-output',
    "if(process.argv.includes('--version')) r.stdout='incorrect-version\\n';",
  );
  const result = await run(candidate, 'wrong-output');
  expect(result.status).toBe(1);
  expect(result.report.passed).toBe(false);
  expect(JSON.stringify(result.report.results)).toContain('incorrect-version');
}, 90_000);

test('should reject a candidate that returns the wrong exit status', async () => {
  const candidate = await mutant(
    'wrong-status',
    "if(process.argv.includes('--version')) r.status=7;",
  );
  const result = await run(candidate, 'wrong-status');
  expect(result.status).toBe(1);
  expect(JSON.stringify(result.report.results)).toContain('candidate=7');
}, 90_000);

test('should reject a candidate that writes during help', async () => {
  const candidate = await mutant(
    'unexpected-write',
    "if(process.argv.includes('--help')) require('node:fs').writeFileSync('unexpected','changed');",
  );
  const result = await run(candidate, 'unexpected-write');
  expect(result.status).toBe(1);
  expect(JSON.stringify(result.report.results)).toContain('unexpected writes');
}, 90_000);

function output(stdout: string): Execution {
  return { status: 0, signal: null, error: null, stdout, stderr: '' };
}
function differences(
  reference: string,
  candidate: string,
): ReadonlyArray<string> {
  return compareExecutions({
    reference: output(reference),
    candidate: output(candidate),
    referenceRoot: '/tmp/reference',
    candidateRoot: '/tmp/candidate',
  });
}

test('should preserve array order and missing versus null while accepting JSON key order', () => {
  expect(differences('{"a":1,"b":2}', '{ "b":2,"a":1 }')).toEqual([]);
  expect(differences('{"a":null}', '{}')).not.toEqual([]);
  expect(differences('[1,2]', '[2,1]')).not.toEqual([]);
  expect(differences('[]', '[null]')).not.toEqual([]);
  expect(differences('0', '"0"')).not.toEqual([]);
  expect(differences('', 'null')).not.toEqual([]);
  expect(
    differences('/tmp/reference/project', '/tmp/candidate/project'),
  ).toEqual([]);
  expect(differences('meaningful text', 'meaningful text\n')).not.toEqual([]);
});

test('should reject exact byte, entry kind, mode, missing file, and binary differences', () => {
  const file = {
    kind: 'file',
    mode: 0o644,
    bytes: Buffer.from('terms\r\n').toString('base64'),
  };
  const reference = { 'LICENSE.md': file };
  for (const candidate of [
    {},
    {
      'LICENSE.md': {
        ...file,
        bytes: Buffer.from('terms\n').toString('base64'),
      },
    },
    { 'LICENSE.md': { ...file, kind: 'symlink' } },
    { 'LICENSE.md': { ...file, mode: 0o755 } },
    {
      'LICENSE.md': {
        ...file,
        bytes: Buffer.from([0, 255]).toString('base64'),
      },
    },
  ]) {
    expect(compareTrees({ reference, candidate })).not.toEqual([]);
  }
  expect(compareTrees({ reference: {}, candidate: {} })).toEqual([]);
});

test('should fail independent assertions even when both implementations return the same wrong output', () => {
  const version = scenarios.find((scenario) => scenario.id === 'version');
  expect(() => version?.verify?.(output('incorrect-version'), {})).toThrow();
});

test('should keep inventory IDs unique, dependencies acyclic, source owners present, and scenario coverage explicit', async () => {
  const ids = contracts.capabilities.map((capability) => capability.id);
  expect(new Set(ids).size).toBe(ids.length);
  expect(new Set(scenarios.map((scenario) => scenario.id)).size).toBe(
    scenarios.length,
  );
  for (const capability of contracts.capabilities) {
    for (const dependency of capability.dependsOn)
      expect(ids.indexOf(dependency)).toBeLessThan(ids.indexOf(capability.id));
    for (const path of capability.sourceOwners)
      expect(await Bun.file(path).exists()).toBe(true);
    expect(capability.requiredScenarios.length).toBeGreaterThan(0);
  }
  for (const scenario of scenarios)
    for (const id of scenario.capabilities) expect(ids).toContain(id);
});

test('should reject an unreviewed reference revision before running an executable', () => {
  const result = spawnSync(
    'bun',
    [
      'tests/migration/compare.ts',
      '--reference',
      executable,
      '--candidate',
      '/does/not/exist',
      '--reference-revision',
      '0'.repeat(40),
      '--report',
      join(root, 'invalid.json'),
    ],
    { encoding: 'utf8' },
  );
  expect(result.status).toBe(2);
  expect(result.stderr).toContain('Reference revision must match');
});

test('should report incomplete candidates and failed prerequisites instead of losing evidence', async () => {
  const path = join(root, 'unsupported.cjs');
  await writeFile(
    path,
    "#!/usr/bin/env node\nconsole.error('Unsupported candidate command'); process.exit(2);\n",
  );
  await chmod(path, 0o755);
  const result = await run(path, 'unsupported');
  expect(result.status).toBe(1);
  expect(JSON.stringify(result.report.results)).toContain(
    'Scenario precondition failed',
  );
}, 90_000);

test('should reject invalid UTF-8 output instead of replacing bytes and hiding a difference', () => {
  const result = execute({
    executable: process.execPath,
    args: ['-e', 'process.stdout.write(Buffer.from([255]))'],
    cwd: root,
    env: process.env,
  });
  expect(result.error).toContain('invalid UTF-8');
});
