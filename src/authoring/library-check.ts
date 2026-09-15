/** @fileoverview Validates a working library's complete rule and asset inventory without generating or writing files. */

import { join } from 'node:path';
import { buildRules } from '../builds';
import { readLibraryLicenses, licensePaths } from '../formats/manifest';
import {
  assetDirectory,
  assetOwner,
  isAssetPath,
  isRuleFile,
  requireAllowedTarget,
} from '../formats/assets';
import { groupId } from '../formats/validation';
import { markdownTargets } from '../formats/markdown-links';
import { readTree, textFile, ProjectError } from '../project-files/files';
import { requireActive } from '../project-files/project';
import {
  libraryRoot,
  libraryManifest,
  readDeclaredTerms,
} from './library-input';
import type { LibraryOptions } from './library';

/** Successful format check with adoption counts and explicit licensing caveats; no files are changed. */
export type LibraryCheckResult = {
  readonly groups: number;
  readonly rules: number;
  readonly warnings: readonly string[];
};

/** Check all groups, rules, links, assets, and declared terms with shared import/build conventions; reject marked unfinished drafts. */
export async function checkLibrary(
  options: LibraryOptions = {},
): Promise<LibraryCheckResult> {
  requireActive(options);
  const root = await libraryRoot(options.directory);
  const files = await readDeclaredTerms(root, await libraryManifest(root));
  const licenses = readLibraryLicenses(files, 'library');
  const terms = licensePaths(licenses);
  const paths = new Set(files.keys());
  for (const directory of ['techs', 'practices', 'assets']) {
    const tree = await readTree(join(root, directory));
    for (const [relative, bytes] of tree?.files ?? []) {
      requireActive(options);
      const path = `${directory}/${relative}`;
      paths.add(path);
      if (!isAssetPath(path) || path.endsWith('.md') || terms.includes(path))
        files.set(path, textFile(bytes, path));
    }
  }
  const groups = new Set<string>();
  let rules = 0;
  const spellings = new Map<string, string>();
  for (const path of paths) {
    const parts = path.split('/');
    for (let count = 1; count <= parts.length; count++) {
      const prefix = parts.slice(0, count).join('/');
      const key = prefix
        .normalize('NFC')
        .toLowerCase()
        .toUpperCase()
        .normalize('NFC');
      const old = spellings.get(key);
      if (old !== undefined && old !== prefix)
        throw new ProjectError(
          'path-collision',
          `${path}: collides with ${old}.`,
        );
      spellings.set(key, prefix);
    }
    if (path === 'rule-library.json' || terms.includes(path)) continue;
    if (isAssetPath(path)) {
      const directory = assetDirectory(path);
      const owner = directory === null ? null : assetOwner(directory);
      if (directory === null || (owner !== null && !paths.has(owner)))
        throw new ProjectError(
          'invalid-asset',
          `${path}: missing rule owner or invalid asset directory.`,
        );
    } else {
      const id = groupId(parts.slice(0, 2).join('/'), path);
      groups.add(id);
      if (path !== `${id}/_group.json` && !isRuleFile(path))
        throw new ProjectError(
          'invalid-library',
          `${path}: use a rule Markdown file, _group.json, or a conventional assets directory.`,
        );
      if (isRuleFile(path)) {
        rules++;
        if (files.get(path)?.includes('<!-- code-rules:draft -->'))
          throw new ProjectError(
            'incomplete-rule',
            `${path}: complete the draft and remove its code-rules:draft marker.`,
          );
      }
    }
    if (path.endsWith('.md'))
      for (const target of markdownTargets(files.get(path) ?? '', path)) {
        requireAllowedTarget(path, target, terms);
        if (!paths.has(target))
          throw new ProjectError(
            'missing-link',
            `${path}: missing link destination ${target}.`,
          );
      }
  }
  // A private in-memory envelope lets the established builder validate the complete library without fetching or publishing it.
  const repository = 'https://code-rules.invalid/library.git';
  const ref = 'working-tree';
  buildRules({
    configuration: {
      schemaVersion: 1,
      sources: {
        library: { repository, ref, groups: '*', exclude: {}, replace: {} },
      },
    },
    snapshots: {
      library: {
        repository,
        ref,
        resolvedCommit: '0'.repeat(40),
        groups: [...groups].sort(),
        groupSelection: '*',
        files: Object.fromEntries(files),
        filePaths: [...paths],
      },
    },
    localFiles: {},
    toolVersion: 'library-check',
  });
  requireActive(options);
  return {
    groups: groups.size,
    rules,
    warnings:
      licenses.length === 0
        ? ['License is undeclared. Decide terms before sharing this library.']
        : licenses.some((license) => license.spdxExpression === null)
          ? [
              'The license has no SPDX expression. Declare the library terms explicitly.',
            ]
          : [],
  };
}
