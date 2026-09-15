/** @fileoverview Checks explicit and wildcard group selection, exclusions, replacements, and local rules through buildRules. */

import { expect, test } from 'bun:test';
import { buildRules, BuildError } from './index';
import type { BuildInput } from './index';
import {
  group,
  ruleId,
  metadata,
  ruleText,
  source,
  snapshot,
  input,
  generated,
  localInput,
  freezeInput,
} from './build-test-fixtures';

/** Supply a complete two-group snapshot with an explicit wildcard selection marker. */
function wildcardInput(): BuildInput {
  const repository = 'https://github.com/example/rules.git';
  return {
    configuration: {
      schemaVersion: 1,
      sources: { all: { ...source(repository), groups: '*' } },
    },
    snapshots: {
      all: {
        ...snapshot(repository, 'Retries'),
        groupSelection: '*',
        groups: [group, 'techs/typescript'],
        files: {
          ...snapshot(repository, 'Retries').files,
          'techs/typescript/_group.json': metadata,
          'techs/typescript/check-results.md': ruleText('Check results'),
        },
      },
    },
    localFiles: {},
    toolVersion: 'test',
  };
}

const scopes: ReadonlyArray<{
  pattern: 'practices/*' | 'techs/*';
  groups: ReadonlyArray<string>;
  excluded: string;
}> = [
  { pattern: 'practices/*', groups: [group], excluded: 'techs/typescript' },
  { pattern: 'techs/*', groups: ['techs/typescript'], excluded: group },
];

test.each([...scopes])(
  'should expand only the requested kind: $pattern',
  ({ pattern, groups, excluded }) => {
    const build = wildcardInput();
    const saved = build.snapshots.all;
    if (!saved) throw new Error('Missing all-groups fixture');
    const configuration = {
      schemaVersion: 1,
      sources: { all: { ...source(saved.repository), groups: pattern } },
    };
    const selected = {
      ...build,
      configuration,
      snapshots: { all: { ...saved, groupSelection: pattern, groups } },
    };
    const files = buildRules(selected).files;
    expect(files[`groups/${excluded}.md`]).toBeUndefined();
    for (const id of groups) expect(files[`groups/${id}.md`]).toBeDefined();
    expect(JSON.parse(files['provenance.json'] ?? '')).toMatchObject({
      sources: [{ groupSelection: pattern, groups }],
    });
    expect(() =>
      buildRules({ ...selected, snapshots: build.snapshots }),
    ).toThrow('run sync');
  },
);

test('should allow a technology-only local group alongside a practices wildcard', () => {
  const build = wildcardInput();
  const saved = build.snapshots.all;
  if (!saved) throw new Error('Missing all-groups fixture');
  const configuration = {
    schemaVersion: 1,
    sources: { all: { ...source(saved.repository), groups: 'practices/*' } },
  };
  const output = buildRules({
    ...build,
    configuration,
    snapshots: {
      all: { ...saved, groups: [group], groupSelection: 'practices/*' },
    },
    localFiles: { 'techs/typescript/_group.json': metadata },
  });
  expect(output.files['groups/techs/typescript.md']).toBeDefined();
  expect(
    output.files['rules/all/techs/typescript/check-results.md'],
  ).toBeUndefined();
});

test('should expand every group while recording both selection intent and concrete IDs', () => {
  const files = buildRules(wildcardInput()).files;
  expect(files[`rules/all/${ruleId}.md`]).toBeDefined();
  expect(files['rules/all/techs/typescript/check-results.md']).toBeDefined();
  expect(JSON.parse(files['provenance.json'] ?? '')).toMatchObject({
    sources: [{ groupSelection: '*', groups: [group, 'techs/typescript'] }],
  });
});

test('should reject an old partial snapshot rather than interpreting it as a whole library', () => {
  const build = wildcardInput();
  expect(() =>
    buildRules({
      ...build,
      snapshots: {
        all: snapshot('https://github.com/example/rules.git', 'Partial'),
      },
    }),
  ).toThrow('run sync');
});

