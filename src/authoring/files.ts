/** @fileoverview Publishes authored files under a writer lock without overwriting new-file collisions. */

import {
  lstat,
  mkdir,
  readdir,
  mkdtemp,
  writeFile,
  link,
  unlink,
  rename,
  rmdir,
  rm,
} from 'node:fs/promises';
import { dirname, join, basename, relative } from 'node:path';
import { relativePath } from '../formats/validation';
import {
  missing,
  readBytes,
  digest,
  ProjectError,
} from '../project-files/files';

/** A replacement must carry the exact previous bytes; null means the file must not exist. */
export type AuthoredFile = {
  readonly path: string;
  readonly text: string;
  readonly before: Uint8Array | null;
};

/** Read a regular authored file or return null only when absent. */
export async function optionalBytes(path: string): Promise<Uint8Array | null> {
  try {
    return await readBytes(path);
  } catch (error) {
    if (missing(error)) return null;
    throw error;
  }
}

/** Validate each existing path component and create missing directories, tracking only directories created here. */
export async function ensureDirectory(
  path: string,
  created: string[],
): Promise<void> {
  const parent = dirname(path);
  if (parent !== path) await ensureDirectory(parent, created);
  await rejectCaseAlias(path);
  try {
    const stat = await lstat(path);
    if (!stat.isDirectory() || stat.isSymbolicLink())
      throw new ProjectError(
        'unsafe-path',
        `${path}: expected a directory without symlinks.`,
      );
  } catch (error) {
    if (!missing(error)) throw error;
    try {
      await mkdir(path);
      created.push(path);
    } catch (failure) {
      if (!(
        failure instanceof Error &&
        'code' in failure &&
        failure.code === 'EEXIST'
      ))
        throw failure;
      await ensureDirectory(path, created);
    }
  }
}

async function rejectCaseAlias(path: string): Promise<void> {
  if (dirname(path) === path) return;
  const name = basename(path);
  const names = await readdir(dirname(path));
  if (
    names.some(
      (entry) =>
        entry !== name &&
        entry.normalize('NFC').toLowerCase() ===
          name.normalize('NFC').toLowerCase(),
    )
  )
    throw new ProjectError(
      'path-collision',
      `${path}: conflicts with an existing path's spelling.`,
    );
}

async function verify(file: AuthoredFile): Promise<void> {
  await rejectCaseAlias(file.path);
  const current = await optionalBytes(file.path);
  if (
    file.before === null
      ? current !== null
      : current === null || digest(current) !== digest(file.before)
  )
    throw new ProjectError(
      'concurrent-change',
      `${file.path}: already exists or changed; no overwrite was performed.`,
    );
}

/** Publish new files first and at most one configuration replacement last; roll back unchanged new files on failure. Caller holds the project writer lock. */
export async function publishAuthored(
  root: string,
  files: readonly AuthoredFile[],
): Promise<void> {
  for (const file of files)
    relativePath(relative(root, file.path), 'authoring target');
  if (new Set(files.map((file) => file.path)).size !== files.length)
    throw new Error('Duplicate authoring target.');
  if (files.filter((file) => file.before !== null).length > 1)
    throw new Error('An authoring operation can replace at most one file.');
  const directories: string[] = [];
  const published: AuthoredFile[] = [];
  const stage = await mkdtemp(join(root, '.code-rules-authoring-'));
  try {
    for (const file of files) {
      await ensureDirectory(dirname(file.path), directories);
      await verify(file);
    }
    const ordered = [
      ...files.filter((file) => file.before === null),
      ...files.filter((file) => file.before !== null),
    ];
    for (const [index, file] of ordered.entries()) {
      const staged = join(stage, String(index));
      await writeFile(staged, file.text, { flag: 'wx' });
      await verify(file);
      if (file.before === null) {
        await link(staged, file.path);
        // Drop the staging link immediately so ordinary project readers still see single-link files.
        published.push(file);
        await unlink(staged);
      } else {
        await rename(staged, file.path);
      }
    }
  } catch (error) {
    // Remove staging hard links before reading files for rollback.
    await rm(stage, { recursive: true, force: true });
    for (const file of published.reverse()) {
      const current = await optionalBytes(file.path);
      if (
        current !== null &&
        digest(current) === digest(Buffer.from(file.text))
      )
        await unlink(file.path);
    }
    throw error;
  } finally {
    await rm(stage, { recursive: true, force: true });
    for (const path of directories.reverse()) {
      try {
        await rmdir(path);
      } catch (error) {
        if (!(
          error instanceof Error &&
          'code' in error &&
          ['ENOTEMPTY', 'EEXIST', 'ENOENT'].includes(String(error.code))
        ))
          throw error;
      }
    }
  }
}
