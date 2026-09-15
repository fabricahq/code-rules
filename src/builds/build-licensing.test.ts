/** @fileoverview Checks library terms, SPDX declarations, notices, attribution, and provenance through buildRules. */

import { expect, test } from 'bun:test';
import { buildRules } from './index';
import type { BuildInput, FileContents } from './index';
import {
  commit,
  group,
  ruleId,
  ruleText,
  source,
  snapshot,
  input,
  localInput,
} from './build-test-fixtures';

/** Replace Fabrica's manifest and supplied files while retaining the standard two-library fixture. */
function withLibraryManifest(
  manifest: string,
  libraryFiles: FileContents = {},
): BuildInput {
  const build = input();
  return {
    ...build,
    snapshots: {
      ...build.snapshots,
      fabrica: snapshot(
        'https://github.com/fabrica/rules.git',
        'Licensed retries',
        {
          ...libraryFiles,
          'rule-library.json': manifest,
        },
      ),
    },
  };
}

test('should record no license paths when the manifest omits licensing', () => {
  const output = buildRules(withLibraryManifest('{"formatVersion":1}'));
  for (const path of [`rules/fabrica/${ruleId}.md`, `groups/${group}.md`]) {
    expect(output.files[path]).not.toContain('Library license:');
    expect(output.files[path]).not.toContain('Library license and notices:');
    expect(output.files[path]).toContain('**Rule source:**');
  }
  const provenance: unknown = JSON.parse(
    output.files['provenance.json'] ?? 'null',
  );
  expect(provenance).toMatchObject({
    sources: expect.arrayContaining([
      expect.objectContaining({ name: 'fabrica', licenseFiles: [] }),
    ]),
  });
});

test('should deduplicate notices while numbering them in declaration order', () => {
  /** Encode a supported manifest with the supplied notice order and a fixed license path. */
  const manifest = (notices: ReadonlyArray<string>): string =>
    JSON.stringify({
      formatVersion: 1,
      license: { file: 'LICENSE.md', notices },
    });
  const files = {
    'LICENSE.md': '',
    'A-NOTICE.txt': 'A',
    'Z-NOTICE.txt': 'Z',
  };
  const first = withLibraryManifest(
    manifest(['Z-NOTICE.txt', 'LICENSE.md', 'A-NOTICE.txt', 'A-NOTICE.txt']),
    files,
  );
  const second = withLibraryManifest(
    manifest(['A-NOTICE.txt', 'Z-NOTICE.txt']),
    files,
  );
  const output = buildRules(first);
  expect(output.files['libraries/fabrica/licenses/LICENSE.md']).toBe('');
  expect(output.files['libraries/fabrica/licenses/notices/001.md']).toBe('Z');
  expect(output.files['libraries/fabrica/licenses/notices/002.md']).toBe('A');
  expect(
    output.files['libraries/fabrica/licenses/notices/003.md'],
  ).toBeUndefined();
  expect(
    buildRules(second).files['libraries/fabrica/licenses/notices/001.md'],
  ).toBe('A');
  expect(output).toEqual(buildRules(first));
  const provenance: unknown = JSON.parse(
    output.files['provenance.json'] ?? 'null',
  );
  expect(provenance).toMatchObject({
    sources: expect.arrayContaining([
      expect.objectContaining({
        name: 'fabrica',
        licenseFiles: ['A-NOTICE.txt', 'LICENSE.md', 'Z-NOTICE.txt'],
      }),
    ]),
  });
});

test.each([{}, { spdxExpression: 'MIT' }])(
  'should reject misspelled license fields instead of silently dropping them: %j',
  (declaration) => {
    const build = withLibraryManifest(
      JSON.stringify({
        formatVersion: 1,
        license: {
          file: 'LICENSE.md',
          notices: [],
          spdxExpresion: 'MIT',
          ...declaration,
        },
      }),
      { 'LICENSE.md': 'Retained library terms.' },
    );
    expect(() => buildRules(build)).toThrow(
      'fabrica/rule-library.json: license: unknown field spdxExpresion',
    );
  },
);

