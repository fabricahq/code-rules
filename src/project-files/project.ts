/** @fileoverview Loads project input and inventories managed directories for optimistic change detection. */

import { realpath } from 'node:fs/promises';
import { dirname, basename, resolve, join } from 'node:path';
import type { BuildInput, FileContents } from '../builds';
import { configuration } from '../configuration';
import { json } from '../formats/validation';
import {
  digest,
  readBytes,
  readTree,
  treeDigest,
  textFile,
  ProjectError,
} from './files';
import type { Tree } from './files';
import { localText } from './snapshots';

/** Caller-selected configuration and rendering limits; the configuration's directory contains local, vendor, and generated. */
export type ProjectOptions = {
  readonly configPath?: string;
  readonly toolVersion?: string;
  readonly signal?: AbortSignal;
  readonly indexMaxBytes?: number;
  readonly groupInlineMaxBytes?: number;
};

/** A coherent baseline to compare immediately before applying output. */
export type ProjectState = {
  readonly input: unknown;
  readonly configBytes: Uint8Array;
  readonly local: Tree;
  readonly vendor: Tree;
  readonly generated: Tree;
};

/** Canonicalize the parent while still checking the configuration file itself without following symlinks. */
export async function projectLocation(
  options: ProjectOptions,
): Promise<{ root: string; configPath: string }> {
  const path = resolve(options.configPath ?? '.code-rules/config.json');
  const root = await realpath(dirname(path));
  return { root, configPath: join(root, basename(path)) };
}

/** Stop before any new side effects when caller cancellation is requested. */
export function requireActive(options: ProjectOptions): void {
  if (options.signal?.aborted)
    throw new ProjectError('cancelled', 'Project operation cancelled.');
}

/** Read project-owned input and both managed outputs, rejecting unsupported filesystem entries. */
export async function readProject(
  root: string,
  configPath: string,
): Promise<ProjectState> {
  const configBytes = await readBytes(configPath);
  const input = json(textFile(configBytes, configPath), configPath);
  configuration(input);
  return {
    input,
    configBytes,
    local: await readTree(join(root, 'local')),
    vendor: await readTree(join(root, 'vendor')),
    generated: await readTree(join(root, 'generated')),
  };
}

/** Verify that the exact inputs and existing output have not changed during fetching or generation. */
export async function assertUnchanged(
  root: string,
  configPath: string,
  before: ProjectState,
  options: ProjectOptions,
): Promise<void> {
  requireActive(options);
  const after = await readProject(root, configPath);
  if (
    digest(before.configBytes) !== digest(after.configBytes) ||
    ['local', 'vendor', 'generated'].some((key) => {
      if (key === 'local')
        return treeDigest(before.local) !== treeDigest(after.local);
      if (key === 'vendor')
        return treeDigest(before.vendor) !== treeDigest(after.vendor);
      return treeDigest(before.generated) !== treeDigest(after.generated);
    })
  )
    throw new ProjectError(
      'concurrent-change',
      'Project inputs or managed output changed during the operation. Retry after edits finish.',
    );
}

/** Populate the offline builder's inputs using project-owned text and explicit rendering options. */
export function buildInput(
  state: ProjectState,
  snapshots: BuildInput['snapshots'],
  options: ProjectOptions,
): BuildInput {
  return {
    configuration: state.input,
    snapshots,
    localFiles: localText(state.local?.files ?? new Map()),
    localFilePaths: [...(state.local?.files.keys() ?? [])],
    toolVersion: options.toolVersion ?? '0.0.0-development',
    ...(options.indexMaxBytes === undefined
      ? {}
      : { indexMaxBytes: options.indexMaxBytes }),
    ...(options.groupInlineMaxBytes === undefined
      ? {}
      : { groupInlineMaxBytes: options.groupInlineMaxBytes }),
  };
}

/** Encode generated text without changing its content. */
export function generatedBytes(
  files: FileContents,
): ReadonlyMap<string, Uint8Array> {
  return new Map(
    Object.entries(files).map(([path, text]) => [path, Buffer.from(text)]),
  );
}

/** Sorted file changes, relative to the configuration directory; empty directories carry no rule content. */
export type FileChanges = {
  readonly added: readonly string[];
  readonly changed: readonly string[];
  readonly removed: readonly string[];
};

/** Compare bytes without timestamps, keeping repeated operations quiet and deterministic. */
export function compareFiles(
  before: ReadonlyMap<string, Uint8Array>,
  after: ReadonlyMap<string, Uint8Array>,
): FileChanges {
  return {
    added: [...after.keys()].filter((path) => !before.has(path)).sort(),
    changed: [...after]
      .filter(([path, bytes]) => {
        const old = before.get(path);
        return old !== undefined && digest(old) !== digest(bytes);
      })
      .map(([path]) => path)
      .sort(),
    removed: [...before.keys()].filter((path) => !after.has(path)).sort(),
  };
}

/** Combine path-prefixed managed outputs for user-facing change reports. */
export function managedFiles(
  vendor: ReadonlyMap<string, Uint8Array>,
  generated: ReadonlyMap<string, Uint8Array>,
): ReadonlyMap<string, Uint8Array> {
  return new Map(
    [...vendor]
      .map(([path, bytes]): [string, Uint8Array] => [`vendor/${path}`, bytes])
      .concat(
        [...generated].map(([path, bytes]) => [`generated/${path}`, bytes]),
      ),
  );
}
