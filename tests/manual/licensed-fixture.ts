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
