/** @fileoverview Runs project setup, authoring, sync, build, and check through the development CLI. */

import { sync } from './sync';
import { runAuthoring, authoringHelp, UsageError } from './authoring/cli';
import { buildProject, checkProject } from './project';

/** Execute one command and return its exit status; check reports stale files without modifying them. */
export async function main(args: readonly string[]): Promise<number> {
  const [command, ...rest] = args;
  if (command === '--help' || command === '-h') {
    console.log(
      'Usage: bun src/cli.ts <command> [options]\n  sync | build | check [--config path/to/config.json]\n' +
        authoringHelp,
    );
    return 0;
  }
  const controller = new AbortController();
  const cancel = (): void => controller.abort();
  process.once('SIGINT', cancel);
  process.once('SIGTERM', cancel);
  try {
    const authored = await runAuthoring(args, controller.signal);
    if (authored !== null) {
      console.log(JSON.stringify(authored, null, 2));
      return 0;
    }
    if (
      !['sync', 'build', 'check'].includes(command ?? '') ||
      (rest.length !== 0 &&
        (rest.length !== 2 || rest[0] !== '--config' || !rest[1]))
    ) {
      console.error(
        'Usage: bun src/cli.ts <sync|build|check> [--config path/to/config.json]',
      );
      return 2;
    }
    const options = {
      signal: controller.signal,
      ...(rest[1] === undefined ? {} : { configPath: rest[1] }),
    };
    const result = await (command === 'sync'
      ? sync(options)
      : command === 'build'
        ? buildProject(options)
        : checkProject(options));
    console.log(JSON.stringify(result, null, 2));
    return command === 'check' &&
      Object.values(result).some((paths) => paths.length > 0)
      ? 1
      : 0;
  } catch (error) {
    console.error(
      error instanceof Error ? error.message : 'Project operation failed.',
    );
    return error instanceof UsageError ? 2 : 1;
  } finally {
    process.removeListener('SIGINT', cancel);
    process.removeListener('SIGTERM', cancel);
  }
}

if (import.meta.main) process.exitCode = await main(process.argv.slice(2));
