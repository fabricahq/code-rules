/** @fileoverview Loads a commit-pinned MIT rule adaptation and its retained notices for the offline licensing walkthrough. */

import { readFile } from 'node:fs/promises';
import type { BuildInput } from '../../src/builds';

/** Return an illustrative compatible library whose adapted rule uses the declared library-wide MIT license and retains real upstream attribution. */
export async function licensedLibraryExampleInput(): Promise<BuildInput> {
  const folder = new URL('./fixtures/licensed-library/', import.meta.url);
  const [rule, license, notice] = await Promise.all(
    ['prefer-for-of.md', 'LICENSE.md', 'NOTICE.md'].map((path) =>
      readFile(new URL(path, folder), 'utf8'),
    ),
  );
  if (rule === undefined || license === undefined || notice === undefined)
    throw new Error('Missing licensed library fixture');
  const groups = ['techs/javascript'];
  const repository = 'example/licensed-code-rules';
  const ref = 'v1.0.0';
  return {
    configuration: {
      schemaVersion: 1,
      sources: {
        licensed: { repository, ref, groups: '*', exclude: {}, replace: {} },
      },
      localGroups: [],
    },
    snapshots: {
      licensed: {
        repository,
        ref,
        resolvedCommit: 'b'.repeat(40),
        groups,
        groupSelection: '*',
        files: {
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
          'LICENSE.md': license,
          'NOTICE.md': notice,
          'techs/javascript/prefer-for-of.md': rule,
          'rule-library.json':
            JSON.stringify(
              {
                formatVersion: 1,
                license: {
                  expression: 'MIT',
                  file: 'LICENSE.md',
                  notices: ['NOTICE.md'],
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
