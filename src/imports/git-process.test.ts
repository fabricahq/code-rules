/** @fileoverview Verifies Git subprocess cancellation, output limits, and secret-safe diagnostics. */

import { expect, test } from 'bun:test';
import { tmpdir } from 'node:os';
import { gitProcess } from './git-process';

test('terminates a running Git alias and its child on timeout', async () => {
  await expect(
    gitProcess(
      tmpdir(),
      ['-c', 'alias.pause=!sleep 30', 'pause'],
      AbortSignal.timeout(100),
      4096,
    ),
  ).rejects.toMatchObject({ code: 'timed-out' });
});

test('terminates a running Git process on caller cancellation', async () => {
  const controller = new AbortController();
  const timer = setTimeout(
    () => controller.abort('caller-private-reason'),
    100,
  );
  try {
    await expect(
      gitProcess(
        tmpdir(),
        ['-c', 'alias.pause=!sleep 30', 'pause'],
        controller.signal,
        4096,
      ),
    ).rejects.toMatchObject({
      code: 'cancelled',
      message: expect.not.stringContaining('private-reason'),
    });
  } finally {
    clearTimeout(timer);
  }
});

test('bounds captured Git output', async () => {
  await expect(
    gitProcess(tmpdir(), ['--version'], AbortSignal.timeout(2000), 1),
  ).rejects.toMatchObject({ code: 'limit-exceeded' });
});

test('discards diagnostic text that could expose a rewritten URL credential', async () => {
  const result = await gitProcess(
    tmpdir(),
    [
      '-c',
      'alias.fail=!printf "https://user:secret@example.invalid" >&2; exit 1',
      'fail',
    ],
    AbortSignal.timeout(2000),
    4096,
  );
  expect(result.status).not.toBe(0);
  expect(result.output.toString()).toBe('');
});
