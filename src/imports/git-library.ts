/** @fileoverview Fetches one exact Git revision and exposes its tree and original blob contents. */

import type { LibrarySource } from '../configuration-types';
import { selectVersion } from './version-selection';
import type { VersionSelection } from './version-selection';
import { gitProcess } from './git-process';
import { ImportError } from './errors';

/** Fixed resource ceilings per library; fetch disk usage is not bounded by retained-file limits. */
export const IMPORT_LIMITS = {
  durationMs: 120_000,
  treeBytes: 8 * 1024 * 1024,
  fileBytes: 8 * 1024 * 1024,
  retainedBytes: 64 * 1024 * 1024,
  files: 10_000,
};

/** One Git tree entry; mode distinguishes ordinary blobs from symlinks and submodules. */
export type TreeEntry = {
  readonly path: string;
  readonly oid: string;
  readonly mode: string;
  readonly size: number;
};

/** Fail without publishing Git's stderr, which can contain authentication data. */
async function command(
  cwd: string,
  args: ReadonlyArray<string>,
  signal: AbortSignal,
  maxBytes: number,
): Promise<Buffer> {
  const result = await gitProcess(cwd, args, signal, maxBytes);
  if (result.status !== 0)
    throw new ImportError(
      'git-failed',
      'Git could not read the fetched library.',
    );
  return result.output;
}

/** A fetched commit plus the selected tag and version when a range was requested. */
export type FetchedRevision = {
  readonly commit: string;
} & (
  | { readonly resolvedTag: string; readonly resolvedVersion: string }
  | { readonly resolvedTag?: never; readonly resolvedVersion?: never }
);

/** Resolve a request to an exact fetch target and preserve advertisement identity for range selections. */
async function fetchTarget(
  cwd: string,
  source: LibrarySource,
  signal: AbortSignal,
): Promise<{
  readonly ref: string;
  readonly selection: VersionSelection | undefined;
}> {
  switch (source.parsedRef.kind) {
    case 'commit':
      return { ref: source.parsedRef.sha, selection: undefined };
    case 'tag':
      return { ref: source.parsedRef.name, selection: undefined };
    case 'version': {
      const selection = await selectVersion(
        cwd,
        source.repository,
        source.parsedRef.range,
        signal,
      );
      return { ref: `refs/tags/${selection.tag}`, selection };
    }
  }
}

/** Fetch a configured exact revision or the highest matching tag and verify its immutable identity. */
export async function fetchRevision(
  cwd: string,
  source: LibrarySource,
  signal: AbortSignal,
): Promise<FetchedRevision> {
  const version = (await command(cwd, ['--version'], signal, 4096)).toString();
  const match = /git version (\d+)\.(\d+)/u.exec(version);
  if (
    !match ||
    Number(match[1]) < 2 ||
    (Number(match[1]) === 2 && Number(match[2]) < 30)
  )
    throw new ImportError('git-unavailable', 'Git 2.30 or later is required.');
  await command(
    cwd,
    ['init', '--bare', '--quiet', '--template='],
    signal,
    4096,
  );
  const remote = source.repository;
  const { ref, selection } = await fetchTarget(cwd, source, signal);
  const fetched = await gitProcess(
    cwd,
    [
      'fetch',
      '--quiet',
      '--depth=1',
      '--no-tags',
      '--no-recurse-submodules',
      remote,
      ref,
    ],
    signal,
    64 * 1024,
  );
  if (fetched.status !== 0) {
    const reachable = await gitProcess(
      cwd,
      ['ls-remote', '--exit-code', remote, 'HEAD'],
      signal,
      64 * 1024,
    );
    throw new ImportError(
      reachable.status === 0 || reachable.status === 2
        ? 'ref-not-found'
        : 'not-found-or-no-access',
      reachable.status === 0 || reachable.status === 2
        ? 'The requested ref is missing or the server refused it; no other revision was selected.'
        : 'Repository not found or no access; check its name and Git credentials.',
    );
  }
  if (selection !== undefined) {
    const fetchedObject = (
      await command(cwd, ['rev-parse', '--verify', 'FETCH_HEAD'], signal, 4096)
    )
      .toString()
      .trim();
    if (fetchedObject !== selection.object)
      throw new ImportError(
        'ref-changed',
        'The selected version tag changed during import; retry to resolve it again.',
      );
  }
  const resolved = await gitProcess(
    cwd,
    ['rev-parse', '--verify', 'FETCH_HEAD^{commit}'],
    signal,
    4096,
  );
  if (resolved.status !== 0)
    throw new ImportError(
      'unsupported-content',
      'The requested tag does not resolve to a commit.',
    );
  const commit = resolved.output.toString().trim().toLowerCase();
  if (
    !/^[0-9a-f]{40}$/u.test(commit) ||
    (source.parsedRef.kind === 'commit' && source.parsedRef.sha !== commit)
  )
    throw new ImportError(
      'git-failed',
      'Git returned a different or unsupported commit identity.',
    );
  return {
    commit,
    ...(selection === undefined
      ? {}
      : { resolvedTag: selection.tag, resolvedVersion: selection.version }),
  };
}

