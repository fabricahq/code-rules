/** @fileoverview Checks applicability indexes, complete navigation, and UTF-8 pagination limits through buildRules. */

import { expect, test } from 'bun:test';
import { buildRules } from './index';
import type { BuildInput } from './index';
import {
  group,
  ruleId,
  metadata,
  ruleText,
  input,
  generated,
  localInput,
  indexedPaths,
} from './build-test-fixtures';

test('should split indexes at complete entries with bounded UTF-8 bytes and resolvable links', () => {
  const files = Object.fromEntries(
    Array.from({ length: 30 }, (_, index) => [
      `${group}/rule-${String(index).padStart(2, '0')}.md`,
      ruleText(`Check retries ${index}`, 'Full guidance '.repeat(400)).replace(
        'When planning, implementing, or reviewing retries.',
        'When adding retries. '.repeat(8) + '验证重试。',
      ),
    ]),
  );
  const build = { ...localInput(files), indexMaxBytes: 3000 };
  const output = buildRules(build).files;
  const parts = Object.keys(output).filter((path) =>
    path.startsWith(`groups/${group}.part-`),
  );
  expect(parts.length).toBeGreaterThan(1);
  expect(
    indexedPaths(
      `groups/${group}.md`,
      output[`groups/${group}.md`] ?? '',
    ).filter((path) => parts.includes(path)),
  ).toEqual(
    parts.sort(
      (a, b) =>
        Number(a.match(/part-(\d+)/u)?.[1]) -
        Number(b.match(/part-(\d+)/u)?.[1]),
    ),
  );
  const linkedRules: Array<string> = [];
  for (const [path, content] of Object.entries(output)) {
    if (path.startsWith('rules/') || path === 'provenance.json') continue;
    expect(Buffer.byteLength(content, 'utf8')).toBeLessThanOrEqual(
      build.indexMaxBytes,
    );
    for (const target of indexedPaths(path, content)) {
      expect(output[target]).toBeDefined();
      if (target.startsWith('rules/')) linkedRules.push(target);
    }
  }
  expect(linkedRules.length).toBe(30);
  expect(new Set(linkedRules).size).toBe(30);
  for (const path of linkedRules)
    expect(output[path]).toContain('Full guidance '.repeat(400).trim());
  const reordered = {
    ...build,
    localFiles: Object.fromEntries(Object.entries(build.localFiles).reverse()),
  };
  expect(buildRules(reordered).files).toEqual(output);
});

test('should split the project group index while retaining every group and reading instruction', () => {
  const groups = Array.from(
    { length: 20 },
    (_, index) => `techs/language-${index}`,
  );
  const build: BuildInput = {
    configuration: { schemaVersion: 1, sources: {} },
    snapshots: {},
    toolVersion: 'test',
    indexMaxBytes: 2000,
    localFiles: Object.fromEntries(
      groups.map((id) => [`${id}/_group.json`, metadata]),
    ),
  };
  const output = buildRules(build).files;
  const parts = indexedPaths('RULES.md', output['RULES.md'] ?? '').filter(
    (path) => path.startsWith('RULES.part-'),
  );
  expect(parts.length).toBeGreaterThan(1);
  const destinations = parts
    .flatMap((path) => indexedPaths(path, output[path] ?? ''))
    .filter((path) => path.startsWith('groups/techs/'));
  expect(destinations.sort()).toEqual(
    groups.map((id) => `groups/${id}.md`).sort(),
  );
  for (const path of ['RULES.md', ...parts])
    expect(Buffer.byteLength(output[path] ?? '', 'utf8')).toBeLessThanOrEqual(
      2000,
    );
});

test('should retain a single index at the exact byte limit and split below it without truncation', () => {
  const files = Object.fromEntries(
    Array.from({ length: 10 }, (_, index) => [
      `${group}/rule-${index}.md`,
      ruleText(`Retry ${index}`),
    ]),
  );
  const build = { ...localInput(files), groupInlineMaxBytes: 0 };
  const full = generated(build, `groups/${group}.md`);
  const boundary = Buffer.byteLength(full, 'utf8');
  expect(
    generated({ ...build, indexMaxBytes: boundary }, `groups/${group}.md`),
  ).toBe(full);
  expect(
    generated({ ...build, indexMaxBytes: boundary - 1 }, `groups/${group}.md`),
  ).toContain('Part 1');
});

test.each([0, -1, 1.5, NaN, Infinity])(
  'should reject invalid index byte budgets: %s',
  (indexMaxBytes) => {
    expect(() => buildRules({ ...input(), indexMaxBytes })).toThrow(
      'indexMaxBytes',
    );
  },
);

test('should fail explicitly when a single entry cannot fit its index budget', () => {
  const text = ruleText('Large metadata').replace(
    'When planning, implementing, or reviewing retries.',
    'Read when '.repeat(1000),
  );
  expect(() =>
    buildRules({
      ...localInput({ [`${ruleId}.md`]: text }),
      indexMaxBytes: 2000,
    }),
  ).toThrow('an index entry');
});

test('should report an oversized complete part directory rather than drop navigation links', () => {
  const files = Object.fromEntries(
    Array.from({ length: 100 }, (_, index) => [
      `${group}/rule-${index}.md`,
      ruleText(`Retry ${index}`).replace(
        'When planning, implementing, or reviewing retries.',
        'Read when '.repeat(25),
      ),
    ]),
  );
  expect(() =>
    buildRules({ ...localInput(files), indexMaxBytes: 2000 }),
  ).toThrow('complete index part directory');
});

test('should rebuild changed applicability into the index and full definition while preserving the rule ID', () => {
  const first = localInput({ [`${ruleId}.md`]: ruleText('Retry') });
  const next = localInput({
    [`${ruleId}.md`]: ruleText('Retry').replace(
      'When planning, implementing, or reviewing retries.',
      'When changing API retries.',
    ),
  });
  const before = buildRules(first).files;
  const after = buildRules(next).files;
  expect(Object.keys(after)).toEqual(Object.keys(before));
  expect(after[`groups/${group}.md`]).not.toBe(before[`groups/${group}.md`]);
  expect(after[`rules/local/${ruleId}.md`]).not.toBe(
    before[`rules/local/${ruleId}.md`],
  );
  expect(after['provenance.json']).toBe(before['provenance.json']);
});