test('should check expansion against snapshot contents and reject duplicate recorded groups', () => {
  const build = wildcardInput();
  const saved = build.snapshots.all;
  if (!saved) throw new Error('Missing all-groups fixture');
  expect(() =>
    buildRules({
      ...build,
      snapshots: { all: { ...saved, groups: [group] } },
    }),
  ).toThrow('wildcard snapshot groups differ');
  expect(() =>
    buildRules({
      ...build,
      snapshots: {
        all: { ...saved, groups: [group, group, 'techs/typescript'] },
      },
    }),
  ).toThrow('duplicate entries');
});

test('should validate orphan rules and malformed groups instead of silently omitting them', () => {
  const build = wildcardInput();
  const saved = build.snapshots.all;
  if (!saved) throw new Error('Missing all-groups fixture');
  const { ['techs/typescript/_group.json']: omitted, ...orphanFiles } =
    saved.files;
  expect(omitted).toBeDefined();
  expect(() =>
    buildRules({
      ...build,
      snapshots: { all: { ...saved, files: orphanFiles } },
    }),
  ).toThrow('_group.json');
  expect(() =>
    buildRules({
      ...build,
      snapshots: {
        all: {
          ...saved,
          files: { ...saved.files, 'techs/typescript/_group.json': '{}' },
        },
      },
    }),
  ).toThrow('name');
  expect(() =>
    buildRules({
      ...build,
      snapshots: {
        all: {
          ...saved,
          files: { ...saved.files, 'techs/Bad/_group.json': metadata },
        },
      },
    }),
  ).toThrow('invalid group ID');
});

test('should preserve rule exclusions and local replacements after expansion', () => {
  const build = wildcardInput();
  const replacement = `local/${group}/local-retries.md`;
  const configuration = {
    schemaVersion: 1,
    sources: {
      all: {
        ...source(
          'https://github.com/example/rules.git',
          { 'techs/typescript/check-results': 'Covered elsewhere.' },
          { [ruleId]: { file: replacement, reason: 'Three attempts.' } },
        ),
        groups: '*',
      },
    },
  };
  const files = buildRules({
    ...build,
    configuration,
    localFiles: {
      [`${group}/local-retries.md`]: ruleText('Exactly three attempts'),
    },
  }).files;
  expect(files['rules/all/techs/typescript/check-results.md']).toBeUndefined();
  expect(files[`rules/local/${group}/local-retries.md`]).toContain(
    'Exactly three attempts',
  );
  expect(files[`rules/local/${group}/local-retries.md`]).toContain(
    `local:${group}/local-retries`,
  );
});

test('should allow local metadata to describe a group adopted by a wildcard', () => {
  const build = wildcardInput();
  const configuration = {
    schemaVersion: 1,
    sources: {
      all: {
        ...source('https://github.com/example/rules.git'),
        groups: '*',
      },
    },
  };
  const output = buildRules({
    ...build,
    configuration,
    localFiles: {
      'techs/typescript/_group.json': JSON.stringify({
        name: 'Project TypeScript',
        description: 'Local scope.',
        whenToRead: ['Before changing project TypeScript.'],
      }),
    },
  }).files;
  expect(output['RULES.md']).toContain('Project TypeScript');
  expect(output['groups/techs/typescript.md']).toContain(
    'all:techs/typescript/check-results',
  );
});

test.each([
  { groups: ['*'] },
  { groups: ['*', group] },
  { groups: 'techs/**' },
  { groups: ['practices/*'] },
  { groups: ['techs/*', 'practices/*'] },
  { groups: 'practices/testing/*' },
  { groups: null },
])('should reject mixed wildcard lists or other patterns: %j', ({ groups }) => {
  const build = wildcardInput();
  const configuration = {
    schemaVersion: 1,
    sources: {
      all: {
        ...source('https://github.com/example/rules.git'),
        groups,
      },
    },
  };
  expect(() => buildRules({ ...build, configuration })).toThrow(BuildError);
});

