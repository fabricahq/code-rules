/** @fileoverview Defines the initial executable migration scenarios and independent safety assertions using original project fixtures. */

import assert from 'node:assert/strict';
import { mkdir, writeFile, readFile, symlink } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { exampleFiles } from '../fixtures/import-fixture';
import type { Execution, Tree } from './observe';

/** Each step acts on both workspaces and checks each executable independently of parity. */
export type Scenario = {
  readonly id: string;
  readonly capabilities: ReadonlyArray<string>;
  readonly args: ReadonlyArray<string>;
  readonly status: number;
  readonly readOnly: boolean;
  readonly prepare?: (root: string) => Promise<void>;
  readonly verify?: (result: Execution, tree: Tree) => void;
};
const group = [
  '--name',
  'Testing',
  '--description',
  'Verify behavior.',
  '--when-to-read',
  'Before changing behavior.',
];
const rule = [
  '--title',
  'Limit attempts',
  '--when-to-read',
  'Before changing retries.',
  '--impact',
  'HIGH',
  '--impact-description',
  'Avoid excess requests.',
];

/** Decode a required textual output file, failing when the candidate omits it or changes its type. */
function fileText(tree: Tree, path: string): string {
  const entry = tree[path];
  assert.equal(entry?.kind, 'file', `Expected file ${path}`);
  assert.ok(entry);
  return Buffer.from(entry.bytes, 'base64').toString('utf8');
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function object(value: unknown): Record<string, unknown> {
  assert.ok(isRecord(value), 'Expected JSON object');
  return value;
}

function ruleIds(value: unknown): ReadonlyArray<unknown> {
  const rules = object(value).rules;
  assert.ok(Array.isArray(rules), 'Expected provenance rules array');
  return rules.map((rule: unknown) => object(rule).id);
}

async function write(root: string, path: string, text: string): Promise<void> {
  await mkdir(dirname(join(root, path)), { recursive: true });
  await writeFile(join(root, path), text);
}

/** Stable step order exercises stateful workflows; failures remain visible and do not approve any capability. */
export const scenarios: ReadonlyArray<Scenario> = [
  {
    id: 'help.top',
    capabilities: ['cli.discovery'],
    args: ['--help'],
    status: 0,
    readOnly: true,
    verify: (result) => assert.match(result.stdout, /library/),
  },
  {
    id: 'help.nested',
    capabilities: ['cli.discovery'],
    args: ['library', 'init', '--help'],
    status: 0,
    readOnly: true,
    verify: (result) => assert.match(result.stdout, /--license-file/),
  },
  {
    id: 'version',
    capabilities: ['cli.discovery'],
    args: ['--version'],
    status: 0,
    readOnly: true,
    verify: (result) => assert.equal(result.stdout.trim(), '0.1.0-rc.1'),
  },
  {
    id: 'usage.unknown',
    capabilities: ['cli.discovery'],
    args: ['build', '--misspelled'],
    status: 2,
    readOnly: true,
    verify: (result) => assert.match(result.stderr, /--help/),
  },
  {
    id: 'project.init',
    capabilities: ['authoring.project'],
    args: ['init'],
    status: 0,
    readOnly: false,
    verify: (_, tree) =>
      assert.deepEqual(JSON.parse(fileText(tree, '.code-rules/config.json')), {
        schemaVersion: 1,
        sources: {},
      }),
  },
  {
    id: 'project.init-repeat',
    capabilities: ['authoring.project'],
    args: ['init'],
    status: 0,
    readOnly: true,
  },
  {
    id: 'project.missing-input',
    capabilities: ['authoring.project'],
    args: ['local', 'add', 'group', 'practices/missing', '--non-interactive'],
    status: 2,
    readOnly: true,
  },
  {
    id: 'local.group',
    capabilities: ['authoring.project'],
    args: ['local', 'add', 'group', 'practices/testing', ...group],
    status: 0,
    readOnly: false,
  },
  {
    id: 'local.rule',
    capabilities: ['authoring.project'],
    args: [
      'local',
      'add',
      'rule',
      'practices/testing/budget',
      ...rule,
      '--body-file',
      'body.md',
    ],
    status: 0,
    readOnly: false,
    prepare: (root) => write(root, 'body.md', 'Allow three attempts.\n'),
  },
  {
    id: 'local.collision',
    capabilities: ['authoring.project', 'files.safe-write'],
    args: [
      'local',
      'add',
      'rule',
      'practices/testing/budget',
      ...rule,
      '--body-file',
      'body.md',
    ],
    status: 1,
    readOnly: true,
  },
  {
    id: 'build.local',
    capabilities: ['project.offline', 'builds.resolve', 'builds.render'],
    args: ['build'],
    status: 0,
    readOnly: false,
    verify: (_, tree) => {
      const value = object(
        JSON.parse(fileText(tree, '.code-rules/generated/provenance.json')),
      );
      assert.equal(value.toolVersion, '0.1.0-rc.1');
      assert.deepEqual(ruleIds(value), ['local:practices/testing/budget']);
      assert.match(fileText(tree, '.code-rules/generated/RULES.md'), /Testing/);
    },
  },
  {
    id: 'build.repeat',
    capabilities: ['project.offline'],
    args: ['build'],
    status: 0,
    readOnly: true,
    verify: (result) =>
      assert.deepEqual(JSON.parse(result.stdout), {
        added: [],
        changed: [],
        removed: [],
      }),
  },
  {
    id: 'check.clean',
    capabilities: ['project.offline'],
    args: ['check'],
    status: 0,
    readOnly: true,
  },
  {
    id: 'check.stale',
    capabilities: ['project.offline'],
    args: ['check'],
    status: 1,
    readOnly: true,
    prepare: (root) =>
      write(root, '.code-rules/generated/RULES.md', 'Stale output.\n'),
    verify: (_, tree) =>
      assert.equal(
        fileText(tree, '.code-rules/generated/RULES.md'),
        'Stale output.\n',
      ),
  },
  {
    id: 'build.repair',
    capabilities: ['project.offline'],
    args: ['build'],
    status: 0,
    readOnly: false,
    verify: (_, tree) =>
      assert.notEqual(
        fileText(tree, '.code-rules/generated/RULES.md'),
        'Stale output.\n',
      ),
  },
  {
    id: 'source.add',
    capabilities: ['authoring.project'],
    args: [
      'add',
      'source',
      'team',
      '--repository',
      'https://github.com/fixture/team.git',
      '--version',
      '^1.0.0',
      '--groups',
      'practices/testing',
    ],
    status: 0,
    readOnly: false,
    verify: (_, tree) => assert.equal(tree['.code-rules/vendor'], undefined),
  },
  {
    id: 'sync.licensed',
    capabilities: [
      'project.sync',
      'imports.git',
      'formats.versions',
      'formats.licenses',
      'files.snapshots',
    ],
    args: ['sync'],
    status: 0,
    readOnly: false,
    verify: (_, tree) => {
      assert.equal(
        fileText(
          tree,
          '.code-rules/generated/libraries/team/licenses/LICENSE.md',
        ),
        exampleFiles['LICENSE.txt'],
      );
      assert.equal(
        fileText(
          tree,
          '.code-rules/generated/libraries/team/licenses/notices/001.md',
        ),
        exampleFiles['NOTICE.txt'],
      );
      const rules = ruleIds(
        JSON.parse(fileText(tree, '.code-rules/generated/provenance.json')),
      );
      assert.deepEqual(rules, [
        'local:practices/testing/budget',
        'team:practices/testing/verify-retries',
      ]);
      assert.ok(
        Object.values(tree).some(
          (entry) =>
            entry.bytes === Buffer.from([0, 255, 128, 10]).toString('base64'),
        ),
      );
    },
  },
  {
    id: 'sync.repeat',
    capabilities: ['project.sync'],
    args: ['sync'],
    status: 0,
    readOnly: true,
  },
  {
    id: 'check.offline',
    capabilities: ['project.offline'],
    args: ['check'],
    status: 0,
    readOnly: true,
  },
  {
    id: 'build.replacement',
    capabilities: ['builds.resolve', 'builds.render'],
    args: ['build'],
    status: 0,
    readOnly: false,
    prepare: async (root) => {
      const path = join(root, '.code-rules/config.json');
      const config = object(JSON.parse(await readFile(path, 'utf8')));
      const team = object(object(config.sources).team);
      object(team.replace)['practices/testing/verify-retries'] = {
        file: 'local/practices/testing/budget.md',
        reason: 'Our project budget.',
      };
      await writeFile(path, JSON.stringify(config, null, 2));
    },
    verify: (_, tree) =>
      assert.deepEqual(
        ruleIds(
          JSON.parse(fileText(tree, '.code-rules/generated/provenance.json')),
        ),
        ['local:practices/testing/budget'],
      ),
  },
  {
    id: 'library.init',
    capabilities: ['authoring.library', 'formats.licenses'],
    args: [
      'library',
      'init',
      '--directory',
      'library',
      '--spdx',
      'CC0-1.0',
      '--license-file',
      'terms.txt',
      '--notice-file',
      'notice.txt',
    ],
    status: 0,
    readOnly: false,
    prepare: async (root) => {
      await write(root, 'terms.txt', 'Original migration terms.\r\n');
      await write(root, 'notice.txt', 'Original migration notice.\r\n');
    },
    verify: (_, tree) => {
      assert.equal(
        fileText(tree, 'library/LICENSE.md'),
        'Original migration terms.\r\n',
      );
      assert.equal(
        fileText(tree, 'library/NOTICE.md'),
        'Original migration notice.\r\n',
      );
    },
  },
  {
    id: 'library.group',
    capabilities: ['authoring.library'],
    args: [
      'library',
      'add',
      'group',
      'practices/testing',
      '--directory',
      'library',
      ...group,
    ],
    status: 0,
    readOnly: false,
  },
  {
    id: 'library.draft',
    capabilities: ['authoring.library'],
    args: [
      'library',
      'add',
      'rule',
      'practices/testing/budget',
      '--directory',
      'library',
      ...rule,
    ],
    status: 0,
    readOnly: false,
    verify: (_, tree) =>
      assert.match(
        fileText(tree, 'library/practices/testing/budget.md'),
        /code-rules:draft/,
      ),
  },
  {
    id: 'library.reject-draft',
    capabilities: ['authoring.library'],
    args: ['library', 'check', '--directory', 'library'],
    status: 1,
    readOnly: true,
  },
  {
    id: 'library.check',
    capabilities: ['authoring.library'],
    args: ['library', 'check', '--directory', 'library'],
    status: 0,
    readOnly: true,
    prepare: async (root) => {
      const path = join(root, 'library/practices/testing/budget.md');
      const draft = await readFile(path, 'utf8');
      await writeFile(
        path,
        draft.slice(0, draft.indexOf('\n---\n', 4) + 5) +
          '\nAllow three attempts.\n',
      );
    },
    verify: (result) => {
      const value = object(JSON.parse(result.stdout));
      assert.equal(value.groups, 1);
      assert.equal(value.rules, 1);
    },
  },
  {
    id: 'sync.failure-preserves',
    capabilities: ['project.sync', 'files.safe-write'],
    args: ['sync'],
    status: 1,
    readOnly: true,
    prepare: async (root) => {
      const path = join(root, '.code-rules/config.json');
      const config = object(JSON.parse(await readFile(path, 'utf8')));
      const team = object(object(config.sources).team);
      delete team.version;
      team.ref = 'missing-tag';
      await writeFile(path, JSON.stringify(config, null, 2));
    },
  },
  {
    id: 'authoring.symlink',
    capabilities: ['files.safe-write', 'authoring.library'],
    args: ['library', 'init', '--directory', 'linked'],
    status: 1,
    readOnly: true,
    prepare: async (root) => {
      await mkdir(join(root, 'outside'));
      await write(root, 'outside/sentinel', 'Preserve me.');
      await symlink('outside', join(root, 'linked'));
    },
  },
];
