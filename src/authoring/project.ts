/** @fileoverview Creates project scaffolding and local definitions, and records sources without fetching them. */

import { dirname, basename, resolve, join } from 'node:path';
import { realpath, lstat } from 'node:fs/promises';
import { configuration } from '../configuration';
import {
  groupId,
  ruleGroup,
  groupMetadata,
  json,
  object,
} from '../formats/validation';
import type { GroupMetadata } from '../formats/validation';
import {
  ProjectError,
  textFile,
  readTree,
  missing,
} from '../project-files/files';
import { requireActive } from '../project-files/project';
import { withWriter } from '../project-files/apply';
import { loadSnapshots } from '../project-files/snapshots';
import { renderGroup, renderRuleDraft, localReadme } from './templates';
import type { RuleMetadata } from './templates';
import { ensureDirectory, optionalBytes, publishAuthored } from './files';
import type { AuthoredFile } from './files';

/** Configuration location and caller cancellation shared by authoring operations. */
export type AuthoringOptions = {
  readonly configPath?: string;
  readonly signal?: AbortSignal;
};

/** Result names authored paths and the next explicit command; generated files are not changed. */
export type AuthoringResult = {
  readonly files: readonly string[];
  readonly next: string;
};

async function canonicalParent(path: string): Promise<string> {
  try {
    return await realpath(path);
  } catch (error) {
    if (!missing(error)) throw error;
    return join(await canonicalParent(dirname(path)), basename(path));
  }
}

async function location(
  configPath: string | undefined,
): Promise<{ root: string; configPath: string }> {
  const path = resolve(configPath ?? '.code-rules/config.json');
  const rawRoot = dirname(path);
  const root = join(await canonicalParent(dirname(rawRoot)), basename(rawRoot));
  return { root, configPath: join(root, basename(path)) };
}

async function readConfiguration(
  path: string,
): Promise<{ bytes: Uint8Array; value: Record<string, unknown> }> {
  const bytes = await optionalBytes(path);
  if (bytes === null)
    throw new ProjectError(
      'needs-init',
      `${path}: missing configuration; run init first.`,
    );
  const value = object(json(textFile(bytes, path), path), path);
  configuration(value);
  return { bytes, value };
}

async function existingLocation(
  configPath: string | undefined,
): Promise<{ root: string; configPath: string }> {
  const target = await location(configPath);
  const stat = await lstat(target.root).catch((error) => {
    if (missing(error))
      throw new ProjectError(
        'needs-init',
        'Project is not initialized; run init first.',
      );
    throw error;
  });
  if (!stat.isDirectory() || stat.isSymbolicLink())
    throw new ProjectError(
      'unsafe-path',
      `${target.root}: expected a directory without symlinks.`,
    );
  await readConfiguration(target.configPath);
  return target;
}

/** Create an empty project and local orientation, preserving existing valid configuration and README bytes. */
export async function initializeProject(
  options: AuthoringOptions = {},
): Promise<AuthoringResult> {
  requireActive(options);
  const target = await location(options.configPath);
  requireActive(options);
  await ensureDirectory(target.root, []);
  return withWriter(target.root, async () => {
    requireActive(options);
    const old = await optionalBytes(target.configPath);
    if (old !== null)
      configuration(json(textFile(old, target.configPath), target.configPath));
    // Inventory before scaffolding so a linked or case-colliding local tree never receives writes.
    await readTree(join(target.root, 'local'));
    const readmePath = join(target.root, 'local/README.md');
    const readme = await optionalBytes(readmePath);
    const files: AuthoredFile[] = [];
    if (old === null)
      files.push({
        path: target.configPath,
        text: JSON.stringify({ schemaVersion: 1, sources: {} }, null, 2) + '\n',
        before: null,
      });
    if (readme === null)
      files.push({ path: readmePath, text: localReadme, before: null });
    await publishAuthored(target.root, files, options.signal);
    return {
      files: files.map((file) => file.path),
      next: 'Add a local group and rule, then run build. Or add a source and run sync.',
    };
  });
}

