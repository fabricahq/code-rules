/** @fileoverview Collects explicit or interactive authoring inputs before invoking project operations. */

import { createInterface } from 'node:readline/promises';
import { stdin, stderr } from 'node:process';
import { readFile } from 'node:fs/promises';
import { textFile } from '../project-files/files';
import { groupId, ruleGroup } from '../formats/validation';
import type { GroupMetadata } from '../formats/validation';
import {
  initializeProject,
  addLocalGroup,
  addLocalRule,
  addSource,
  hasLocalRuleGroup,
} from './project';
import type { AuthoringResult } from './project';
import {
  initializeLibrary,
  addLibraryGroup,
  addLibraryRule,
  hasLibraryGroup,
} from './library';
import type { LibraryTerms } from './library';
import { checkLibrary } from './library-check';
import { UsageError } from '../cli/arguments';
import type { Invocation } from '../cli/arguments';
import type { LibraryCheckResult } from './library-check';

async function licenseText(path: string): Promise<string> {
  return new TextDecoder('utf-8', { fatal: true, ignoreBOM: true }).decode(
    await readFile(path),
  );
}

/** Run an authoring command, or return null for other CLI commands. Prompt only on a terminal and never hold a writer lock while prompting. */
export async function runAuthoring(
  request: Invocation,
  signal: AbortSignal,
): Promise<AuthoringResult | LibraryCheckResult | null> {
  const { command, id, flags } = request;
  if (command.action.type !== 'authoring') return null;
  const { kind, target } = command.action;
  const directory = flags.get('directory')?.[0];
  const libraryOptions = {
    signal,
    ...(directory === undefined ? {} : { directory }),
  };
  const configPath = flags.get('config')?.[0];
  const projectOptions = {
    signal,
    ...(configPath === undefined ? {} : { configPath }),
  };
  if (kind === 'check') return checkLibrary(libraryOptions);
  if (kind === 'init' && target === 'project')
    return initializeProject(projectOptions);
  if (kind === 'init' && flags.has('notice-file') && !flags.has('spdx'))
    throw new UsageError('--notice-file requires --spdx and --license-file.');
  if (kind === 'init' && flags.has('spdx') !== flags.has('license-file'))
    throw new UsageError('Supply --spdx and --license-file together.');
  if (kind === 'group') groupId(id, 'group');
  if (kind === 'rule') {
    if (id.endsWith('.md'))
      throw new UsageError('Use a rule ID without the .md extension.');
    ruleGroup(`${id}.md`, 'rule');
    if (
      !flags.has('create-group') &&
      [...flags.keys()].some((key) => key.startsWith('group-'))
    )
      throw new UsageError('Group metadata requires --create-group.');
  }
  if (kind === 'source' && flags.has('ref') && flags.has('version'))
    throw new UsageError('Specify exactly one of --ref or --version.');
  let terminal: ReturnType<typeof createInterface> | undefined;
  async function ask(label: string): Promise<string> {
    if (!stdin.isTTY || !stderr.isTTY || flags.has('non-interactive'))
      throw new UsageError(
        `${label} Missing input; supply explicit flags in noninteractive mode.`,
      );
    terminal ??= createInterface({ input: stdin, output: stderr });
    return (await terminal.question(label + ' ', { signal })).trim();
  }
  async function required(key: string, label: string): Promise<string> {
    const value = flags.get(key)?.[0] ?? (await ask(`${label} (--${key}):`));
    if (!value) throw new UsageError(`Provide --${key}.`);
    return value;
  }
  async function group(prefix: string): Promise<GroupMetadata> {
    return {
      name: await required(prefix + 'name', 'Group name'),
      description: await required(prefix + 'description', 'Group scope'),
      whenToRead: flags.get(prefix + 'when-to-read') ?? [
        await required(
          prefix + 'when-to-read',
          'When should an agent read this group?',
        ),
      ],
    };
  }
  try {
    if (kind === 'init') {
      let terms: LibraryTerms | undefined;
      const spdx = flags.get('spdx')?.[0];
      if (spdx !== undefined) {
        const licensePath = await required(
          'license-file',
          'Existing license text file',
        );
        const noticePath = flags.get('notice-file')?.[0];
        terms = {
          spdxExpression: spdx,
          license: await licenseText(licensePath),
          ...(noticePath === undefined
            ? {}
            : { notice: await licenseText(noticePath) }),
        };
      }
      return await initializeLibrary({
        ...libraryOptions,
        ...(terms === undefined ? {} : { terms }),
      });
    }
    if (kind === 'group') {
      const metadata = await group('');
      return target === 'library'
        ? await addLibraryGroup(id, metadata, libraryOptions)
        : await addLocalGroup(id, metadata, projectOptions);
    }
    if (kind === 'rule') {
      const metadata = {
        title: await required('title', 'Action-oriented title'),
        whenToRead: await required(
          'when-to-read',
          'When should an agent read this rule?',
        ),
        impact: await required(
          'impact',
          'Impact (CRITICAL, HIGH, MEDIUM-HIGH, MEDIUM, LOW-MEDIUM, LOW)',
        ),
        impactDescription: await required(
          'impact-description',
          'Consequence the rule prevents',
        ),
      };
      let createGroup = flags.has('create-group');
      const idGroup = ruleGroup(`${id}.md`, 'rule');
      const exists =
        createGroup ||
        (target === 'library'
          ? await hasLibraryGroup(idGroup, directory)
          : await hasLocalRuleGroup(idGroup, configPath));
      if (!exists) {
        if (!stdin.isTTY || !stderr.isTTY || flags.has('non-interactive'))
          throw new UsageError(
            `Missing group ${idGroup}. Run ${target === 'library' ? 'library' : 'local'} add group ${idGroup} or supply --create-group and group metadata.`,
          );
        createGroup = /^(y|yes)$/iu.test(
          await ask(`Create missing group ${idGroup}? [y/N]`),
        );
        if (!createGroup)
          throw new UsageError('Cancelled; no rule was created.');
      }
      const bodyPath = flags.get('body-file')?.[0];
      const options = {
        ...(target === 'library' ? libraryOptions : projectOptions),
        ...(createGroup ? { group: await group('group-') } : {}),
        ...(bodyPath === undefined
          ? {}
          : { body: textFile(await readFile(bodyPath), bodyPath) }),
      };
      if (signal.aborted) throw new Error('Authoring cancelled.');
      return target === 'library'
        ? await addLibraryRule(id, metadata, options)
        : await addLocalRule(id, metadata, options);
    }
    const repository = await required('repository', 'Git repository URL');
    let revision: { ref: string } | { version: string };
    if (flags.has('ref'))
      revision = { ref: await required('ref', 'Exact tag or commit') };
    else if (flags.has('version'))
      revision = {
        version: await required('version', 'npm version constraint'),
      };
    else {
      const choice = await ask('Revision kind (ref or version):');
      if (choice !== 'ref' && choice !== 'version')
        throw new UsageError('Choose ref or version.');
      revision =
        choice === 'ref'
          ? { ref: await required('ref', 'Exact tag or commit') }
          : { version: await required('version', 'npm version constraint') };
    }
    const selections =
      flags.get('groups') ??
      (
        await required(
          'groups',
          'Groups (comma-separated IDs, *, practices/*, or techs/*)',
        )
      )
        .split(',')
        .map((value) => value.trim());
    const first = selections[0];
    const groups =
      selections.length === 1 &&
      ['*', 'practices/*', 'techs/*'].includes(first ?? '')
        ? first
        : selections;
    if (signal.aborted) throw new Error('Authoring cancelled.');
    return await addSource(
      id,
      { repository, ...revision, groups, exclude: {}, replace: {} },
      projectOptions,
    );
  } finally {
    terminal?.close();
  }
}