test.each([
  {
    name: 'invalid JSON',
    manifest: '{',
    location: 'fabrica/rule-library.json: invalid JSON',
  },
  {
    name: 'non-object manifest',
    manifest: 'null',
    location: 'fabrica/rule-library.json: expected an object',
  },
  {
    name: 'missing version',
    manifest: '{}',
    location: 'fabrica/rule-library.json: formatVersion',
  },
  {
    name: 'unsupported version',
    manifest: '{"formatVersion":2}',
    location: 'fabrica/rule-library.json: formatVersion',
  },
  {
    name: 'null license',
    manifest: '{"formatVersion":1,"license":null}',
    location: 'fabrica/rule-library.json: license',
  },
  {
    name: 'missing license path',
    manifest: '{"formatVersion":1,"license":{"notices":[]}}',
    location: 'fabrica/rule-library.json: license.file',
  },
  {
    name: 'escaping license path',
    manifest:
      '{"formatVersion":1,"license":{"file":"../LICENSE.md","notices":[]}}',
    location: 'fabrica/rule-library.json: license.file',
  },
  {
    name: 'missing notices',
    manifest: '{"formatVersion":1,"license":{"file":"LICENSE.md"}}',
    location: 'fabrica/rule-library.json: license.notices',
  },
  {
    name: 'non-array notices',
    manifest:
      '{"formatVersion":1,"license":{"file":"LICENSE.md","notices":"NOTICE.txt"}}',
    location: 'fabrica/rule-library.json: license.notices',
  },
  {
    name: 'non-string notice',
    manifest:
      '{"formatVersion":1,"license":{"file":"LICENSE.md","notices":["NOTICE.txt",7]}}',
    location: 'fabrica/rule-library.json: license.notices[1]',
  },
  {
    name: 'blank notice',
    manifest:
      '{"formatVersion":1,"license":{"file":"LICENSE.md","notices":["NOTICE.txt"," "]}}',
    location: 'fabrica/rule-library.json: license.notices[1]',
  },
  {
    name: 'escaping notice',
    manifest:
      '{"formatVersion":1,"license":{"file":"LICENSE.md","notices":["NOTICE.txt","../other.txt"]}}',
    location: 'fabrica/rule-library.json: license.notices[1]',
  },
  {
    name: 'missing notice file',
    manifest:
      '{"formatVersion":1,"license":{"file":"LICENSE.md","notices":["NOTICE.txt","MISSING.txt"]}}',
    location:
      'fabrica/rule-library.json: license.notices[1]: missing declared file "MISSING.txt"',
  },
])(
  'should identify the manifest field when rejecting $name',
  ({ manifest, location }) => {
    const build = withLibraryManifest(manifest, {
      'LICENSE.md': 'Terms',
      'NOTICE.txt': 'Notice',
    });
    expect(() => buildRules(build)).toThrow(location);
  },
);

test.each([
  'MIT',
  '(MIT OR Apache-2.0) AND BSD-3-Clause',
  'GPL-2.0-only WITH Classpath-exception-2.0',
  'LGPL-2.1+',
  'LicenseRef-Acme-Custom',
])('should preserve a valid SPDX declaration: %s', (spdxExpression) => {
  const output = buildRules(
    withLibraryManifest(
      JSON.stringify({
        formatVersion: 1,
        license: { spdxExpression, file: 'LICENSE.md', notices: [] },
      }),
      { 'LICENSE.md': 'Publisher-provided terms' },
    ),
  ).files;
  const provenance = JSON.parse(output['provenance.json'] ?? '{}');
  const declaration = provenance.sources.find(
    (entry: { name: string }) => entry.name === 'fabrica',
  ).licenses[0];
  expect(declaration.spdxExpression).toBe(spdxExpression);
  expect(declaration).not.toHaveProperty('expression');
});

test.each([
  '',
  '   ',
  null,
  42,
  'Not-A-Listed-License',
  'MIT OR',
  '(MIT AND Apache-2.0',
  'MIT WITH Not-A-Listed-Exception',
  'MIT\nOR Apache-2.0',
  'LicenseRef-',
])('should reject an invalid SPDX declaration: %j', (spdxExpression) => {
  expect(() =>
    buildRules(
      withLibraryManifest(
        JSON.stringify({
          formatVersion: 1,
          license: { spdxExpression, file: 'LICENSE.md', notices: [] },
        }),
        { 'LICENSE.md': 'Publisher-provided terms' },
      ),
    ),
  ).toThrow('rule-library.json: license.spdxExpression');
});

test.each([{}, { spdxExpression: 'MIT' }])(
  'should explain the renamed field even when its replacement exists: %j',
  (fields) => {
    expect(() =>
      buildRules(
        withLibraryManifest(
          JSON.stringify({
            formatVersion: 1,
            license: {
              ...fields,
              expression: 'MIT',
              file: 'LICENSE.md',
              notices: [],
            },
          }),
          { 'LICENSE.md': 'Publisher-provided terms' },
        ),
      ),
    ).toThrow('renamed to license.spdxExpression');
  },
);

