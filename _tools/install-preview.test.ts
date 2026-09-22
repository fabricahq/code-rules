/** @fileoverview Runs the preview installer directly with controlled platforms and downloads. */
import { expect, test } from 'bun:test';
import { execFileSync, spawnSync } from 'node:child_process';
import {
  mkdtempSync,
  readdirSync,
  statSync,
  symlinkSync,
  mkdirSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { fakeGitHubCLI } from './fake-github-cli';

test('installs the matching executable and preserves the old file on failure', () => {
  const executable = '#!/bin/sh\nprintf "preview works\\n"\n';
  for (const shell of ['sh', 'bash']) {
    for (const [platform, artifact] of [
      ['Darwin arm64', '70'],
      ['Darwin x86_64', '71'],
      ['Linux aarch64', '72'],
      ['Linux x86_64', '73'],
    ]) {
      for (const mode of [
        'success',
        'fresh',
        'partial',
        'empty',
        'directory',
        'symlink',
        'directory-symlink',
        'unsupported',
      ]) {
        const directory = mkdtempSync(join(tmpdir(), 'preview install '));
        try {
          const bin = join(directory, 'bin');
          mkdirSync(bin);
          writeFileSync(
            join(bin, 'uname'),
            '#!/bin/sh\nprintf "%s\\n" "$FIXTURE_PLATFORM"\n',
            { mode: 0o755 },
          );
          const binaryRequest = [
            'api',
            '--hostname',
            'github.com',
            `/repos/fabricahq/code-rules/actions/artifacts/${artifact}/zip`,
          ];
          // The installer remains real; gh returns the selected download outcome without network access.
          const ghCalls = fakeGitHubCLI(bin, [
            {
              args: binaryRequest,
              stdout:
                mode === 'partial'
                  ? 'partial'
                  : mode === 'empty'
                    ? ''
                    : executable,
              exitCode: mode === 'partial' ? 1 : 0,
            },
          ]);
          const destination = join(directory, 'code-rules');
          const other = join(directory, 'other');
          if (mode === 'directory') mkdirSync(destination);
          else if (mode === 'directory-symlink') {
            mkdirSync(other);
            symlinkSync(other, destination);
          } else if (mode === 'symlink') {
            writeFileSync(other, 'keep symlink target');
            symlinkSync(other, destination);
          } else if (mode !== 'fresh')
            writeFileSync(destination, 'old executable');
          const result = spawnSync(
            shell,
            [
              fileURLToPath(new URL('./install-preview.sh', import.meta.url)),
              'github.com',
              'fabricahq/code-rules',
              '70',
              '71',
              '72',
              '73',
            ],
            {
              cwd: directory,
              env: {
                ...process.env,
                PATH: `${bin}:${process.env.PATH}`,
                FIXTURE_PLATFORM:
                  mode === 'unsupported' ? 'Linux riscv64' : platform,
              },
              encoding: 'utf8',
            },
          );
          if (['success', 'fresh', 'symlink'].includes(mode)) {
            expect(result.status).toBe(0);
            expect(result.stdout).toBe('Downloaded and wrote ./code-rules\n');
            expect(readFileSync(destination, 'utf8')).toBe(executable);
            expect(statSync(destination).mode & 0o100).toBe(0o100);
            expect(execFileSync(destination, { encoding: 'utf8' })).toBe(
              'preview works\n',
            );
            if (mode === 'symlink')
              expect(readFileSync(other, 'utf8')).toBe('keep symlink target');
          } else {
            expect(result.status).not.toBe(0);
            expect(result.stdout).toBe('');
            if (mode === 'directory' || mode === 'directory-symlink') {
              expect(readdirSync(destination)).toEqual([]);
              expect(result.stderr).toMatch(
                /Cannot install: \/.*\/code-rules is an existing folder\./,
              );
              expect(result.stderr).toContain(
                'Nothing was changed. Run this command from a different directory.',
              );
            } else
              expect(readFileSync(destination, 'utf8')).toBe('old executable');
          }
          expect(ghCalls()).toEqual(
            ['directory', 'directory-symlink', 'unsupported'].includes(mode)
              ? []
              : [binaryRequest],
          );
          expect(
            readdirSync(directory).filter((name) =>
              name.startsWith('.code-rules.'),
            ),
          ).toEqual([]);
        } finally {
          rmSync(directory, { recursive: true, force: true });
        }
      }
    }
  }
}, 30_000);