test('should include new and empty groups from a complete updated snapshot deterministically', () => {
  const build = wildcardInput();
  const saved = build.snapshots.all;
  if (!saved) throw new Error('Missing all-groups fixture');
  const updated = {
    ...build,
    snapshots: {
      all: {
        ...saved,
        resolvedCommit: 'b'.repeat(40),
        groups: ['techs/typescript', 'practices/design', group],
        files: { ...saved.files, 'practices/design/_group.json': metadata },
      },
    },
  };
  expect(buildRules(build).files['groups/practices/design.md']).toBeUndefined();
  const first = buildRules(updated);
  expect(first.files['groups/practices/design.md']).toBeDefined();
  freezeInput(updated);
  expect(buildRules(updated)).toEqual(first);
});

test('should accept a complete empty library and preserve its source record', () => {
  const build = wildcardInput();
  const saved = build.snapshots.all;
  if (!saved) throw new Error('Missing all-groups fixture');
  const files = buildRules({
    ...build,
    snapshots: {
      all: {
        ...saved,
        groups: [],
        files: { 'rule-library.json': '{"formatVersion":1}' },
      },
    },
  }).files;
  expect(JSON.parse(files['provenance.json'] ?? '')).toMatchObject({
    sources: [{ groupSelection: '*', groups: [] }],
    rules: [],
  });
});

test('should not discover declared Markdown license assets as additional groups', () => {
  const build = wildcardInput();
  const saved = build.snapshots.all;
  if (!saved) throw new Error('Missing all-groups fixture');
  const text = ruleText('Licensed retries');
  const files = buildRules({
    ...build,
    snapshots: {
      all: {
        ...saved,
        files: {
          ...saved.files,
          [`${ruleId}.md`]: text,
          'rule-library.json': JSON.stringify({
            formatVersion: 1,
            license: {
              spdxExpression: 'MIT',
              file: 'techs/licenses/LICENSE.md',
              notices: [],
            },
          }),
          'techs/licenses/LICENSE.md': 'Retained fixture terms.',
        },
      },
    },
  }).files;
  expect(files['groups/techs/licenses.md']).toBeUndefined();
  expect(files[`rules/all/${ruleId}.md`]).toContain(
    '**Declared license:** MIT',
  );
});

test('should exclude only the owning source without listing inactive rules in the index', () => {
  const build = {
    ...input(),
    configuration: {
      schemaVersion: 1,
      sources: {
        fabrica: source('https://github.com/fabrica/rules.git', {
          [ruleId]: 'Covered locally.',
        }),
        acme: source('https://github.com/acme/rules.git'),
      },
    },
  };
  const output = generated(build, `groups/${group}.md`);
  expect(output).not.toContain(`fabrica:${ruleId}`);
  expect(output).toContain(`acme:${ruleId}`);
  expect(generated(build, 'RULES.md')).not.toContain(`fabrica:${ruleId}`);
  expect(generated(build, 'RULES.md')).not.toContain('Covered locally');
});

