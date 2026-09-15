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
import { requireActive } from '../project-files/project';
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

async function restoreClaim(claim: string, target: string): Promise<void> {
  // Linking is exclusive: never overwrite a path an editor recreated while its old file was set aside.
  await link(claim, target);
  await unlink(claim);
}

/** Publish under the writer lock, claiming existing files before comparison and installing replacements exclusively. Preserve uncertain claims for manual recovery. */
export async function publishAuthored(
  root: string,
  files: readonly AuthoredFile[],
  signal?: AbortSignal,
): Promise<void> {
  const options = signal === undefined ? {} : { signal };
  requireActive(options);
  for (const file of files)
    relativePath(relative(root, file.path), 'authoring target');
  if (new Set(files.map((file) => file.path)).size !== files.length)
    throw new Error('Duplicate authoring target.');
  if (files.filter((file) => file.before !== null).length > 1)
    throw new Error('An authoring operation can replace at most one file.');
  const directories: string[] = [];
  const published: AuthoredFile[] = [];
  const pending = (await readdir(root)).filter((name) =>
    name.startsWith('.code-rules-authoring-'),
  );
  if (pending.length)
    throw new ProjectError(
      'recovery-required',
      `Inspect retained authoring files before retrying: ${pending.map((name) => join(root, name)).join(', ')}`,
    );
  const stage = await mkdtemp(join(root, '.code-rules-authoring-'));
  let retainStage = false;
  try {
    for (const file of files) {
      requireActive(options);
      await ensureDirectory(dirname(file.path), directories);
      await verify(file);
    }
    const ordered = [
      ...files.filter((file) => file.before === null),
      ...files.filter((file) => file.before !== null),
    ];
    for (const [index, file] of ordered.entries()) {
      requireActive(options);
      const staged = join(stage, `new-${index}`);
      await writeFile(staged, file.text, { flag: 'wx' });
      await verify(file);
      requireActive(options);
      if (file.before === null) {
        await link(staged, file.path);
        published.push(file);
        await unlink(staged);
        requireActive(options);
      } else {
        const claim = join(stage, `previous-${index}`);
        // Record how to restore the original if the process exits while its pathname is claimed.
        await writeFile(
          join(stage, 'recovery.json'),
          JSON.stringify({ target: file.path, previous: claim }, null, 2),
        );
        requireActive(options);
        await rename(file.path, claim);
        try {
          const actual = await readBytes(claim);
          if (digest(actual) !== digest(file.before))
            throw new ProjectError(
              'concurrent-change',
              `${file.path}: changed during publication; preserving the editor's bytes.`,
            );
          requireActive(options);
          await link(staged, file.path);
        } catch (error) {
          try {
            await restoreClaim(claim, file.path);
          } catch {
            retainStage = true;
            throw new ProjectError(
              'recovery-required',
              `${file.path}: another file occupies the target. Its bytes were preserved; the previous file is retained at ${claim}.`,
              { cause: error },
            );
          }
          throw error;
        }
        // The exclusive link is the replacement commit point. Cancellation after it leaves a complete result.
        await unlink(staged);
      }
    }
  } catch (error) {
    for (const [index, file] of published.reverse().entries()) {
      const claim = join(stage, `rollback-${index}`);
      try {
        await rename(file.path, claim);
      } catch (failure) {
        if (missing(failure)) continue;
        retainStage = true;
        throw new ProjectError(
          'recovery-required',
          `Could not claim ${file.path} for rollback; inspect ${stage}.`,
          { cause: failure },
        );
      }
      try {
        const current = await readBytes(claim);
        if (digest(current) !== digest(Buffer.from(file.text)))
          await restoreClaim(claim, file.path);
      } catch (failure) {
        retainStage = true;
        throw new ProjectError(
          'recovery-required',
          `Preserved uncertain rollback content at ${claim}; inspect it alongside ${file.path}.`,
          { cause: failure },
        );
      }
    }
    throw error;
  } finally {
    if (!retainStage) await rm(stage, { recursive: true, force: true });
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
