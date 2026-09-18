/**
 * @fileoverview Checks every YAML file in .github/workflows for dependency pins and disabled checkout credential persistence.
 * Fixed commits and image digests prevent upstream tag changes from silently changing CI code.
 * These configuration checks do not verify who published the code or whether it is safe.
 */

import { expect, test } from 'bun:test';
import { readdirSync, readFileSync } from 'node:fs';
import { parse } from 'yaml';

// Local actions use the checked-out source. External actions require a full commit SHA,
// and Docker images require a SHA-256 digest; tags such as @v3 or :latest fail.
function expectImmutableReference(reference: string) {
  if (reference.startsWith('./')) return;
  if (reference.startsWith('docker://')) {
    expect(reference).toMatch(/^docker:\/\/.+@sha256:[a-f0-9]{64}$/);
    return;
  }
  expect(reference).toMatch(/^[\w.-]+\/[\w./-]+@[a-f0-9]{40}$/);
}

// Discover workflows automatically so newly added files inherit the same policy.
for (const file of readdirSync('.github/workflows').filter((name) =>
  /\.ya?ml$/.test(name),
)) {
  test(`${file} pins external references and disables checkout credential persistence`, () => {
    const workflow = parse(readFileSync(`.github/workflows/${file}`, 'utf8'));
    for (const job of Object.values(workflow.jobs) as Array<{
      uses?: string;
      steps?: Array<{
        uses?: string;
        with?: { 'persist-credentials'?: boolean };
      }>;
    }>) {
      if (job.uses) expectImmutableReference(job.uses);
      for (const step of job.steps ?? []) {
        if (step.uses) expectImmutableReference(step.uses);
        // PR code and build tools do not need the token after checkout finishes.
        if (step.uses?.startsWith('actions/checkout@')) {
          expect(step.with?.['persist-credentials']).toBe(false);
        }
      }
    }
  });
}