test('should retain unidentified legacy terms without guessing an SPDX declaration', () => {
  const output = buildRules(
    withLibraryManifest(
      JSON.stringify({
        formatVersion: 1,
        license: { file: 'LICENSE.md', notices: [] },
      }),
      { 'LICENSE.md': 'MIT-like terms' },
    ),
  ).files;
  const provenance = JSON.parse(output['provenance.json'] ?? '{}');
  expect(
    provenance.sources.find(
      (entry: { name: string }) => entry.name === 'fabrica',
    ).licenses[0].spdxExpression,
  ).toBeNull();
  expect(output[`rules/fabrica/${ruleId}.md`]).not.toContain(
    '**Declared license:**',
  );
});

test('should describe imported libraries without licensing and omit library output for local-only rules', () => {
  const output = buildRules(input()).files;
  const readme = output['libraries/fabrica/README.md'];
  expect(readme).toContain(
    '[https://github.com/fabrica/rules.git](https://github.com/fabrica/rules)',
  );
  expect(readme).toContain('**Requested revision:** v1.0.0');
  expect(readme).toContain(`https://github.com/fabrica/rules/tree/${commit}`);
  expect(readme).toContain('No library license declaration was supplied.');
  expect(readme).toContain('[provenance.json](../../provenance.json)');
  expect(readme).toContain('[RULES.md](../../RULES.md)');
  expect(
    Object.keys(output).filter((path) => path.includes('/licenses/')),
  ).toEqual([]);
  const local = buildRules(
    localInput({ [`${ruleId}.md`]: ruleText('Local') }),
  ).files;
  expect(
    Object.keys(local).filter((path) => path.startsWith('libraries/')),
  ).toEqual([]);
});

test.each([0, 8192])(
  'should standardize arbitrary source license paths in delivery mode %s',
  (groupInlineMaxBytes) => {
    const licenseText = '\ufeffPublisher terms ©\r\nSecond line\r\n';
    const build = withLibraryManifest(
      JSON.stringify({
        formatVersion: 1,
        license: {
          spdxExpression: 'MIT',
          file: 'legal/custom.txt',
          notices: ['credits/z.txt', 'other/z.txt'],
        },
      }),
      {
        'legal/custom.txt': licenseText,
        'credits/z.txt': 'First notice\r\n',
        'other/z.txt': 'Second notice\n',
        [`${ruleId}.md`]: ruleText(
          'License links',
          '[Terms](../../legal/custom.txt#terms) and [notice](../../credits/z.txt).',
        ),
      },
    );
    const output = buildRules({ ...build, groupInlineMaxBytes }).files;
    const readme = output['libraries/fabrica/README.md'];
    expect(readme).toContain('**Declared license:** MIT');
    expect(readme).toContain('[License text](licenses/LICENSE.md)');
    expect(readme).toContain('[Notice 1](licenses/notices/001.md)');
    expect(readme).toContain('[Notice 2](licenses/notices/002.md)');
    expect(
      Object.keys(output).some((path) => path.startsWith('licenses/')),
    ).toBe(false);
    expect(output['libraries/fabrica/licenses/LICENSE.md']).toBe(licenseText);
    expect(output['libraries/fabrica/licenses/notices/001.md']).toBe(
      'First notice\r\n',
    );
    expect(output['libraries/fabrica/licenses/notices/002.md']).toBe(
      'Second notice\n',
    );
    expect(
      output['libraries/fabrica/licenses/legal/custom.txt'],
    ).toBeUndefined();
    expect(output[`rules/fabrica/${ruleId}.md`]).toContain(
      '../../../../libraries/fabrica/licenses/LICENSE.md#terms',
    );
    expect(output[`rules/fabrica/${ruleId}.md`]).not.toContain(
      'vendor/fabrica/',
    );
    if (groupInlineMaxBytes) {
      expect(output[`groups/${group}.md`]).toContain(
        '../../libraries/fabrica/licenses/LICENSE.md#terms',
      );
      expect(output[`groups/${group}.md`]).toContain(
        '../../libraries/fabrica/licenses/notices/001.md',
      );
    }
    const provenance = JSON.parse(output['provenance.json'] ?? '{}');
    const rule = provenance.rules.find(
      (entry: { id: string }) => entry.id === `fabrica:${ruleId}`,
    );
    expect(rule.licenses[0]).toEqual({
      spdxExpression: 'MIT',
      files: ['vendor/fabrica/legal/custom.txt'],
      attributionFiles: [
        'vendor/fabrica/credits/z.txt',
        'vendor/fabrica/other/z.txt',
      ],
      generatedFiles: ['libraries/fabrica/licenses/LICENSE.md'],
      generatedAttributionFiles: [
        'libraries/fabrica/licenses/notices/001.md',
        'libraries/fabrica/licenses/notices/002.md',
      ],
    });
  },
);

