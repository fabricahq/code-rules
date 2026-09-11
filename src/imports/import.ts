/** @fileoverview Coordinates complete multi-library imports without writing a consuming workspace. */

import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { configuration } from '../configuration';
import type { LibrarySource, ProjectConfig } from '../configuration-types';
import { ValidationError } from '../formats/validation';
import { ImportError, requireActive } from './errors';
import { fetchRevision, libraryTree, IMPORT_LIMITS } from './git-library';
import { selectLibrary } from './selection';
import type { ImportedLibrary, ImportOptions, ImportOutput } from './types';

/** Parse project input without leaking the shared parser's error type across the Imports boundary. */
function projectConfig(input: unknown): ProjectConfig {
  try {
    return configuration(input);
  } catch (error) {
    if (error instanceof ValidationError)
      throw new ImportError('invalid-configuration', error.message);
    throw error;
  }
}

/** Retrieve one source with a bounded lifetime and remove temporary Git state on every exit. */
async function importSource(
  source: LibrarySource,
  options: ImportOptions,
): Promise<ImportedLibrary> {
  const timeout = AbortSignal.timeout(IMPORT_LIMITS.durationMs);
  const signal =
    options.signal === undefined
      ? timeout
      : AbortSignal.any([timeout, options.signal]);
  let temporary: string | undefined;
  try {
    requireActive(signal);
    temporary = await mkdtemp(join(tmpdir(), 'code-rules-import-'));
    const commit = await fetchRevision(temporary, source, signal);
    const tree = await libraryTree(temporary, commit, signal);
    const library = await selectLibrary(
      temporary,
      source,
      commit,
      tree,
      signal,
    );
    requireActive(signal);
    return library;
  } catch (error) {
    if (error instanceof ImportError)
      throw new ImportError(error.code, error.message, source.name);
    if (error instanceof ValidationError)
      throw new ImportError('invalid-library', error.message, source.name);
    throw new ImportError(
      'io-error',
      'Could not read or stage the library in temporary storage.',
      source.name,
    );
  } finally {
    if (temporary !== undefined) {
      try {
        await rm(temporary, { recursive: true, force: true });
      } catch {
        throw new ImportError(
          'io-error',
          'Could not remove temporary import storage.',
          source.name,
        );
      }
    }
  }
}

/** Import all configured sources in alias order; return no partial result and never install project files. */
export async function importLibraries(
  input: unknown,
  options: ImportOptions = {},
): Promise<ImportOutput> {
  if (process.platform !== 'darwin' && process.platform !== 'linux')
    throw new ImportError(
      'unsupported-content',
      'Imports currently supports macOS and Linux.',
    );
  const config = projectConfig(input);
  if (options.signal !== undefined) requireActive(options.signal);
  const libraries: Array<readonly [string, ImportedLibrary]> = [];
  for (const source of config.sources)
    libraries.push([source.name, await importSource(source, options)]);
  return Object.fromEntries(libraries);
}
