/** @fileoverview Selects library groups and referenced attachments while preserving exact source bytes. */

import type { LibrarySource } from '../configuration-types';
import { collectLibraryLicensePaths } from '../formats/manifest';
import {
  assetDirectory,
  assetOwner,
  isAssetPath,
  isRuleFile,
  requireAllowedTarget,
} from '../formats/assets';
import { markdownTargets } from '../formats/markdown-links';
import { groupId, groupMetadata, relativePath } from '../formats/validation';
import { ImportError, requireActive } from './errors';
import { IMPORT_LIMITS, readBlobs } from './git-library';
import type { TreeEntry } from './git-library';
import type { ImportedLibrary } from './types';

/** Decode authored text without replacing invalid bytes or changing retained file contents. */
function utf8(bytes: Uint8Array, path: string, preserveBOM = false): string {
  try {
    return new TextDecoder('utf-8', {
      fatal: true,
      ignoreBOM: preserveBOM,
    }).decode(bytes);
  } catch {
    throw new ImportError('invalid-library', `${path}: expected UTF-8 text.`);
  }
}

/** Fold Unicode case variants, including final sigma and sharp S, before comparing retained path spellings. */
function pathKey(path: string): string {
  return path.normalize('NFC').toLowerCase().toUpperCase().normalize('NFC');
}

/** Register portable path spellings and reject metadata replacement or ambiguous directories. */
function registerPath(path: string, paths: Map<string, string>): void {
  relativePath(path, path);
  const parts = path.split('/');
  if (
    pathKey(path) === '_SOURCE.JSON' ||
    parts.some((part) => pathKey(part) === '.GIT')
  )
    throw new ImportError(
      'unsupported-content',
      `${path}: reserved import path.`,
    );
  for (let length = 1; length <= parts.length; length++) {
    const prefix = parts.slice(0, length).join('/');
    const key = pathKey(prefix);
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

/** Preserve complete selected directories, adding shared assets only when a permitted Markdown reference requires them. */
async function retainFiles(
  cwd: string,
  requested: Set<string>,
  tree: ReadonlyMap<string, TreeEntry>,
  signal: AbortSignal,
  licensePaths: ReadonlyArray<string> = [],
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
      if (path.endsWith('.md') && !licensePaths.includes(path)) {
        for (const target of markdownTargets(utf8(bytes, path), path)) {
          requireAllowedTarget(path, target, licensePaths);
          if (!tree.has(target))
            throw new ImportError(
              'invalid-library',
              `${path}: missing link destination: ${target}`,
            );
          if (assetDirectory(target) === 'assets/') {
            for (const shared of tree.keys())
              if (shared.startsWith('assets/')) requested.add(shared);
          }
          // Rule links are references, not an instruction to adopt another group.
        }
      }
    }
  }
  return new Map(
    [...retained].sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0)),
  );
}

/** Expand a pattern from the immutable tree, including empty groups and rejecting orphan group content. */
function selectedGroups(
  source: LibrarySource,
  tree: ReadonlyMap<string, TreeEntry>,
  licensePaths: ReadonlyArray<string>,
): ReadonlyArray<string> {
  if (typeof source.groups !== 'string') return source.groups;
  const roots =
    source.groups === '*'
      ? ['techs', 'practices']
      : [source.groups.split('/')[0]];
  const groups = new Set<string>();
  for (const path of tree.keys()) {
    if (licensePaths.includes(path)) continue;
    const parts = path.split('/');
    if (!roots.includes(parts[0])) continue;
    const id = groupId(parts.slice(0, 2).join('/'), `${source.name}/${path}`);
    if (!tree.has(`${id}/_group.json`))
      throw new ImportError(
        'invalid-library',
        `${id}: missing required _group.json.`,
      );
    groups.add(id);
  }
  return [...groups].sort();
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
  const groups = selectedGroups(source, tree, licensePaths);
  const requested = new Set([manifestPath, ...licensePaths]);
  for (const group of groups) {
    requested.add(`${group}/_group.json`);
    for (const path of tree.keys())
      if (path.startsWith(`${group}/`)) requested.add(path);
  }
  // Validate file kinds before layout, so unsupported Git objects retain their diagnostics.
  selectedEntries(requested, tree, new Map());
  for (const path of requested) {
    if (
      path === manifestPath ||
      licensePaths.includes(path) ||
      groups.some((group) => path === `${group}/_group.json`)
    )
      continue;
    const directory = assetDirectory(path);
    if (directory !== null) {
      const owner = assetOwner(directory);
      if (
        owner !== null &&
        (!tree.has(owner) || !isRuleFile(owner) || licensePaths.includes(owner))
      )
        throw new ImportError(
          'invalid-library',
          `${path}: assets directory has no adjacent owning rule: ${owner}`,
        );
    } else if (isAssetPath(path) || !isRuleFile(path)) {
      throw new ImportError(
        'invalid-library',
        `${path}: supporting files must live in an owned assets directory or library-root assets/`,
      );
    }
  }
  const files = await retainFiles(cwd, requested, tree, signal, licensePaths);
  const textFiles: Array<readonly [string, string]> = [];
  for (const [path, bytes] of files) {
    if (
      path === manifestPath ||
      licensePaths.includes(path) ||
      groups.some(
        (group) =>
          path === `${group}/_group.json` ||
          (path.startsWith(`${group}/`) && isRuleFile(path)),
      )
    ) {
      const text = utf8(bytes, path, licensePaths.includes(path));
      if (groups.some((group) => path === `${group}/_group.json`))
        groupMetadata(text, `${source.name}/${path}`);
      textFiles.push([path, text]);
    }
  }
  return {
    files,
    snapshot: {
      repository: source.repository,
      ...(source.version === undefined
        ? { ref: source.ref }
        : { version: source.version }),
      resolvedCommit: commit,
      groups,
      groupSelection: source.groups,
      files: Object.fromEntries(textFiles),
      filePaths: [...files.keys()],
    },
  };
}
