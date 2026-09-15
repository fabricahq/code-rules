/** @fileoverview Runs the development sync, build, and check commands; publishing and authoring commands are separate work. */

import { sync } from './sync';
import { buildProject, checkProject } from './project';

/** Execute one command and return its exit status; check reports stale files without modifying them. */
export async function main(args: readonly string[]): Promise<number> {
  const [command, ...rest] = args;
  if (command === '--help' || command === '-h') {
    console.log(
      'Usage: bun src/cli.ts <sync|build|check> [--config path/to/config.json]',
    );
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
  const controller = new AbortController();
  const cancel = (): void => controller.abort();
  process.once('SIGINT', cancel);
  process.once('SIGTERM', cancel);
  try {
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
    return 1;
  } finally {
    process.removeListener('SIGINT', cancel);
    process.removeListener('SIGTERM', cancel);
  }
}

if (import.meta.main) process.exitCode = await main(process.argv.slice(2));
