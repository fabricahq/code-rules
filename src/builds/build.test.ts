/** @fileoverview Checks offline Builds behavior through its public interface using original rule fixtures. */

import { describe, expect, test } from 'bun:test';
import { buildRules, BuildError } from './index';
import type { BuildInput, FileContents, LibrarySnapshot } from './index';

const commit = 'a'.repeat(40);
const group = 'practices/testing';
const ruleId = `${group}/verify-retries`;
const metadata = JSON.stringify({
  name: 'Testing',
  description: 'Check externally visible behavior.',
  whenToRead: ['Changing behavior, including production code.'],
});

/** Create a complete Markdown rule with an optional body for public Builds tests. */
function ruleText(
  title: string,
  body = 'Verify the retry limit before shipping.',
): string {
  return `---\ntitle: ${title}\nimpact: HIGH\nimpactDescription: Prevent unbounded retries.\ntags: testing, retries\n---\n\n## ${title}\n\n${body}\n`;
}

/** Declare one imported testing group with source-scoped exceptions. */
function source(
  repository: string,
  exclude: Record<string, string> = {},
  replace: Record<string, { file: string; reason: string }> = {},
): object {
  return { repository, ref: 'v1.0.0', groups: [group], exclude, replace };
}

/** Create a pinned library snapshot with optional files overriding the defaults. */
function snapshot(
  repository: string,
  title: string,
  extra: FileContents = {},
): LibrarySnapshot {
  return {
    repository,
    ref: 'v1.0.0',
    resolvedCommit: commit,
    groups: [group],
    files: {
      'rule-library.json': '{"formatVersion":1}',
      [`${group}/_group.json`]: metadata,
      [`${ruleId}.md`]: ruleText(title),
      ...extra,
    },
  };
}

/** Create two independent libraries sharing a testing group and rule path. */
function input(): BuildInput {
  return {
    configuration: {
      schemaVersion: 1,
      sources: { fabrica: source('fabrica/rules'), acme: source('acme/rules') },
      localGroups: [],
    },
    snapshots: {
      fabrica: snapshot('fabrica/rules', 'Fabrica retries'),
      acme: snapshot('acme/rules', 'Acme retries'),
    },
    localFiles: {},
    toolVersion: '0.0.0-test',
  };
}

/** Read one generated file through Builds, failing when the expected path is absent. */
function generated(build: BuildInput, path: string): string {
  const content = buildRules(build).files[path];
  if (content === undefined) throw new Error(`Expected generated ${path}`);
  return content;
}

/** Create a project with one local-only testing group and supplied definitions. */
function localInput(localFiles: FileContents): BuildInput {
  return {
    configuration: { schemaVersion: 1, sources: {}, localGroups: [group] },
    snapshots: {},
    localFiles: { [`${group}/_group.json`]: metadata, ...localFiles },
    toolVersion: 'test',
  };
}

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
      fabrica: snapshot('fabrica/rules', 'Licensed retries', {
        ...libraryFiles,
        'rule-library.json': manifest,
      }),
    },
  };
}

