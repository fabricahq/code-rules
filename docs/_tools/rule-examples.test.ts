/** @fileoverview Checks that copyable documentation rule examples satisfy the public builder's metadata contract. */

import { expect, test } from 'bun:test';
import { readFile } from 'node:fs/promises';
import { buildRules } from '../../src/builds/index';

test.each([
  [
    '../src/components/HomeHero.astro',
    /<pre><code>(---[\s\S]*?)<\/code><\/pre>/u,
  ],
  ['../src/content/docs/guides/write-rules.md', /```md\n(---[\s\S]*?)\n```/u],
])('builds the copyable rule in %s', async (path, pattern) => {
  const document = await readFile(new URL(path, import.meta.url), 'utf8');
  const rule = pattern.exec(document)?.[1];
  if (rule === undefined) throw new Error(`Missing rule example in ${path}`);
  const result = buildRules({
    configuration: {
      schemaVersion: 1,
      sources: {},
      localGroups: ['practices/testing'],
    },
    snapshots: {},
    localFiles: {
      'practices/testing/_group.json': JSON.stringify({
        name: 'Testing',
        description: 'Verify behavior.',
        whenToRead: ['Changing behavior.'],
      }),
      'practices/testing/example.md': rule,
    },
    toolVersion: 'documentation-example',
  });
  expect(result.files['rules/local/practices/testing/example.md']).toContain(
    'Verify retry limits',
  );
});
