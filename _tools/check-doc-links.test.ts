/** @fileoverview Tests the documentation link checker as a command against isolated built-site fixtures. */

import { expect, test } from 'bun:test';
import {
  mkdtempSync,
  mkdirSync,
  writeFileSync,
  copyFileSync,
  rmSync,
} from 'node:fs';
import { join, dirname } from 'node:path';
import { tmpdir } from 'node:os';
import { spawnSync } from 'node:child_process';

/** Run the real script in an isolated checkout layout, then remove all temporary files. */
function checkSite(pages: Readonly<Record<string, string>>): {
  status: number | null;
  stdout: string;
  stderr: string;
} {
  const workspace = mkdtempSync(join(tmpdir(), 'code-rules-links-'));
  try {
    const script = join(workspace, '_tools', 'check-doc-links.py');
    mkdirSync(dirname(script), { recursive: true });
    copyFileSync(join(import.meta.dir, 'check-doc-links.py'), script);
    for (const [path, content] of Object.entries(pages)) {
      const file = join(workspace, 'docs', 'dist', path);
      mkdirSync(dirname(file), { recursive: true });
      writeFileSync(file, content);
    }
    const result = spawnSync('python3', [script], { encoding: 'utf8' });
    if (result.error !== undefined) throw result.error;
    return {
      status: result.status,
      stdout: result.stdout,
      stderr: result.stderr,
    };
  } finally {
    rmSync(workspace, { recursive: true, force: true });
  }
}

test('should accept page-relative, root-relative, encoded, directory, and same-page links', () => {
  const result = checkSite({
    'index.html':
      '<h1 id="start">Home</h1><a href="#start">self</a><a href="/guide/#section%20one">root</a><a href="guide/">directory</a><a href="a%20file.txt?download=1">asset</a><a href="https://example.com/missing">external</a><a href="//example.com/missing">external</a><a href="mailto:example@example.com">email</a>',
    'guide/index.html':
      '<h1 id="section one">Guide</h1><a href="../#start">up</a><a href="?mode=1#section%20one">self</a>',
    'a file.txt': '',
  });
  expect(result.status).toBe(0);
  expect(result.stdout).toContain(
    'Checked local links and fragments in 2 pages.',
  );
  expect(result.stderr).toBe('');
});

test('should report every broken local link with its source page', () => {
  const result = checkSite({
    'index.html':
      '<a href="missing/">missing</a><a href="#absent">anchor</a><a href="../outside">escape</a>',
  });
  expect(result.status).toBe(1);
  expect(result.stderr).toContain('index.html: missing destination: missing/');
  expect(result.stderr).toContain('index.html: missing anchor: #absent');
  expect(result.stderr).toContain('index.html: outside output: ../outside');
});

test('should fail explicitly when there are no built pages', () => {
  const result = checkSite({});
  expect(result.status).toBe(1);
  expect(result.stderr).toContain(
    'No built pages found. Run bun run docs:build first.',
  );
});
