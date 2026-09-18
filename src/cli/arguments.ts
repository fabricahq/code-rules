/** @fileoverview Resolves CLI commands and validates their arguments without reading or changing project files. */

import { commands, commandGroups } from './commands';
import type { Command } from './commands';

/** Invalid command syntax or missing noninteractive input, reported with contextual help and exit 2. */
export class UsageError extends Error {
  constructor(
    message: string,
    readonly commandPath = '',
  ) {
    super(message);
  }
}

/** A command or navigation group and its remaining, unconsumed arguments. */
export type Selection = {
  readonly path: string;
  readonly command: Command | null;
  readonly rest: readonly string[];
};

/** Resolve the longest command path, rejecting unknown words with the nearest group's help context. */
export function selectCommand(args: readonly string[]): Selection {
  let path = '';
  let offset = 0;
  while (offset < args.length) {
    const word = args[offset];
    if (word === undefined || word.startsWith('-')) break;
    const next = path ? path + ' ' + word : word;
    const command = commands.find((entry) => entry.path === next);
    if (command) return { path: next, command, rest: args.slice(offset + 1) };
    if (!Object.hasOwn(commandGroups, next))
      throw new UsageError('Unknown command "' + word + '".', path);
    path = next;
    offset++;
  }
  return { path, command: null, rest: args.slice(offset) };
}

/** Validated flags and optional positional ID; help requests may omit required inputs. */
export type Invocation = {
  readonly command: Command;
  readonly id: string;
  readonly flags: ReadonlyMap<string, readonly string[]>;
  readonly help: boolean;
};

/** Parse only the selected command's declared options, respecting repeatability and value boundaries. */
export function parseArguments(
  command: Command,
  args: readonly string[],
): Invocation {
  const flags = new Map<string, string[]>();
  let id = '';
  let help = false;
  for (let i = 0; i < args.length; i++) {
    const arg = args[i] ?? '';
    if (arg === '--help' || arg === '-h') {
      help = true;
      continue;
    }
    if (!arg.startsWith('-')) {
      if (command.argument !== null && id === '') {
        id = arg;
        continue;
      }
      throw new UsageError('Unexpected argument "' + arg + '".', command.path);
    }
    const key = arg.slice(2);
    const option = command.options.find(
      (entry) => entry.name === key && arg.startsWith('--'),
    );
    if (!option)
      throw new UsageError('Unknown option ' + arg + '.', command.path);
    if (flags.has(key) && !option.repeat)
      throw new UsageError('Option --' + key + ' cannot repeat.', command.path);
    const value = option.value === null ? 'true' : args[++i];
    if (
      value === undefined ||
      value.startsWith('--') ||
      value === '-h' ||
      value.trim() === ''
    )
      throw new UsageError('Provide a value for --' + key + '.', command.path);
    flags.set(key, [...(flags.get(key) ?? []), value]);
  }
  if (command.argument !== null && id === '' && !help)
    throw new UsageError('Missing <' + command.argument + '>.', command.path);
  return { command, id, flags, help };
}
