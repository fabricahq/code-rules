/** @fileoverview Exercises the release guard against isolated package metadata and license files as CI would invoke it. */

import { test, expect } from 'bun:test';
import {
  mkdtemp,
  mkdir,
  copyFile,
  writeFile,
  symlink,
  rm,
} from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

for (const scenario of [
  'missing-license',
  'valid',
  'wrong-tag',
  'wrong-channel',
] as const) {
  test(`release guard reports ${scenario} against the staged package`, async () => {
    const root = await mkdtemp(join(tmpdir(), 'code-rules-release-guard-'));
    try {
      await mkdir(join(root, '_tools'));
      await copyFile(
        new URL('./release-check.ts', import.meta.url),
        join(root, '_tools/release-check.ts'),
      );
      await symlink(
        resolve(import.meta.dir, '../node_modules'),
        join(root, 'node_modules'),
      );
      await writeFile(
        join(root, 'package.json'),
        JSON.stringify({
          type: 'module',
          version: '1.0.0',
          license: 'LicenseRef-Test',
        }),
      );
      if (scenario !== 'missing-license')
        await writeFile(join(root, 'LICENSE.md'), 'Original test terms.');
      const result = spawnSync(process.execPath, ['_tools/release-check.ts'], {
        cwd: root,
        encoding: 'utf8',
        env: {
          ...process.env,
          RELEASE_TAG: scenario === 'wrong-tag' ? 'v2.0.0' : 'v1.0.0',
          RELEASE_PRERELEASE: scenario === 'wrong-channel' ? 'true' : 'false',
        },
      });
      expect(result.status).toBe(scenario === 'valid' ? 0 : 1);
      if (scenario === 'missing-license')
        expect(result.stderr).toContain('Declare the approved tool license');
      if (scenario === 'wrong-tag')
        expect(result.stderr).toContain('exactly match');
      if (scenario === 'wrong-channel')
        expect(result.stderr).toContain('prerelease status');
    } finally {
      await rm(root, { recursive: true, force: true });
    }
  });
}