/** Create local group metadata without changing an existing local definition or imported content. */
export async function addLocalGroup(
  id: string,
  metadata: GroupMetadata,
  options: AuthoringOptions = {},
): Promise<AuthoringResult> {
  groupId(id, 'group');
  const text = renderGroup(metadata);
  requireActive(options);
  const target = await existingLocation(options.configPath);
  requireActive(options);
  return withWriter(target.root, async () => {
    requireActive(options);
    await readConfiguration(target.configPath);
    await readTree(join(target.root, 'local'));
    const path = join(target.root, 'local', id, '_group.json');
    await publishAuthored(
      target.root,
      [{ path, text, before: null }],
      options.signal,
    );
    return { files: [path], next: 'Add a rule to this group, then run build.' };
  });
}

async function groupAvailable(
  root: string,
  input: unknown,
  id: string,
): Promise<boolean> {
  const local = await readTree(join(root, 'local'));
  const metadata = local?.files.get(`${id}/_group.json`);
  if (metadata !== undefined) {
    groupMetadata(textFile(metadata, id), id);
    return true;
  }
  const vendor = await readTree(join(root, 'vendor'));
  if (vendor === null) return false;
  const snapshots = loadSnapshots(input, vendor.files);
  return Object.values(snapshots).some((snapshot) => {
    const text = snapshot.files[`${id}/_group.json`];
    if (text === undefined) return false;
    groupMetadata(text, id);
    return snapshot.groups.includes(id);
  });
}

/** Check current local or verified imported metadata before offering interactive missing-group creation. Writes recheck under the writer lock. */
export async function hasLocalRuleGroup(
  id: string,
  configPath?: string,
): Promise<boolean> {
  groupId(id, 'group');
  const target = await existingLocation(configPath);
  return groupAvailable(
    target.root,
    (await readConfiguration(target.configPath)).value,
    id,
  );
}

/** Create a rule draft or supplied body, optionally creating its missing local group in the same operation. */
export async function addLocalRule(
  id: string,
  metadata: RuleMetadata,
  options: {
    readonly configPath?: string;
    readonly signal?: AbortSignal;
    readonly body?: string;
    readonly group?: GroupMetadata;
  } = {},
): Promise<AuthoringResult> {
  requireActive(options);
  const group = ruleGroup(`${id}.md`, 'rule');
  const text = await renderRuleDraft(id, metadata, options.body);
  const groupText =
    options.group === undefined ? undefined : renderGroup(options.group);
  const target = await existingLocation(options.configPath);
  requireActive(options);
  return withWriter(target.root, async () => {
    requireActive(options);
    const config = await readConfiguration(target.configPath);
    await readTree(join(target.root, 'local'));
    const files: AuthoredFile[] = [];
    if (groupText !== undefined) {
      files.push({
        path: join(target.root, 'local', group, '_group.json'),
        text: groupText,
        before: null,
      });
    } else if (!(await groupAvailable(target.root, config.value, group))) {
      throw new ProjectError(
        'missing-group',
        `No metadata for ${group}; run local add group ${group}, or supply --create-group and its metadata.`,
      );
    }
    const path = join(target.root, 'local', `${id}.md`);
    files.push({ path, text, before: null });
    await publishAuthored(target.root, files, options.signal);
    return {
      files: files.map((file) => file.path),
      next:
        options.body === undefined
          ? 'Complete the draft and remove unused template prompts before running build.'
          : 'Review the rule, then run build.',
    };
  });
}

/** Validate and add one source declaration, preserving other selections and exceptions. Run sync separately to fetch. */
export async function addSource(
  alias: string,
  source: unknown,
  options: AuthoringOptions = {},
): Promise<AuthoringResult> {
  requireActive(options);
  const target = await existingLocation(options.configPath);
  requireActive(options);
  return withWriter(target.root, async () => {
    requireActive(options);
    const config = await readConfiguration(target.configPath);
    const sources = object(config.value.sources, 'sources');
    if (Object.hasOwn(sources, alias))
      throw new ProjectError(
        'source-exists',
        `Source ${alias} already exists; edit its configuration explicitly.`,
      );
    const value = { ...config.value, sources: { ...sources, [alias]: source } };
    configuration(value);
    await publishAuthored(
      target.root,
      [
        {
          path: target.configPath,
          before: config.bytes,
          text: JSON.stringify(value, null, 2) + '\n',
        },
      ],
      options.signal,
    );
    return {
      files: [target.configPath],
      next: 'Run sync to import this source and regenerate resolved rules.',
    };
  });
}
