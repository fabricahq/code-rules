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

/** Invalid command syntax or missing noninteractive author input; the CLI reports exit status 2. */
export class UsageError extends Error {}

/** Development CLI syntax, including explicit equivalents for every interactive question. */
export const authoringHelp = `
  init [--config path]
  local add group <group-id> --name text --description text --when-to-read text
  local add rule <rule-id> --title text --when-to-read text --impact level --impact-description text [--body-file path]
  add source <alias> --repository url (--ref tag-or-sha | --version range) --groups selector

All authoring commands accept --config path and --non-interactive.
Group --when-to-read and source --groups can repeat. Wildcards *, practices/*, and techs/* must be used alone.
Rule IDs omit .md. Rules use existing local or imported groups. To create a missing group with a rule, pass:
  --create-group --group-name text --group-description text --group-when-to-read text
On a terminal, omitted required inputs are prompted. Draft rules must be completed before build.
Authoring does not fetch sources or regenerate output; run build after local edits or sync after adding a source.
`;

type Kind = 'init' | 'group' | 'rule' | 'source';
type Invocation = {
  kind: Kind;
  id: string;
  flags: ReadonlyMap<string, readonly string[]>;
};
const booleanFlags = ['non-interactive', 'create-group'];

function invocation(args: readonly string[]): Invocation | null {
  let kind: Kind;
  let offset: number;
  if (args[0] === 'init') {
    kind = 'init';
    offset = 1;
  } else if (args[0] === 'local') {
    if (args[1] !== 'add' || (args[2] !== 'group' && args[2] !== 'rule'))
      throw new UsageError('Use local add group <id> or local add rule <id>.');
    kind = args[2];
    offset = 4;
  } else if (args[0] === 'add') {
    if (args[1] !== 'source') throw new UsageError('Use add source <alias>.');
    kind = 'source';
    offset = 3;
  } else return null;
  const id = kind === 'init' ? '' : args[offset - 1];
  if (id === undefined || id.startsWith('--'))
    throw new UsageError(`Missing ${kind} ID.`);
  const allowed = [
    'config',
    'non-interactive',
    ...(kind === 'group'
      ? ['name', 'description', 'when-to-read']
      : kind === 'rule'
        ? [
            'title',
            'when-to-read',
            'impact',
            'impact-description',
            'body-file',
            'create-group',
            'group-name',
            'group-description',
            'group-when-to-read',
          ]
        : kind === 'source'
          ? ['repository', 'ref', 'version', 'groups']
          : []),
  ];
  const repeat =
    kind === 'group'
      ? ['when-to-read']
      : kind === 'rule'
        ? ['group-when-to-read']
        : ['groups'];
  const flags = new Map<string, string[]>();
  for (let i = offset; i < args.length; i++) {
    const arg = args[i] ?? '';
    const key = arg.slice(2);
    if (!arg.startsWith('--') || !allowed.includes(key))
      throw new UsageError(`Unknown option ${arg}.`);
    if (flags.has(key) && !repeat.includes(key))
      throw new UsageError(`Option --${key} cannot repeat.`);
    const value = booleanFlags.includes(key) ? 'true' : args[++i];
    if (value === undefined || value.startsWith('--') || value.trim() === '')
      throw new UsageError(`Provide a value for --${key}.`);
    flags.set(key, [...(flags.get(key) ?? []), value]);
  }
  return { kind, id, flags };
}

/** Run an authoring command, or return null for other CLI commands. Prompt only on a terminal and never hold a writer lock while prompting. */
export async function runAuthoring(
  args: readonly string[],
  signal: AbortSignal,
): Promise<AuthoringResult | null> {
  const request = invocation(args);
  if (request === null) return null;
  const { kind, id, flags } = request;
  const configPath = flags.get('config')?.[0];
  const projectOptions = {
    signal,
    ...(configPath === undefined ? {} : { configPath }),
  };
  if (kind === 'init') return initializeProject(projectOptions);
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
    if (kind === 'group')
      return await addLocalGroup(id, await group(''), projectOptions);
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
      if (!createGroup && !(await hasLocalRuleGroup(idGroup, configPath))) {
        if (!stdin.isTTY || !stderr.isTTY || flags.has('non-interactive'))
          throw new UsageError(
            `Missing group ${idGroup}. Run local add group ${idGroup} or supply --create-group and group metadata.`,
          );
        createGroup = /^(y|yes)$/iu.test(
          await ask(`Create missing group ${idGroup}? [y/N]`),
        );
        if (!createGroup)
          throw new UsageError('Cancelled; no rule was created.');
      }
      const bodyPath = flags.get('body-file')?.[0];
      const options = {
        ...projectOptions,
        ...(createGroup ? { group: await group('group-') } : {}),
        ...(bodyPath === undefined
          ? {}
          : { body: textFile(await readFile(bodyPath), bodyPath) }),
      };
      if (signal.aborted) throw new Error('Authoring cancelled.');
      return await addLocalRule(id, metadata, options);
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
