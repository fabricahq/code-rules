/** @fileoverview Injects one publication failure in a child CLI, leaving other filesystem operations real. */

import { mock } from 'bun:test';
import * as fs from 'node:fs/promises';
let calls = 0;
const original = fs.link;
mock.module('node:fs/promises', () => ({
  ...fs,
  link: async (existing: string, target: string): Promise<void> => {
    if (++calls === 2) throw new Error('Simulated publication failure');
    await original(existing, target);
  },
}));
const { main } = await import('../../src/cli');
process.exitCode = await main(process.argv.slice(2));
