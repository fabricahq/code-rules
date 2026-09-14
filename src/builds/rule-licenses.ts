/** @fileoverview Parses explicit license and attribution declarations without inferring permission or certifying SPDX expressions. */

import type { Attribution, LicenseDeclaration } from './types';
import {
  compare,
  field,
  invalid,
  nonempty,
  object,
  relativePath,
  requiredFile,
  strings,
} from './validation';

/** Preserve a declared single-line expression; its legal meaning and correspondence to the files remain the publisher's responsibility. */
export function licenseExpression(value: unknown, location: string): string {
  const expression = nonempty(value, location);
  if (/[\x00-\x1f\x7f]/u.test(expression))
    return invalid(location, 'expected a single-line license expression');
  return expression;
}

/** Resolve explicit file paths from the source root, preserving complete license text outside the generated rule. */
function declarationPaths(
  value: unknown,
  location: string,
): ReadonlyArray<string> {
  return strings(value, location)
    .map((path) => relativePath(path, location))
    .sort(compare);
}

/** Parse optional rule-specific terms; omission inherits a library default while an explicit list must contain at least one declaration. */
export function ruleLicenses(
  value: unknown,
  location: string,
): ReadonlyArray<LicenseDeclaration> | null {
  if (value === undefined) return null;
  if (!Array.isArray(value) || value.length === 0)
    return invalid(location, 'expected a non-empty license declaration array');
  const entries: ReadonlyArray<unknown> = value;
  return entries.map((entry, index) => {
    const where = `${location}[${index}]`;
    const data = object(entry, where);
    const expression = licenseExpression(
      field(data, 'expression'),
      `${where}.expression`,
    );
    const files = declarationPaths(field(data, 'files'), `${where}.files`);
    if (!files.length)
      return invalid(`${where}.files`, 'expected at least one license file');
    const attributionFiles = field(data, 'attributionFiles');
    return {
      expression,
      files,
      attributionFiles:
        attributionFiles === undefined
          ? []
          : declarationPaths(attributionFiles, `${where}.attributionFiles`),
    };
  });
}

/** Read explicit external-source citations for adaptations; these are declarations, not authenticated origins or license grants. */
export function ruleAttribution(
  value: unknown,
  location: string,
): ReadonlyArray<Attribution> {
  if (value === undefined) return [];
  if (!Array.isArray(value))
    return invalid(location, 'expected an attribution array');
  const entries: ReadonlyArray<unknown> = value;
  return entries.map((entry, index) => {
    const where = `${location}[${index}]`;
    const data = object(entry, where);
    const url = nonempty(field(data, 'url'), `${where}.url`);
    let parsed: URL;
    try {
      parsed = new URL(url);
    } catch {
      return invalid(`${where}.url`, 'expected an absolute HTTP(S) URL');
    }
    if (
      !['http:', 'https:'].includes(parsed.protocol) ||
      parsed.username ||
      parsed.password
    )
      return invalid(
        `${where}.url`,
        'expected an absolute HTTP(S) URL without credentials',
      );
    return {
      url: parsed.href,
      description: nonempty(field(data, 'description'), `${where}.description`),
    };
  });
}

/** Collect unique source-relative license and notice paths without assuming a license from their names. */
export function licensePaths(
  licenses: ReadonlyArray<LicenseDeclaration>,
): ReadonlyArray<string> {
  return [
    ...new Set(
      licenses.flatMap((license) => [
        ...license.files,
        ...license.attributionFiles,
      ]),
    ),
  ].sort(compare);
}

/** Require every rule-declared license and attribution file in its owning snapshot or local input. */
export function requireLicenseFiles(
  licenses: ReadonlyArray<LicenseDeclaration>,
  sourceFiles: ReadonlyMap<string, string>,
  location: string,
): void {
  for (const path of licensePaths(licenses))
    requiredFile(sourceFiles, path, location);
}
