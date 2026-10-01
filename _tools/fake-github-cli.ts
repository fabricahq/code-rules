/** @fileoverview Replaces gh in shell tests with declared responses and a record of every invocation. */

import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

/** Quote a value as one POSIX shell word. */
function shellQuote(value: string): string {
  return `'${value.replaceAll("'", `'\\''`)}'`;
}

/**
 * Write a network-free gh executable into an existing test bin directory.
 * Exact argument matches return the declared stdout and exit code; unexpected calls fail.
 * Returns a reader for the recorded argument lists. The caller owns directory cleanup.
 * Throws if a declared value contains a NUL byte, which shell arguments and literals cannot carry.
 */
export function fakeGitHubCLI(
  binDirectory: string,
  responses: { args: string[]; stdout: string; exitCode?: number }[],
): () => string[][] {
  const callsPath = join(binDirectory, 'gh-calls');
  writeFileSync(callsPath, '');
  const branches = responses.map(({ args, stdout, exitCode }) => {
    for (const value of [...args, stdout]) {
      if (value.includes('\0'))
        throw new Error('Fake gh values cannot contain NUL.');
    }
    const conditions = [
      `[ "$#" -eq ${args.length} ]`,
      ...args.map(
        (arg, index) => `[ "\${${index + 1}}" = ${shellQuote(arg)} ]`,
      ),
    ];
    return `if ${conditions.join(' && ')}; then
  printf '%s' ${shellQuote(stdout)}
  exit ${exitCode ?? 0}
fi`;
  });
  // POSIX sh keeps each call to a few milliseconds; a JavaScript runtime per call dominated test time under load.
  // Each call records its argument count, then its arguments, all NUL-terminated, so any argument text round-trips.
  writeFileSync(
    join(binDirectory, 'gh'),
    `#!/bin/sh
printf '%s\\0' "$#" "$@" >> ${shellQuote(callsPath)}
${branches.join('\n')}
printf 'Unexpected gh invocation:' >&2
printf ' %s' "$@" >&2
printf '\\n' >&2
exit 1
`,
    { mode: 0o755 },
  );
  return () => {
    const fields = readFileSync(callsPath, 'utf8').split('\0');
    const calls: string[][] = [];
    for (let index = 0; index < fields.length - 1;) {
      const count = Number(fields[index]);
      calls.push(fields.slice(index + 1, index + 1 + count));
      index += count + 1;
    }
    return calls;
  };
}
