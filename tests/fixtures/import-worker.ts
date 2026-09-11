/** @fileoverview Exercises Imports and Builds in a child with isolated Git configuration. */

import { importLibraries, ImportError } from '../../src/imports';
import { buildRules } from '../../src/builds';

const input: {
  configuration: unknown;
  build?: boolean;
  localFiles?: Record<string, string>;
  cancel?: boolean;
} = JSON.parse(process.argv[2] ?? '{}');
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
    }),
  );
}
