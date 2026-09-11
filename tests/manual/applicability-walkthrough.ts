/** @fileoverview Runs the Runbooks walkthrough against the real builder and captures inspectable examples in its generated-file panel. */

import assert from 'node:assert/strict';
import { mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { buildRules, BuildError } from '../../src/builds';
import type { BuildInput, FileContents } from '../../src/builds';
import { buildsExampleInput } from './builds-fixture';

const action = process.argv[2];
const choice = process.argv[3];
const configuredCaptureDirectory = process.env.GENERATED_FILES;
assert(
  configuredCaptureDirectory,
  'Launch this walkthrough through Runbooks to capture its files.',
);
const captureDirectory = configuredCaptureDirectory;
let input: BuildInput = structuredClone(buildsExampleInput);

/** Return expected file contents, or fail with the missing path; empty files remain valid. */
function requireFile({
  files,
  path,
}: {
  readonly files: FileContents;
  readonly path: string;
}): string {
  const content = files[path];
  assert(content !== undefined, `Missing expected file: ${path}`);
  return content;
}

async function capture(path: string, content: string): Promise<void> {
  const destination = join(captureDirectory, path);
  await mkdir(join(destination, '..'), { recursive: true });
  await writeFile(destination, content);
}

async function captureBuild(
  prefix: string,
): Promise<ReturnType<typeof buildRules>> {
  const result = buildRules(input);
  // Each experiment owns its output directory; remove obsolete parts only after a successful build.
  await rm(join(captureDirectory, prefix, 'generated'), {
    recursive: true,
    force: true,
  });
  for (const [path, content] of Object.entries(result.files)) {
    await capture(`${prefix}/generated/${path}`, content);
  }
  // Keep the builder's relative source links usable inside the file panel.
  for (const [path, content] of Object.entries(input.localFiles)) {
    await capture(`${prefix}/local/${path}`, content);
  }
  for (const [source, snapshot] of Object.entries(input.snapshots)) {
    for (const [path, content] of Object.entries(snapshot.files)) {
      await capture(`${prefix}/vendor/${source}/${path}`, content);
    }
  }
  await capture(
    `${prefix}/config.json`,
    JSON.stringify(input.configuration, null, 2) + '\n',
  );
  return result;
}

switch (action) {
  case 'build': {
    const { files } = await captureBuild('01-deliverable');
    const index = requireFile({ files, path: 'practices/testing.md' });
    const replacement = requireFile({
      files,
      path: 'rules/example/practices/testing/verify-retries.md',
    });
    assert(index.includes('Full rules are included below.'));
    assert(index.includes('exactly three attempts'));
    assert(index.includes('**Why it matters:**'));
    assert(index.includes('Verify the project retry budget'));
    assert(index.includes('Stop retries after success'));
    assert(!index.includes('legacy-backoff'));
    assert(!Object.keys(files).some((path) => path.includes('legacy-backoff')));
    assert(replacement.includes('exactly three attempts'));
    assert(replacement.includes('example:practices/testing/verify-retries'));
    console.log(
      'Built the real example. Open 01-deliverable/generated/RULES.md in Files.',
    );
    console.log(
      'Then inspect practices/testing.md and rules/example/practices/testing/verify-retries.md.',
    );
    console.log(
      'Verified: replacement keeps its ID; excluded backoff rule is absent; local success rule is indexed.',
    );
    console.log(
      'config.json, local/, vendor/, and generated/provenance.json show where those decisions came from.',
    );
    break;
  }
  case 'delivery': {
    const inline = buildRules(input).files;
    input = { ...input, groupInlineMaxBytes: 0 };
    const { files } = await captureBuild('01-summary-only');
    const summaries = requireFile({ files, path: 'practices/testing.md' });
    assert(summaries.includes('This file contains summaries only.'));
    assert(summaries.includes('**Read full rule:**'));
    assert(!summaries.includes('exactly three attempts'));
    for (const [path, content] of Object.entries(files)) {
      if (path.startsWith('rules/') || path === 'provenance.json')
        assert.equal(content, inline[path]);
    }
    console.log(
      'Compare 01-deliverable with 01-summary-only. Full rule files and provenance are identical; only group delivery changes.',
    );
    break;
  }
  case 'scenario': {
    assert(
      choice === 'implementation' || choice === 'validation',
      'Choose an implementation or validation scenario.',
    );
    const prefix = `02-${choice}`;
    await captureBuild(prefix);
    const explanation =
      choice === 'implementation'
        ? `# Before writing a TypeScript retry client\n\nNo function exists yet. Start at generated/RULES.md.\n\n1. Read techs/typescript.md: designing the retry return type makes the TypeScript rule relevant.\n2. Read practices/testing.md: new behavior needs tests even before test files exist.\n3. Read practices/code-design.md: the operation will coordinate request execution, retry decisions, and its final result.\n4. Skip techs/go.md for this TypeScript task.\n5. Read the full definitions included in these small groups before implementing. Larger groups require following their Read full rule links. Plan an explicit exhausted result, exactly three total attempts, and an immediate stop after success.\n\nThe design rule allows cohesive inline steps. Creating helper functions is not itself the goal.\n`
        : `# Review an attempt-limit change\n\nThe proposed TypeScript change raises total attempts from three to five. No test file changed.\n\n1. Start at generated/RULES.md and include the testing practice because behavior changed.\n2. Read the full retry-budget rule in practices/testing.md; its separate file is rules/example/practices/testing/verify-retries.md.\n3. The effective replacement requires exactly three attempts. A deterministic all-failure test can demonstrate five calls and establish the mismatch.\n4. Cite example:practices/testing/verify-retries, the changed limit, and that observable behavior.\n5. Code-design relevance alone proves no violation. Inspect whether a step's purpose is obscured; helper count is insufficient evidence. Impact describes why a rule matters, while finding severity depends on the observed consequence.\n\nThis is a guided interpretation of the example, not an automated review of your application.\n`;
    await capture(`${prefix}/START-HERE.md`, explanation);
    console.log(explanation);
    break;
  }
  case 'budget': {
    assert(
      choice === 'default' || choice === 'small' || choice === 'tiny',
      'Choose a budget preset.',
    );
    const group = 'practices/testing';
    const snapshot = input.snapshots.example;
    assert(snapshot !== undefined, 'Missing expected snapshot: example');
    const source = buildsExampleInput.configuration.sources.example;
    const configuration = {
      ...buildsExampleInput.configuration,
      sources: {
        example: { ...source, groups: [group], exclude: {}, replace: {} },
      },
    };
    const snapshots = {
      example: {
        ...snapshot,
        groups: [group],
        files: {
          'rule-library.json': requireFile({
            files: snapshot.files,
            path: 'rule-library.json',
          }),
          [`${group}/_group.json`]: requireFile({
            files: snapshot.files,
            path: `${group}/_group.json`,
          }),
          ...Object.fromEntries(
            Array.from({ length: 40 }, (_, index) => [
              `${group}/scenario-${String(index + 1).padStart(2, '0')}.md`,
              `---\ntitle: Verify retry scenario ${index + 1}\nwhenToRead: When planning or reviewing retry scenario ${index + 1}.\nimpact: HIGH\nimpactDescription: Catch retry failures.\ntags: testing, retries\n---\n\n## Verify retry scenario ${index + 1}\n\nThis illustrative body must remain complete in its own file.\n`,
            ]),
          ),
        },
      },
    };
    const budget =
      choice === 'default' ? 24 * 1024 : choice === 'small' ? 3000 : 100;
    input = {
      ...input,
      configuration,
      snapshots,
      localFiles: {},
      indexMaxBytes: budget,
    };
    if (choice === 'tiny') {
      assert.throws(
        () => buildRules(input),
        (error: unknown) => {
          assert(error instanceof BuildError);
          assert(error.message.includes('indexMaxBytes'));
          console.log(`Expected refusal at ${budget} bytes: ${error.message}`);
          return true;
        },
      );
      await capture(
        '03-budget-tiny/RESULT.md',
        '# Expected refusal\n\nA 100-byte limit cannot fit the index instructions and one complete entry. Builds rejects the request instead of silently truncating it. No build output is returned.\n',
      );
      break;
    }
    const prefix = `03-budget-${choice}`;
    const { files } = await captureBuild(prefix);
    const indexes = Object.entries(files).filter(
      ([path]) =>
        path === 'RULES.md' ||
        path.startsWith('RULES.part-') ||
        path.startsWith(`${group}.`),
    );
    const bodies = Object.keys(files).filter((path) =>
      path.startsWith('rules/'),
    );
    assert.equal(bodies.length, 40);
    assert(
      indexes.every(
        ([, content]) => Buffer.byteLength(content, 'utf8') <= budget,
      ),
    );
    const partCount = indexes.filter(([path]) =>
      path.includes('.part-'),
    ).length;
    assert(choice === 'small' ? partCount > 0 : partCount === 0);
    const report = `# Index budget: ${budget} UTF-8 bytes\n\n${bodies.length} full definitions, ${partCount} numbered index parts.\n\n| Index | UTF-8 bytes |\n| --- | ---: |\n${indexes.map(([path, content]) => `| ${path} | ${Buffer.byteLength(content, 'utf8')} |`).join('\n')}\n\nOpen generated/practices/testing.md. If split, read every listed part; all 40 bodies remain available under generated/rules/.\n`;
    await capture(`${prefix}/SIZES.md`, report);
    console.log(report);
    break;
  }
  case 'metadata': {
    const snapshot = input.snapshots.example;
    assert(snapshot !== undefined, 'Missing expected snapshot: example');
    const path = 'practices/testing/verify-retries.md';
    const snapshots = {
      example: {
        ...snapshot,
        files: {
          ...snapshot.files,
          [path]: requireFile({ files: snapshot.files, path }).replace(
            /^whenToRead:.*\n/m,
            '',
          ),
        },
      },
    };
    assert.throws(
      () => buildRules({ ...input, snapshots }),
      (error: unknown) => {
        assert(error instanceof BuildError);
        assert(error.message.includes('whenToRead'));
        console.log(`Expected metadata rejection: ${error.message}`);
        return true;
      },
    );
    console.log(
      'Even replaced source rules must have valid metadata. Add whenToRead to each rule before rebuilding.',
    );
    break;
  }
  case 'authoring': {
    const snapshot = input.snapshots.example;
    assert(snapshot !== undefined, 'Missing expected snapshot: example');
    const guidance = await readFile(
      new URL(
        '../../docs/src/content/docs/reference/rule-authoring.md',
        import.meta.url,
      ),
      'utf8',
    );
    await capture('04-authoring/rule-rubric-and-template.md', guidance);
    await capture(
      '04-authoring/name-retry-stages.md',
      requireFile({
        files: snapshot.files,
        path: 'practices/code-design/name-retry-stages.md',
      }),
    );
    console.log(
      'Open 04-authoring/rule-rubric-and-template.md and name-retry-stages.md.',
    );
    console.log(
      'whenToRead: decide relevance before code exists. Body: implementation choices. Validation: observable evidence and exceptions.',
    );
    console.log(
      'One shared template; add implementation and validation headings only when they contribute useful guidance.',
    );
    break;
  }
  default:
    throw new Error(
      'Unknown walkthrough action. Use the controls in the runbook.',
    );
}
