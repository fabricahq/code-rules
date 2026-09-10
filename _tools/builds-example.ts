/** Create an isolated example workspace for reviewing generated agent guidance. */
import { mkdtemp, mkdir, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { buildRules } from '../src/builds';
import type { BuildInput } from '../src/builds';

function exampleRule(title: string, obligation: string): string {
  return `---\ntitle: ${title}\nimpact: HIGH\nimpactDescription: Catch retry failures before they reach users.\ntags: testing, retries\n---\n\n## ${title}\n\n${obligation}\n\n### Verification\n\nCheck the retry count and final result with a deterministic test.\n`;
}

const group = 'practices/testing';
const groupMetadata = JSON.stringify(
  {
    name: 'Testing',
    description: 'Verify behavior with meaningful tests.',
    whenToRead: ['Changing behavior, including code outside test files.'],
  },
  null,
  2,
);
const input: BuildInput = {
  toolVersion: '0.0.0-example',
  configuration: {
    schemaVersion: 1,
    sources: {
      example: {
        repository: 'example/rules',
        ref: 'v1.0.0',
        groups: [group],
        exclude: {},
        replace: {
          [`${group}/verify-retries`]: {
            file: `local/${group}/retry-budget.md`,
            reason: 'Our API allows exactly three attempts.',
          },
        },
      },
    },
    localGroups: [],
  },
  snapshots: {
    example: {
      repository: 'example/rules',
      ref: 'v1.0.0',
      resolvedCommit: 'a'.repeat(40),
      groups: [group],
      files: {
        'rule-library.json': '{"formatVersion":1}',
        [`${group}/_group.json`]: groupMetadata,
        [`${group}/verify-retries.md`]: exampleRule(
          'Verify retry limits',
          'Verify that retries stop at the configured limit.',
        ),
      },
    },
  },
  localFiles: {
    [`${group}/retry-budget.md`]: exampleRule(
      'Verify the project retry budget',
      'Assert that a failed request makes exactly three attempts before returning the error.',
    ),
    [`${group}/verify-success.md`]: exampleRule(
      'Stop retries after success',
      'Assert that a successful request triggers no further attempts.',
    ),
  },
};

const result = buildRules(input);
const directory = await mkdtemp(join(tmpdir(), 'code-rules-builds-'));
const workspaceFiles = {
  'config.json': `${JSON.stringify(input.configuration, null, 2)}\n`,
  ...Object.fromEntries(
    Object.entries(input.localFiles).map(([path, content]) => [
      `local/${path}`,
      content,
    ]),
  ),
  ...Object.fromEntries(
    Object.entries(input.snapshots).flatMap(([name, snapshot]) =>
      Object.entries(snapshot.files).map(([path, content]) => [
        `vendor/${name}/${path}`,
        content,
      ]),
    ),
  ),
  ...Object.fromEntries(
    Object.entries(result.files).map(([path, content]) => [
      `generated/${path}`,
      content,
    ]),
  ),
};
for (const [path, content] of Object.entries(workspaceFiles)) {
  const destination = join(directory, path);
  await mkdir(dirname(destination), { recursive: true });
  await writeFile(destination, content);
}
console.log(`Example workspace: ${directory}`);
console.log(`Agent index: ${join(directory, 'generated/RULES.md')}`);
console.log(
  'Source repository and commit are illustrative; no libraries were fetched.',
);
