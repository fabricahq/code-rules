/** @fileoverview Defines shared and rule-owned asset directories and the boundaries of local supporting links. */

import { posix } from 'node:path';
import { invalid, ruleGroup } from './validation';

/** Return the owning asset directory, or null for paths outside the reserved asset locations. */
export function assetDirectory(path: string): string | null {
  const parts = path.split('/');
  if (parts[0] === 'assets') return parts.length > 1 ? 'assets/' : null;
  const index = parts.indexOf('assets', 2);
  if (index < 2 || !parts[index + 1] || parts.length <= index + 2) return null;
  return `${parts.slice(0, index + 2).join('/')}/`;
}

/** Identify reserved asset paths even when their owner segment is missing. */
export function isAssetPath(path: string): boolean {
  return path.split('/').includes('assets');
}

/** Return the adjacent directory owned by a rule, including rules nested below a group. */
export function ruleAssetDirectory(rule: string): string {
  return `${posix.dirname(rule)}/assets/${posix.basename(rule, '.md')}/`;
}

/** Identify rule filenames without interpreting Markdown assets or underscore-prefixed supporting files as rules. */
export function isRuleFile(path: string): boolean {
  return (
    !isAssetPath(path) &&
    path.endsWith('.md') &&
    !posix.basename(path).startsWith('_')
  );
}

/** Return the rule owning an asset directory; the library-wide directory has no individual owner. */
export function assetOwner(directory: string): string | null {
  if (directory === 'assets/') return null;
  const parts = directory.split('/').filter(Boolean);
  const name = parts.pop();
  parts.pop();
  return `${parts.join('/')}/${name}.md`;
}

/** Reject supporting references outside the source's own assets, shared assets, other rules, or declared license files. */
export function requireAllowedTarget(
  file: string,
  target: string,
  licenseFiles: ReadonlyArray<string>,
): void {
  if (target === file || licenseFiles.includes(target)) return;
  const targetDirectory = assetDirectory(target);
  const ownDirectory = assetDirectory(file) ?? ruleAssetDirectory(file);
  if (
    targetDirectory === 'assets/' ||
    (targetDirectory !== null && targetDirectory === ownDirectory)
  )
    return;
  if (/^(techs|practices)\/[^/]+\//u.test(target) && isRuleFile(target)) {
    ruleGroup(target, file);
    return;
  }
  invalid(
    file,
    `unsupported supporting-file link: ${target}; use this rule's assets directory or the library-root assets directory`,
  );
}
