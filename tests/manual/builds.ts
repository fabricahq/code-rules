/** @fileoverview Writes a temporary workspace for manually inspecting generated agent guidance; leaves files in place for review. */

import { mkdtemp, mkdir, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { buildRules } from '../../src/builds';
import { buildsExampleInput as input } from './builds-fixture';

const result = buildRules(input);
const directory = await mkdtemp(join(tmpdir(), 'code-rules-builds-'));
const workspaceFiles = {
  'scenarios.md':
    'Implementation: plan a TypeScript retry client before writing any code. Inspect TypeScript, testing, and code-design indexes; the Go rule is unrelated.\n\nValidation: review a change that increases the retry attempt count without changing tests. Read the project retry-budget replacement and check its three-attempt contract. Reading the code-design rule alone does not establish a violation.\n',
  'config.json': `${JSON.stringify(input.configuration, null, 2)}\n`,
  ...Object.fromEntries(
    Object.entries(input.localFiles).map(([path, content]) => [
      `local/${path}`,
      content,
    ]),
  ),
  ...Object.fromEntries(
    Object.entries(input.snapshots).flatMap(([name, snapshot]) =>
      Object.entries(snapshot.files).map(([path, content]) => [
        `vendor/${name}/${path}`,
        content,
      ]),
    ),
  ),
  ...Object.fromEntries(
    Object.entries(result.files).map(([path, content]) => [
      `generated/${path}`,
      content,
    ]),
  ),
};
for (const [path, content] of Object.entries(workspaceFiles)) {
  const destination = join(directory, path);
  await mkdir(dirname(destination), { recursive: true });
  await writeFile(destination, content);
}
console.log(`Manual test workspace: ${directory}`);
console.log(`Agent index: ${join(directory, 'generated/RULES.md')}`);
console.log(
  'Source repository and commit are illustrative; no libraries were fetched.',
);
console.log(`Selection scenarios: ${join(directory, 'scenarios.md')}`);
