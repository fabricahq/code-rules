/** @fileoverview Selects library groups and referenced attachments while preserving exact source bytes. */

import type { LibrarySource } from '../configuration-types';
import { collectLibraryLicensePaths } from '../formats/manifest';
import { markdownTargets } from '../formats/markdown-links';
import { groupMetadata, relativePath } from '../formats/validation';
import { ImportError, requireActive } from './errors';
import { IMPORT_LIMITS, readBlobs } from './git-library';
import type { TreeEntry } from './git-library';
import type { ImportedLibrary } from './types';

/** Decode authored text without replacing invalid bytes or changing retained file contents. */
function utf8(bytes: Uint8Array, path: string): string {
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(bytes);
  } catch {
    throw new ImportError('invalid-library', `${path}: expected UTF-8 text.`);
  }
}

/** Register portable path spellings and reject metadata replacement or ambiguous directories. */
function registerPath(path: string, paths: Map<string, string>): void {
  relativePath(path, path);
  const parts = path.split('/');
  if (
    path.normalize('NFC').toLowerCase() === '_source.json' ||
    parts.some((part) => part.toLowerCase() === '.git')
  )
    throw new ImportError(
      'unsupported-content',
      `${path}: reserved import path.`,
    );
  for (let length = 1; length <= parts.length; length++) {
    const prefix = parts.slice(0, length).join('/');
    const key = prefix.normalize('NFC').toLowerCase();
    const existing = paths.get(key);
    if (existing !== undefined && existing !== prefix)
      throw new ImportError(
        'unsupported-content',
        `${path}: case or Unicode path collision with ${existing}.`,
      );
    paths.set(key, prefix);
  }
}

/** Verify selected entries are ordinary bounded files before asking Git for their bytes. */
function selectedEntries(
  requested: ReadonlySet<string>,
  tree: ReadonlyMap<string, TreeEntry>,
  retained: ReadonlyMap<string, Uint8Array>,
): ReadonlyArray<TreeEntry> {
  const entries: Array<TreeEntry> = [];
  const spellings = new Map<string, string>();
  let bytes = 0;
  for (const path of [...requested].sort()) {
    registerPath(path, spellings);
    const entry = tree.get(path);
    if (entry === undefined)
      throw new ImportError(
        'invalid-library',
        `${path}: missing required library file.`,
      );
    if (entry.mode !== '100644' && entry.mode !== '100755')
      throw new ImportError(
        'unsupported-content',
        `${path}: symlinks and submodules are unsupported.`,
      );
    bytes += entry.size;
    if (
      entry.size > IMPORT_LIMITS.fileBytes ||
      bytes > IMPORT_LIMITS.retainedBytes ||
      requested.size > IMPORT_LIMITS.files
    )
      throw new ImportError(
        'limit-exceeded',
        'Selected files exceed import size limits.',
      );
    if (!retained.has(path)) entries.push(entry);
  }
  return entries;
}

/** Preserve requested files and recursively discover standard Markdown dependencies in bounded waves. */
async function retainFiles(
  cwd: string,
  requested: Set<string>,
  tree: ReadonlyMap<string, TreeEntry>,
  signal: AbortSignal,
): Promise<ReadonlyMap<string, Uint8Array>> {
  const retained = new Map<string, Uint8Array>();
  while (retained.size < requested.size) {
    requireActive(signal);
    const entries = selectedEntries(requested, tree, retained);
    for (const [path, bytes] of await readBlobs(cwd, entries, signal)) {
      if (
        new TextDecoder()
          .decode(bytes.subarray(0, 128))
          .replace(/\r\n/gu, '\n')
          .startsWith('version https://git-lfs.github.com/spec/v1\n')
      )
        throw new ImportError(
          'unsupported-content',
          `${path}: Git LFS objects are unsupported.`,
        );
      retained.set(path, bytes);
      if (path.endsWith('.md')) {
        for (const target of markdownTargets(utf8(bytes, path), path)) {
          if (tree.has(target)) requested.add(target);
        }
      }
    }
  }
  return new Map(
    [...retained].sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0)),
  );
}

/** Return a Builds projection and byte-preserving files from one immutable Git tree. */
export async function selectLibrary(
  cwd: string,
  source: LibrarySource,
  commit: string,
  tree: ReadonlyMap<string, TreeEntry>,
  signal: AbortSignal,
): Promise<ImportedLibrary> {
  const manifestPath = 'rule-library.json';
  const manifestFiles = await retainFiles(
    cwd,
    new Set([manifestPath]),
    tree,
    signal,
  );
  const manifestBytes = manifestFiles.get(manifestPath);
  if (manifestBytes === undefined)
    throw new ImportError('invalid-library', 'Missing rule-library.json.');
  const manifestText = utf8(manifestBytes, manifestPath);
  const licensePaths = collectLibraryLicensePaths(
    new Map([[manifestPath, manifestText]]),
    source.name,
    new Set(tree.keys()),
  );
  const requested = new Set([manifestPath, ...licensePaths]);
  for (const group of source.groups) {
    requested.add(`${group}/_group.json`);
    for (const path of tree.keys())
      if (path.startsWith(`${group}/`)) requested.add(path);
  }
  const files = await retainFiles(cwd, requested, tree, signal);
  const textFiles: Array<readonly [string, string]> = [];
  for (const [path, bytes] of files) {
    if (
      path === manifestPath ||
      source.groups.some(
        (group) =>
          path === `${group}/_group.json` ||
          (path.startsWith(`${group}/`) && path.endsWith('.md')),
      )
    ) {
      const text = utf8(bytes, path);
      if (source.groups.some((group) => path === `${group}/_group.json`))
        groupMetadata(text, `${source.name}/${path}`);
      textFiles.push([path, text]);
    }
  }
  return {
    files,
    snapshot: {
      repository: source.repository,
      ref: source.ref,
      resolvedCommit: commit,
      groups: source.groups,
      files: Object.fromEntries(textFiles),
      filePaths: [...files.keys()],
    },
  };
}
