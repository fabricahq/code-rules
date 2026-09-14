/** @fileoverview Coordinates validation, rule resolution, and rendering for the offline Builds interface. */

import type { BuildInput, BuildOutput } from './types';
import { configuration } from './configuration';
import { files, nonempty, invalid } from './validation';
import { resolveRules } from './resolve';
import { renderGeneratedFiles } from './render';

/**
 * Return the complete generated file map after applying source exclusions, replacements, and local rules.
 * Paths are relative to generated/; identical inputs yield identical bytes without I/O or input mutation.
 * Throws BuildError with invalid-input for malformed data or needs-sync for missing/mismatched snapshots.
 * Checks snapshot identity, while callers remain responsible for authenticating and verifying its contents.
 */
export function buildRules(input: BuildInput): BuildOutput {
  const config = configuration(input.configuration);
  nonempty(input.toolVersion, 'toolVersion');
  const indexMaxBytes =
    input.indexMaxBytes === undefined ? 24 * 1024 : input.indexMaxBytes;
  if (!Number.isSafeInteger(indexMaxBytes) || indexMaxBytes <= 0)
    invalid('indexMaxBytes', 'expected a positive safe integer byte budget');
  const groupInlineMaxBytes =
    input.groupInlineMaxBytes === undefined
      ? 8 * 1024
      : input.groupInlineMaxBytes;
  if (!Number.isSafeInteger(groupInlineMaxBytes) || groupInlineMaxBytes < 0)
    invalid(
      'groupInlineMaxBytes',
      'expected a non-negative safe integer byte budget',
    );
  const localFiles = files(input.localFiles, 'local');
  const resolved = resolveRules(config, input.snapshots, localFiles);
  return renderGeneratedFiles(resolved, input.toolVersion, {
    indexMaxBytes,
    groupInlineMaxBytes,
  });
}
