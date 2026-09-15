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

/** Translate a source failure without publishing arbitrary I/O or subprocess diagnostic text. */
function sourceFailure(error: unknown, source: string): ImportError {
  if (error instanceof ImportError)
    return new ImportError(error.code, error.message, source);
  if (error instanceof ValidationError)
    return new ImportError('invalid-library', error.message, source);
  return new ImportError(
    'io-error',
    'Could not read or stage the library in temporary storage.',
    source,
  );
}

/** Retrieve one source and clean up temporary state, preserving the primary error when cleanup also fails. */
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
  let outcome:
    { readonly library: ImportedLibrary } | { readonly error: ImportError };
  let cleanupFailure: ImportError | undefined;
  try {
    requireActive(signal);
    temporary = await mkdtemp(join(tmpdir(), 'code-rules-import-'));
    const revision = await fetchRevision(temporary, source, signal);
    const commit = revision.commit;
    const tree = await libraryTree(temporary, commit, signal);
    const library = await selectLibrary(
      temporary,
      source,
      commit,
      tree,
      signal,
    );
    requireActive(signal);
    outcome = {
      library: {
        ...library,
        snapshot: {
          ...library.snapshot,
          ...(revision.resolvedTag === undefined
            ? {}
            : {
                resolvedTag: revision.resolvedTag,
                resolvedVersion: revision.resolvedVersion,
              }),
        },
      },
    };
  } catch (error) {
    outcome = { error: sourceFailure(error, source.name) };
  } finally {
    if (temporary !== undefined) {
      try {
        await rm(temporary, {
          recursive: true,
          force: true,
          maxRetries: 3,
          retryDelay: 100,
        });
      } catch {
        cleanupFailure = new ImportError(
          'io-error',
          'Could not remove temporary import storage.',
          source.name,
        );
      }
    }
  }
  if ('error' in outcome) {
    if (cleanupFailure !== undefined) outcome.error.cause = cleanupFailure;
    throw outcome.error;
  }
  if (cleanupFailure !== undefined) throw cleanupFailure;
  return outcome.library;
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
