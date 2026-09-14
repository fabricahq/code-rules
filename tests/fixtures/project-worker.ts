/** @fileoverview Runs real project operations with child-only Git routing and optional filesystem fault injection. */

import { mock } from 'bun:test';
import * as filesystem from 'node:fs/promises';
import { basename } from 'node:path';

// This worker is test-owned. Faults occur at filesystem boundaries after all other operations use real files.
const fault = process.env.PROJECT_TEST_FAULT;
if (fault) {
  const rename = filesystem.rename;
  mock.module('node:fs/promises', () => ({
    ...filesystem,
    rename: async (from: string, to: string): Promise<void> => {
      if (basename(from) === 'new-generated' && fault === 'rename-error')
        throw new Error('Simulated disk failure');
      await rename(from, to);
      if (
        (basename(from) === 'new-vendor' && fault === 'crash') ||
        (basename(to) === '.code-rules-cleanup' && fault === 'cleanup-crash')
      )
        process.kill(process.pid, 'SIGKILL');
    },
  }));
}
const { main } = await import('../../src/cli');
process.exitCode = await main(process.argv.slice(2));
