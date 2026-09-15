/** @fileoverview Exercises Imports and Builds in a child with isolated Git configuration. */

import { mock } from 'bun:test';
import * as filesystem from 'node:fs/promises';
import { importLibraries, ImportError } from '../../src/imports';
import { buildRules } from '../../src/builds';

const input: {
  configuration: unknown;
  build?: boolean;
  localFiles?: Record<string, string>;
  cancel?: boolean;
  cleanupFailure?: boolean;
} = JSON.parse(process.argv[2] ?? '{}');
if (input.cleanupFailure) {
  const originalRemove = filesystem.rm;
  mock.module('node:fs/promises', () => ({
    ...filesystem,
    /** Simulate a cleanup I/O failure after removing the fixture's temporary data. */
    rm: async (...args: Parameters<typeof filesystem.rm>): Promise<void> => {
      await originalRemove(...args);
      throw new Error('Simulated cleanup failure');
    },
  }));
}
const controller = new AbortController();
if (input.cancel) controller.abort();
try {
  const libraries = await importLibraries(input.configuration, {
    signal: controller.signal,
  });
  const snapshots = Object.fromEntries(
    Object.entries(libraries).map(([name, library]) => [
      name,
      library.snapshot,
    ]),
  );
  const files = Object.fromEntries(
    Object.entries(libraries).map(([name, library]) => [
      name,
      Object.fromEntries(
        [...library.files].map(([path, bytes]) => [
          path,
          Buffer.from(bytes).toString('base64'),
        ]),
      ),
    ]),
  );
  const generated = input.build
    ? buildRules({
        configuration: input.configuration,
        snapshots,
        localFiles: input.localFiles ?? {},
        toolVersion: 'test',
      }).files
    : null;
  process.stdout.write(JSON.stringify({ snapshots, files, generated }));
} catch (error) {
  process.stdout.write(
    JSON.stringify({
      error: error instanceof ImportError ? error.code : 'build-error',
      message: error instanceof Error ? error.message : 'Unknown failure',
      cause:
        error instanceof Error && error.cause instanceof Error
          ? error.cause.message
          : null,
    }),
  );
}
