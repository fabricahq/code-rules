/** @fileoverview Rejects mutable external action references before workflow changes merge. */

import { expect, test } from 'bun:test';
import { readdirSync, readFileSync } from 'node:fs';
import { parse } from 'yaml';

// Check both step actions and job-level reusable workflows, while permitting local code.
function expectImmutableReference(reference: string) {
  if (reference.startsWith('./')) return;
  if (reference.startsWith('docker://')) {
    expect(reference).toMatch(/^docker:\/\/.+@sha256:[a-f0-9]{64}$/);
    return;
  }
  expect(reference).toMatch(/^[\w.-]+\/[\w./-]+@[a-f0-9]{40}$/);
}

for (const file of readdirSync('.github/workflows').filter((name) =>
  /\.ya?ml$/.test(name),
)) {
  test(`${file} pins external actions and workflows to immutable revisions`, () => {
    const workflow = parse(readFileSync(`.github/workflows/${file}`, 'utf8'));
    for (const job of Object.values(workflow.jobs) as Array<{
      uses?: string;
      steps?: Array<{ uses?: string }>;
    }>) {
      if (job.uses) expectImmutableReference(job.uses);
      for (const step of job.steps ?? []) {
        if (step.uses) expectImmutableReference(step.uses);
      }
    }
  });
}
