/** @fileoverview Injects editor saves and cancellation at real CLI filesystem boundaries. */

import { mock } from 'bun:test';
import * as fs from 'node:fs/promises';
import { basename } from 'node:path';
const original = { ...fs };
const mode = process.env.AUTHORING_FAULT;
let fired = false;
let links = 0;
const edited = '{"schemaVersion":1,"sources":{},"editor":"saved"}\n';
mock.module('node:fs/promises', () => ({
  ...fs,
  rename: async (from: string, to: string): Promise<void> => {
    if (
      !fired &&
      mode === 'config-edit' &&
      (basename(from) === 'config.json' || basename(to) === 'config.json')
    ) {
      fired = true;
      await original.writeFile(
        basename(from) === 'config.json' ? from : to,
        edited,
      );
    }
    if (
      !fired &&
      mode === 'rollback-edit' &&
      basename(from) === '_group.json'
    ) {
      fired = true;
      await original.writeFile(from, 'Editor saved this group.');
    }
    await original.rename(from, to);
    if (
      !fired &&
      mode === 'rollback-recreated' &&
      basename(from) === '_group.json'
    ) {
      fired = true;
      await original.writeFile(from, 'Editor saved this group.');
    }
    if (
      !fired &&
      mode === 'cancel-after-claim' &&
      basename(from) === 'config.json'
    ) {
      fired = true;
      process.emit('SIGTERM');
    }
    if (
      !fired &&
      mode === 'config-recreated' &&
      basename(from) === 'config.json'
    ) {
      fired = true;
      await original.writeFile(from, edited);
    }
  },
  link: async (from: string, to: string): Promise<void> => {
    if (
      ['rollback-edit', 'rollback-recreated'].includes(mode ?? '') &&
      ++links === 2
    )
      throw new Error('Publication failed');
    await original.link(from, to);
  },
  unlink: async (path: string): Promise<void> => {
    if (
      !fired &&
      mode === 'rollback-edit' &&
      basename(path) === '_group.json'
    ) {
      fired = true;
      await original.writeFile(path, 'Editor saved this group.');
    }
    await original.unlink(path);
  },
  writeFile: async (
    ...args: Parameters<typeof fs.writeFile>
  ): Promise<void> => {
    await original.writeFile(...args);
    if (
      !fired &&
      mode === 'cancel' &&
      String(args[0]).includes('.code-rules-authoring-')
    ) {
      fired = true;
      process.emit('SIGINT');
    }
  },
}));
const { main } = await import('../../src/cli');
process.exitCode = await main(process.argv.slice(2));
