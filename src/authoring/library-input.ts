/** @fileoverview Reads library-owned inputs without traversing unrelated repository content or linked paths. */

import { dirname, basename, join, resolve } from 'node:path';
import { realpath, lstat } from 'node:fs/promises';
import { readLibraryLicenses, licensePaths } from '../formats/manifest';
import { json, object, relativePath } from '../formats/validation';
import { ProjectError, textFile } from '../project-files/files';
import { optionalBytes } from './files';

/** Resolve an existing parent without accepting a symlink at the requested library root. */
export async function libraryRoot(
  directory: string | undefined,
): Promise<string> {
  const raw = resolve(directory ?? '.');
  return join(await realpath(dirname(raw)), basename(raw));
}

/** Read the required manifest from a real library directory; initialization is a separate action. */
export async function libraryManifest(root: string): Promise<string> {
  const stat = await lstat(root);
  if (!stat.isDirectory() || stat.isSymbolicLink())
    throw new ProjectError(
      'unsafe-path',
      `${root}: expected a directory without symlinks.`,
    );
  const bytes = await optionalBytes(join(root, 'rule-library.json'));
  if (bytes === null)
    throw new ProjectError('needs-init', `${root}: run library init first.`);
  return textFile(bytes, 'rule-library.json');
}

async function containedText(root: string, path: string): Promise<string> {
  relativePath(path, 'license path');
  const parts = path.split('/');
  for (let count = 1; count < parts.length; count++) {
    const parent = join(root, ...parts.slice(0, count));
    const stat = await lstat(parent);
    if (!stat.isDirectory() || stat.isSymbolicLink())
      throw new ProjectError(
        'unsafe-path',
        `${parent}: expected a directory without symlinks.`,
      );
  }
  const bytes = await optionalBytes(join(root, path));
  if (bytes === null)
    throw new ProjectError(
      'missing-license',
      `${path}: missing declared license or notice.`,
    );
  return textFile(bytes, path);
}

/** Validate declared terms and read their exact text, or use planned new files during initialization. */
export async function readDeclaredTerms(
  root: string,
  manifest: string,
  planned: ReadonlyMap<string, string> = new Map(),
): Promise<Map<string, string>> {
  const input = object(
    json(manifest, 'rule-library.json'),
    'rule-library.json',
  );
  const paths = new Set<string>();
  if (input.license !== undefined) {
    const license = object(input.license, 'license');
    if (typeof license.file === 'string') paths.add(license.file);
    if (Array.isArray(license.notices))
      for (const notice of license.notices)
        if (typeof notice === 'string') paths.add(notice);
  }
  const files = new Map([['rule-library.json', manifest]]);
  const declarations = readLibraryLicenses(files, 'library', paths);
  for (const path of licensePaths(declarations))
    files.set(path, planned.get(path) ?? (await containedText(root, path)));
  return files;
}
