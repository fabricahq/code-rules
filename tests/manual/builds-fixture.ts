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
  return `---\ntitle: ${title}\nwhenToRead: ${whenToRead}\nimpact: HIGH\nimpactDescription: ${impactDescription}\n---\n\n## ${title}\n\n${obligation}${implementationSection}\n\n### Validation\n\n${validation}\n`;
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
// Original examples make TypeScript large enough to demonstrate automatic summary delivery.
const additionalTypeScriptRules = {
  'validate-retry-options': {
    title: 'Validate retry options at the boundary',
    whenToRead:
      'When accepting retry settings from configuration, user input, or an external caller.',
    impactDescription:
      'Invalid limits or delays can produce unbounded retries or requests that never execute.',
    obligation:
      'Validate runtime retry options before starting an operation. Require a positive integer attempt limit and finite non-negative delays.',
    implementation:
      'Parse external configuration once into a validated policy. Return a useful configuration error before invoking the request callback when a value is invalid.',
    validation:
      'Exercise zero, negative, fractional, and non-finite limits. Verify that invalid options fail before any request is sent; a TypeScript annotation alone does not validate runtime input.',
  },
  'narrow-caught-errors': {
    title: 'Narrow caught values before inspecting them',
    whenToRead:
      'When a TypeScript catch block inspects a failure to decide whether to retry.',
    impactDescription:
      'Assuming every thrown value is an Error can hide the original failure behind another exception.',
    obligation:
      'Treat caught values as unknown and establish their shape before reading properties. Preserve unexpected values when returning or reporting the final failure.',
    implementation:
      'Use a type guard or an instanceof check appropriate to the client contract. Define how the policy handles string throws, null, and unfamiliar objects without asserting that they have a status code.',
    validation:
      'Drive the failure path with an Error, a string, null, and an unfamiliar object. Confirm that classification does not throw and that the final result retains the relevant original failure.',
  },
  'handle-every-outcome': {
    title: 'Handle every retry outcome explicitly',
    whenToRead:
      'When consuming a discriminated union representing success, exhausted retries, or cancellation.',
    impactDescription:
      'An unhandled outcome can silently fall through and leave callers with an incorrect result.',
    obligation:
      'Handle each declared outcome explicitly. Use an exhaustive check so adding an outcome causes affected consumers to be reconsidered.',
    implementation:
      'Switch on the discriminant and keep the success value and failure details in their respective branches. Place an exhaustiveness check after the handled cases instead of treating every remaining variant as success.',
    validation:
      'Check a consumer for each union member. When a new member is introduced, confirm that consumers must handle it; a default branch returning a success-shaped value is insufficient.',
  },
  'propagate-cancellation': {
    title: 'Propagate cancellation through requests and waits',
    whenToRead:
      'When implementing a retry API that accepts an AbortSignal or another cancellation contract.',
    impactDescription:
      'Ignoring cancellation can keep requests running after the caller has abandoned the operation.',
    obligation:
      'Apply cancellation to both the request and the delay between attempts. Stop scheduling attempts after cancellation and expose the documented cancellation outcome.',
    implementation:
      'Pass the signal to supported dependencies and make retry waits interruptible. Check cancellation before starting another request, including when cancellation arrives just after a previous failure.',
    validation:
      'Cancel before the first attempt, during a request, and during a retry wait. Verify that subsequent attempts do not start and that each path produces the promised cancellation result.',
  },
  'settle-request-promises': {
    title: 'Account for every request promise',
    whenToRead:
      'When starting asynchronous requests or asynchronous callbacks inside a retry operation.',
    impactDescription:
      'Unobserved rejections can escape the result contract and make failures hard to diagnose.',
    obligation:
      'Await or return each request promise through the operation result. Deliberate background work needs an explicit owner and failure-handling contract.',
    implementation:
      'Keep the attempt loop connected to the promise returned to the caller. If an asynchronous hook is supported, document whether the operation waits for it and how its failure affects the result.',
    validation:
      'Reject a request and any supported asynchronous hook. Verify that each rejection reaches its documented handler, with no unhandled promise rejection or premature success result.',
  },
  'preserve-caller-policy': {
    title: 'Keep caller retry policies immutable',
    whenToRead:
      'When accepting a retry policy object that callers may reuse for multiple operations.',
    impactDescription:
      'Mutating a shared policy can cause concurrent or later operations to use unexpected limits.',
    obligation:
      'Keep per-operation counters and state separate from caller-owned policy objects. Do not decrement or overwrite configuration values as attempts proceed.',
    implementation:
      'Accept policy inputs as readonly and store mutable attempt state inside the operation. If nested policy data needs normalization, create owned values rather than mutating the caller object.',
    validation:
      'Reuse the same policy for two operations and verify independent attempt counts. Compare the policy before and after success and failure, including any nested configuration that the operation reads.',
  },
  'make-delay-units-explicit': {
    title: 'Make retry delay units explicit',
    whenToRead:
      'When a retry API accepts, computes, or passes durations across an interface.',
    impactDescription:
      'Confusing seconds and milliseconds can produce excessive traffic or unexpectedly long waits.',
    obligation:
      'State duration units in the API and convert them at explicit boundaries. Use names such as delayMs when the value is measured in milliseconds.',
    implementation:
      'Normalize external durations into the operation’s chosen unit once. Keep the configured delay, computed backoff, and timer argument consistent, and validate the result of any conversion.',
    validation:
      'Use a known duration to check the timer argument and verify the documented unit. Include conversion and upper-bound cases; a bare number type provides no evidence that units agree.',
  },
  'inject-retry-timing': {
    title: 'Make retry timing controllable in tests',
    whenToRead:
      'When retry behavior depends on timers, elapsed time, or random jitter.',
    impactDescription:
      'Tests tied to wall-clock waits and random timing can be slow or fail intermittently.',
    obligation:
      'Provide a way to control time and randomness when testing the retry policy. Keep production timing defaults behind the same behavior contract.',
    implementation:
      'Use the project’s supported fake timers or accept a clock, wait function, and random source where needed. Test policy decisions with controlled values while keeping a focused check of production timer integration.',
    validation:
      'Drive a known sequence of delays without real waiting and verify attempt order and deadline handling. Repeat the scenario with the same inputs and confirm the same outcome.',
  },
};
const otherRules = {
  ...Object.fromEntries(
    Object.entries(additionalTypeScriptRules).map(([slug, rule]) => [
      `${technologyGroup}/${slug}.md`,
      exampleRule(rule),
    ]),
  ),
  [`${technologyGroup}/_group.json`]: JSON.stringify(
    {
      name: 'TypeScript',
      description: 'Express retry outcomes in TypeScript.',
      whenToRead: ['When the intended or existing code uses TypeScript.'],
    },
    null,
    2,
  ),
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
  [`${designGroup}/_group.json`]: JSON.stringify(
    {
      name: 'Code design',
      description:
        'Organize code around clear responsibilities and understandable interactions.',
      whenToRead: [
        'Before planning, writing, changing, or reviewing how code is organized, how responsibilities are divided, or how functions and modules work together.',
      ],
    },
    null,
    2,
  ),
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
  [`${unrelatedGroup}/_group.json`]: JSON.stringify(
    {
      name: 'Go',
      description: 'Write Go retry APIs.',
      whenToRead: ['When the intended or existing code uses Go.'],
    },
    null,
    2,
  ),
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
        repository: 'https://github.com/example/rules.git',
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
  },
  snapshots: {
    example: {
      repository: 'https://github.com/example/rules.git',
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
