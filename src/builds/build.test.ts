/** @fileoverview Checks offline Builds behavior through its public interface using original rule fixtures. */

import { describe, expect, test } from 'bun:test';
import { posix } from 'node:path';
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
  return `---\ntitle: ${title}\nwhenToRead: When planning, implementing, or reviewing retries.\nimpact: HIGH\nimpactDescription: Prevent unbounded retries.\ntags: testing, retries\n---\n\n## ${title}\n\n${body}\n`;
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
    const output = generated(build, `rules/local/${ruleId}.md`);
    expect(output.split('# Local retries').length - 1).toBe(1);
    const index = generated(build, 'RULES.md');
    expect(index).toContain(
      '**Open group:** [Testing](groups/practices/testing.md)',
    );
    expect(index).toContain('Changing behavior, including production code.');
    expect(index).not.toContain('**local: Testing**');
    expect(index).not.toContain('Check externally visible behavior.');
  });

  test('should combine overlapping groups while preserving both source-qualified identities', () => {
    const output = generated(input(), `groups/${group}.md`);
    expect(output).toStartWith('# Testing\n');
    expect(output).toContain(`Group ID: \`${group}\``);
    expect(output).toContain(`fabrica:${ruleId}`);
    expect(output).toContain(`acme:${ruleId}`);
    expect(output).toContain('Fabrica retries');
    expect(output).toContain('Acme retries');
    expect(output).toContain(`../../rules/fabrica/${ruleId}.md`);
    expect(generated(input(), `rules/fabrica/${ruleId}.md`)).toContain(
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
    const output = generated(build, `groups/${group}.md`);
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
    const output = generated(build, `groups/${group}.md`);
    expect(output).toContain(`Rule ID: \`fabrica:${ruleId}\``);
    expect(output).not.toContain(`local:${group}/bounded`);
    expect(output).not.toContain('## Fabrica retries');
    expect(output).toContain(`../../rules/fabrica/${ruleId}.md`);
    expect(generated(build, `rules/fabrica/${ruleId}.md`)).toContain(
      `**Rule source:** [Original rule](../../../../../local/${group}/bounded.md)`,
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
    expect(generated(build, `groups/${group}.md`)).toContain(
      `local:${group}/project-contract`,
    );
  });

  test('should build a local-only group with one rule', () => {
    const build = localInput({ [`${ruleId}.md`]: ruleText('Local retries') });
    expect(generated(build, `groups/${group}.md`)).toContain(`local:${ruleId}`);
    const provenance: unknown = JSON.parse(generated(build, 'provenance.json'));
    expect(provenance).toMatchObject({ sources: [] });
  });

  test('should preserve an empty selected group and support a project with no selected groups', () => {
    expect(generated(localInput({}), `groups/${group}.md`)).toContain(
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
      `rules/fabrica/${ruleId}.md`,
    );
    expect(output).toContain('../../../../../vendor/fabrica/LICENSE.md');
    expect(output).toContain('../../../../../vendor/fabrica/NOTICE.txt');
    expect(output).toContain(
      `https://github.com/fabrica/rules/blob/${commit}/guides/retries.md#limits`,
    );
    expect(output).toContain('[This rule](#verification)');
    expect(output).toContain('Credit: original fixture author.');
  });

  test('should record no license paths when the manifest omits licensing', () => {
    const output = buildRules(withLibraryManifest('{"formatVersion":1}'));
    for (const path of [`rules/fabrica/${ruleId}.md`, `groups/${group}.md`]) {
      expect(output.files[path]).not.toContain('Library license:');
      expect(output.files[path]).not.toContain(
        'Library default license and notices:',
      );
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
      `rules/fabrica/${ruleId}.md`,
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
    for (const name of ['fabrica', 'acme']) {
      const output = generated(
        { ...build, snapshots },
        `rules/${name}/${ruleId}.md`,
      );
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
      `rules/local/${ruleId}.md`,
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
    `rules/local/${group}/sample.md`,
  );
  expect(result).toContain('Keep  double spaces and **strong** text.');
  expect(result).toContain(
    `[![icon](../../../../../local/${group}/icon.png)](../../../../../local/${group}/guide.md?mode=1#part)`,
  );
  expect(result).toContain(`../../../../../local/${group}/guide.md#other`);
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

/** Follow the Markdown links emitted by an index, keeping destinations relative to generated/. */
function indexedPaths(path: string, text: string): Array<string> {
  return Array.from(text.matchAll(/\]\(([^)]+\.md)\)/gu), (match) =>
    posix.normalize(posix.join(posix.dirname(path), match[1] ?? '')),
  );
}

test.each([0, 8192])(
  'should preserve replacement identity and exclusions with inline budget %s',
  (groupInlineMaxBytes) => {
    const id = `${group}/verify-retries`;
    const build: BuildInput = {
      ...input(),
      groupInlineMaxBytes,
      configuration: {
        schemaVersion: 1,
        sources: {
          fabrica: source(
            'fabrica/rules',
            { [`${group}/obsolete`]: 'No longer applicable.' },
            {
              [id]: {
                file: `local/${group}/replacement.md`,
                reason: 'Project retry contract.',
              },
            },
          ),
        },
        localGroups: [],
      },
      snapshots: {
        fabrica: snapshot('fabrica/rules', 'Superseded', {
          [`${group}/obsolete.md`]: ruleText('Excluded', 'Excluded body.'),
        }),
      },
      localFiles: {
        [`${group}/replacement.md`]: ruleText(
          'Project retries',
          'Use exactly three attempts.',
        ).replace(
          'When planning, implementing, or reviewing retries.',
          'When adding requests to the project API.',
        ),
        [`${group}/success.md`]: ruleText(
          'Stop after success',
          'Never retry a successful result.',
        ),
      },
    };
    const output = buildRules(build).files;
    const index = output[`groups/${group}.md`] ?? '';
    expect(index).toContain(
      '**When to read:** When adding requests to the project API.',
    );
    expect(index).toContain('**Impact:** HIGH');
    expect(index.includes('Use exactly three attempts.')).toBe(
      groupInlineMaxBytes > 0,
    );
    expect(index).toContain('**Why it matters:** Prevent unbounded retries.');
    expect(index).toContain('regardless of impact');
    expect(index).toContain(
      'Assess findings from concrete evidence and consequences.',
    );
    expect(output[`rules/fabrica/${id}.md`]).toContain(
      'Use exactly three attempts.',
    );
    expect(
      Object.keys(output).filter((path) => path.startsWith('rules/')),
    ).toEqual([`rules/fabrica/${id}.md`, `rules/local/${group}/success.md`]);
    expect(
      indexedPaths(`groups/${group}.md`, index).filter((path) =>
        path.startsWith('rules/'),
      ),
    ).toEqual([`rules/fabrica/${id}.md`, `rules/local/${group}/success.md`]);
    expect(Object.values(output).join('\n')).not.toContain('Excluded body.');
    expect(Object.values(output).join('\n')).not.toContain('title: Superseded');
  },
);

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
  const build = { ...localInput(files), indexMaxBytes: 2000 };
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
    expect(Buffer.byteLength(content, 'utf8')).toBeLessThanOrEqual(2000);
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
    configuration: { schemaVersion: 1, sources: {}, localGroups: groups },
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

test('should relocate links and attribution correctly for deeply nested individual definitions', () => {
  const path = `${group}/nested/retry.md`;
  const build = localInput({
    [path]: ruleText(
      'Nested',
      '[Guide](../guide.md?mode=1#validation)\n\n[Here](#validation)',
    ),
    [`${group}/guide.md`]: ruleText('Guide'),
  });
  const output = generated(build, `rules/local/${path}`);
  expect(output).toContain(
    `../../../../../../local/${group}/guide.md?mode=1#validation`,
  );
  expect(output).toContain('[Here](#validation)');
  expect(output).toContain(
    `**Rule source:** [Original rule](../../../../../../local/${path})`,
  );
});

test('should reject generated paths that would require a rule file to also be a directory', () => {
  const build = localInput({
    [`${group}/retry.md`]: ruleText('Retry'),
    [`${group}/retry.md/nested.md`]: ruleText('Nested retry'),
  });
  expect(() => buildRules(build)).toThrow('generated path conflicts with file');
});

test('should inline a complete group at the default UTF-8 boundary and use summaries one byte above it', () => {
  const file = `${group}/retry.md`;
  const body = '验证重试。\n\n```ts\nconst attempts = 3;\n```\n\nEnd.';
  const initial = localInput({ [file]: ruleText('Retry', body) });
  const size = Buffer.byteLength(
    generated(initial, `groups/${group}.md`),
    'utf8',
  );
  const build = localInput({
    [file]: ruleText('Retry', body + 'x'.repeat(8192 - size)),
  });
  const inline = buildRules(build).files;
  expect(Buffer.byteLength(inline[`groups/${group}.md`] ?? '', 'utf8')).toBe(
    8192,
  );
  expect(inline[`groups/${group}.md`]).toContain(
    'Full rules are included below.',
  );
  expect(inline[`groups/${group}.md`]).toContain('const attempts = 3;');

  const larger = buildRules(
    localInput({ [file]: ruleText('Retry', body + 'x'.repeat(8193 - size)) }),
  ).files;
  expect(larger[`groups/${group}.md`]).toContain(
    'This file contains summaries only.',
  );
  expect(larger[`groups/${group}.md`]).toContain('**Read full rule:** [Retry]');
  expect(larger[`groups/${group}.md`]).not.toContain('const attempts = 3;');
  expect(larger[`rules/local/${file}`]).toContain('const attempts = 3;');
});

test('should change only group delivery when inline is disabled or the page budget is smaller', () => {
  const build = localInput({
    [`${ruleId}.md`]: ruleText('Retry', 'Complete guidance. '.repeat(150)),
  });
  const inline = buildRules(build).files;
  const disabled = buildRules({ ...build, groupInlineMaxBytes: 0 }).files;
  const bounded = buildRules({ ...build, indexMaxBytes: 2000 }).files;
  expect(inline[`groups/${group}.md`]).toContain(
    'Full rules are included below.',
  );
  for (const output of [disabled, bounded]) {
    expect(output[`groups/${group}.md`]).toContain(
      'This file contains summaries only.',
    );
    expect(output[`rules/local/${ruleId}.md`]).toBe(
      inline[`rules/local/${ruleId}.md`],
    );
    expect(output['provenance.json']).toBe(inline['provenance.json']);
    expect(output['RULES.md']).toBe(inline['RULES.md']);
  }
  expect(
    Buffer.byteLength(bounded[`groups/${group}.md`] ?? '', 'utf8'),
  ).toBeLessThanOrEqual(2000);
});

test.each([-1, 1.5, NaN, Infinity, Number.MAX_SAFE_INTEGER + 1])(
  'should reject an invalid inline byte budget: %s',
  (groupInlineMaxBytes) => {
    expect(() => buildRules({ ...input(), groupInlineMaxBytes })).toThrow(
      'groupInlineMaxBytes',
    );
  },
);

test('should keep empty groups explicit without claiming to include full rules', () => {
  const output = generated(localInput({}), `groups/${group}.md`);
  expect(output).toContain('No active rules in this group.');
  expect(output).not.toContain('Full rules are included below.');
});

test('should preserve inline licenses, relative links, references, and same-file destinations across rules', () => {
  const firstPath = `${group}/one/retry.md`;
  const secondPath = `${group}/two/retry.md`;
  const body =
    '### Validation\n\n[Here](#validation) and [Guide][guide].\n\n[guide]: guide.txt\n\n```md\n# Preserve this code\n[guide]: untouched\n```';
  const build: BuildInput = {
    ...input(),
    configuration: {
      schemaVersion: 1,
      sources: { fabrica: source('fabrica/rules') },
      localGroups: [],
    },
    snapshots: {
      fabrica: snapshot('fabrica/rules', 'Base', {
        'rule-library.json': JSON.stringify({
          formatVersion: 1,
          license: { file: 'LICENSE.md', notices: [] },
        }),
        'LICENSE.md': 'Library terms.',
        [firstPath]: ruleText('Retry one', body),
        [secondPath]: ruleText('Retry two', body),
        [`${group}/one/guide.txt`]: 'First guide.',
        [`${group}/two/guide.txt`]: 'Second guide.',
      }),
    },
  };
  const files = buildRules(build).files;
  const inline = files[`groups/${group}.md`] ?? '';
  expect(inline).toContain('Full rules are included below.');
  expect(inline).toContain('\n## Rules\n\n### Retry one\n');
  expect(inline).toContain('\n##### Validation\n');
  expect(inline).toContain(
    '```md\n# Preserve this code\n[guide]: untouched\n```',
  );
  expect(inline).toContain('[LICENSE.md](../../../vendor/fabrica/LICENSE.md)');
  for (const path of [firstPath, secondPath]) {
    expect(inline).toContain(`../../rules/fabrica/${path}#validation`);
    expect(inline).toContain(
      `../../../vendor/fabrica/${posix.dirname(path)}/guide.txt`,
    );
    expect(inline).toContain(
      `code-rules-${encodeURIComponent(`fabrica:${path.slice(0, -3)}`)}-guide`,
    );
    expect(files[`rules/fabrica/${path}`]).toContain('[Here](#validation)');
    expect(files[`rules/fabrica/${path}`]).toContain('### Validation');
  }
});

test('should nest embedded headings while preserving links inside them and code examples', () => {
  const body =
    '# [Details](#details)\n\n## Validation\n\n```md\n# Keep this heading literal\n```';
  const build = localInput({ [`${ruleId}.md`]: ruleText('Retry', body) });
  const output = generated(build, `groups/${group}.md`);
  expect(output).toContain(
    `##### [Details](../../rules/local/${ruleId}.md#details)`,
  );
  expect(output).toContain('\n###### Validation\n');
  expect(output).toContain('```md\n# Keep this heading literal\n```');
});

test.each([0, 8192])(
  'should separate group pages from effective rules with inline budget %s',
  (groupInlineMaxBytes) => {
    const output = buildRules({
      ...localInput({ [`${ruleId}.md`]: ruleText('Retry') }),
      groupInlineMaxBytes,
    }).files;
    const page = output[`groups/${group}.md`] ?? '';
    expect(
      [
        ...new Set(Object.keys(output).map((path) => path.split('/')[0])),
      ].sort(),
    ).toEqual(['RULES.md', 'groups', 'provenance.json', 'rules']);
    expect(output['RULES.md']).toContain(`(groups/${group}.md)`);
    expect(page).toContain(`Group ID: \`${group}\``);
    expect(page).toContain('\n## Rules\n\n### Retry\n');
    expect(output[`rules/local/${ruleId}.md`]).toStartWith('# Retry\n');
    expect(page).toContain(`(../../rules/local/${ruleId}.md)`);
    expect(page).toContain('[RULES.md](../../RULES.md)');
    expect(page).toContain('[provenance.json](../../provenance.json)');
    expect(output[`rules/local/${ruleId}.md`]).toContain(
      `Rule ID: \`local:${ruleId}\``,
    );
    if (groupInlineMaxBytes > 0)
      expect(page).toContain(
        `**Rule source:** [Original rule](../../../local/${ruleId}.md)`,
      );
  },
);

test('should put complete guidance before source details and preserved metadata in both full-text formats', () => {
  const body =
    'Treat caught values as unknown.\n\n## Implementation\n\nNarrow before inspecting.\n\n## Validation\n\nExercise unfamiliar values.';
  const files = buildRules(
    localInput({ [`${ruleId}.md`]: ruleText('Retry', body) }),
  ).files;
  for (const [path, sectionHeading] of [
    [`rules/local/${ruleId}.md`, '##'],
    [`groups/${group}.md`, '####'],
  ]) {
    const page = files[path ?? ''] ?? '';
    const guidance = page.indexOf(`\n${sectionHeading} Guidance\n`);
    const validation = page.indexOf(`\n${sectionHeading}# Validation\n`);
    const source = page.indexOf(`\n${sectionHeading} Source and attribution\n`);
    const metadata = page.indexOf(`\n${sectionHeading}# Source metadata\n`);
    expect(guidance).toBeGreaterThan(0);
    expect(validation).toBeGreaterThan(guidance);
    expect(source).toBeGreaterThan(validation);
    expect(page.indexOf('**Rule source:**')).toBeGreaterThan(source);
    expect(metadata).toBeGreaterThan(source);
    expect(page.indexOf('```yaml')).toBeGreaterThan(metadata);
    expect(page).toContain('tags: testing, retries');
    expect(page).not.toContain('Project-authored definition.');
    expect(page).not.toContain('stated below');
  }
});