describe('buildRules', () => {
  test('should show each rule title once and keep ordinary prose readable as Markdown', () => {
    const build = localInput({ [`${ruleId}.md`]: ruleText('Local retries') });
    const output = generated(build, `${group}.md`);
    expect(output.split('## Local retries').length - 1).toBe(1);
    expect(generated(build, 'RULES.md')).toContain(
      'Check externally visible behavior.',
    );
  });

  test('should combine overlapping groups while preserving both source-qualified identities', () => {
    const output = generated(input(), `${group}.md`);
    expect(output).toStartWith('# Testing\n');
    expect(output).toContain(`Group ID: \`${group}\``);
    expect(output).toContain(`fabrica:${ruleId}`);
    expect(output).toContain(`acme:${ruleId}`);
    expect(output).toContain('Fabrica retries');
    expect(output).toContain('Acme retries');
    expect(output).toContain(
      `https://github.com/fabrica/rules/blob/${commit}/${ruleId}.md`,
    );
    const index = generated(input(), 'RULES.md');
    expect(index).toContain(
      '[Source versions and rule origins](provenance.json)',
    );
    expect(index).not.toContain('## Sources and licenses');
    expect(index).not.toContain('## Exceptions');
    expect(index).toContain('**acme: Testing**');
    expect(index).toContain('**fabrica: Testing**');
    expect(index).toContain('Changing behavior, including production code');
  });

  test('should exclude only the owning source without listing inactive rules in the index', () => {
    const build = {
      ...input(),
      configuration: {
        schemaVersion: 1,
        sources: {
          fabrica: source('fabrica/rules', { [ruleId]: 'Covered locally.' }),
          acme: source('acme/rules'),
        },
        localGroups: [],
      },
    };
    const output = generated(build, `${group}.md`);
    expect(output).not.toContain(`fabrica:${ruleId}`);
    expect(output).toContain(`acme:${ruleId}`);
    expect(generated(build, 'RULES.md')).not.toContain(`fabrica:${ruleId}`);
    expect(generated(build, 'RULES.md')).not.toContain('Covered locally');
  });

  test('should preserve the replaced ID and both origins without adding the local definition twice', () => {
    const build = {
      ...input(),
      configuration: {
        schemaVersion: 1,
        sources: {
          fabrica: source(
            'fabrica/rules',
            {},
            {
              [ruleId]: {
                file: `local/${group}/bounded.md`,
                reason: 'Use our exact retry budget.',
              },
            },
          ),
          acme: source('acme/rules'),
        },
        localGroups: [],
      },
      localFiles: { [`${group}/bounded.md`]: ruleText('Project retry budget') },
    };
    const output = generated(build, `${group}.md`);
    expect(output).toContain(`Rule ID: \`fabrica:${ruleId}\``);
    expect(output).not.toContain(`local:${group}/bounded`);
    expect(output).not.toContain('## Fabrica retries');
    expect(output).toContain(
      `[Active definition](../../local/${group}/bounded.md)`,
    );
    const provenance = generated(build, 'provenance.json');
    expect(provenance).toContain(`"file": "${group}/bounded.md"`);
    expect(provenance).toContain(`"file": "${ruleId}.md"`);
    expect(provenance).toContain('Use our exact retry budget.');
  });

  test('should add local rules to an imported group', () => {
    const build = {
      ...input(),
      localFiles: {
        [`${group}/project-contract.md`]: ruleText(
          'Check the project contract',
        ),
      },
    };
    expect(generated(build, `${group}.md`)).toContain(
      `local:${group}/project-contract`,
    );
  });

  test('should build a local-only group with one rule', () => {
    const build = localInput({ [`${ruleId}.md`]: ruleText('Local retries') });
    expect(generated(build, `${group}.md`)).toContain(`local:${ruleId}`);
    const provenance: unknown = JSON.parse(generated(build, 'provenance.json'));
    expect(provenance).toMatchObject({ sources: [] });
  });

  test('should preserve an empty selected group and support a project with no selected groups', () => {
    expect(generated(localInput({}), `${group}.md`)).toContain(
      'No active rules',
    );
    const build = {
      configuration: { schemaVersion: 1, sources: {}, localGroups: [] },
      snapshots: {},
      localFiles: {},
      toolVersion: 'test',
    };
    expect(Object.keys(buildRules(build).files)).toEqual([
      'RULES.md',
      'provenance.json',
    ]);
  });

  test('should produce identical bytes when source and file insertion order changes', () => {
    const first = input();
    const second = {
      ...first,
      configuration: {
        schemaVersion: 1,
        sources: {
          acme: source('acme/rules'),
          fabrica: source('fabrica/rules'),
        },
        localGroups: [],
      },
      snapshots: Object.fromEntries(
        Object.entries(first.snapshots)
          .reverse()
          .map(([name, value]) => [
            name,
            {
              ...value,
              files: Object.fromEntries(Object.entries(value.files).reverse()),
            },
          ]),
      ),
    };
    const before = JSON.stringify(first);
    expect(buildRules(first)).toEqual(buildRules(second));
    expect(JSON.stringify(first)).toBe(before);
  });

  test('should retain attribution, license links, and pinned links to documents outside the snapshot', () => {
    const build = input();
    const licensed = snapshot('fabrica/rules', 'Licensed retries', {
      'rule-library.json':
        '{"formatVersion":1,"license":{"file":"LICENSE.md","notices":["NOTICE.txt"]}}',
      'LICENSE.md': 'Example terms.',
      'NOTICE.txt': 'Original example attribution.',
      [`${ruleId}.md`]: ruleText(
        'Licensed retries',
        '[Terms](../../LICENSE.md)\n\n[Background](../../guides/retries.md#limits)\n\n[This rule](#verification)\n\nCredit: original fixture author.',
      ),
    });
    const output = generated(
      { ...build, snapshots: { ...build.snapshots, fabrica: licensed } },
      `${group}.md`,
    );
    expect(output).toContain('../../vendor/fabrica/LICENSE.md');
    expect(output).toContain('../../vendor/fabrica/NOTICE.txt');
    expect(output).toContain(
      `https://github.com/fabrica/rules/blob/${commit}/guides/retries.md#limits`,
    );
    expect(output).toContain(`../../vendor/fabrica/${ruleId}.md#verification`);
    expect(output).toContain('Credit: original fixture author.');
  });

  test('should record no license paths when the manifest omits licensing', () => {
    const output = buildRules(withLibraryManifest('{"formatVersion":1}'));
    const provenance: unknown = JSON.parse(
      output.files['provenance.json'] ?? 'null',
    );
    expect(provenance).toMatchObject({
      sources: expect.arrayContaining([
        expect.objectContaining({ name: 'fabrica', licenseFiles: [] }),
      ]),
    });
  });

  test('should emit unique sorted license paths when declarations repeat or change order', () => {
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
    expect(output).toEqual(buildRules(second));
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

  test('should relocate reference definitions and images while leaving fenced code unchanged', () => {
    const body =
      '[More][context]\n\n[context]: ../../README.md\n\n![Diagram](../../diagram.png)\n\n```md\n[Literal](../../README.md)\n```';
    const build = input();
    const output = generated(
      {
        ...build,
        snapshots: {
          ...build.snapshots,
          fabrica: snapshot('fabrica/rules', 'Links', {
            [`${ruleId}.md`]: ruleText('Links', body),
          }),
        },
      },
      `${group}.md`,
    );
    expect(output).toContain(
      `]: https://github.com/fabrica/rules/blob/${commit}/README.md`,
    );
    expect(output).toContain(
      `https://raw.githubusercontent.com/fabrica/rules/${commit}/diagram.png`,
    );
    expect(output).toContain('```md\n[Literal](../../README.md)\n```');
  });

  test('should keep reference labels distinct across rules and relocate images nested in links', () => {
    const build = input();
    const body =
      '[Reference][details]\n\n[details]: ../../README.md\n\n[![Diagram](../../diagram.png)](../../overview.md)';
    const snapshots = {
      fabrica: snapshot('fabrica/rules', 'Fabrica references', {
        [`${ruleId}.md`]: ruleText('Fabrica references', body),
      }),
      acme: snapshot('acme/rules', 'Acme references', {
        [`${ruleId}.md`]: ruleText('Acme references', body),
      }),
    };
    const output = generated({ ...build, snapshots }, `${group}.md`);
    for (const name of ['fabrica', 'acme']) {
      const label = `code-rules-${encodeURIComponent(`${name}:${ruleId}`)}-details`;
      expect(output).toContain(`[Reference][${label}]`);
      expect(output).toContain(
        `[${label}]: https://github.com/${name}/rules/blob/${commit}/README.md`,
      );
      expect(output).toContain(
        `https://raw.githubusercontent.com/${name}/rules/${commit}/diagram.png`,
      );
      expect(output).toContain(
        `https://github.com/${name}/rules/blob/${commit}/overview.md`,
      );
    }
  });

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

  test('should preserve extra attribution metadata and CRLF-authored rules', () => {
    const text = ruleText('Attribution')
      .replace(
        'tags: testing, retries',
        'tags: [testing, retries]\nsource:\n  author: Original fixture author',
      )
      .replaceAll('\n', '\r\n');
    const output = generated(
      localInput({ [`${ruleId}.md`]: text }),
      `${group}.md`,
    );
    expect(output).toContain('author: Original fixture author');
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

  test('should reject a reused replacement and a replacement in another group', () => {
    const file = `local/${group}/shared.md`;
    const config = {
      schemaVersion: 1,
      sources: {
        fabrica: source(
          'fabrica/rules',
          {},
          { [ruleId]: { file, reason: 'Replace.' } },
        ),
        acme: source(
          'acme/rules',
          {},
          { [ruleId]: { file, reason: 'Replace.' } },
        ),
      },
      localGroups: [],
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
          'fabrica/rules',
          {},
          {
            [ruleId]: {
              file: 'local/techs/typescript/shared.md',
              reason: 'Replace.',
            },
          },
        ),
        acme: source('acme/rules'),
      },
      localGroups: [],
    };
    expect(() => buildRules({ ...input(), configuration: other })).toThrow(
      'target group',
    );
  });

  test('should reject undeclared local groups and missing local metadata', () => {
    expect(() =>
      buildRules({
        ...input(),
        localFiles: { 'techs/go/rule.md': ruleText('Go rule') },
      }),
    ).toThrow('undeclared group');
    expect(() => buildRules({ ...localInput({}), localFiles: {} })).toThrow(
      '_group.json',
    );
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
      ruleText('Duplicate').replace(
        'impact: HIGH',
        'impact: HIGH\nimpact: LOW',
      ),
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

  test('should identify input errors without generating partial output', () => {
    try {
      buildRules(localInput({ [`${ruleId}.md`]: 'Invalid' }));
      throw new Error('Expected a build error');
    } catch (error) {
      expect(error).toBeInstanceOf(BuildError);
      if (!(error instanceof BuildError)) throw error;
      expect(error.code).toBe('invalid-input');
      expect(error.message).toContain(`local:${ruleId}.md`);
    }
  });
});

/** Freeze nested fixture data so accidental mutations fail at the public Builds boundary. */
function freezeInput(value: unknown): void {
  if (value === null || typeof value !== 'object') return;
  for (const child of Object.values(value)) freezeInput(child);
  Object.freeze(value);
}

test('should build repeatedly without mutating frozen configuration or snapshots', () => {
  const build = input();
  const before = structuredClone(build);
  freezeInput(build);
  expect(buildRules(build)).toEqual(buildRules(build));
  expect(build).toEqual(before);
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

test('should preserve surrounding Markdown while relocating nested images and multiple links', () => {
  const body =
    'Keep  double spaces and **strong** text.\n\n[![icon](./icon.png)](./guide.md?mode=1#part) then [guide][doc].\n\n[doc]: ./guide.md#other\n\n```md\n[untouched](./missing.md)\n```';
  const result = generated(
    localInput({
      [`${group}/sample.md`]: ruleText('Sample', body),
      [`${group}/icon.png`]: 'image bytes',
      [`${group}/guide.md`]: ruleText('Guide'),
    }),
    `${group}.md`,
  );
  expect(result).toContain('Keep  double spaces and **strong** text.');
  expect(result).toContain(
    `[![icon](../../local/${group}/icon.png)](../../local/${group}/guide.md?mode=1#part)`,
  );
  expect(result).toContain(`../../local/${group}/guide.md#other`);
  expect(result).toContain('```md\n[untouched](./missing.md)\n```');
});

test.each([
  { body: '[bad](./%GG)', message: 'invalid encoded link' },
  {
    body: '<a href="./file.md">file</a>',
    message: 'use Markdown links for relative references',
  },
  { body: '[bad](./%5Cfile)', message: 'unsafe relative link' },
])('should reject unsupported references: $message', ({ body, message }) => {
  expect(() =>
    buildRules(
      localInput({ [`${group}/sample.md`]: ruleText('Sample', body) }),
    ),
  ).toThrow(message);
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