test('should use the local ID and preserve both origins without adding the local definition twice', () => {
  const build = {
    ...input(),
    configuration: {
      schemaVersion: 1,
      sources: {
        fabrica: source(
          'https://github.com/fabrica/rules.git',
          {},
          {
            [ruleId]: {
              file: `local/${group}/bounded.md`,
              reason: 'Use our exact retry budget.',
            },
          },
        ),
        acme: source('https://github.com/acme/rules.git'),
      },
    },
    localFiles: {
      [`${group}/bounded.md`]: ruleText(
        'Project retry budget',
        '[Cases](assets/bounded/cases.md)',
      ).replace('impact: HIGH', 'impact: MEDIUM'),
      [`${group}/assets/bounded/cases.md`]:
        'Three failures produce exactly three requests.',
    },
  };
  const output = generated(build, `groups/${group}.md`);
  expect(output).toContain(`Rule ID: \`local:${group}/bounded\``);
  expect(output).not.toContain(`fabrica:${ruleId}`);
  expect(output).not.toContain('## Fabrica retries');
  expect(output).toContain(`../../rules/local/${group}/bounded.md`);
  expect(generated(build, `rules/local/${group}/bounded.md`)).toContain(
    `**Rule source:** [Original rule](../../../../../local/${group}/bounded.md)`,
  );
  const provenance = generated(build, 'provenance.json');
  expect(provenance).toContain(`"file": "${group}/bounded.md"`);
  expect(provenance).toContain(`"file": "${ruleId}.md"`);
  expect(provenance).toContain('Use our exact retry budget.');
  const replaced = buildRules(build).files;
  const excluded = buildRules({
    ...build,
    configuration: {
      ...build.configuration,
      sources: {
        ...build.configuration.sources,
        fabrica: source('https://github.com/fabrica/rules.git', {
          [ruleId]: 'Use our exact retry budget.',
        }),
      },
    },
  }).files;
  const agentFiles = (files: Readonly<Record<string, string>>) =>
    Object.fromEntries(
      Object.entries(files).filter(([path]) => path !== 'provenance.json'),
    );
  expect(agentFiles(replaced)).toEqual(agentFiles(excluded));
  expect(replaced['provenance.json']).not.toBe(excluded['provenance.json']);
});

test('should reject a reused replacement and a replacement in another group', () => {
  const file = `local/${group}/shared.md`;
  const config = {
    schemaVersion: 1,
    sources: {
      fabrica: source(
        'https://github.com/fabrica/rules.git',
        {},
        { [ruleId]: { file, reason: 'Replace.' } },
      ),
      acme: source(
        'https://github.com/acme/rules.git',
        {},
        { [ruleId]: { file, reason: 'Replace.' } },
      ),
    },
  };
  expect(() =>
    buildRules({
      ...input(),
      configuration: config,
      localFiles: { [`${group}/shared.md`]: ruleText('Shared') },
    }),
  ).toThrow('reused');
  const other = {
    schemaVersion: 1,
    sources: {
      fabrica: source(
        'https://github.com/fabrica/rules.git',
        {},
        {
          [ruleId]: {
            file: 'local/techs/typescript/shared.md',
            reason: 'Replace.',
          },
        },
      ),
      acme: source('https://github.com/acme/rules.git'),
    },
  };
  expect(() => buildRules({ ...input(), configuration: other })).toThrow(
    'target group',
  );
});

test('should reject local rules without group metadata', () => {
  expect(() =>
    buildRules({
      ...input(),
      localFiles: { 'techs/go/rule.md': ruleText('Go rule') },
    }),
  ).toThrow('local/techs/go/_group.json');
  expect(() =>
    buildRules({
      ...localInput({}),
      localFiles: {
        'practices/testing/example.md': ruleText('Missing metadata'),
      },
    }),
  ).toThrow('local/practices/testing/_group.json');
});

test('should read only the local root README as directory documentation', () => {
  const files = buildRules(
    localInput({
      'README.md': '# Author rules here without frontmatter.',
      [`${group}/example.md`]: ruleText('Project rule'),
      [`${group}/assets/example/_group.json`]:
        'Supporting asset, not metadata.',
    }),
  ).files;
  const provenance = JSON.parse(files['provenance.json']!);
  expect(provenance.rules.map((rule: { id: string }) => rule.id)).toEqual([
    `local:${group}/example`,
  ]);
  expect(provenance.groups.map((group: { id: string }) => group.id)).toEqual([
    group,
  ]);
  expect(() =>
    buildRules(localInput({ 'notes.md': 'Misplaced supporting prose.' })),
  ).toThrow('invalid rule path');
});

test('should validate a local group description even when imported metadata is valid', () => {
  expect(() =>
    buildRules({ ...input(), localFiles: { [`${group}/_group.json`]: '{}' } }),
  ).toThrow('name');
});
