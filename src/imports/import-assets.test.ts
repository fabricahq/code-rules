/** @fileoverview Verifies asset ownership, complete directory retention, and local-link boundaries through real Git imports and generated rules. */

import { afterEach, beforeEach, expect, test } from 'bun:test';
import {
  addLibrary,
  createImportFixture,
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
  await removeImportFixture(fixture);
});
const rule =
  '---\ntitle: Verify retries\nwhenToRead: When changing retries.\nimpact: HIGH\nimpactDescription: Prevent excess attempts.\n---\n\n';
const files = {
  'rule-library.json': '{"formatVersion":1}',
  'practices/testing/_group.json':
    '{"name":"Testing","description":"Check behavior","whenToRead":["When changing behavior."]}',
  'practices/testing/verify-retries.md':
    rule + '[Explanation](assets/verify-retries/explanation.md)',
  'practices/testing/assets/verify-retries/explanation.md':
    '[Diagram](/assets/flow.svg)',
  'practices/testing/assets/verify-retries/unlinked.json': '{"attempts":3}',
  'assets/flow.svg': '<svg/>',
  'assets/unlinked.bin': new Uint8Array([0, 255]),
  'assets/_group.json': 'This is an asset, not group metadata.',
  'notes/unrelated.txt': 'Do not import',
};
function configuration() {
  return {
    schemaVersion: 1,
    sources: { one: fixtureSource('one') },
  };
}

test('retains complete owned and referenced shared asset directories without adopting Markdown assets', async () => {
  await addLibrary(fixture, 'one', files);
  const result = runImport(fixture, configuration(), { build: true }) as {
    error?: string;
    files: { one: Record<string, string> };
    generated: Record<string, string>;
  };
  expect(result.error).toBeUndefined();
  expect(
    result.files.one['practices/testing/assets/verify-retries/unlinked.json'],
  ).toBeDefined();
  expect(result.files.one['assets/unlinked.bin']).toBe(
    Buffer.from([0, 255]).toString('base64'),
  );
  expect(result.files.one['notes/unrelated.txt']).toBeUndefined();
  expect(
    Object.keys(result.generated).filter((path) =>
      path.startsWith('rules/one/'),
    ),
  ).toEqual(['rules/one/practices/testing/verify-retries.md']);
});

test.each([
  'notes/context.md',
  'practices/testing/assets/other/context.md',
  'assets/missing.md',
])(
  'rejects an unsupported or missing supporting reference: %s',
  async (target) => {
    await addLibrary(fixture, 'one', {
      ...files,
      'practices/testing/verify-retries.md': rule + `[Context](/${target})`,
      ...(target === 'assets/missing.md' ? {} : { [target]: 'Context' }),
    });
    expect(runImport(fixture, configuration())).toMatchObject({
      error: 'invalid-library',
    });
  },
);

test('omits unreferenced shared assets but retains unlinked owned assets', async () => {
  await addLibrary(fixture, 'one', {
    ...files,
    'practices/testing/verify-retries.md': rule + 'Check retries.',
    'practices/testing/assets/verify-retries/explanation.md': 'No shared link.',
  });
  const result = runImport(fixture, configuration()) as {
    error?: string;
    files: { one: Record<string, string> };
    generated: Record<string, string>;
  };
  expect(result.error).toBeUndefined();
  expect(result.files.one['assets/flow.svg']).toBeUndefined();
  expect(
    result.files.one['practices/testing/assets/verify-retries/unlinked.json'],
  ).toBeDefined();
});

test('keeps references to unselected rules without adopting their group', async () => {
  await addLibrary(fixture, 'one', {
    ...files,
    'practices/testing/verify-retries.md':
      rule + '[Other rule](../../techs/go/retry.md)',
    'techs/go/retry.md': rule + 'Return errors.',
  });
  const result = runImport(fixture, configuration(), { build: true });
  expect(result).toMatchObject({
    snapshots: { one: { groups: ['practices/testing'] } },
    generated: {
      'rules/one/practices/testing/verify-retries.md':
        expect.stringContaining('/techs/go/retry.md'),
    },
  });
  expect(JSON.stringify(result)).not.toContain('"groups/techs/go.md"');
});

test('rejects orphan asset directories even when no rule links to them', async () => {
  await addLibrary(fixture, 'one', {
    ...files,
    'practices/testing/assets/missing/example.json': '{}',
  });
  expect(runImport(fixture, configuration())).toMatchObject({
    error: 'invalid-library',
    message: expect.stringContaining('no adjacent owning rule'),
  });
});

test('rejects shared assets depending on a rule-private directory', async () => {
  await addLibrary(fixture, 'one', {
    ...files,
    'assets/context.md':
      '[Private](/practices/testing/assets/verify-retries/explanation.md)',
  });
  expect(runImport(fixture, configuration())).toMatchObject({
    error: 'invalid-library',
    message: expect.stringContaining('unsupported supporting-file link'),
  });
});

test('recognizes nested rules and keeps asset metadata out of wildcard discovery', async () => {
  await addLibrary(fixture, 'one', {
    ...files,
    'practices/testing/nested/retry.md':
      rule + '[Context](assets/retry/context.md)',
    'practices/testing/nested/assets/retry/context.md':
      'Supporting text without frontmatter.',
    'practices/testing/nested/assets/retry/_group.json':
      'Arbitrary asset data.',
  });
  expect(
    runImport(
      fixture,
      {
        schemaVersion: 1,
        sources: { one: { ...fixtureSource('one'), groups: '*' } },
      },
      { build: true },
    ),
  ).toMatchObject({
    snapshots: { one: { groups: ['practices/testing'] } },
    generated: {
      'rules/one/practices/testing/nested/retry.md': expect.any(String),
    },
  });
});
