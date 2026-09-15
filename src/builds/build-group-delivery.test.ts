/** @fileoverview Checks inline and summary group delivery while preserving standalone rules through buildRules. */

import { expect, test } from 'bun:test';
import { buildRules } from './index';
import type { BuildInput } from './index';
import {
  group,
  ruleId,
  ruleText,
  source,
  snapshot,
  input,
  generated,
  localInput,
  indexedPaths,
} from './build-test-fixtures';

test.each([0, 8192])(
  'should preserve replacement identity and exclusions with inline budget %s',
  (groupInlineMaxBytes) => {
    const id = `${group}/verify-retries`;
    const build: BuildInput = {
      ...input(),
      groupInlineMaxBytes,
      configuration: {
        schemaVersion: 1,
        sources: {
          fabrica: source(
            'https://github.com/fabrica/rules.git',
            { [`${group}/obsolete`]: 'No longer applicable.' },
            {
              [id]: {
                file: `local/${group}/replacement.md`,
                reason: 'Project retry contract.',
              },
            },
          ),
        },
        localGroups: [],
      },
      snapshots: {
        fabrica: snapshot(
          'https://github.com/fabrica/rules.git',
          'Superseded',
          {
            [`${group}/obsolete.md`]: ruleText('Excluded', 'Excluded body.'),
          },
        ),
      },
      localFiles: {
        [`${group}/replacement.md`]: ruleText(
          'Project retries',
          'Use exactly three attempts.',
        ).replace(
          'When planning, implementing, or reviewing retries.',
          'When adding requests to the project API.',
        ),
        [`${group}/success.md`]: ruleText(
          'Stop after success',
          'Never retry a successful result.',
        ),
      },
    };
    const output = buildRules(build).files;
    const index = output[`groups/${group}.md`] ?? '';
    expect(index).toContain(
      '**When to read:** When adding requests to the project API.',
    );
    expect(index).toContain('**Impact:** HIGH');
    expect(index.includes('Use exactly three attempts.')).toBe(
      groupInlineMaxBytes > 0,
    );
    expect(index).toContain('**Why it matters:** Prevent unbounded retries.');
    expect(index).toContain('regardless of impact');
    expect(index).toContain(
      'Assess findings from concrete evidence and consequences.',
    );
    expect(output[`rules/fabrica/${id}.md`]).toContain(
      'Use exactly three attempts.',
    );
    expect(
      Object.keys(output).filter(
        (path) => path.startsWith('rules/') && path !== 'rules/README.md',
      ),
    ).toEqual([`rules/fabrica/${id}.md`, `rules/local/${group}/success.md`]);
    expect(
      indexedPaths(`groups/${group}.md`, index).filter((path) =>
        path.startsWith('rules/'),
      ),
    ).toEqual([`rules/fabrica/${id}.md`, `rules/local/${group}/success.md`]);
    expect(Object.values(output).join('\n')).not.toContain('Excluded body.');
    expect(Object.values(output).join('\n')).not.toContain('title: Superseded');
  },
);

test('should inline a complete group at the default UTF-8 boundary and use summaries one byte above it', () => {
  const file = `${group}/retry.md`;
  const body = '验证重试。\n\n```ts\nconst attempts = 3;\n```\n\nEnd.';
  const initial = localInput({ [file]: ruleText('Retry', body) });
  const size = Buffer.byteLength(
    generated(initial, `groups/${group}.md`),
    'utf8',
  );
  const build = localInput({
    [file]: ruleText('Retry', body + 'x'.repeat(8192 - size)),
  });
  const inline = buildRules(build).files;
  expect(Buffer.byteLength(inline[`groups/${group}.md`] ?? '', 'utf8')).toBe(
    8192,
  );
  expect(inline[`groups/${group}.md`]).toContain(
    'Full rules are included below.',
  );
  expect(inline[`groups/${group}.md`]).toContain('const attempts = 3;');

  const larger = buildRules(
    localInput({ [file]: ruleText('Retry', body + 'x'.repeat(8193 - size)) }),
  ).files;
  expect(larger[`groups/${group}.md`]).toContain(
    'This file contains summaries only.',
  );
  expect(larger[`groups/${group}.md`]).toContain('**Read full rule:** [Retry]');
  expect(larger[`groups/${group}.md`]).not.toContain('const attempts = 3;');
  expect(larger[`rules/local/${file}`]).toContain('const attempts = 3;');
});

test('should change only group delivery when inline is disabled or the page budget is smaller', () => {
  const build = localInput({
    [`${ruleId}.md`]: ruleText('Retry', 'Complete guidance. '.repeat(150)),
  });
  const inline = buildRules(build).files;
  const disabled = buildRules({ ...build, groupInlineMaxBytes: 0 }).files;
  const bounded = buildRules({ ...build, indexMaxBytes: 2000 }).files;
  expect(inline[`groups/${group}.md`]).toContain(
    'Full rules are included below.',
  );
  for (const output of [disabled, bounded]) {
    expect(output[`groups/${group}.md`]).toContain(
      'This file contains summaries only.',
    );
    expect(output[`rules/local/${ruleId}.md`]).toBe(
      inline[`rules/local/${ruleId}.md`],
    );
    expect(output['provenance.json']).toBe(inline['provenance.json']);
    expect(output['RULES.md']).toBe(inline['RULES.md']);
  }
  expect(
    Buffer.byteLength(bounded[`groups/${group}.md`] ?? '', 'utf8'),
  ).toBeLessThanOrEqual(2000);
});

test.each([-1, 1.5, NaN, Infinity, Number.MAX_SAFE_INTEGER + 1])(
  'should reject an invalid inline byte budget: %s',
  (groupInlineMaxBytes) => {
    expect(() => buildRules({ ...input(), groupInlineMaxBytes })).toThrow(
      'groupInlineMaxBytes',
    );
  },
);

test('should keep empty groups explicit without claiming to include full rules', () => {
  const output = generated(localInput({}), `groups/${group}.md`);
  expect(output).toContain('No active rules in this group.');
  expect(output).not.toContain('Full rules are included below.');
});

test.each([0, 8192])(
  'should separate group pages from effective rules with inline budget %s',
  (groupInlineMaxBytes) => {
    const output = buildRules({
      ...localInput({ [`${ruleId}.md`]: ruleText('Retry') }),
      groupInlineMaxBytes,
    }).files;
    const page = output[`groups/${group}.md`] ?? '';
    expect(
      [
        ...new Set(Object.keys(output).map((path) => path.split('/')[0])),
      ].sort(),
    ).toEqual(['RULES.md', 'groups', 'provenance.json', 'rules']);
    expect(output['RULES.md']).toContain(`(groups/${group}.md)`);
    expect(page).toContain(`Group ID: \`${group}\``);
    expect(page).toContain('\n## Rules\n\n### Retry\n');
    expect(output[`rules/local/${ruleId}.md`]).toStartWith('# Retry\n');
    expect(page).toContain(`(../../rules/local/${ruleId}.md)`);
    expect(page).toContain('[RULES.md](../../RULES.md)');
    expect(page).toContain('[provenance.json](../../provenance.json)');
    expect(output[`rules/local/${ruleId}.md`]).toContain(
      `Rule ID: \`local:${ruleId}\``,
    );
    if (groupInlineMaxBytes > 0)
      expect(page).toContain(
        `**Rule source:** [Original rule](../../../local/${ruleId}.md)`,
      );
  },
);
