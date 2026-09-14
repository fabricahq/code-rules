/** @fileoverview Validates library license declarations and resolves their files within an in-memory source snapshot. */

import parseSpdxExpression from 'spdx-expression-parse';

import type { LicenseDeclaration } from './types';

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

/** Parse the library manifest into an object with a supported formatVersion, or throw BuildError. */
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

/** Validate notice paths in declaration order, retaining indexed locations for diagnostics; throw BuildError for invalid entries. */
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
 * Return the validated license path followed by notice paths in declaration order, retaining duplicates.
 * Return an empty array when licensing is unspecified; throw BuildError for an invalid declaration.
 */
function licenseAndNoticeDeclarations(
  manifest: Record<string, unknown>,
  sourceName: string,
): ReadonlyArray<DeclaredPath> {
  const value = field(manifest, 'license');
  if (value === undefined) return [];
  const location = `${sourceName}/${LIBRARY_MANIFEST}: license`;
  const license = object(value, location);
  const file = declaredPath(field(license, 'file'), `${location}.file`);
  const notices = noticePaths(field(license, 'notices'), `${location}.notices`);
  return [file, ...notices];
}

/** Throw BuildError at the first declaration whose path is absent from the snapshot; empty files count as present. */
function requireDeclaredFiles(
  sourceFiles: ReadonlyMap<string, string>,
  declarations: ReadonlyArray<DeclaredPath>,
): void {
  for (const { path, location } of declarations) {
    if (!sourceFiles.has(path)) {
      invalid(location, `missing declared file ${JSON.stringify(path)}`);
    }
  }
}

/**
 * Validate the library manifest and read its declared expression, license, and notice files.
 * Return the single library-wide declaration with notices deduplicated in declaration order, or an empty array when licensing is unspecified.
 * Throw BuildError for an invalid manifest or a declared file missing from the snapshot.
 */
export function readLibraryLicenses(
  sourceFiles: ReadonlyMap<string, string>,
  sourceName: string,
): ReadonlyArray<LicenseDeclaration> {
  const manifest = libraryManifest(sourceFiles, sourceName);
  const declarations = licenseAndNoticeDeclarations(manifest, sourceName);
  requireDeclaredFiles(sourceFiles, declarations);
  const first = declarations[0];
  if (first === undefined) return [];
  const license = object(
    field(manifest, 'license'),
    `${sourceName}/rule-library.json: license`,
  );
  if (field(license, 'expression') !== undefined)
    return invalid(
      `${sourceName}/rule-library.json: license.expression`,
      'renamed to license.spdxExpression; move the declaration to that field',
    );
  const expression = field(license, 'spdxExpression');
  const file = first.path;
  return [
    {
      spdxExpression:
        expression === undefined
          ? null
          : spdxExpression(
              expression,
              `${sourceName}/rule-library.json: license.spdxExpression`,
            ),
      files: [file],
      attributionFiles: [
        ...new Set(
          declarations
            .slice(1)
            .map(({ path }) => path)
            .filter((path) => path !== file),
        ),
      ],
    },
  ];
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
