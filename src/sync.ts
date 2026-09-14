/** @fileoverview Coordinates importing libraries, generating resolved rules, and applying complete project changes. */

import { importLibraries } from './imports';
import { buildRules } from './builds';
import { applyChanges, withWriter } from './project-files/apply';
import { vendorFiles } from './project-files/snapshots';
import {
  projectLocation,
  readProject,
  requireActive,
  buildInput,
  generatedBytes,
  assertUnchanged,
  compareFiles,
  managedFiles,
} from './project-files/project';
import type { ProjectOptions, FileChanges } from './project-files/project';
export type { ProjectOptions, FileChanges } from './project-files/project';
export { ProjectError } from './project-files/files';

/** Import every source and generate resolved rules before replacing managed output; preserve authored input and recover interrupted writes. */
export async function sync(options: ProjectOptions = {}): Promise<FileChanges> {
  requireActive(options);
  const { root, configPath } = await projectLocation(options);
  return withWriter(root, async () => {
    requireActive(options);
    const state = await readProject(root, configPath);
    const libraries = await importLibraries(
      state.input,
      options.signal === undefined ? {} : { signal: options.signal },
    );
    const snapshots = Object.fromEntries(
      Object.entries(libraries).map(([alias, library]) => [
        alias,
        library.snapshot,
      ]),
    );
    const generated = generatedBytes(
      buildRules(buildInput(state, snapshots, options)).files,
    );
    const vendor = vendorFiles(libraries);
    const changes = compareFiles(
      managedFiles(
        state.vendor?.files ?? new Map(),
        state.generated?.files ?? new Map(),
      ),
      managedFiles(vendor, generated),
    );
    await applyChanges(root, { vendor, generated }, () =>
      assertUnchanged(root, configPath, state, options),
    );
    return changes;
  });
}
