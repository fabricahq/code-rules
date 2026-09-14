/** @fileoverview Loads a commit-pinned MIT rule adaptation and its retained notices for the offline licensing walkthrough. */

import { readFile } from 'node:fs/promises';
import type { BuildInput } from '../../src/builds';

/** Return a local-only example whose attribution points to the real upstream definition instead of inventing an imported snapshot. */
export async function licensedExampleInput(): Promise<BuildInput> {
  const folder = new URL('./fixtures/licensed-rule/', import.meta.url);
  const [rule, license, notice] = await Promise.all([
    readFile(new URL('prefer-for-of.md', folder), 'utf8'),
    readFile(new URL('LICENSE.md', folder), 'utf8'),
    readFile(new URL('NOTICE.md', folder), 'utf8'),
  ]);
  return {
    configuration: {
      schemaVersion: 1,
      sources: {},
      localGroups: ['techs/javascript'],
    },
    snapshots: {},
    localFiles: {
      'techs/javascript/_group.json':
        JSON.stringify(
          {
            name: 'JavaScript',
            description: 'JavaScript iteration guidance.',
            whenToRead: [
              'When planning, changing, or reviewing JavaScript or TypeScript behavior.',
            ],
          },
          null,
          2,
        ) + '\n',
      'techs/javascript/prefer-for-of.md': rule,
      'licenses/unicorn/LICENSE.md': license,
      'licenses/unicorn/NOTICE.md': notice,
    },
    toolVersion: 'licensed-rule-demonstration',
  };
}

/** Return an illustrative compatible library whose adapted rule inherits the declared MIT default and retains real upstream attribution. */
export async function licensedLibraryExampleInput(): Promise<BuildInput> {
  const localExample = await licensedExampleInput();
  const rule = await readFile(
    new URL(
      './fixtures/licensed-rule/library-prefer-for-of.md',
      import.meta.url,
    ),
    'utf8',
  );
  const groups = ['techs/javascript'];
  const repository = 'example/licensed-code-rules';
  const ref = 'v1.0.0';
  return {
    configuration: {
      schemaVersion: 1,
      sources: {
        licensed: { repository, ref, groups, exclude: {}, replace: {} },
      },
      localGroups: [],
    },
    snapshots: {
      licensed: {
        repository,
        ref,
        resolvedCommit: 'b'.repeat(40),
        groups,
        files: {
          ...localExample.localFiles,
          'techs/javascript/prefer-for-of.md': rule,
          'rule-library.json':
            JSON.stringify(
              {
                formatVersion: 1,
                license: {
                  expression: 'MIT',
                  file: 'licenses/unicorn/LICENSE.md',
                  notices: ['licenses/unicorn/NOTICE.md'],
                },
              },
              null,
              2,
            ) + '\n',
        },
      },
    },
    localFiles: {},
    toolVersion: 'licensed-library-demonstration',
  };
}