test('should retain separate generated licenses even when every rule from one library is excluded', () => {
  const manifest = JSON.stringify({
    formatVersion: 1,
    license: { file: 'terms.txt', notices: [] },
  });
  const build: BuildInput = {
    ...input(),
    configuration: {
      schemaVersion: 1,
      sources: {
        fabrica: source('https://github.com/fabrica/rules.git', {
          [ruleId]: 'Not adopted',
        }),
        acme: source('https://github.com/acme/rules.git'),
      },
    },
    snapshots: {
      fabrica: snapshot('https://github.com/fabrica/rules.git', 'Fabrica', {
        'rule-library.json': manifest,
        'terms.txt': 'Fabrica terms',
      }),
      acme: snapshot('https://github.com/acme/rules.git', 'Acme', {
        'rule-library.json': manifest,
        'terms.txt': 'Acme terms',
      }),
    },
  };
  const output = buildRules(build).files;
  expect(output['libraries/fabrica/licenses/LICENSE.md']).toBe('Fabrica terms');
  expect(output['libraries/acme/licenses/LICENSE.md']).toBe('Acme terms');
  expect(output[`rules/fabrica/${ruleId}.md`]).toBeUndefined();
  expect(
    Object.keys(output).filter((path) => path.includes('/notices/')),
  ).toEqual([]);
});

test('should preserve declared library expressions separately from retained files', () => {
  const build = withLibraryManifest(
    JSON.stringify({
      formatVersion: 1,
      license: {
        spdxExpression: 'MIT OR Apache-2.0',
        file: 'LICENSE.md',
        notices: ['NOTICE.txt'],
      },
    }),
    { 'LICENSE.md': 'Full terms', 'NOTICE.txt': 'Copyright notice' },
  );
  const output = buildRules(build).files;
  const provenance = JSON.parse(output['provenance.json']!);
  expect(
    provenance.sources.find(
      (entry: { name: string }) => entry.name === 'fabrica',
    ).licenses,
  ).toEqual([
    {
      spdxExpression: 'MIT OR Apache-2.0',
      files: ['LICENSE.md'],
      attributionFiles: ['NOTICE.txt'],
      generatedFiles: ['libraries/fabrica/licenses/LICENSE.md'],
      generatedAttributionFiles: ['libraries/fabrica/licenses/notices/001.md'],
    },
  ]);
  expect(
    provenance.rules.find(
      (entry: { id: string }) => entry.id === `fabrica:${ruleId}`,
    ),
  ).toMatchObject({
    licenseBasis: 'library',
    licenses: [
      {
        spdxExpression: 'MIT OR Apache-2.0',
        files: ['vendor/fabrica/LICENSE.md'],
        attributionFiles: ['vendor/fabrica/NOTICE.txt'],
      },
    ],
  });
  expect(output[`rules/fabrica/${ruleId}.md`]).toContain(
    '**Declared license:** MIT OR Apache-2.0',
  );
});

test.each([0, 8192])(
  'should preserve library-wide terms and per-rule attribution in delivery mode %s',
  (groupInlineMaxBytes) => {
    const metadata = `attribution:\n  - url: https://example.com/source/commit/rule.md\n    description: Adapted from the original.\n`;
    const build = withLibraryManifest(
      '{"formatVersion":1,"license":{"spdxExpression":"MIT","file":"LICENSE.md","notices":["NOTICE.md"]}}',
      {
        'LICENSE.md': 'Library terms',
        'NOTICE.md': 'Original attribution notice',
        [`${ruleId}.md`]: ruleText('Adapted rule').replace(
          '---\n',
          `---\n${metadata}`,
        ),
        [`${group}/another.md`]: ruleText('Another rule'),
      },
    );
    const output = buildRules({ ...build, groupInlineMaxBytes }).files;
    const provenance = JSON.parse(output['provenance.json'] ?? '{}');
    const imported = provenance.rules.filter(
      (entry: { origin: { source: string } }) =>
        entry.origin.source === 'fabrica',
    );
    expect(imported).toHaveLength(2);
    for (const entry of imported) {
      expect(entry).toMatchObject({
        licenseBasis: 'library',
        licenses: [
          {
            spdxExpression: 'MIT',
            files: ['vendor/fabrica/LICENSE.md'],
            attributionFiles: ['vendor/fabrica/NOTICE.md'],
          },
        ],
      });
    }
    expect(output[`rules/fabrica/${ruleId}.md`]).toContain('**Attribution:**');
    expect(output[`rules/fabrica/${ruleId}.md`]).toContain(
      '**Declared license:** MIT',
    );
    if (groupInlineMaxBytes)
      expect(output[`groups/${group}.md`]).toContain(
        '**Declared license:** MIT',
      );
  },
);

