/** @fileoverview Checks native Markdown discovery against the pinned reference and independent expected targets. */

import { deepStrictEqual, equal } from 'node:assert';
import { resolve } from 'node:path';
import { markdownTargets } from '../../src/formats/markdown-links';

const candidate = process.argv[2];
if (!candidate) throw new Error('Expected Go adapter path');
const cases: ReadonlyArray<{ text: string; expected: string[] }> = [
  { text: 'plain', expected: [] },
  { text: '[one](a.md) [two](a.md#part)', expected: ['techs/go/a.md'] },
  { text: '![image](assets/r/a.png)', expected: ['techs/go/assets/r/a.png'] },
  {
    text: '[link][ref]\n\n[ref]: /assets/a%20b.md "Title"',
    expected: ['assets/a b.md'],
  },
  { text: '[unused]: /assets/a.md', expected: ['assets/a.md'] },
  { text: '`[code](none)`\n\n```\n[code](none)\n```', expected: [] },
  {
    text: '[external](https://example.com) [fragment](#part)',
    expected: ['techs/go/r.md'],
  },
  {
    text: '[nested](assets/r/a(b).png)',
    expected: ['techs/go/assets/r/a(b).png'],
  },
  {
    text: '[entity](assets/r/a&amp;b.png)',
    expected: ['techs/go/assets/r/a&b.png'],
  },
  {
    text: '---\nfield: "[skip](secret)"\n---\n[body](a.md)',
    expected: ['techs/go/a.md'],
  },
];
for (const { text, expected } of cases) {
  const input = { text, file: 'techs/go/r.md' };
  deepStrictEqual(markdownTargets(text, input.file), expected);
  const result = Bun.spawnSync([resolve(candidate)], {
    stdin: Buffer.from(
      JSON.stringify({
        operation: 'markdownTargets',
        input,
        location: 'links',
      }) + '\n',
    ),
    stdout: 'pipe',
    stderr: 'pipe',
    timeout: 10_000,
  });
  equal(result.exitCode, 0, result.stderr.toString());
  deepStrictEqual(JSON.parse(result.stdout.toString()), {
    ok: true,
    value: expected,
  });
}
console.log(
  `PASS: ${cases.length} Markdown discovery comparisons with independent expectations`,
);
