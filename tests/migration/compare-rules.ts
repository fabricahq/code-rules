/** @fileoverview Compares native rules functions with pinned TypeScript behavior and independent shared expectations. */
import { deepStrictEqual } from 'node:assert';
import { resolve } from 'node:path';
import { configuration } from '../../src/configuration';
import {
  groupId,
  groupMetadata,
  ruleGroup,
  ValidationError,
} from '../../src/formats/validation';
import identityCases from './identities/cases.json';
import metadataCases from './group-metadata/cases.json';
import contracts from '../../migration/contracts.json';
import approvedDifferences from '../../migration/approved-differences.json';

const cases = [...identityCases, ...metadataCases];
const referenceRevision = contracts.referenceRevision;
deepStrictEqual(
  approvedDifferences,
  [],
  'Global comparison policies must remain unchanged for this function slice',
);

/** Call existing reference functions; configuration exposes its private selector parser. */
function reference(test: (typeof cases)[number]): unknown {
  try {
    let value: unknown;
    if (test.operation === 'selection') {
      const config = configuration({
        schemaVersion: 1,
        sources: {
          team: {
            repository: 'https://github.com/fixture/team.git',
            ref: 'v1.0.0',
            ...(Object.hasOwn(test, 'input') ? { groups: test.input } : {}),
            exclude: {},
            replace: {},
          },
        },
      });
      const source = config.sources[0];
      if (!source)
        throw new Error('Reference configuration returned no source');
      value = source.groups;
    } else {
      if (typeof test.input !== 'string')
        throw new Error('Expected text fixture');
      if (test.operation === 'groupMetadata')
        value = groupMetadata(test.input, test.location);
      else if (test.operation === 'groupID')
        value = groupId(test.input, test.location);
      else if (test.operation === 'ruleGroup')
        value = ruleGroup(test.input, test.location);
      else throw new Error(`Unknown fixture operation: ${test.operation}`);
    }
    return { ok: true, value };
  } catch (error) {
    if (!(error instanceof ValidationError)) throw error;
    return {
      ok: false,
      error: {
        name: error.name,
        message: error.message,
        location: error.message.slice(0, error.message.indexOf(':')),
      },
    };
  }
}

const candidate = process.argv[2];
if (!candidate)
  throw new Error('Usage: bun tests/migration/compare-rules.ts <Go adapter>');
const owners = ['src', 'package.json', 'bun.lock'];
const drift = Bun.spawnSync([
  'git',
  'diff',
  '--exit-code',
  referenceRevision,
  '--',
  ...owners,
]);
const untracked = Bun.spawnSync([
  'git',
  'ls-files',
  '--others',
  '--exclude-standard',
  '--',
  ...owners,
]);
if (
  drift.exitCode !== 0 ||
  untracked.exitCode !== 0 ||
  untracked.stdout.length !== 0
) {
  throw new Error('TypeScript reference differs from the pinned revision');
}
const requests =
  cases
    .map(({ operation, input, location }) =>
      JSON.stringify({ operation, input, location }),
    )
    .join('\n') + '\n';
const processResult = Bun.spawnSync([resolve(candidate)], {
  stdin: Buffer.from(requests),
  timeout: 30_000,
  maxBuffer: 4 * 1024 * 1024,
  // The native adapter must run without finding Node, Bun, or any other executable.
  env: { PATH: '/nonexistent', LANG: 'C', TZ: 'UTC' },
});
if (processResult.exitCode !== 0 || processResult.stderr.length !== 0) {
  throw new Error(`Native adapter failed: ${processResult.stderr.toString()}`);
}
const lines = processResult.stdout.toString().trimEnd().split('\n');
deepStrictEqual(lines.length, cases.length, 'one native result per request');
for (const [index, test] of cases.entries()) {
  const line = lines[index];
  if (line === undefined)
    throw new Error(`Missing native response: ${test.id}`);
  const native: unknown = JSON.parse(line);
  const expected = test.expected;
  // The user approved clearer identity diagnostics and trimmed metadata in Go.
  // Those cases retain explicit old and new expectations; no output is normalized.
  deepStrictEqual(
    reference(test),
    ('referenceExpected' in test ? test.referenceExpected : undefined) ??
      expected,
    `${test.id}: TypeScript vs expectation`,
  );
  deepStrictEqual(native, expected, `${test.id}: Go vs expectation`);
}
console.log(
  `PASS: ${cases.length} shared rules cases; TypeScript + Go match independent expectations`,
);
console.log(
  `Approved behavior differences: ${cases.filter((test) => 'referenceExpected' in test && test.referenceExpected !== undefined).length} cases`,
);
console.log(`TypeScript reference: ${referenceRevision}`);
console.log(
  `Go adapter SHA-256: ${new Bun.CryptoHasher('sha256').update(await Bun.file(resolve(candidate)).arrayBuffer()).digest('hex')}`,
);
