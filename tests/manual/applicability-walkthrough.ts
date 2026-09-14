/** @fileoverview Runs the Runbooks walkthrough against the real builder and captures inspectable examples in its generated-file panel. */

import assert from 'node:assert/strict';
import { mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { buildRules, BuildError } from '../../src/builds';
import type { BuildInput, FileContents } from '../../src/builds';
import { buildsExampleInput } from './builds-fixture';
import { licensedLibraryExampleInput } from './licensed-fixture';

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
  case 'groups': {
    assert(
      choice === 'all' || choice === 'practices' || choice === 'techs',
      'Choose all, practices, or techs.',
    );
    const pattern =
      choice === 'all'
        ? '*'
        : choice === 'practices'
          ? 'practices/*'
          : 'techs/*';
    const snapshot = buildsExampleInput.snapshots.example;
    assert(snapshot, 'Missing example snapshot.');
    const groups = snapshot.groups
      .filter((id) => pattern === '*' || id.startsWith(pattern.slice(0, -1)))
      .sort();
    input = {
      configuration: {
        schemaVersion: 1,
        sources: {
          example: {
            repository: snapshot.repository,
            ref: snapshot.ref,
            groups: pattern,
            exclude: {},
            replace: {},
          },
        },
        localGroups: [],
      },
      snapshots: { example: { ...snapshot, groupSelection: pattern, groups } },
      localFiles: {},
      toolVersion: 'group-selection-demonstration',
    };
    const prefix = `06-groups-${choice}`;
    const { files } = await captureBuild(prefix);
    const provenance = JSON.parse(
      requireFile({ files, path: 'provenance.json' }),
    );
    assert.equal(provenance.sources[0].groupSelection, pattern);
    assert.deepEqual(provenance.sources[0].groups, groups);
    for (const id of snapshot.groups) {
      assert.equal(
        Object.hasOwn(files, `groups/${id}.md`),
        groups.includes(id),
      );
    }
    await capture(
      `${prefix}/README.md`,
      `# Selected groups\n\nSelector: \`${pattern}\`\n\nIncluded groups:\n\n${groups.map((id) => `- [${id}](generated/groups/${id}.md)`).join('\n')}\n\nOpen [config.json](config.json) to see the selector and [provenance.json](generated/provenance.json) to see the concrete expansion. This chapter uses the original library definitions to compare group selection.\n`,
    );
    console.log(
      `Open ${prefix}/README.md: ${pattern} includes ${groups.join(', ')}.`,
    );
    break;
  }
  case 'library-licenses': {
    input = await licensedLibraryExampleInput();
    const { files } = await captureBuild('05-licensed-library');
    const provenance = JSON.parse(
      requireFile({ files, path: 'provenance.json' }),
    );
    assert(
      requireFile({ files, path: 'libraries/licensed/README.md' }).includes(
        '**Declared license:** MIT',
      ),
    );
    assert.equal(provenance.sources[0].licenses[0].spdxExpression, 'MIT');
    assert.equal(provenance.sources[0].groupSelection, '*');
    assert.deepEqual(provenance.sources[0].groups, ['techs/javascript']);
    assert.equal(provenance.rules[0].licenseBasis, 'library');
    assert.equal(provenance.rules[0].origin.source, 'licensed');
    assert.deepEqual(provenance.rules[0].licenses, [
      {
        spdxExpression: 'MIT',
        files: ['vendor/licensed/LICENSE.md'],
        attributionFiles: ['vendor/licensed/NOTICE.md'],
        generatedFiles: ['libraries/licensed/licenses/LICENSE.md'],
        generatedAttributionFiles: [
          'libraries/licensed/licenses/notices/001.md',
        ],
      },
    ]);
    for (const path of [
      'rules/licensed/techs/javascript/prefer-for-of.md',
      'groups/techs/javascript.md',
    ]) {
      const rendered = requireFile({ files, path });
      assert(rendered.includes('**Declared license:** MIT'));
      assert(rendered.includes('libraries/licensed/licenses/LICENSE.md'));
      assert(rendered.includes('libraries/licensed/licenses/notices/001.md'));
      assert(!rendered.includes('vendor/licensed/'));
      assert(rendered.includes('5d9d745c5365b6fdb824db1122ff982dd824b11a'));
    }
    const retainedLicense = await readFile(
      join(captureDirectory, '05-licensed-library/vendor/licensed/LICENSE.md'),
    );
    assert.deepEqual(
      retainedLicense,
      await readFile(
        new URL('./fixtures/licensed-library/LICENSE.md', import.meta.url),
      ),
    );
    assert.equal(
      requireFile({ files, path: 'libraries/licensed/licenses/LICENSE.md' }),
      retainedLicense.toString('utf8'),
    );
    assert.equal(
      requireFile({
        files,
        path: 'libraries/licensed/licenses/notices/001.md',
      }),
      await readFile(
        new URL('./fixtures/licensed-library/NOTICE.md', import.meta.url),
        'utf8',
      ),
    );
    await capture(
      '05-licensed-library/README.md',
      '# Follow a library license into generated rules\n\nStart with [the library overview](generated/libraries/licensed/README.md) for the repository, revision, license, and provenance links.\n\n1. Open [config.json](config.json): the project selects every group from the `licensed` source with `groups: "*"`.\n2. Open [the library manifest](vendor/licensed/rule-library.json): it declares MIT and names the license and notice files once for the library.\n3. Open [the source rule](vendor/licensed/techs/javascript/prefer-for-of.md): it has attribution but no rule-level `licenses` field.\n4. Open [the full generated rule](generated/rules/licensed/techs/javascript/prefer-for-of.md): its footer displays the library-wide MIT declaration. Follow the license and notice links into `generated/libraries/licensed/licenses/`: `LICENSE.md` and `notices/001.md` are unchanged copies with standard destinations.\n5. Open [the group](generated/groups/techs/javascript.md): the inline full rule retains the same license and attribution links.\n6. Open [provenance.json](generated/provenance.json): the source records `groupSelection: "*"`, the concrete `groups` list, and MIT with library-relative paths; the effective rule records `licenseBasis: library` and paths starting with `vendor/licensed/`. Original rule license paths resolve from this directory, above generated/. Both source and rule declarations include `generatedFiles` and `generatedAttributionFiles`, relative to generated/, mapping each original file to its output copy.\n\nThe example retains the original ESLint Unicorn attribution and complete MIT text. The licensed library repository `example/licensed-code-rules`, tag, and commit are illustrative; only the original attribution points to a real upstream source. This fixture supplies a compatible snapshot offline and does not fetch a repository. Code Rules preserves declared terms rather than certifying them.\n',
    );
    console.log(
      'Open 05-licensed-library/README.md to trace the library license through the source rule, generated footer, and provenance.',
    );
    break;
  }
  case 'build': {
    const { files } = await captureBuild('01-deliverable');
    const index = requireFile({ files, path: 'groups/practices/testing.md' });
    const replacement = requireFile({
      files,
      path: 'rules/example/practices/testing/verify-retries.md',
    });
    const typescriptIndex = requireFile({
      files,
      path: 'groups/techs/typescript.md',
    });
    assert(typescriptIndex.includes('This file contains summaries only.'));
    assert(typescriptIndex.includes('**Read full rule:**'));
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
      'Compare groups/practices/testing.md (full text) with groups/techs/typescript.md (index), then follow a Read full rule link.',
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
    const summaries = requireFile({
      files,
      path: 'groups/practices/testing.md',
    });
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
        ? `# Before writing a TypeScript retry client\n\nNo function exists yet. Start at generated/RULES.md.\n\n1. Read groups/techs/typescript.md: designing the retry return type makes the TypeScript rule relevant.\n2. Read groups/practices/testing.md: new behavior needs tests even before test files exist.\n3. Read groups/practices/code-design.md: the operation will coordinate request execution, retry decisions, and its final result.\n4. Skip groups/techs/go.md for this TypeScript task.\n5. Read the full definitions included in the testing and code-design groups. Follow the Read full rule links for relevant rules in the TypeScript index. Plan an explicit exhausted result, exactly three total attempts, and an immediate stop after success.\n\nThe design rule allows cohesive inline steps. Creating helper functions is not itself the goal.\n`
        : `# Review an attempt-limit change\n\nThe proposed TypeScript change raises total attempts from three to five. No test file changed.\n\n1. Start at generated/RULES.md and include the testing practice because behavior changed.\n2. Read the full retry-budget rule in groups/practices/testing.md; its separate file is rules/example/practices/testing/verify-retries.md.\n3. The effective replacement requires exactly three attempts. A deterministic all-failure test can demonstrate five calls and establish the mismatch.\n4. Cite example:practices/testing/verify-retries, the changed limit, and that observable behavior.\n5. Code-design relevance alone proves no violation. Inspect whether a step's purpose is obscured; helper count is insufficient evidence. Impact describes why a rule matters, while finding severity depends on the observed consequence.\n\nThis is a guided interpretation of the example, not an automated review of your application.\n`;
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
              `---\ntitle: Verify retry scenario ${index + 1}\nwhenToRead: When planning or reviewing retry scenario ${index + 1}.\nimpact: HIGH\nimpactDescription: Catch retry failures.\n---\n\n## Verify retry scenario ${index + 1}\n\nThis illustrative body must remain complete in its own file.\n`,
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
        path.startsWith(`groups/${group}.`),
    );
    const bodies = Object.keys(files).filter(
      (path) => path.startsWith('rules/') && path !== 'rules/README.md',
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
    const report = `# Index budget: ${budget} UTF-8 bytes\n\n${bodies.length} full definitions, ${partCount} numbered index parts.\n\n| Index | UTF-8 bytes |\n| --- | ---: |\n${indexes.map(([path, content]) => `| ${path} | ${Buffer.byteLength(content, 'utf8')} |`).join('\n')}\n\nOpen generated/groups/practices/testing.md. If split, read every listed part; all 40 bodies remain available under generated/rules/.\n`;
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
