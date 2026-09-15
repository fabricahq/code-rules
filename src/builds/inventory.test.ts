/** @fileoverview Checks that Builds uses a complete attachment inventory without silently dropping rule obligations. */

import { expect, test } from 'bun:test';
import { buildRules } from './index';
import type { LibrarySnapshot } from './index';
import { exampleFiles } from '../../tests/fixtures/import-fixture';

/** Build one original library with caller-controlled text and retained-file paths. */
function build(snapshot: LibrarySnapshot): string | undefined {
  return buildRules({
    configuration: {
      schemaVersion: 1,
      sources: {
        example: {
          repository: 'https://github.com/fixture/one.git',
          ref: 'v1',
          groups: ['practices/testing'],
          exclude: {},
          replace: {},
        },
      },
    },
    snapshots: { example: snapshot },
    localFiles: {},
    toolVersion: 'test',
  }).files['groups/practices/testing.md'];
}

const text = Object.fromEntries(
  Object.entries(exampleFiles).filter(
    (entry): entry is [string, string] => typeof entry[1] === 'string',
  ),
);
const snapshot: LibrarySnapshot = {
  repository: 'https://github.com/fixture/one.git',
  ref: 'v1',
  resolvedCommit: 'a'.repeat(40),
  groups: ['practices/testing'],
  files: text,
  filePaths: Object.keys(exampleFiles),
};

test('links to binary attachments present only in the inventory', () => {
  expect(build(snapshot)).toContain(
    '../../../vendor/example/assets/images/flow.png',
  );
  expect(build(snapshot)).toContain(
    '../../../vendor/example/assets/special.pdf',
  );
});

test('rejects text absent from the explicit inventory', () => {
  expect(() => build({ ...snapshot, filePaths: [] })).toThrow(
    'text file missing from inventory',
  );
});

test('rejects rule paths omitted from the text map instead of dropping the rule', () => {
  expect(() =>
    build({
      ...snapshot,
      filePaths: [
        ...Object.keys(exampleFiles),
        'practices/testing/missing-rule.md',
      ],
    }),
  ).toThrow('rule text missing from snapshot');
});

test('validates inventory containment and duplicate paths', () => {
  expect(() => build({ ...snapshot, filePaths: ['../outside'] })).toThrow(
    'contained relative path',
  );
  expect(() =>
    build({ ...snapshot, filePaths: ['duplicate', 'duplicate'] }),
  ).toThrow('duplicate entries');
});