/** Read a bounded recursive tree without checkout filters or filename quoting. */
export async function libraryTree(
  cwd: string,
  commit: string,
  signal: AbortSignal,
): Promise<ReadonlyMap<string, TreeEntry>> {
  const output = await command(
    cwd,
    ['ls-tree', '-r', '-l', '-z', commit],
    signal,
    IMPORT_LIMITS.treeBytes,
  );
  let text: string;
  try {
    text = new TextDecoder('utf-8', { fatal: true }).decode(output);
  } catch {
    throw new ImportError(
      'unsupported-content',
      'Library filenames must be UTF-8.',
    );
  }
  const entries = new Map<string, TreeEntry>();
  for (const record of text.split('\0')) {
    if (!record) continue;
    const row =
      /^(\d{6}) (?:blob|commit) ([0-9a-f]{40}) +([0-9]+|-)\t([\s\S]+)$/u.exec(
        record,
      );
    if (
      !row ||
      row[1] === undefined ||
      row[2] === undefined ||
      row[3] === undefined ||
      row[4] === undefined
    )
      throw new ImportError(
        'unsupported-content',
        'Unsupported Git tree entry.',
      );
    entries.set(row[4], {
      path: row[4],
      mode: row[1],
      oid: row[2],
      size: row[3] === '-' ? 0 : Number(row[3]),
    });
    if (entries.size > IMPORT_LIMITS.files)
      throw new ImportError(
        'limit-exceeded',
        'Library tree exceeds 10,000 files.',
      );
  }
  return entries;
}

/** Read one dependency wave through a single batch process, checking Git's size framing before retaining bytes. */
export async function readBlobs(
  cwd: string,
  entries: ReadonlyArray<TreeEntry>,
  signal: AbortSignal,
): Promise<ReadonlyMap<string, Uint8Array>> {
  if (entries.length === 0) return new Map();
  const expectedBytes = entries.reduce((sum, entry) => sum + entry.size, 0);
  const result = await gitProcess(
    cwd,
    ['cat-file', '--batch'],
    signal,
    expectedBytes + entries.length * 128 + 4096,
    entries.map((entry) => entry.oid).join('\n') + '\n',
  );
  if (result.status !== 0)
    throw new ImportError('git-failed', 'Could not read library blobs.');
  let offset = 0;
  const files = new Map<string, Uint8Array>();
  for (const entry of entries) {
    const end = result.output.indexOf(10, offset);
    const header = result.output.subarray(offset, end).toString();
    if (end < 0 || header !== `${entry.oid} blob ${entry.size}`)
      throw new ImportError(
        'git-failed',
        'Git returned inconsistent blob metadata.',
      );
    offset = end + 1;
    if (result.output[offset + entry.size] !== 10)
      throw new ImportError(
        'git-failed',
        'Git returned incomplete blob contents.',
      );
    files.set(
      entry.path,
      Uint8Array.from(result.output.subarray(offset, offset + entry.size)),
    );
    offset += entry.size + 1;
  }
  if (offset !== result.output.length)
    throw new ImportError('git-failed', 'Git returned unexpected blob data.');
  return files;
}
