/** @fileoverview Defines an original retry-client library with exclusions, replacement, and local additions for manual walkthroughs. */

import type { BuildInput } from '../../src/builds';

/** Create a complete rule document for the manual test scenario. */
function exampleRule(
  title: string,
  obligation: string,
  whenToRead = 'When planning, implementing, or reviewing retry behavior.',
  validation = 'Check the retry count and final result with a deterministic test.',
  implementation?: string,
): string {
  const implementationSection = implementation
    ? `\n\n### Implementation\n\n${implementation}`
    : '';
  return `---\ntitle: ${title}\nwhenToRead: ${whenToRead}\nimpact: HIGH\nimpactDescription: Catch retry failures before they reach users.\ntags: testing, retries\n---\n\n## ${title}\n\n${obligation}${implementationSection}\n\n### Validation\n\n${validation}\n`;
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
const technologyGroup = 'techs/typescript';
const designGroup = 'practices/code-design';
const unrelatedGroup = 'techs/go';
const groups = [group, technologyGroup, designGroup, unrelatedGroup];
const otherRules = {
  [`${technologyGroup}/_group.json`]: JSON.stringify({
    name: 'TypeScript',
    description: 'Express retry outcomes in TypeScript.',
    whenToRead: ['Planning, writing, or reviewing TypeScript.'],
  }),
  [`${technologyGroup}/retry-outcome.md`]: exampleRule(
    'Represent retry exhaustion explicitly',
    'Distinguish a successful result from exhausted attempts in the return type.',
    'When designing or reviewing a TypeScript retry API.',
    'Check that callers can distinguish exhausted attempts from a successful result using the declared return type.',
  ),
  [`${designGroup}/_group.json`]: JSON.stringify({
    name: 'Code design',
    description: 'Keep multi-step operations understandable.',
    whenToRead: [
      'Before planning, writing, changing, or reviewing a function that coordinates multiple steps, such as parsing input, validating it, calling another operation, or constructing a result.',
    ],
  }),
  [`${designGroup}/name-retry-stages.md`]: exampleRule(
    'Name the retry stages',
    'Make request execution, retry decisions, and final results recognizable as separate steps.',
    'When planning, writing, changing, or reviewing an operation that coordinates several steps.',
    'Read the operation in order and identify any stage whose purpose is obscured. A short, cohesive function does not need extraction merely to become smaller; helper count alone is insufficient evidence of a violation.',
    'Make each step understandable. Extract parsing, validation, or result construction when those details obscure the operation. Keep cohesive inline steps when their purpose is already clear.',
  ),
  [`${unrelatedGroup}/_group.json`]: JSON.stringify({
    name: 'Go',
    description: 'Write Go retry APIs.',
    whenToRead: ['Planning or changing Go code.'],
  }),
  [`${unrelatedGroup}/retry-errors.md`]: exampleRule(
    'Return retry errors to the caller',
    'Return the final failure to the caller of the Go retry function.',
    'When writing or reviewing a retry function in Go.',
    'Drive repeated failures and verify that the caller receives the last error.',
  ),
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
        [`${group}/legacy-backoff.md`]: exampleRule(
          'Use the library backoff schedule',
          'Use the original library backoff schedule.',
        ),
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
} satisfies BuildInput;
