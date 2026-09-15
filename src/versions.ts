/** @fileoverview Shares strict Git-tag version recognition and npm range validation between Imports and offline Builds. */

import { parse, validRange } from 'semver';
import { invalid, nonempty } from './formats/validation';

/** Validate npm range syntax while preserving the user's declared constraint for provenance. */
export function versionConstraint(value: unknown, location: string): string {
  const range = nonempty(value, location);
  if (range.length > 1024 || validRange(range) === null)
    return invalid(
      location,
      'expected an npm semantic version range, such as ^1.2.0 or >=1.2.0 <2.0.0',
    );
  return range;
}

/** Recognize a complete SemVer tag with an optional lowercase v; never coerce partial or prefixed tags. */
export function tagVersion(tag: string): string | null {
  if (!/^v?\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.+-]+)?$/u.test(tag)) return null;
  const parsed = parse(tag.startsWith('v') ? tag.slice(1) : tag);
  if (parsed === null) return null;
  return (
    parsed.version + (parsed.build.length ? `+${parsed.build.join('.')}` : '')
  );
}
