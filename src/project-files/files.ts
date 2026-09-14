/** @fileoverview Reads contained project trees and compares original bytes without following symlinks. */

import { constants } from 'node:fs';
import { lstat, open, readdir, mkdir, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { createHash } from 'node:crypto';
import { relativePath } from '../formats/validation';

/** A project operation could not safely read, validate, or replace managed files. */
export class ProjectError extends Error {
  constructor(
    readonly code: string,
    message: string,
    options?: ErrorOptions,
  ) {
    super(message, options);
    this.name = 'ProjectError';
  }
}

/** Original file bytes and directory inventory; null distinguishes an absent tree from an empty one. */
export type Tree = {
  files: ReadonlyMap<string, Uint8Array>;
  directories: readonly string[];
} | null;

/** Identify an absent filesystem entry without hiding permission or device failures. */
export function missing(error: unknown): boolean {
  return error instanceof Error && 'code' in error && error.code === 'ENOENT';
}

/** Hash original bytes, including binary content and line endings. */
export function digest(bytes: Uint8Array): string {
  return createHash('sha256').update(bytes).digest('hex');
}

/** Read one regular, unlinked file without following a final symlink; bound memory use. */
export async function readBytes(path: string): Promise<Uint8Array> {
  const handle = await open(
    path,
    constants.O_RDONLY | constants.O_NOFOLLOW | constants.O_NONBLOCK,
  );
  try {
    const stat = await handle.stat();
    if (!stat.isFile() || stat.nlink !== 1 || stat.size > 64 * 1024 * 1024)
      throw new ProjectError(
        'unsafe-file',
        `${path}: expected a regular file of at most 64 MiB, without hard links.`,
      );
    const bytes = await handle.readFile();
    const after = await handle.stat();
    if (
      after.size !== stat.size ||
      after.mtimeMs !== stat.mtimeMs ||
      after.ctimeMs !== stat.ctimeMs
    )
      throw new ProjectError(
        'concurrent-change',
        `${path}: changed while being read.`,
      );
    return bytes;
  } finally {
    await handle.close();
  }
}

/** Decode required text strictly; binary assets remain byte arrays elsewhere. */
export function textFile(bytes: Uint8Array, path: string): string {
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(bytes);
  } catch {
    throw new ProjectError('invalid-text', `${path}: expected UTF-8 text.`);
  }
}

/** Inventory a tree, rejecting links, special files, and excessive input before using its contents. */
export async function readTree(
  root: string,
  limits: { entries: number; bytes: number; depth: number } = {
    entries: 30000,
    bytes: 256 * 1024 * 1024,
    depth: 64,
  },
): Promise<Tree> {
  let stat;
  try {
    stat = await lstat(root);
  } catch (error) {
    if (missing(error)) return null;
    throw error;
  }
  if (!stat.isDirectory() || stat.isSymbolicLink())
    throw new ProjectError(
      'unsafe-path',
      `${root}: expected a directory, not a symlink or file.`,
    );
  const files = new Map<string, Uint8Array>();
  const directories: string[] = [];
  let size = 0;
  async function visit(prefix: string): Promise<void> {
    for (const entry of (
      await readdir(join(root, prefix), { withFileTypes: true })
    ).sort((a, b) => (a.name < b.name ? -1 : 1))) {
      const path = prefix ? `${prefix}/${entry.name}` : entry.name;
      relativePath(path, root);
      if (
        files.size + directories.length >= limits.entries ||
        path.split('/').length > limits.depth
      )
        throw new ProjectError(
          'input-limit',
          `${root}: tree exceeds the file or depth limit.`,
        );
      if (entry.isDirectory()) {
        const current = await lstat(join(root, path));
        if (!current.isDirectory() || current.isSymbolicLink())
          throw new ProjectError(
            'unsafe-path',
            `${path}: directory changed during traversal.`,
          );
        directories.push(path);
        await visit(path);
      } else if (entry.isFile()) {
        const bytes = await readBytes(join(root, path));
        size += bytes.length;
        if (size > limits.bytes)
          throw new ProjectError(
            'input-limit',
            `${root}: tree exceeds its byte limit.`,
          );
        files.set(path, bytes);
      } else
        throw new ProjectError(
          'unsafe-path',
          `${root}/${path}: links and special files are unsupported.`,
        );
    }
  }
  await visit('');
  return { files, directories };
}

/** Stable identity includes empty directories and distinguishes absent output. */
export function treeDigest(tree: Tree): string {
  return digest(
    Buffer.from(
      JSON.stringify(
        tree === null
          ? null
          : {
              directories: [...tree.directories].sort(),
              files: [...tree.files]
                .sort(([a], [b]) => (a < b ? -1 : 1))
                .map(([path, bytes]) => [path, digest(bytes)]),
            },
      ),
    ),
  );
}

/** Write a fresh staging tree; reject portable path collisions before creating any file. */
export async function writeTree(
  root: string,
  files: ReadonlyMap<string, Uint8Array>,
): Promise<void> {
  const paths = new Map<string, string>();
  const directories = new Set<string>();
  const spellings = new Map<string, string>();
  for (const path of files.keys()) {
    relativePath(path, 'output');
    const segments = path.split('/');
    for (let length = 1; length <= segments.length; length++) {
      const spelling = segments.slice(0, length).join('/');
      const key = spelling.normalize('NFC').toLowerCase();
      const previous = spellings.get(key);
      if (previous !== undefined && previous !== spelling)
        throw new ProjectError(
          'path-collision',
          `Output paths collide: ${previous} and ${spelling}.`,
        );
      spellings.set(key, spelling);
    }
    const normalized = path.normalize('NFC').toLowerCase();
    if (paths.has(normalized))
      throw new ProjectError(
        'path-collision',
        `Output paths collide: ${path}.`,
      );
    paths.set(normalized, path);
    const parts = normalized.split('/');
    while (parts.length > 1) {
      parts.pop();
      directories.add(parts.join('/'));
    }
  }
  for (const path of paths.keys())
    if (directories.has(path))
      throw new ProjectError(
        'path-collision',
        `Output file is also a directory: ${path}.`,
      );
  await mkdir(root);
  for (const [path, bytes] of files) {
    await mkdir(dirname(join(root, path)), { recursive: true });
    await writeFile(join(root, path), bytes, { flag: 'wx', mode: 0o644 });
  }
}
