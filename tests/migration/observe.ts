/** @fileoverview Captures executable results and workspace entries without following symlinks; reports strict contract differences. */

import { spawnSync } from 'node:child_process';
import { lstat, readdir, readFile, readlink } from 'node:fs/promises';
import { join } from 'node:path';
import { createHash } from 'node:crypto';
import { isDeepStrictEqual } from 'node:util';

/** Exact entry content and mode; directories include empty ones and symlinks remain links. */
export type Entry = {
  readonly kind: string;
  readonly mode: number;
  readonly bytes: string;
};
/** Sorted relative paths to filesystem entries, including non-file entries. */
export type Tree = Readonly<Record<string, Entry>>;
/** Captured process outcome; spawn failures and signals never masquerade as exit codes. */
export type Execution = {
  readonly status: number | null;
  readonly signal: string | null;
  readonly error: string | null;
  readonly stdout: string;
  readonly stderr: string;
};

/** Hash the exact executable bytes, resolving symlinks as readFile does. */
export async function executableHash(path: string): Promise<string> {
  return createHash('sha256')
    .update(await readFile(path))
    .digest('hex');
}

/** Read all entries recursively, preserving binary bytes as base64 and never traversing a symlink. */
export async function readWorkspace(root: string): Promise<Tree> {
  const entries: Record<string, Entry> = {};
  async function visit(relative: string): Promise<void> {
    const path = join(root, relative);
    const stat = await lstat(path);
    const kind = stat.isSymbolicLink()
      ? 'symlink'
      : stat.isDirectory()
        ? 'directory'
        : stat.isFile()
          ? 'file'
          : 'special';
    const bytes =
      kind === 'file'
        ? (await readFile(path)).toString('base64')
        : kind === 'symlink'
          ? await readlink(path)
          : '';
    if (relative) entries[relative] = { kind, mode: stat.mode & 0o777, bytes };
    if (kind === 'directory') {
      for (const child of (await readdir(path)).sort())
        await visit(relative ? `${relative}/${child}` : child);
    }
  }
  await visit('');
  return entries;
}

/** Run a real executable with closed stdin, bounded output, and a 30-second deadline. */
export function execute(options: {
  readonly executable: string;
  readonly args: ReadonlyArray<string>;
  readonly cwd: string;
  readonly env: NodeJS.ProcessEnv;
}): Execution {
  const result = spawnSync(options.executable, options.args, {
    cwd: options.cwd,
    env: options.env,
    input: '',
    timeout: 30_000,
    maxBuffer: 8 * 1024 * 1024,
  });
  let stdout = '';
  let stderr = '';
  let error = result.error?.message ?? null;
  try {
    stdout = new TextDecoder('utf-8', { fatal: true }).decode(
      result.stdout ?? new Uint8Array(),
    );
    stderr = new TextDecoder('utf-8', { fatal: true }).decode(
      result.stderr ?? new Uint8Array(),
    );
  } catch {
    error =
      'Executable emitted invalid UTF-8; stdout(base64)=' +
      (result.stdout?.toString('base64') ?? '') +
      '; stderr(base64)=' +
      (result.stderr?.toString('base64') ?? '');
  }
  return {
    status: result.status,
    signal: result.signal,
    error,
    stdout,
    stderr,
  };
}

/** Compare exact tree entries, retaining paths and both values for each discrepancy. */
export function compareTrees(options: {
  readonly reference: Tree;
  readonly candidate: Tree;
}): Array<string> {
  return [
    ...new Set([
      ...Object.keys(options.reference),
      ...Object.keys(options.candidate),
    ]),
  ]
    .sort()
    .flatMap((path) =>
      isDeepStrictEqual(options.reference[path], options.candidate[path])
        ? []
        : [
            `${path}: reference=${JSON.stringify(options.reference[path])} candidate=${JSON.stringify(options.candidate[path])}`,
          ],
    );
}

/** Compare process outcomes; only paired workspace roots in text streams may differ. JSON object key order is incidental; arrays and missing keys are preserved. */
export function compareExecutions(options: {
  readonly reference: Execution;
  readonly candidate: Execution;
  readonly referenceRoot: string;
  readonly candidateRoot: string;
}): Array<string> {
  const differences: Array<string> = [];
  for (const key of ['status', 'signal', 'error'] as const) {
    if (options.reference[key] !== options.candidate[key])
      differences.push(
        `${key}: reference=${options.reference[key]} candidate=${options.candidate[key]}`,
      );
  }
  for (const key of ['stdout', 'stderr'] as const) {
    const reference = options.reference[key].replaceAll(
      options.referenceRoot,
      '<workspace>',
    );
    const candidate = options.candidate[key].replaceAll(
      options.candidateRoot,
      '<workspace>',
    );
    let equal = reference === candidate;
    if (!equal && key === 'stdout') {
      try {
        const a: unknown = JSON.parse(reference);
        const b: unknown = JSON.parse(candidate);
        equal = isDeepStrictEqual(a, b);
      } catch {
        /* Non-JSON output is contractual text. */
      }
    }
    if (!equal)
      differences.push(
        `${key}: reference=${JSON.stringify(reference)} candidate=${JSON.stringify(candidate)}`,
      );
  }
  return differences;
}
