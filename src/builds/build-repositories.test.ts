/** @fileoverview Checks host-aware source links and Git repository identities through buildRules. */

import { describe, expect, test } from 'bun:test';
import { buildRules, BuildError } from './index';
import type { BuildInput } from './index';
import {
  commit,
  group,
  ruleId,
  source,
  snapshot,
  ruleText,
  generated,
  input,
} from './build-test-fixtures';

/** Create one imported rule from an explicit Git address, optionally referencing unvendored documentation and an image. */
function repositoryInput(repository: string, body: string): BuildInput {
  return {
    configuration: {
      schemaVersion: 1,
      sources: { remote: source(repository) },
      localGroups: [],
    },
    snapshots: {
      remote: snapshot(repository, 'Retry guidance', {
        [`${ruleId}.md`]: ruleText('Retry guidance', body),
      }),
    },
    localFiles: {},
    toolVersion: 'test',
  };
}

describe('Git repository addresses', () => {
  test.each([
    [
      'https://github.com/example/rules.git',
      'https://github.com/example/rules/blob',
      'https://raw.githubusercontent.com/example/rules',
    ],
    [
      'git@github.com:example/rules.git',
      'https://github.com/example/rules/blob',
      'https://raw.githubusercontent.com/example/rules',
    ],
    [
      'ssh://git@GITHUB.COM:22/example/rules.git',
      'https://github.com/example/rules/blob',
      'https://raw.githubusercontent.com/example/rules',
    ],
    [
      'https://gitlab.com/example/engineering/rules.git',
      'https://gitlab.com/example/engineering/rules/-/blob',
      'https://gitlab.com/example/engineering/rules/-/raw',
    ],
    [
      'git@gitlab.com:example/engineering/rules.git',
      'https://gitlab.com/example/engineering/rules/-/blob',
      'https://gitlab.com/example/engineering/rules/-/raw',
    ],
  ])(
    'should build pinned document and image links for %s',
    (repository, fileBase, rawBase) => {
      const build = repositoryInput(
        repository,
        '[Details](../../guide.md#retry) ![Diagram](../../image.png)',
      );
      const files = buildRules(build).files;
      for (const path of [`groups/${group}.md`]) {
        expect(files[path]).toContain(`${fileBase}/${commit}/${ruleId}.md`);
        expect(files[path]).toContain(`${fileBase}/${commit}/guide.md#retry`);
        expect(files[path]).toContain(`${rawBase}/${commit}/image.png`);
      }
      const readme = files['libraries/remote/README.md'];
      expect(readme).toContain(
        `](${fileBase.replace(/(?:\/-)?\/blob$/u, '')})`,
      );
      expect(readme).toContain(
        `](${fileBase.replace(/\/blob$/u, '/tree')}/${commit})`,
      );
      expect(files['provenance.json']).toContain(JSON.stringify(repository));
    },
  );

  test.each([
    'https://git.example.org:8443/Team/Rules.git',
    'ssh://alice@git.example.org:2222/srv/git/Rules.git',
    'git@git.example.org:Team/Rules.git',
    'git@git.example.org:/srv/Team/Rules.git',
    'ssh://git@[::1]:2222/Rules.git',
    'https://gitlab.example.org/team/rules.git',
    'https://github.com.evil.test/team/rules.git',
    'ssh://git@github.com:2222/team/rules.git',
  ])(
    'should link retained source files without guessing a web interface for %s',
    (repository) => {
      const output = generated(
        repositoryInput(repository, 'Follow the retry contract.'),
        `groups/${group}.md`,
      );
      expect(output).toContain(`../../../vendor/remote/${ruleId}.md`);
      expect(output).not.toContain('/blob/');
      const readme = generated(
        repositoryInput(repository, 'Guidance.'),
        'libraries/remote/README.md',
      );
      expect(readme.replaceAll('\\', '')).toContain(
        `**Repository:** ${repository}`,
      );
      expect(readme).toContain(`**Resolved commit:** ${commit}`);
      expect(readme).not.toContain('](https://github.com/');
      expect(() =>
        buildRules(repositoryInput(repository, '[Missing](../../missing.md)')),
      ).toThrow('include the target in the source snapshot');
    },
  );

  test('should relocate retained assets for an unknown host in generated groups', () => {
    const repository = 'ssh://git@git.example.org/srv/rules.git';
    const build = repositoryInput(
      repository,
      '[Details](../../guide.md#retry) ![Diagram](../../image.png)',
    );
    const fixture = build.snapshots.remote;
    if (!fixture) throw new Error('Missing remote fixture');
    const complete = {
      ...build,
      snapshots: {
        remote: {
          ...fixture,
          files: {
            ...fixture.files,
            'guide.md': '# Retry',
            'image.png': 'fixture image',
          },
        },
      },
    };
    const files = buildRules(complete).files;
    expect(files[`groups/${group}.md`]).toContain(
      '../../../vendor/remote/guide.md#retry',
    );
    expect(files[`groups/${group}.md`]).toContain(
      '../../../vendor/remote/image.png',
    );
  });

  test.each([
    'example/rules',
    'http://example.org/rules.git',
    'file:///tmp/rules',
    'git://example.org/rules.git',
    'ext::sh -c command',
    'git::https://github.com/example/rules.git',
    'https://user:secret@example.org/rules.git',
    'https://token@example.org/rules.git',
    'ssh://git:secret@example.org/rules.git',
    'https://example.org/rules.git?ref=main',
    'https://example.org/rules.git#main',
    ' https://example.org/rules.git',
    'https://example.org/ru\nles.git',
    'https://example.org/a/../rules.git',
    'https://example.org/a/%2e%2e/rules.git',
    'https://example.org/a%2fb/rules.git',
    'https://example.org/a%5cb/rules.git',
    'https://example.org/a%00b/rules.git',
    'https://example.org/%zz/rules.git',
    'https://example.org//rules.git',
    'https://example.org\\evil/rules.git',
    'git@example.org:../rules.git',
    'ssh://git@example.org/a/./rules.git',
    'https://example.org/',
    'ssh://-option.example/rules.git',
    'https://@example.org/rules.git',
    'ssh://git:@example.org/rules.git',
    'https://example.org/\uD800.git',
    '-git@example.org:rules.git',
  ])(
    'should reject unsafe or ambiguous repository syntax: %s',
    (repository) => {
      expect(() =>
        buildRules(repositoryInput(repository, 'Guidance.')),
      ).toThrow(BuildError);
      try {
        buildRules(repositoryInput(repository, 'Guidance.'));
      } catch (error) {
        expect(String(error)).not.toContain('secret');
      }
    },
  );

  test('should recognize standard GitHub transport aliases as duplicates', () => {
    const build = input();
    const configuration = {
      schemaVersion: 1,
      sources: {
        first: source('https://github.com/Example/Rules.git'),
        second: source('git@github.com:example/rules'),
      },
      localGroups: [],
    };
    expect(() => buildRules({ ...build, configuration })).toThrow(
      'more than once',
    );
  });

  test('should canonicalize URL escapes for hosted identity and safely encode browser links', () => {
    const configuration = {
      schemaVersion: 1,
      sources: {
        first: source('https://github.com/example/rules.git'),
        second: source('https://github.com/example/%72ules%2egit'),
      },
      localGroups: [],
    };
    expect(() => buildRules({ ...input(), configuration })).toThrow(
      'more than once',
    );
    const output = generated(
      repositoryInput('https://github.com/example/rules)text.git', 'Guidance.'),
      `groups/${group}.md`,
    );
    expect(output).toContain(
      `https://github.com/example/rules%29text/blob/${commit}/`,
    );
  });

  test('should preserve literal percent escapes in scp paths', () => {
    const output = generated(
      repositoryInput('git@github.com:example/%72ules.git', 'Guidance.'),
      `groups/${group}.md`,
    );
    expect(output).toContain(
      `https://github.com/example/%2572ules/blob/${commit}/`,
    );
  });

  test.each([
    ['ssh://git@host.example/Rules.git', 'git@host.example:Rules.git'],
    ['git@host.example:/Rules.git', 'git@host.example:Rules.git'],
    [
      'https://host.example/Team/Rules.git',
      'https://host.example/Team/rules.git',
    ],
    [
      'ssh://git@host.example:2222/Rules.git',
      'ssh://git@host.example:2223/Rules.git',
    ],
  ])(
    'should preserve distinct generic repository identities: %s and %s',
    (first, second) => {
      const build: BuildInput = {
        configuration: {
          schemaVersion: 1,
          sources: { first: source(first), second: source(second) },
          localGroups: [],
        },
        snapshots: {
          first: snapshot(first, 'First'),
          second: snapshot(second, 'Second'),
        },
        localFiles: {},
        toolVersion: 'test',
      };
      expect(generated(build, `groups/${group}.md`)).toContain(
        `second:${ruleId}`,
      );
    },
  );
});
