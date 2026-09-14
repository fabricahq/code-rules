/** @fileoverview Checks configuration, snapshot, and authored metadata validation with precise public Builds diagnostics. */

import { expect, test } from 'bun:test';
import { buildRules } from './index';
import type { BuildInput } from './index';
import {
  group,
  ruleId,
  ruleText,
  source,
  snapshot,
  input,
  localInput,
} from './build-test-fixtures';

test('should reject misspelled configuration fields instead of silently ignoring policy', () => {
  const configuration = {
    schemaVersion: 1,
    sources: {
      fabrica: {
        repository: 'fabrica/rules',
        ref: 'v1.0.0',
        groups: [group],
        exclude: {},
        replace: {},
        excludes: { [ruleId]: 'Omit.' },
      },
    },
    localGroups: [],
  };
  expect(() => buildRules({ ...input(), configuration })).toThrow(
    'unknown field excludes',
  );
});

test('should reject a missing or mismatched snapshot with a sync diagnostic', () => {
  expect(() => buildRules({ ...input(), snapshots: {} })).toThrow('run sync');
  const build = input();
  const mismatch = { ...snapshot('fabrica/rules', 'Rule'), ref: 'v2' };
  expect(() =>
    buildRules({
      ...build,
      snapshots: { ...build.snapshots, fabrica: mismatch },
    }),
  ).toThrow('run sync');
});

test('should reject a commit ref that disagrees with its snapshot', () => {
  const build = {
    ...input(),
    configuration: {
      schemaVersion: 1,
      sources: {
        fabrica: {
          repository: 'fabrica/rules',
          ref: 'b'.repeat(40),
          groups: [group],
          exclude: {},
          replace: {},
        },
      },
      localGroups: [],
    },
    snapshots: {
      fabrica: { ...snapshot('fabrica/rules', 'Rule'), ref: 'b'.repeat(40) },
    },
  };
  expect(() => buildRules(build)).toThrow('resolvedCommit differs');
});

test.each([
  [
    'invalid source',
    {
      schemaVersion: 1,
      sources: { local: source('fabrica/rules') },
      localGroups: [],
    },
    'reserved source',
  ],
  [
    'duplicate repository',
    {
      schemaVersion: 1,
      sources: {
        fabrica: source('fabrica/rules'),
        acme: source('Fabrica/Rules'),
      },
      localGroups: [],
    },
    'more than once',
  ],
  [
    'overlapping local group',
    {
      schemaVersion: 1,
      sources: { fabrica: source('fabrica/rules') },
      localGroups: [group],
    },
    'cannot also',
  ],
  [
    'unknown version',
    { schemaVersion: 2, sources: {}, localGroups: [] },
    'version 1',
  ],
  [
    'missing target',
    {
      schemaVersion: 1,
      sources: {
        fabrica: source('fabrica/rules', {
          [`${group}/missing`]: 'Unneeded.',
        }),
        acme: source('acme/rules'),
      },
      localGroups: [],
    },
    'target is missing',
  ],
  [
    'contradictory exception',
    {
      schemaVersion: 1,
      sources: {
        fabrica: source(
          'fabrica/rules',
          { [ruleId]: 'Omit.' },
          { [ruleId]: { file: `local/${ruleId}.md`, reason: 'Replace.' } },
        ),
      },
      localGroups: [],
    },
    'both excluded and replaced',
  ],
  [
    'escaping replacement',
    {
      schemaVersion: 1,
      sources: {
        fabrica: source(
          'fabrica/rules',
          {},
          { [ruleId]: { file: 'local/../escape.md', reason: 'Replace.' } },
        ),
      },
      localGroups: [],
    },
    'contained relative path',
  ],
])('should reject %s', (_name, configuration, message) => {
  expect(() => buildRules({ ...input(), configuration })).toThrow(message);
});

test.each([
  [
    'invalid impact',
    ruleText('Invalid').replace('impact: HIGH', 'impact: URGENT'),
    'unknown impact',
  ],
  ['missing metadata', '## No frontmatter', 'frontmatter'],
  [
    'duplicate YAML key',
    ruleText('Duplicate').replace('impact: HIGH', 'impact: HIGH\nimpact: LOW'),
    'invalid YAML',
  ],
  ['empty body', ruleText('Empty', '').split('## Empty')[0] ?? '', 'body'],
  [
    'escaping link',
    ruleText('Link', '[Escape](../../../secret.md)'),
    'escapes source root',
  ],
  [
    'missing local link',
    ruleText('Link', '[Missing](missing.txt)'),
    'missing local link',
  ],
])('should reject %s', (_name, text, message) => {
  expect(() => buildRules(localInput({ [`${ruleId}.md`]: text }))).toThrow(
    message,
  );
});

test('should reject missing license text and malformed group metadata', () => {
  const build = input();
  const missing = snapshot('fabrica/rules', 'Rule', {
    'rule-library.json':
      '{"formatVersion":1,"license":{"file":"LICENSE.md","notices":[]}}',
  });
  expect(() =>
    buildRules({
      ...build,
      snapshots: { ...build.snapshots, fabrica: missing },
    }),
  ).toThrow('LICENSE.md');
  expect(() =>
    buildRules(localInput({ [`${group}/_group.json`]: '{}' })),
  ).toThrow('name');
});

