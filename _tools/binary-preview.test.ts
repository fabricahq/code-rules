/** @fileoverview Executes the preview workflow's API script against GitHub response fixtures. */

import { expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';
import { parse } from 'yaml';

const workflow = parse(
  readFileSync('.github/workflows/comment-binary-preview.yml', 'utf8'),
);
const script = workflow.jobs.comment.steps[0].with.script;
const marker = '<!-- code-rules-binary-preview -->';

// Exercise the shipped script without credentials or external writes.
async function preview(
  options: {
    event?: string;
    conclusion?: string;
    path?: string;
    artifacts?: object[];
    state?: string;
    head?: string;
    foreign?: boolean;
    previous?: { id: number; user: { login: string }; body: string };
  } = {},
) {
  const sha = 'a'.repeat(40);
  const writes: Array<Record<string, unknown>> = [];
  const warnings: string[] = [];
  const pr = {
    number: 38,
    state: options.state ?? 'open',
    head: { sha: options.head ?? sha, repo: { id: 99 } },
    base: { repo: { id: options.foreign ? 2 : 1 } },
  };
  const artifact = {
    id: 70,
    name: 'unpublished-native-candidates',
    expired: false,
    size_in_bytes: 100,
    expires_at: '2026-09-24T12:00:00Z',
  };
  const responses: Record<string, object[]> = {
    artifacts: options.artifacts ?? [artifact],
    pulls: [pr],
    comments: options.previous ? [options.previous] : [],
  };
  const github = {
    paginate: async (route: string) => responses[route],
    rest: {
      actions: { listWorkflowRunArtifacts: 'artifacts' },
      repos: { listPullRequestsAssociatedWithCommit: 'pulls' },
      pulls: { get: async () => ({ data: pr }) },
      issues: {
        listComments: 'comments',
        createComment: async (args: Record<string, unknown>) => {
          writes.push({ method: 'create', ...args });
        },
        updateComment: async (args: Record<string, unknown>) => {
          writes.push({ method: 'update', ...args });
        },
      },
    },
  };
  await runInNewContext(`(async () => { ${script}\n })()`, {
    github,
    core: { warning: (message: string) => warnings.push(message) },
    context: {
      repo: { owner: 'fabricahq', repo: 'code-rules' },
      serverUrl: 'https://github.com',
      payload: {
        repository: { id: 1 },
        workflow_run: {
          id: 500,
          run_attempt: 2,
          head_sha: sha,
          event: options.event ?? 'pull_request',
          conclusion: options.conclusion ?? 'success',
          path: options.path ?? '.github/workflows/package-binaries.yml',
          pull_requests: [],
        },
      },
    },
  });
  return { writes, warnings };
}

test('posts downloadable candidates for an open fork PR even without event PR metadata', async () => {
  const { writes } = await preview();
  expect(writes).toHaveLength(1);
  expect(writes[0]).toMatchObject({ method: 'create', issue_number: 38 });
  expect(writes[0]!.body).toContain(
    'https://github.com/fabricahq/code-rules/actions/runs/500/artifacts/70',
  );
  expect(writes[0]!.body).toContain('expire 2026-09-24T12:00:00Z');
  expect(writes[0]!.body).toContain('unpublished review builds');
});

test('updates its existing bot comment instead of posting another', async () => {
  const { writes } = await preview({
    previous: {
      id: 8,
      user: { login: 'github-actions[bot]' },
      body: `${marker}\n<!-- run:499 attempt:1 -->`,
    },
  });
  expect(writes).toHaveLength(1);
  expect(writes[0]).toMatchObject({ method: 'update', comment_id: 8 });
});

test('does not edit a human comment containing the preview marker', async () => {
  const { writes } = await preview({
    previous: { id: 8, user: { login: 'reviewer' }, body: marker },
  });
  expect(writes[0]!.method).toBe('create');
});

for (const options of [
  { event: 'workflow_dispatch' },
  { conclusion: 'failure' },
  { path: '.github/workflows/another.yml' },
  { state: 'closed' },
  { head: 'b'.repeat(40) },
  { foreign: true },
  { artifacts: [] },
  {
    artifacts: [
      {
        name: 'unpublished-native-candidates',
        expired: true,
        size_in_bytes: 100,
      },
    ],
  },
  {
    artifacts: [
      {
        name: 'unpublished-native-candidates',
        expired: false,
        size_in_bytes: 0,
      },
    ],
  },
  { artifacts: [{ name: 'other', expired: false, size_in_bytes: 100 }] },
]) {
  test(`does not post an unavailable or inapplicable preview: ${JSON.stringify(options)}`, async () => {
    expect((await preview(options)).writes).toEqual([]);
  });
}

for (const stamp of ['run:501 attempt:1', 'run:500 attempt:3']) {
  test(`does not replace newer preview ${stamp}`, async () => {
    const { writes } = await preview({
      previous: {
        id: 8,
        user: { login: 'github-actions[bot]' },
        body: `${marker}\n<!-- ${stamp} -->`,
      },
    });
    expect(writes).toEqual([]);
  });
}

test('keeps builds read-only and limits the notification to API access', () => {
  const packaging = parse(
    readFileSync('.github/workflows/package-binaries.yml', 'utf8'),
  );
  expect(packaging.permissions).toEqual({ contents: 'read' });
  expect(packaging.jobs.install.steps[0].with.ref).toBe(
    '${{ github.event.pull_request.head.sha || github.sha }}',
  );
  expect(workflow.on.workflow_run).toEqual({
    workflows: [packaging.name],
    types: ['completed'],
  });
  expect(workflow.permissions).toEqual({
    actions: 'read',
    contents: 'read',
    'pull-requests': 'write',
  });
  expect(workflow.jobs.comment.steps).toHaveLength(1);
  expect(workflow.jobs.comment.steps[0].uses).toBe(
    'actions/github-script@v9.0.0',
  );
  expect(workflow.jobs.comment.steps[0].run).toBeUndefined();
});
