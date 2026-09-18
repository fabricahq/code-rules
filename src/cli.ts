#!/usr/bin/env node
/** @fileoverview Runs project and library commands with contextual help, usage errors, and cancellation. */

import { sync } from './sync';
import { toolVersion } from './version';
import { runAuthoring } from './authoring/cli';
import { buildProject, checkProject } from './project';
import { selectCommand, parseArguments, UsageError } from './cli/arguments';
import { renderHelp, helpHint } from './cli/help';

/** Execute a command and return its exit status; help and usage errors never modify project files. */
export async function main(args: readonly string[]): Promise<number> {
  if (args.length === 1 && (args[0] === '--version' || args[0] === '-v')) {
    console.log(toolVersion);
    return 0;
  }
  const controller = new AbortController();
  const cancel = (): void => controller.abort();
  process.once('SIGINT', cancel);
  process.once('SIGTERM', cancel);
  let path = '';
  try {
    const selection = selectCommand(args);
    path = selection.path;
    if (selection.command === null) {
      const unexpected = selection.rest.find(
        (arg) => arg !== '--help' && arg !== '-h',
      );
      if (unexpected !== undefined)
        throw new UsageError('Unknown option ' + unexpected + '.', path);
      console.log(renderHelp(path, null));
      return 0;
    }
    const request = parseArguments(selection.command, selection.rest);
    if (request.help) {
      console.log(renderHelp(path, selection.command));
      return 0;
    }
    const authored = await runAuthoring(request, controller.signal);
    if (authored !== null) {
      console.log(JSON.stringify(authored, null, 2));
      return 0;
    }
    const action = request.command.action;
    if (action.type !== 'project')
      throw new Error('Unsupported command handler.');
    const configPath = request.flags.get('config')?.[0];
    const options = {
      signal: controller.signal,
      toolVersion,
      ...(configPath === undefined ? {} : { configPath }),
    };
    const result = await (action.kind === 'sync'
      ? sync(options)
      : action.kind === 'build'
        ? buildProject(options)
        : checkProject(options));
    console.log(JSON.stringify(result, null, 2));
    return action.kind === 'check' &&
      Object.values(result).some((paths) => paths.length > 0)
      ? 1
      : 0;
  } catch (error) {
    console.error(
      error instanceof Error ? error.message : 'Project operation failed.',
    );
    if (error instanceof UsageError)
      console.error('\n' + helpHint(error.commandPath || path));
    return error instanceof UsageError ? 2 : 1;
  } finally {
    process.removeListener('SIGINT', cancel);
    process.removeListener('SIGTERM', cancel);
  }
}

if (import.meta.main) process.exitCode = await main(process.argv.slice(2));
