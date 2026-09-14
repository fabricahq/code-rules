/** @fileoverview Checks composition, deterministic output, input immutability, and errors through the public Builds interface. */

import { expect, test } from 'bun:test';
import { buildRules, BuildError } from './index';
import {
  commit,
  group,
  ruleId,
  ruleText,
  source,
  input,
  generated,
  localInput,
  freezeInput,
} from './build-test-fixtures';

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

test('should add local rules to an imported group', () => {
  const build = {
    ...input(),
    localFiles: {
      [`${group}/project-contract.md`]: ruleText('Check the project contract'),
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
    'groups/README.md',
    'provenance.json',
    'rules/README.md',
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

test('should build repeatedly without mutating frozen configuration or snapshots', () => {
  const build = input();
  const before = structuredClone(build);
  freezeInput(build);
  expect(buildRules(build)).toEqual(buildRules(build));
  expect(build).toEqual(before);
});