test('should report duplicate repositories before an invalid ref on the same source', () => {
  const build = input();
  const configuration = {
    schemaVersion: 1,
    sources: {
      acme: source('acme/rules'),
      fabrica: { ...source('ACME/rules'), ref: 'bad ref' },
    },
    localGroups: [],
  };
  expect(() => buildRules({ ...build, configuration })).toThrow(
    'sources.fabrica: repository ACME/rules is declared more than once',
  );
});

test('should validate malformed excluded rules before applying exclusions', () => {
  const build = input();
  const excluded = {
    ...build,
    configuration: {
      schemaVersion: 1,
      sources: { fabrica: source('fabrica/rules', { [ruleId]: 'Unused' }) },
      localGroups: [],
    },
    snapshots: {
      fabrica: snapshot('fabrica/rules', 'Invalid', {
        [`${ruleId}.md`]: 'missing frontmatter',
      }),
    },
  };
  expect(() => buildRules(excluded)).toThrow(
    'expected YAML frontmatter followed by Markdown',
  );
});

test('should reject YAML aliases through the public builder', () => {
  const text = ruleText('Sample').replace(
    'tags: testing, retries',
    'attribution: &author Example\ntags: *author',
  );
  expect(() =>
    buildRules(localInput({ [`${group}/sample.md`]: text })),
  ).toThrow('YAML aliases are unsupported');
});

test.each([
  {
    change: { groups: ['practices/testing', 'invalid/group'] },
    expected: 'sources.fabrica.groups[1]: invalid group ID',
  },
  {
    change: {
      replace: {
        [ruleId]: { file: 'local/../outside.md', reason: 'Override' },
      },
    },
    expected: `sources.fabrica.replace.${ruleId}.file: expected a contained relative path`,
  },
  {
    change: {
      replace: { [ruleId]: { file: 'elsewhere/rule.md', reason: 'Override' } },
    },
    expected: `sources.fabrica.replace.${ruleId}.file: replacement files must be under local/`,
  },
  {
    change: { exclude: { 'wrong/group/rule': 'Unused' } },
    expected: 'sources.fabrica.exclude.wrong/group/rule: invalid group ID',
  },
])(
  'should retain the configuration field location: $expected',
  ({ change, expected }) => {
    const build = input();
    const configuration = {
      schemaVersion: 1,
      sources: { fabrica: { ...source('fabrica/rules'), ...change } },
      localGroups: [],
    };
    expect(() => buildRules({ ...build, configuration })).toThrow(expected);
  },
);

test('should retain local group indices before output sorting', () => {
  expect(() =>
    buildRules({
      ...input(),
      configuration: {
        schemaVersion: 1,
        sources: {},
        localGroups: ['techs/typescript', 'invalid/group'],
      },
    }),
  ).toThrow('localGroups[1]: invalid group ID');
});

test.each(
  [undefined, '', '   ', null, 7, ['Changing code.']].map((value) => ({
    value,
  })),
)(
  'should reject missing or invalid rule whenToRead metadata: $value',
  ({ value }) => {
    let text = ruleText('Relevant');
    text = text.replace(
      'whenToRead: When planning, implementing, or reviewing retries.\n',
      value === undefined ? '' : `whenToRead: ${JSON.stringify(value)}\n`,
    );
    expect(() => buildRules(localInput({ [`${ruleId}.md`]: text }))).toThrow(
      '.whenToRead',
    );
  },
);

test('should validate applicability on imported definitions even when excluded and on local replacements', () => {
  const missing = ruleText('Missing').replace(/whenToRead:.*\n/u, '');
  const base: BuildInput = {
    ...input(),
    configuration: {
      schemaVersion: 1,
      sources: { fabrica: source('fabrica/rules', { [ruleId]: 'Unused.' }) },
      localGroups: [],
    },
    snapshots: {
      fabrica: snapshot('fabrica/rules', 'Missing', {
        [`${ruleId}.md`]: missing,
      }),
    },
  };
  expect(() => buildRules(base)).toThrow('.whenToRead');
  expect(() =>
    buildRules({
      ...base,
      configuration: {
        schemaVersion: 1,
        sources: {
          fabrica: source(
            'fabrica/rules',
            {},
            { [ruleId]: { file: `local/${ruleId}.md`, reason: 'Override.' } },
          ),
        },
        localGroups: [],
      },
      snapshots: { fabrica: snapshot('fabrica/rules', 'Valid') },
      localFiles: { [`${ruleId}.md`]: missing },
    }),
  ).toThrow('.whenToRead');
});

test('should build rules without tags and omit absent tags from preserved metadata', () => {
  const text = ruleText('Retry').replace('tags: testing, retries\n', '');
  const files = buildRules(localInput({ [`${ruleId}.md`]: text })).files;
  for (const path of [`rules/local/${ruleId}.md`, `groups/${group}.md`]) {
    expect(files[path]).toContain('Verify the retry limit before shipping.');
    expect(files[path]).not.toContain('tags:');
  }
});

test.each(
  [null, 7, false, '   ', {}, ['testing', 7], ['testing', 'testing']].map(
    (tags) => ({ tags }),
  ),
)('should reject malformed optional tags: $tags', ({ tags }) => {
  const text = ruleText('Retry').replace(
    'tags: testing, retries',
    `tags: ${JSON.stringify(tags)}`,
  );
  expect(() => buildRules(localInput({ [`${ruleId}.md`]: text }))).toThrow(
    `local:${ruleId}.md.tags`,
  );
});
