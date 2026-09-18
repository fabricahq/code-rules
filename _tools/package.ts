/** @fileoverview Builds a Node.js executable and includes the authoring template in the npm package. */

import { mkdir, copyFile, chmod, rm } from 'node:fs/promises';

await rm('dist', { recursive: true, force: true });
await mkdir('dist', { recursive: true });
const result = await Bun.build({
  entrypoints: ['src/cli.ts'],
  outdir: 'dist',
  target: 'node',
  format: 'esm',
  packages: 'external',
  sourcemap: 'external',
});
if (!result.success) throw new AggregateError(result.logs, 'CLI build failed.');
await copyFile('src/authoring/rule-template.md', 'dist/rule-template.md');
await chmod('dist/cli.js', 0o755);
