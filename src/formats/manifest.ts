/** @fileoverview Validates library license declarations and resolves their files within an in-memory source snapshot. */

import parseSpdxExpression from 'spdx-expression-parse';

/** Declared terms and retained files, relative to the owning library root; null expression means legacy unidentified terms. */
export type LicenseDeclaration = {
  readonly spdxExpression: string | null;
  readonly files: ReadonlyArray<string>;
  readonly attributionFiles: ReadonlyArray<string>;
};

import {
  compare,
  field,
  invalid,
  json,
  nonempty,
  object,
  relativePath,
  requiredFile,
} from './validation';

const LIBRARY_MANIFEST = 'rule-library.json';
const SUPPORTED_FORMAT_VERSION = 1;

/** A validated source-relative path paired with the manifest field that declared it for diagnostics. */
type DeclaredPath = {
  readonly path: string;
  readonly location: string;
};

/** License fields with validated paths and their diagnostic context; file presence and SPDX remain unchecked. */
type LicenseFields = {
  readonly metadata: Record<string, unknown>;
  readonly location: string;
  readonly file: DeclaredPath;
  readonly notices: ReadonlyArray<DeclaredPath>;
};

/** Parse the library manifest into an object with a supported formatVersion, or throw ValidationError. */
function libraryManifest(
  sourceFiles: ReadonlyMap<string, string>,
  sourceName: string,
): Record<string, unknown> {
  const location = `${sourceName}/${LIBRARY_MANIFEST}`;
  const text = requiredFile(sourceFiles, LIBRARY_MANIFEST, sourceName);
  const parsed = json(text, location);
  const manifest = object(parsed, location);
  if (field(manifest, 'formatVersion') !== SUPPORTED_FORMAT_VERSION) {
    return invalid(
      `${location}: formatVersion`,
      `only library formatVersion ${SUPPORTED_FORMAT_VERSION} is supported`,
    );
  }
  return manifest;
}

/** Validate a library file path and retain its location for error reporting. */
function declaredPath(value: unknown, location: string): DeclaredPath {
  const text = nonempty(value, location);
  return { path: relativePath(text, location), location };
}

/** Validate notice paths in declaration order, retaining indexed locations for diagnostics; throw ValidationError for invalid entries. */
function noticePaths(
  value: unknown,
  location: string,
): ReadonlyArray<DeclaredPath> {
  if (!Array.isArray(value)) {
    return invalid(location, 'expected an array of notice paths');
  }
  const entries: ReadonlyArray<unknown> = value;
  return entries.map((entry, index) =>
    declaredPath(entry, `${location}[${index}]`),
  );
}

/**
 * Read license fields with validated paths, retaining notice order, duplicates, and diagnostic locations.
 * Return null when licensing is unspecified; throw ValidationError for an invalid object or path declaration.
 */
function licenseFields(
  manifest: Record<string, unknown>,
  sourceName: string,
): LicenseFields | null {
  const value = field(manifest, 'license');
  if (value === undefined) return null;
  const location = `${sourceName}/${LIBRARY_MANIFEST}: license`;
  const license = object(value, location);
  for (const key of Object.keys(license)) {
    if (!['file', 'notices', 'spdxExpression', 'expression'].includes(key))
      invalid(location, `unknown field ${key}`);
  }
  const file = declaredPath(field(license, 'file'), `${location}.file`);
  const notices = noticePaths(field(license, 'notices'), `${location}.notices`);
  return { metadata: license, location, file, notices };
}

/** Throw ValidationError at the first declaration whose path is absent from the snapshot; empty files count as present. */
function requireDeclaredFiles(
  sourcePaths: ReadonlySet<string>,
  declarations: ReadonlyArray<DeclaredPath>,
): void {
  for (const { path, location } of declarations) {
    if (!sourcePaths.has(path)) {
      invalid(location, `missing declared file ${JSON.stringify(path)}`);
    }
  }
}

/**
 * Validate the library manifest and read its declared expression, license, and notice files.
 * Return the single library-wide declaration with notices deduplicated in declaration order, or an empty array when licensing is unspecified.
 * Throw ValidationError for an invalid manifest or a declared file missing from the snapshot.
 */
export function readLibraryLicenses(
  sourceFiles: ReadonlyMap<string, string>,
  sourceName: string,
  sourcePaths: ReadonlySet<string> = new Set(sourceFiles.keys()),
): ReadonlyArray<LicenseDeclaration> {
  const manifest = libraryManifest(sourceFiles, sourceName);
  const fields = licenseFields(manifest, sourceName);
  if (fields === null) return [];
  requireDeclaredFiles(sourcePaths, [fields.file, ...fields.notices]);
  return [normalizedLicense(fields)];
}

/** Reject obsolete or invalid SPDX fields, then deduplicate notices in declaration order and omit the license file itself. */
function normalizedLicense(fields: LicenseFields): LicenseDeclaration {
  const { metadata, location, file, notices } = fields;
  if (field(metadata, 'expression') !== undefined)
    return invalid(
      `${location}.expression`,
      'renamed to license.spdxExpression; move the declaration to that field',
    );
  const expression = field(metadata, 'spdxExpression');
  return {
    spdxExpression:
      expression === undefined
        ? null
        : spdxExpression(expression, `${location}.spdxExpression`),
    files: [file.path],
    attributionFiles: [
      ...new Set(
        notices.map(({ path }) => path).filter((path) => path !== file.path),
      ),
    ],
  };
}

/** Validate SPDX syntax and identifiers, preserving the publisher's declaration without assessing its legal meaning or agreement with the files. */
function spdxExpression(value: unknown, location: string): string {
  const expression = nonempty(value, location);
  if (/[\x00-\x1f\x7f]/u.test(expression))
    return invalid(location, 'expected a single-line license expression');
  try {
    parseSpdxExpression(expression);
  } catch {
    return invalid(
      location,
      'expected an SPDX expression using recognized identifiers or LicenseRef- custom terms',
    );
  }
  return expression;
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

/** Validate the library declaration and return unique retained license paths in code-unit order. */
export function collectLibraryLicensePaths(
  sourceFiles: ReadonlyMap<string, string>,
  sourceName: string,
  sourcePaths: ReadonlySet<string> = new Set(sourceFiles.keys()),
): ReadonlyArray<string> {
  return licensePaths(
    readLibraryLicenses(sourceFiles, sourceName, sourcePaths),
  );
}
