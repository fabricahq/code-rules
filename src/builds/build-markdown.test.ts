/** @fileoverview Checks generated Markdown structure and source-aware reference preservation through buildRules. */

import { expect, test } from 'bun:test';
import { posix } from 'node:path';
import type { Definition, LinkReference, Nodes } from 'mdast';
import { fromMarkdown } from 'mdast-util-from-markdown';
import { buildRules } from './index';
import type { BuildInput } from './index';
import {
  commit,
  group,
  ruleId,
  ruleText,
  source,
  snapshot,
  input,
  generated,
  localInput,
} from './build-test-fixtures';

/** Resolves prose reference links using Markdown's first matching definition. */
function referenceLinks(markdown: string) {
  const definitions: Definition[] = [];
  const references: LinkReference[] = [];
  function collect(node: Nodes) {
    if (node.type === 'definition') definitions.push(node);
    if (node.type === 'linkReference') references.push(node);
    if ('children' in node) node.children.forEach(collect);
  }
  collect(fromMarkdown(markdown));
  return references.map(({ identifier }) => ({
    identifier,
    url: definitions.find((definition) => definition.identifier === identifier)
      ?.url,
  }));
}

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

test('should retain attribution, license links, and pinned links to documents outside the snapshot', () => {
  const build = input();
  const licensed = snapshot(
    'https://github.com/fabrica/rules.git',
    'Licensed retries',
    {
      'rule-library.json':
        '{"formatVersion":1,"license":{"file":"LICENSE.md","notices":["NOTICE.txt"]}}',
      'LICENSE.md': 'Example terms.',
      'NOTICE.txt': 'Original example attribution.',
      [`${ruleId}.md`]: ruleText(
        'Licensed retries',
        '[Terms](../../LICENSE.md)\n\n[Background](../../guides/retries.md#limits)\n\n[This rule](#verification)\n\nCredit: original fixture author.',
      ),
    },
  );
  const output = generated(
    { ...build, snapshots: { ...build.snapshots, fabrica: licensed } },
    `rules/fabrica/${ruleId}.md`,
  );
  expect(output).toContain('../../../../libraries/fabrica/licenses/LICENSE.md');
  expect(output).toContain(
    '../../../../libraries/fabrica/licenses/notices/001.md',
  );
  expect(output).toContain(
    `https://github.com/fabrica/rules/blob/${commit}/guides/retries.md#limits`,
  );
  expect(output).toContain('[This rule](#verification)');
  expect(output).toContain('Credit: original fixture author.');
});

test('should relocate reference definitions and images while leaving fenced code unchanged', () => {
  const body =
    '[More][context]\n\n[context]: ../../README.md\n\n![Diagram](../../diagram.png)\n\n```md\n[Literal](../../README.md)\n```';
  const build = input();
  const output = generated(
    {
      ...build,
      snapshots: {
        ...build.snapshots,
        fabrica: snapshot('https://github.com/fabrica/rules.git', 'Links', {
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
    fabrica: snapshot(
      'https://github.com/fabrica/rules.git',
      'Fabrica references',
      {
        [`${ruleId}.md`]: ruleText('Fabrica references', body),
      },
    ),
    acme: snapshot('https://github.com/acme/rules.git', 'Acme references', {
      [`${ruleId}.md`]: ruleText('Acme references', body),
    }),
  };
  const identifiers = new Set<string>();
  for (const name of ['fabrica', 'acme']) {
    const output = generated(
      { ...build, snapshots },
      `rules/${name}/${ruleId}.md`,
    );
    const references = referenceLinks(output);
    expect(references.map(({ url }) => url)).toEqual([
      `https://github.com/${name}/rules/blob/${commit}/README.md`,
    ]);
    references.forEach(({ identifier }) => identifiers.add(identifier));
    expect(output).toContain(
      `https://raw.githubusercontent.com/${name}/rules/${commit}/diagram.png`,
    );
    expect(output).toContain(
      `https://github.com/${name}/rules/blob/${commit}/overview.md`,
    );
  }
  expect(identifiers.size).toBe(2);
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

test('should preserve inline licenses, relative links, references, and same-file destinations across rules', () => {
  const firstPath = `${group}/one/retry.md`;
  const secondPath = `${group}/two/retry.md`;
  const body =
    '### Validation\n\n[Here](#validation) and [Guide][guide].\n\n[guide]: guide.txt\n\n```md\n# Preserve this code\n[guide]: untouched\n```';
  const build: BuildInput = {
    ...input(),
    configuration: {
      schemaVersion: 1,
      sources: { fabrica: source('https://github.com/fabrica/rules.git') },
      localGroups: [],
    },
    snapshots: {
      fabrica: snapshot('https://github.com/fabrica/rules.git', 'Base', {
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
  expect(inline).toContain(
    '[LICENSE.md](../../libraries/fabrica/licenses/LICENSE.md)',
  );
  const references = referenceLinks(inline);
  expect(references.map(({ url }) => url)).toEqual(
    [firstPath, secondPath].map(
      (path) => `../../../vendor/fabrica/${posix.dirname(path)}/guide.txt`,
    ),
  );
  expect(new Set(references.map(({ identifier }) => identifier)).size).toBe(2);
  for (const path of [firstPath, secondPath]) {
    expect(inline).toContain(`../../rules/fabrica/${path}#validation`);
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

test.each([
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
