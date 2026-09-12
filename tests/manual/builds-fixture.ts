/** @fileoverview Defines an original retry-client library with exclusions, replacement, and local additions for manual walkthroughs. */

import type { BuildInput } from '../../src/builds';

/** Create a complete rule document for the manual test scenario. */
function exampleRule({
  title,
  obligation,
  impactDescription,
  whenToRead = 'When planning, implementing, or reviewing retry behavior.',
  validation = 'Check the retry count and final result with a deterministic test.',
  implementation,
}: {
  readonly title: string;
  readonly obligation: string;
  readonly impactDescription: string;
  readonly whenToRead?: string;
  readonly validation?: string;
  readonly implementation?: string;
}): string {
  const implementationSection = implementation
    ? `\n\n### Implementation\n\n${implementation}`
    : '';
  return `---\ntitle: ${title}\nwhenToRead: ${whenToRead}\nimpact: HIGH\nimpactDescription: ${impactDescription}\ntags: testing, retries\n---\n\n## ${title}\n\n${obligation}${implementationSection}\n\n### Validation\n\n${validation}\n`;
}

const group = 'practices/testing';
const groupMetadata = JSON.stringify(
  {
    name: 'Testing',
    description: 'Verify behavior with meaningful tests.',
    whenToRead: [
      'When adding, changing, reviewing, or diagnosing behavior, even when no test files have changed.',
    ],
  },
  null,
  2,
);
const technologyGroup = 'techs/typescript';
const designGroup = 'practices/code-design';
const unrelatedGroup = 'techs/go';
const groups = [group, technologyGroup, designGroup, unrelatedGroup];
const otherRules = {
  [`${technologyGroup}/_group.json`]: JSON.stringify({
    name: 'TypeScript',
    description: 'Express retry outcomes in TypeScript.',
    whenToRead: ['When the intended or existing code uses TypeScript.'],
  }),
  [`${technologyGroup}/retry-outcome.md`]: exampleRule({
    title: 'Represent retry exhaustion explicitly',
    obligation:
      'Distinguish a successful result from exhausted attempts in the return type.',
    impactDescription:
      'Ambiguous results can cause callers to treat failed requests as successful.',
    whenToRead: 'When designing or reviewing a TypeScript retry API.',
    validation:
      'Check that callers can distinguish exhausted attempts from a successful result using the declared return type.',
  }),
  [`${designGroup}/_group.json`]: JSON.stringify({
    name: 'Code design',
    description:
      'Organize code around clear responsibilities and understandable interactions.',
    whenToRead: [
      'Before planning, writing, changing, or reviewing how code is organized, how responsibilities are divided, or how functions and modules work together.',
    ],
  }),
  [`${designGroup}/name-retry-stages.md`]: exampleRule({
    title: 'Name the retry stages',
    obligation:
      'Make request execution, retry decisions, and final results recognizable as separate steps.',
    impactDescription:
      'Mixing orchestration with low-level details can hide important decisions and make behavior harder to verify or change.',
    whenToRead:
      'Before planning, writing, changing, or reviewing an operation that coordinates request execution, retry decisions, and final results.',
    validation:
      'Read the operation in order and identify any stage whose purpose is obscured. A short, cohesive function does not need extraction merely to become smaller; helper count alone is insufficient evidence of a violation.',
    implementation:
      'Make each step understandable. Extract parsing, validation, or result construction when those details obscure the operation. Keep cohesive inline steps when their purpose is already clear.',
  }),
  [`${unrelatedGroup}/_group.json`]: JSON.stringify({
    name: 'Go',
    description: 'Write Go retry APIs.',
    whenToRead: ['When the intended or existing code uses Go.'],
  }),
  [`${unrelatedGroup}/retry-errors.md`]: exampleRule({
    title: 'Return retry errors to the caller',
    obligation:
      'Return the final failure to the caller of the Go retry function.',
    impactDescription:
      'Swallowed retry errors prevent callers from recognizing and handling a failed operation.',
    whenToRead: 'When writing or reviewing a retry function in Go.',
    validation:
      'Drive repeated failures and verify that the caller receives the last error.',
  }),
};
/** Shared illustrative inputs for the manual example and interactive walkthrough. */
export const buildsExampleInput = {
  toolVersion: '0.0.0-example',
  configuration: {
    schemaVersion: 1,
    sources: {
      example: {
        repository: 'example/rules',
        ref: 'v1.0.0',
        groups,
        exclude: {
          [`${group}/legacy-backoff`]:
            'The project uses a different backoff contract.',
        },
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
      groups,
      files: {
        'rule-library.json': '{"formatVersion":1}',
        ...otherRules,
        [`${group}/legacy-backoff.md`]: exampleRule({
          title: 'Use the library backoff schedule',
          obligation: 'Use the original library backoff schedule.',
          impactDescription:
            'Uncoordinated retries can overload a recovering service.',
        }),
        [`${group}/_group.json`]: groupMetadata,
        [`${group}/verify-retries.md`]: exampleRule({
          title: 'Verify retry limits',
          obligation: 'Verify that retries stop at the configured limit.',
          impactDescription:
            'Unbounded retries can overload the service and keep callers waiting indefinitely.',
        }),
      },
    },
  },
  localFiles: {
    [`${group}/retry-budget.md`]: exampleRule({
      title: 'Verify the project retry budget',
      obligation:
        'Assert that a failed request makes exactly three attempts before returning the error.',
      impactDescription:
        'Exceeding the project retry budget can amplify failing requests and delay recovery.',
    }),
    [`${group}/verify-success.md`]: exampleRule({
      title: 'Stop retries after success',
      obligation:
        'Assert that a successful request triggers no further attempts.',
      impactDescription:
        'Retrying successful requests can duplicate side effects.',
    }),
  },
} satisfies BuildInput;