test.each(['license', 'licenses'])(
  'should reject %s overrides in library rules, local rules, and groups',
  (key) => {
    const override = ruleText('Override').replace(
      '---\n',
      `---\n${key}: MIT\n`,
    );
    const imported = withLibraryManifest(
      '{"formatVersion":1,"license":{"spdxExpression":"MIT","file":"LICENSE.md","notices":[]}}',
      { 'LICENSE.md': 'Library terms', [`${ruleId}.md`]: override },
    );
    expect(() => buildRules(imported)).toThrow(
      'rule-level licenses are unsupported',
    );
    expect(() =>
      buildRules(localInput({ [`${ruleId}.md`]: override })),
    ).toThrow('rule-level licenses are unsupported');
    const groupOverride = JSON.stringify({
      name: 'Testing',
      description: 'Testing',
      whenToRead: ['Changing behavior'],
      [key]: 'MIT',
    });
    expect(() =>
      buildRules(
        localInput({
          [`${group}/_group.json`]: groupOverride,
          [`${ruleId}.md`]: ruleText('Rule'),
        }),
      ),
    ).toThrow('group-level licenses are unsupported');
  },
);

test('should keep replacement licensing independent from upstream library terms', () => {
  const build = withLibraryManifest(
    '{"formatVersion":1,"license":{"spdxExpression":"MIT","file":"LICENSE.md","notices":[]}}',
    { 'LICENSE.md': 'Upstream terms' },
  );
  const configuration = {
    schemaVersion: 1,
    sources: {
      fabrica: source(
        'https://github.com/fabrica/rules.git',
        {},
        {
          [ruleId]: {
            file: `local/${ruleId}.md`,
            reason: 'Original local policy.',
          },
        },
      ),
    },
  };
  const output = buildRules({
    ...build,
    configuration,
    snapshots: { fabrica: build.snapshots.fabrica! },
    localFiles: { [`${ruleId}.md`]: ruleText('Replacement') },
  }).files;
  const provenance = JSON.parse(output['provenance.json']!);
  expect(provenance.rules[0]).toMatchObject({
    id: `local:${ruleId}`,
    licenseBasis: 'undeclared',
    licenses: [],
    upstream: { source: 'fabrica' },
  });
  expect(provenance.sources[0].licenses[0].spdxExpression).toBe('MIT');
});

test.each([
  'attribution: [{url: "javascript:alert(1)", description: Source}]',
  'attribution: [{url: "https://user:secret@example.com", description: Source}]',
])('should reject unsafe attribution URLs: %s', (metadata) => {
  const build = localInput({
    [`${ruleId}.md`]: ruleText('Invalid').replace(
      '---\n',
      `---\n${metadata}\n`,
    ),
  });
  expect(() => buildRules(build)).toThrow('attribution[0].url');
});

test('should reject unsupported license declarations even on excluded rules', () => {
  const definition = ruleText('Excluded').replace(
    '---\n',
    '---\nlicenses: [{spdxExpression: MIT, files: [MISSING.md]}]\n',
  );
  const build: BuildInput = {
    configuration: {
      schemaVersion: 1,
      sources: {
        fabrica: source('https://github.com/fabrica/rules.git', {
          [ruleId]: 'Not adopted.',
        }),
      },
    },
    snapshots: {
      fabrica: snapshot('https://github.com/fabrica/rules.git', 'Excluded', {
        [`${ruleId}.md`]: definition,
      }),
    },
    localFiles: {},
    toolVersion: 'test',
  };
  expect(() => buildRules(build)).toThrow(
    'rule-level licenses are unsupported',
  );
});

test('should reject missing declared license text', () => {
  const build = input();
  const missing = snapshot('https://github.com/fabrica/rules.git', 'Rule', {
    'rule-library.json':
      '{"formatVersion":1,"license":{"file":"LICENSE.md","notices":[]}}',
  });
  expect(() =>
    buildRules({
      ...build,
      snapshots: { ...build.snapshots, fabrica: missing },
    }),
  ).toThrow('LICENSE.md');
});
