/** @fileoverview Persists import identity and byte digests, reconstructing verified offline snapshots. */

import type { LibrarySnapshot } from '../builds';
import type { ImportOutput } from '../imports';
import { configuration } from '../configuration';
import {
  object,
  strings,
  nonempty,
  json,
  relativePath,
} from '../formats/validation';
import { digest, textFile, ProjectError } from './files';

/** Serialize original imported bytes plus a versioned per-library digest record, without embedding rule text twice. */
export function vendorFiles(
  libraries: ImportOutput,
): ReadonlyMap<string, Uint8Array> {
  const result = new Map<string, Uint8Array>();
  for (const [alias, library] of Object.entries(libraries)) {
    const { files: _text, filePaths: _paths, ...identity } = library.snapshot;
    const hashes = Object.fromEntries(
      [...library.files]
        .sort(([a], [b]) => (a < b ? -1 : 1))
        .map(([path, bytes]) => [path, digest(bytes)]),
    );
    for (const [path, bytes] of library.files) {
      if (path === '_source.json')
        throw new ProjectError(
          'reserved-path',
          'Library content cannot overwrite _source.json.',
        );
      result.set(`${alias}/${path}`, bytes);
    }
    result.set(
      `${alias}/_source.json`,
      Buffer.from(
        JSON.stringify(
          { formatVersion: 1, ...identity, files: hashes },
          null,
          2,
        ) + '\n',
      ),
    );
  }
  return result;
}

function projection(
  files: ReadonlyMap<string, Uint8Array>,
): Record<string, string> {
  const entries: [string, string][] = [];
  for (const [path, bytes] of files) {
    try {
      entries.push([
        path,
        new TextDecoder('utf-8', { fatal: true }).decode(bytes),
      ]);
    } catch {
      /* Binary assets stay in filePaths; Builds validates required text independently. */
    }
  }
  return Object.fromEntries(entries);
}

/** Verify every retained byte and reject extra or missing source files before offline generation. */
export function loadSnapshots(
  input: unknown,
  vendor: ReadonlyMap<string, Uint8Array>,
): Record<string, LibrarySnapshot> {
  const config = configuration(input);
  const expected = new Set<string>();
  const snapshots: [string, LibrarySnapshot][] = [];
  for (const source of config.sources) {
    const recordPath = `${source.name}/_source.json`;
    const bytes = vendor.get(recordPath);
    if (!bytes)
      throw new ProjectError(
        'needs-sync',
        `${recordPath}: missing source record; run sync.`,
      );
    expected.add(recordPath);
    const record = object(
      json(textFile(bytes, recordPath), recordPath),
      recordPath,
    );
    const allowed = [
      'formatVersion',
      'repository',
      'ref',
      'version',
      'resolvedTag',
      'resolvedVersion',
      'resolvedCommit',
      'groups',
      'groupSelection',
      'files',
    ];
    if (
      record.formatVersion !== 1 ||
      Object.keys(record).some((key) => !allowed.includes(key))
    )
      throw new ProjectError(
        'invalid-snapshot',
        `${recordPath}: unsupported source record.`,
      );
    const hashes = object(record.files, recordPath);
    const files = new Map<string, Uint8Array>();
    for (const [path, hash] of Object.entries(hashes)) {
      relativePath(path, recordPath);
      if (
        path === '_source.json' ||
        typeof hash !== 'string' ||
        !/^[a-f0-9]{64}$/u.test(hash)
      )
        throw new ProjectError(
          'invalid-snapshot',
          `${recordPath}: invalid file digest.`,
        );
      const full = `${source.name}/${path}`;
      const content = vendor.get(full);
      if (!content || digest(content) !== hash)
        throw new ProjectError(
          'modified-vendor',
          `${full}: missing or modified imported content; run sync.`,
        );
      expected.add(full);
      files.set(path, content);
    }
    const selection = record.groupSelection;
    if (
      selection !== '*' &&
      selection !== 'practices/*' &&
      selection !== 'techs/*' &&
      !Array.isArray(selection)
    )
      throw new ProjectError(
        'invalid-snapshot',
        `${recordPath}: missing or invalid group selection.`,
      );
    snapshots.push([
      source.name,
      {
        repository: nonempty(record.repository, recordPath),
        ...(record.ref === undefined
          ? {}
          : { ref: nonempty(record.ref, recordPath) }),
        ...(record.version === undefined
          ? {}
          : { version: nonempty(record.version, recordPath) }),
        ...(record.resolvedTag === undefined
          ? {}
          : { resolvedTag: nonempty(record.resolvedTag, recordPath) }),
        ...(record.resolvedVersion === undefined
          ? {}
          : { resolvedVersion: nonempty(record.resolvedVersion, recordPath) }),
        resolvedCommit: nonempty(record.resolvedCommit, recordPath),
        groups: strings(record.groups, recordPath),
        groupSelection: Array.isArray(selection)
          ? strings(selection, recordPath)
          : selection,
        files: projection(files),
        filePaths: [...files.keys()].sort(),
      },
    ]);
  }
  if ([...vendor.keys()].some((path) => !expected.has(path)))
    throw new ProjectError(
      'needs-sync',
      'Vendor contains unexpected files or removed sources; run sync.',
    );
  return Object.fromEntries(snapshots);
}

/** Project-owned text projection; non-text supporting assets remain on disk. */
export function localText(
  files: ReadonlyMap<string, Uint8Array>,
): Record<string, string> {
  return projection(files);
}
