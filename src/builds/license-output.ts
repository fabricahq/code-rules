/** @fileoverview Assigns fixed generated destinations to library license and notice files without changing their contents. */

import type { LicenseDeclaration } from './types';
import { requiredFile } from './validation';

/** Map source-root-relative paths to fixed generated-root-relative destinations, preserving unique notice declaration order. */
export function licenseFileMappings(
  source: string,
  license: LicenseDeclaration,
): ReadonlyArray<{
  readonly sourcePath: string;
  readonly generatedPath: string;
  readonly kind: 'license' | 'notice';
}> {
  const root = `licenses/${source}`;
  return [
    ...license.files.map((sourcePath) => ({
      sourcePath,
      generatedPath: `${root}/LICENSE.md`,
      kind: 'license' as const,
    })),
    ...license.attributionFiles.map((sourcePath, index) => ({
      sourcePath,
      generatedPath: `${root}/notices/${String(index + 1).padStart(3, '0')}.md`,
      kind: 'notice' as const,
    })),
  ];
}

/** Return generated paths grouped by file role for provenance and rendered links. */
export function licenseOutputPaths(
  source: string,
  license: LicenseDeclaration,
): {
  readonly generatedFiles: ReadonlyArray<string>;
  readonly generatedAttributionFiles: ReadonlyArray<string>;
} {
  const mappings = licenseFileMappings(source, license);
  return {
    generatedFiles: mappings
      .filter((file) => file.kind === 'license')
      .map((file) => file.generatedPath),
    generatedAttributionFiles: mappings
      .filter((file) => file.kind === 'notice')
      .map((file) => file.generatedPath),
  };
}

/** Copy validated library terms to generated paths unchanged, including for sources with no active rules. */
export function generatedLicenseFiles(
  source: string,
  licenses: ReadonlyArray<LicenseDeclaration>,
  sourceFiles: ReadonlyMap<string, string>,
): ReadonlyMap<string, string> {
  const output = new Map<string, string>();
  for (const license of licenses) {
    for (const { sourcePath, generatedPath } of licenseFileMappings(
      source,
      license,
    ))
      output.set(generatedPath, requiredFile(sourceFiles, sourcePath, source));
  }
  return output;
}
