/** @fileoverview Authors library manifests, groups, and rules using shared templates and protected publication. */

import { join } from 'node:path';
import { readLibraryLicenses } from '../formats/manifest';
import { groupId, ruleGroup, groupMetadata } from '../formats/validation';
import type { GroupMetadata } from '../formats/validation';
import { ProjectError, readTree, textFile } from '../project-files/files';
import {
  libraryRoot,
  libraryManifest,
  readDeclaredTerms,
} from './library-input';
import { requireActive } from '../project-files/project';
import { withWriter } from '../project-files/apply';
import { ensureDirectory, optionalBytes, publishAuthored } from './files';
import type { AuthoredFile } from './files';
import { renderGroup, renderRuleDraft } from './templates';
import type { RuleMetadata } from './templates';
import type { AuthoringResult } from './project';

/** Library root and cancellation; library commands never use a consuming project's configuration. */
export type LibraryOptions = {
  readonly directory?: string;
  readonly signal?: AbortSignal;
};

/** Explicit publisher-supplied terms; text is preserved, never inferred or synthesized. */
export type LibraryTerms = {
  readonly spdxExpression: string;
  readonly license: string;
  readonly notice?: string;
};

const readme = `# Rule library

Describe shared engineering guidance under techs/<group>/ or practices/<group>/.
Start with code-rules library add group, then code-rules library add rule.
Complete drafts and run code-rules library check before committing and tagging a version.

Follow the [canonical authoring rubric](https://github.com/fabricahq/code-rules/blob/main/docs/src/content/docs/reference/rule-authoring.md).
Declare one license for the whole library in rule-library.json and retain its actual license text and notices.
Consumers select this Git repository and a version with code-rules add source, then run code-rules sync.
`;

/** Initialize a library without overwriting authored files; existing manifests are preserved unless new terms would collide. */
export async function initializeLibrary(
  options: LibraryOptions & { readonly terms?: LibraryTerms } = {},
): Promise<AuthoringResult> {
  requireActive(options);
  const root = await libraryRoot(options.directory);
  requireActive(options);
  await ensureDirectory(root, []);
  return withWriter(root, async () => {
    requireActive(options);
    const existing = await optionalBytes(join(root, 'rule-library.json'));
    if (existing !== null && options.terms !== undefined)
      throw new ProjectError(
        'already-initialized',
        'Library manifest exists; edit its license declaration explicitly.',
      );
    const terms = options.terms;
    const manifest =
      existing === null
        ? JSON.stringify(
            {
              formatVersion: 1,
              ...(terms === undefined
                ? {}
                : {
                    license: {
                      spdxExpression: terms.spdxExpression,
                      file: 'LICENSE.md',
                      notices: terms.notice === undefined ? [] : ['NOTICE.md'],
                    },
                  }),
            },
            null,
            2,
          ) + '\n'
        : textFile(existing, 'rule-library.json');
    const files: AuthoredFile[] = [];
    if (existing === null)
      files.push({
        path: join(root, 'rule-library.json'),
        text: manifest,
        before: null,
      });
    if (terms !== undefined) {
      files.push({
        path: join(root, 'LICENSE.md'),
        text: terms.license,
        before: null,
      });
      if (terms.notice !== undefined)
        files.push({
          path: join(root, 'NOTICE.md'),
          text: terms.notice,
          before: null,
        });
    }
    // Validate declarations against supplied files or existing contained files before publishing anything.
    const termFiles = await readDeclaredTerms(
      root,
      manifest,
      new Map(
        files.map((file) => [file.path.slice(root.length + 1), file.text]),
      ),
    );
    if ((await optionalBytes(join(root, 'README.md'))) === null)
      files.push({ path: join(root, 'README.md'), text: readme, before: null });
    await publishAuthored(root, files, options.signal);
    return {
      files: files.map((file) => file.path),
      next:
        readLibraryLicenses(termFiles, 'library').length === 0
          ? 'License is undeclared. Decide terms before sharing. Add a group and rule, then run library check.'
          : 'Add a group and rule, then run library check.',
    };
  });
}

async function validateManifest(root: string): Promise<void> {
  await readDeclaredTerms(root, await libraryManifest(root));
}

/** Create one library group; reject existing paths and invalid metadata without overwriting content. */
export async function addLibraryGroup(
  id: string,
  metadata: GroupMetadata,
  options: LibraryOptions = {},
): Promise<AuthoringResult> {
  groupId(id, 'group');
  const text = renderGroup(metadata);
  requireActive(options);
  const root = await libraryRoot(options.directory);
  await validateManifest(root);
  return withWriter(root, async () => {
    requireActive(options);
    await validateManifest(root);
    await readTree(join(root, id.split('/')[0] ?? ''));
    const path = join(root, id, '_group.json');
    await publishAuthored(root, [{ path, text, before: null }], options.signal);
    return {
      files: [path],
      next: 'Add a library rule, then run library check.',
    };
  });
}

/** Test for valid group metadata before prompting; publication rechecks under the writer lock. */
export async function hasLibraryGroup(
  id: string,
  directory?: string,
): Promise<boolean> {
  groupId(id, 'group');
  const root = await libraryRoot(directory);
  await validateManifest(root);
  await readTree(join(root, id.split('/')[0] ?? ''));
  const bytes = await optionalBytes(join(root, id, '_group.json'));
  if (bytes === null) return false;
  groupMetadata(textFile(bytes, id), id);
  return true;
}

/** Create a complete rule or marked canonical draft, optionally creating its group in the same protected publication. */
export async function addLibraryRule(
  id: string,
  metadata: RuleMetadata,
  options: LibraryOptions & {
    readonly body?: string;
    readonly group?: GroupMetadata;
  } = {},
): Promise<AuthoringResult> {
  requireActive(options);
  const group = ruleGroup(`${id}.md`, 'rule');
  let text = await renderRuleDraft(id, metadata, options.body);
  if (options.body === undefined) text += '\n<!-- code-rules:draft -->\n';
  const root = await libraryRoot(options.directory);
  await validateManifest(root);
  return withWriter(root, async () => {
    requireActive(options);
    await validateManifest(root);
    await readTree(join(root, group.split('/')[0] ?? ''));
    const files: AuthoredFile[] = [];
    if (options.group !== undefined)
      files.push({
        path: join(root, group, '_group.json'),
        text: renderGroup(options.group),
        before: null,
      });
    else if (!(await hasLibraryGroup(group, root)))
      throw new ProjectError(
        'missing-group',
        `Run library add group ${group} first or supply --create-group and metadata.`,
      );
    files.push({ path: join(root, `${id}.md`), text, before: null });
    await publishAuthored(root, files, options.signal);
    return {
      files: files.map((file) => file.path),
      next:
        options.body === undefined
          ? 'Complete the draft and remove its code-rules:draft marker, then run library check.'
          : 'Review the rule and run library check.',
    };
  });
}
