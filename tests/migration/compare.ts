/** @fileoverview Runs a pinned executable comparison in paired temporary workspaces and writes reviewable evidence without approving migration capabilities. */

import assert from 'node:assert/strict';
import { cp, mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { parseArgs } from 'node:util';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import {
  addLibrary,
  createImportFixture,
  fixtureGit,
  removeImportFixture,
} from '../fixtures/import-fixture';
import {
  compareExecutions,
  compareTrees,
  executableHash,
  execute,
  readWorkspace,
} from './observe';
import { scenarios } from './scenarios';
import contracts from '../../migration/contracts.json';
import approvedDifferences from '../../migration/approved-differences.json';

/** A retained step result includes raw streams, complete trees, and separate independent failures. */
type StepResult = {
  readonly id: string;
  readonly differences: ReadonlyArray<string>;
  readonly failures: ReadonlyArray<string>;
};

function git(args: ReadonlyArray<string>): string {
  const result = spawnSync('git', args, {
    cwd: resolve(import.meta.dir, '../..'),
    encoding: 'utf8',
  });
  if (result.status !== 0)
    throw new Error(`Evidence Git command failed: ${result.stderr}`);
  return result.stdout.trim();
}

function environment(root: string): NodeJS.ProcessEnv {
  return {
    PATH: process.env.PATH,
    HOME: join(root, 'home'),
    TMPDIR: join(root, 'scratch'),
    TMP: join(root, 'scratch'),
    TEMP: join(root, 'scratch'),
    LANG: 'C.UTF-8',
    LC_ALL: 'C.UTF-8',
    TZ: 'UTC',
    NO_COLOR: '1',
    GIT_CONFIG_NOSYSTEM: '1',
    GIT_CONFIG_GLOBAL: '/dev/null',
    GIT_TERMINAL_PROMPT: '0',
    GIT_CONFIG_COUNT: '2',
    GIT_CONFIG_KEY_0: `url.file://${root}/.insteadOf`,
    GIT_CONFIG_VALUE_0: 'https://github.com/fixture/',
    GIT_CONFIG_KEY_1: 'protocol.file.allow',
    GIT_CONFIG_VALUE_1: 'always',
  };
}

/** Run every initial scenario, reject stale source evidence, and retain both workspaces even on mismatches. */
async function compare(options: {
  readonly reference: string;
  readonly candidate: string;
  readonly referenceRevision: string;
  readonly report: string;
}): Promise<boolean> {
  assert.equal(
    options.referenceRevision,
    contracts.referenceRevision,
    'Reference revision must match migration/contracts.json; refresh requires review.',
  );
  assert.deepEqual(
    approvedDifferences,
    [],
    'Approved differences require a separately reviewed comparison policy.',
  );
  const owners = [
    'src',
    'package.json',
    'bun.lock',
    '_tools/package.ts',
    'tests/fixtures',
    'tests/package',
  ];
  const drift = git(['diff', contracts.referenceRevision, '--', ...owners]);
  assert.equal(
    drift,
    '',
    'TypeScript inputs changed: prior migration evidence is stale. Propose a reference refresh.',
  );
  assert.equal(
    git(['ls-files', '--others', '--exclude-standard', '--', ...owners]),
    '',
    'Untracked runtime files invalidate reference evidence.',
  );
  const reference = resolve(options.reference);
  const candidate = resolve(options.candidate);
  const root = await mkdtemp(join(tmpdir(), 'code-rules-parity-'));
  const referenceRoot = join(root, 'reference');
  const candidateRoot = join(root, 'candidate');
  const fixture = await createImportFixture();
  try {
    const library = await addLibrary(fixture, 'team');
    fixtureGit(library.directory, ['tag', 'v1.0.0']);
    for (const side of [referenceRoot, candidateRoot]) {
      await cp(fixture.root, side, { recursive: true });
      await mkdir(join(side, 'home'));
      await mkdir(join(side, 'project'));
    }
  } finally {
    await removeImportFixture(fixture);
  }
  const identity = {
    referenceRevision: options.referenceRevision,
    harnessRevision: git(['rev-parse', 'HEAD']),
    harnessDirty: git(['status', '--porcelain']) !== '',
    reference: { path: reference, sha256: await executableHash(reference) },
    candidate: { path: candidate, sha256: await executableHash(candidate) },
    sourceTrees: Object.fromEntries(
      owners.map((path) => [
        path,
        git(['rev-parse', `${contracts.referenceRevision}:${path}`]),
      ]),
    ),
    contractsSha256: createHash('sha256')
      .update(
        await readFile(
          new URL('../../migration/contracts.json', import.meta.url),
        ),
      )
      .digest('hex'),
    rulesCorpus: contracts.rulesCorpus,
    tools: {
      bun: Bun.version,
      node: spawnSync('node', ['--version'], {
        encoding: 'utf8',
      }).stdout.trim(),
      git: git(['--version']),
      platform: process.platform,
      architecture: process.arch,
    },
  };
  const results: Array<StepResult> = [];
  for (const scenario of scenarios) {
    const observations = [];
    const failures: Array<string> = [];
    for (const [side, executable] of [
      [referenceRoot, reference],
      [candidateRoot, candidate],
    ]) {
      assert.ok(side && executable);
      const project = join(side, 'project');
      let preparationError: string | null = null;
      try {
        await scenario.prepare?.(project);
      } catch (error) {
        preparationError = `Scenario precondition failed: ${error instanceof Error ? error.message : String(error)}`;
      }
      const before = await readWorkspace(side);
      const env = environment(side);
      // A missing local transport destination makes accidental Git fetches fail without public network access.
      if (scenario.id === 'check.offline')
        env.GIT_CONFIG_KEY_0 = `url.file://${side}/unavailable/.insteadOf`;
      const execution =
        preparationError === null
          ? execute({ executable, args: scenario.args, cwd: project, env })
          : {
              status: null,
              signal: null,
              error: preparationError,
              stdout: '',
              stderr: '',
            };
      const after = await readWorkspace(side);
      try {
        assert.equal(execution.error, null, `${scenario.id}: process failed`);
        assert.equal(
          execution.signal,
          null,
          `${scenario.id}: process signaled`,
        );
        assert.equal(
          execution.status,
          scenario.status,
          `${scenario.id}: intended exit status; ${execution.stderr}`,
        );
        if (scenario.readOnly)
          assert.deepEqual(
            compareTrees({ reference: before, candidate: after }),
            [],
            `${scenario.id}: unexpected writes`,
          );
        scenario.verify?.(execution, await readWorkspace(project));
      } catch (error) {
        failures.push(
          `${side === referenceRoot ? 'reference' : 'candidate'}: ${error instanceof Error ? error.message : String(error)}`,
        );
      }
      observations.push({ execution, before, after });
    }
    const [a, b] = observations;
    assert.ok(a && b);
    const differences = [
      ...compareExecutions({
        reference: a.execution,
        candidate: b.execution,
        referenceRoot,
        candidateRoot,
      }),
      ...compareTrees({ reference: a.after, candidate: b.after }),
    ];
    await writeFile(
      join(root, `${scenario.id}.json`),
      JSON.stringify(
        { scenario, reference: a, candidate: b, differences, failures },
        null,
        2,
      ) + '\n',
    );
    results.push({ id: scenario.id, differences, failures });
  }
  const passed = results.every(
    (result) => result.differences.length === 0 && result.failures.length === 0,
  );
  const report = {
    schemaVersion: 1,
    passed,
    evidenceRoot: root,
    identity,
    results,
    coverage: contracts.capabilities.map((capability) => ({
      id: capability.id,
      status: capability.status,
      exercisedBy: scenarios
        .filter((scenario) => scenario.capabilities.includes(capability.id))
        .map((scenario) => scenario.id),
      requiredScenarios: capability.requiredScenarios,
    })),
    limitations: [
      'Initial scenarios do not establish full contract coverage or independent review.',
      'Executable hashes identify files; callers must attest their build source and dependency provenance.',
      'Filesystem observations detect final changes, not transient writes or accesses outside controlled roots.',
    ],
  };
  await mkdir(dirname(resolve(options.report)), { recursive: true });
  await writeFile(options.report, JSON.stringify(report, null, 2) + '\n');
  return passed;
}

if (import.meta.main) {
  try {
    const { values } = parseArgs({
      options: {
        reference: { type: 'string' },
        candidate: { type: 'string' },
        'reference-revision': { type: 'string' },
        report: { type: 'string' },
      },
      strict: true,
    });
    assert.ok(
      values.reference &&
        values.candidate &&
        values['reference-revision'] &&
        values.report,
      'Usage: bun tests/migration/compare.ts --reference PATH --candidate PATH --reference-revision SHA --report PATH',
    );
    const passed = await compare({
      reference: values.reference,
      candidate: values.candidate,
      referenceRevision: values['reference-revision'],
      report: values.report,
    });
    console.log(`${passed ? 'PASS' : 'FAIL'}: ${resolve(values.report)}`);
    process.exitCode = passed ? 0 : 1;
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 2;
  }
}
