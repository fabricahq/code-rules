/** @fileoverview Replaces gh in shell tests with declared responses and a record of every invocation. */

import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

/**
 * Write a network-free gh executable into an existing test bin directory.
 * Exact argument matches return the declared stdout and exit code; unexpected calls fail.
 * Returns a reader for the recorded argument lists. The caller owns directory cleanup.
 */
export function fakeGitHubCLI(
  binDirectory: string,
  responses: { args: string[]; stdout: string; exitCode?: number }[],
): () => string[][] {
  const callsPath = join(binDirectory, 'gh-calls.jsonl');
  writeFileSync(callsPath, '');
  writeFileSync(
    join(binDirectory, 'gh'),
    `#!/usr/bin/env bun
const { appendFileSync, writeFileSync } = require('node:fs');
const responses = ${JSON.stringify(responses)};
const args = process.argv.slice(2);
appendFileSync(${JSON.stringify(callsPath)}, JSON.stringify(args) + '\\n');
const response = responses.find(entry => JSON.stringify(entry.args) === JSON.stringify(args));
if (!response) {
  console.error('Unexpected gh invocation:', JSON.stringify(args));
  process.exit(1);
}
writeFileSync(1, response.stdout);
process.exit(response.exitCode ?? 0);
`,
    { mode: 0o755 },
  );
  return () =>
    readFileSync(callsPath, 'utf8')
      .split('\n')
      .filter(Boolean)
      .map((line) => JSON.parse(line) as string[]);
}
