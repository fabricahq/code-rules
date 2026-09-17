/** @fileoverview Exercises command discovery and usage errors through the real CLI without relying on help snapshots. */

import { test, expect } from 'bun:test';
import { mkdtemp, readdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

const entry = fileURLToPath(new URL('../cli.ts', import.meta.url));

async function invoke(
  args: readonly string[],
): Promise<ReturnType<typeof spawnSync>> {
  const directory = await mkdtemp(join(tmpdir(), 'code-rules-help-'));
  try {
    const result = spawnSync(process.execPath, [entry, ...args], {
      cwd: directory,
      encoding: 'utf8',
      timeout: 10000,
    });
    expect(await readdir(directory)).toEqual([]);
    return result;
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
}

test('top-level help supports discovery without dumping authoring options', async () => {
  const result = await invoke(['--help']);
  expect(result.status).toBe(0);
  expect(result.stderr).toBe('');
  expect(result.stdout).toContain(
    'The package manager for your engineering rules',
  );
  expect(result.stdout).toContain('Project commands:');
  expect(result.stdout).toContain('add source');
  expect(result.stdout).toContain('Library commands:');
  expect(result.stdout).not.toContain('--impact-description');
  expect(result.stdout).not.toContain('--license-file');
});

test('bare command groups show navigation without requiring project setup', async () => {
  for (const args of [[], ['local'], ['library'], ['library', 'add']]) {
    const result = await invoke(args);
    expect(result.status).toBe(0);
    expect(result.stdout).toContain('<command>');
    expect(result.stdout).toContain('--help');
  }
});

test('nested help exposes only applicable options, defaults, and examples', async () => {
  const library = await invoke(['library', 'init', '--help']);
  expect(library.status).toBe(0);
  expect(library.stdout).toContain('code-rules library init [options]');
  expect(library.stdout).toContain('--license-file');
  expect(library.stdout).toContain('default: current directory');
  expect(library.stdout).toContain('without prompts');
  expect(library.stdout).toContain('Examples:');
  expect(library.stdout).not.toContain('--config');
  const local = await invoke(['local', 'add', 'rule', '-h']);
  expect(local.status).toBe(0);
  expect(local.stdout).toContain('--body-file');
  expect(local.stdout).toContain('.code-rules/config.json');
  expect(local.stdout).toContain('--non-interactive');
  expect(local.stdout).not.toContain('--license-file');
});

test.each([
  {
    args: ['library', 'unknown'],
    message: 'Unknown command "unknown"',
    hint: 'code-rules library --help',
  },
  {
    args: ['init', '--directory', 'elsewhere'],
    message: 'Unknown option --directory',
    hint: 'code-rules init --help',
  },
  {
    args: ['library', 'init', '--config', 'elsewhere'],
    message: 'Unknown option --config',
    hint: 'code-rules library init --help',
  },
  {
    args: ['local', 'add', 'rule'],
    message: 'Missing <rule-id>',
    hint: 'code-rules local add rule --help',
  },
  {
    args: ['check', '--config'],
    message: 'Provide a value for --config',
    hint: 'code-rules check --help',
  },
  {
    args: ['check', '--config', 'one', '--config', 'two'],
    message: 'cannot repeat',
    hint: 'code-rules check --help',
  },
  {
    args: ['build', 'extra', '--help'],
    message: 'Unexpected argument "extra"',
    hint: 'code-rules build --help',
  },
])(
  'usage failure points to the relevant help: $args',
  async ({ args, message, hint }) => {
    const result = await invoke(args);
    expect(result.status).toBe(2);
    expect(result.stdout).toBe('');
    expect(result.stderr).toContain(message);
    expect(result.stderr).toContain(hint);
  },
);

test('requesting help bypasses authoring prompts and reads even with nonexistent input paths', async () => {
  const result = await invoke([
    'library',
    'init',
    '--license-file',
    'missing',
    '--help',
  ]);
  expect(result.status).toBe(0);
  expect(result.stdout).toContain('--spdx');
});

test('missing noninteractive metadata identifies the field and the command help', async () => {
  const result = await invoke([
    'local',
    'add',
    'group',
    'practices/testing',
    '--non-interactive',
  ]);
  expect(result.status).toBe(2);
  expect(result.stderr).toContain('--name');
  expect(result.stderr).toContain('code-rules local add group --help');
});

test('help stays readable at 80 columns without color or broken quoted examples', async () => {
  const result = await invoke(['local', 'add', 'rule', '--help']);
  expect(result.status).toBe(0);
  const output = String(result.stdout);
  expect(output).not.toContain('\u001b[');
  expect(
    Math.max(...output.split('\n').map((line) => line.length)),
  ).toBeLessThanOrEqual(80);
  const example = output.split('Examples:\n')[1]?.replace(/\\\n\s+/gu, '');
  expect(example).toContain('--title "Limit retries"');
  expect(example).toContain('--when-to-read "When changing retries"');
});
