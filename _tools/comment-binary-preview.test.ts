/** @fileoverview Executes the preview workflow's API script against GitHub response fixtures. */

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
import { runInNewContext } from 'node:vm';
import { parse } from 'yaml';

const workflow = parse(
  readFileSync('.github/workflows/comment-binary-preview.yml', 'utf8'),
);
const script = workflow.jobs.comment.steps[0].with.script;
const marker = '<!-- code-rules-binary-preview -->';
const targets = ['darwin-arm64', 'darwin-amd64', 'linux-arm64', 'linux-amd64'];
const artifacts = targets.map((target, index) => ({
  id: 70 + index,
  name: `code-rules-${target}`,
  expired: false,
  size_in_bytes: 100,
  expires_at: '2026-09-24T12:00:00Z',
}));

// Exercise the shipped script without credentials or external writes.
async function preview(
  options: {
    sameRepository?: boolean;
    authorPermission?: string;
    trigger?: string;
    ref?: string;
    permission?: string;
    runID?: string;
    approvedCommit?: string;
    runRepository?: number;
    runStatus?: string;
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
    user: { login: 'contributor' },
    state: options.state ?? 'open',
    head: {
      sha: options.head ?? sha,
      repo: { id: options.sameRepository ? 1 : 99 },
    },
    base: { repo: { id: options.foreign ? 2 : 1 } },
  };
  const responses: Record<string, object[]> = {
    artifacts: options.artifacts ?? artifacts,
    pulls: [pr],
    comments: options.previous ? [options.previous] : [],
  };
  const run = {
    id: 500,
    run_attempt: 2,
    head_sha: sha,
    repository: { id: options.runRepository ?? 1 },
    status: options.runStatus ?? 'completed',
    event: options.event ?? 'pull_request',
    conclusion: options.conclusion ?? 'success',
    path: options.path ?? '.github/workflows/package-binaries.yml',
    pull_requests: [],
  };
  const github = {
    paginate: async (route: string) => responses[route],
    rest: {
      actions: {
        listWorkflowRunArtifacts: 'artifacts',
        getWorkflowRun: async (args: { run_id: string }) => {
          expect(args.run_id).toBe('500');
          return { data: run };
        },
      },
      repos: {
        listPullRequestsAssociatedWithCommit: 'pulls',
        getCollaboratorPermissionLevel: async (args: { username: string }) => {
          return {
            data: {
              permission:
                args.username === 'maintainer'
                  ? (options.permission ?? 'write')
                  : (options.authorPermission ?? 'read'),
            },
          };
        },
      },
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
    URL,
    core: {
      warning: (message: string) => warnings.push(message),
      notice: () => {},
    },
    context: {
      repo: { owner: 'fabricahq', repo: 'code-rules' },
      serverUrl: 'https://github.com',
      actor: 'maintainer',
      eventName: options.trigger ?? 'workflow_dispatch',
      ref: options.ref ?? 'refs/heads/main',
      payload: {
        repository: { id: 1, default_branch: 'main' },
        workflow_run: run,
        inputs: {
          run_id: options.runID ?? '500',
          commit: options.approvedCommit ?? sha,
        },
      },
    },
  });
  return { writes, warnings };
}

test('advertises a maintainer-approved commit for an open fork PR', async () => {
  const { writes } = await preview();
  expect(writes).toHaveLength(1);
  expect(writes[0]).toMatchObject({ method: 'create', issue_number: 38 });
  for (const [index, label] of [
    'macOS Apple Silicon',
    'macOS Intel',
    'Linux ARM',
    'Linux Intel/AMD',
  ].entries()) {
    expect(writes[0]!.body).toContain(
      `[${label}](https://github.com/fabricahq/code-rules/actions/runs/500/artifacts/${70 + index})`,
    );
  }
  expect(writes[0]!.body).toContain('### CLI preview ready');
  expect(writes[0]!.body).toContain(
    'Packaging and installation checks passed.',
  );
  expect(writes[0]!.body).toContain(
    '**Warning: Runs code from this PR. For isolated testing only; not an official release.**',
  );
  expect(writes[0]!.body).toContain('**Manual testing is optional.**');
  expect(writes[0]!.body).toContain('gh auth login');
  expect(writes[0]!.body).toContain('Then run `./code-rules --help`.');
  expect(writes[0]!.body).toContain('It replaces any existing `./code-rules`');
  expect(writes[0]!.body).toContain(
    `[\`aaaaaaa\`](https://github.com/fabricahq/code-rules/commit/${'a'.repeat(40)})`,
  );
  expect(writes[0]!.body).toMatch(
    /Downloads expire Sep 24, 2026(?:,| at) 12:00\sPM UTC\./,
  );
  expect(writes[0]!.body).toContain('Unreleased preview');
  expect(writes[0]!.body).toContain(`/blob/${'a'.repeat(40)}/LICENSE.md`);
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
  { state: 'closed' },
  { head: 'b'.repeat(40) },
  { foreign: true },
  { artifacts: [] },
  {
    artifacts: [
      {
        name: 'unpublished-native-candidates',
        expired: false,
        size_in_bytes: 100,
      },
    ],
  },
]) {
  test(`does not post an unavailable or inapplicable preview: ${JSON.stringify(options)}`, async () => {
    expect((await preview(options)).writes).toEqual([]);
  });
}

for (const target of targets) {
  test(`does not advertise incomplete or unavailable ${target} downloads`, async () => {
    const selected = artifacts.find((a) => a.name === `code-rules-${target}`)!;
    const others = artifacts.filter((a) => a !== selected);
    for (const invalid of [
      others,
      [...others, selected, selected],
      [...others, { ...selected, expired: true }],
      [...others, { ...selected, size_in_bytes: 0 }],
      [...others, { ...selected, expires_at: 'invalid' }],
    ]) {
      const result = await preview({ artifacts: invalid });
      expect(result.writes).toEqual([]);
      expect(result.warnings).toHaveLength(1);
    }
  });
}

test('uses the earliest expiry and ignores unrelated artifacts', async () => {
  const result = await preview({
    artifacts: [
      ...artifacts.slice(0, 3),
      { ...artifacts[3], expires_at: '2026-09-23T23:30:00Z' },
      { name: 'unrelated', expired: true },
    ],
  });
  expect(result.writes).toHaveLength(1);
  expect(result.writes[0]!.body).toMatch(
    /Downloads expire Sep 23, 2026(?:,| at) 11:30\sPM UTC\./,
  );
});

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
  expect(workflow.on.workflow_dispatch.inputs.commit.required).toBe(true);
  expect(workflow.on.workflow_dispatch.inputs.run_id.required).toBe(true);
  expect(workflow.permissions).toEqual({
    actions: 'read',
    contents: 'read',
    'pull-requests': 'write',
  });
  expect(workflow.jobs.comment.steps).toHaveLength(1);
  expect(workflow.jobs.comment.steps[0].uses).toMatch(
    /^actions\/github-script@[a-f0-9]{40}$/,
  );
  expect(workflow.jobs.comment.steps[0].run).toBeUndefined();
});

test('uploads four unarchived executables with seven-day retention', () => {
  const packaging = parse(
    readFileSync('.github/workflows/package-binaries.yml', 'utf8'),
  );
  const uploads = packaging.jobs.install.steps.filter(
    (step: { uses?: string }) =>
      step.uses?.startsWith('actions/upload-artifact@'),
  );
  expect(
    uploads.map((step: { with: { path: string } }) => step.with.path).sort(),
  ).toEqual(
    targets.map((target) => `native-previews/code-rules-${target}`).sort(),
  );
  for (const step of uploads) {
    expect(step.with.archive).toBe(false);
    expect(step.with['retention-days']).toBe(7);
    expect(step.with['if-no-files-found']).toBe('error');
  }
});

test('extracts exact executable bytes from each packaged target and refuses a missing archive', () => {
  const packaging = parse(
    readFileSync('.github/workflows/package-binaries.yml', 'utf8'),
  );
  const extract = packaging.jobs.install.steps.find(
    (step: { name?: string }) => step.name === 'Extract preview executables',
  ).run;
  const directory = mkdtempSync(join(tmpdir(), 'code-rules-preview-'));
  try {
    mkdirSync(join(directory, 'native-artifacts'));
    const expected = new Map<string, Buffer<ArrayBuffer>>();
    for (const target of targets) {
      const bytes = Buffer.from(`executable for ${target}\0\xff`, 'latin1');
      expected.set(target, bytes);
      writeFileSync(join(directory, 'code-rules'), bytes);
      writeFileSync(join(directory, 'LICENSE.md'), 'Fixture license');
      execFileSync(
        'tar',
        [
          '-czf',
          `native-artifacts/code-rules_candidate_${target.replaceAll('-', '_')}.tar.gz`,
          'code-rules',
          'LICENSE.md',
        ],
        { cwd: directory },
      );
    }
    execFileSync('bash', ['-e', '-o', 'pipefail', '-c', extract], {
      cwd: directory,
    });
    for (const [target, bytes] of expected) {
      expect(
        readFileSync(join(directory, `native-previews/code-rules-${target}`)),
      ).toEqual(bytes);
    }
    rmSync(join(directory, 'native-previews'), { recursive: true });
    rmSync(
      join(
        directory,
        'native-artifacts/code-rules_candidate_linux_arm64.tar.gz',
      ),
    );
    expect(() =>
      execFileSync('bash', ['-e', '-o', 'pipefail', '-c', extract], {
        cwd: directory,
        stdio: 'pipe',
      }),
    ).toThrow();
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});

for (const options of [
  { trigger: 'pull_request_target' },
  { ref: 'refs/heads/contributor-branch' },
  { permission: 'read' },
  { permission: 'none' },
  { runID: '500; echo unsafe' },
  { approvedCommit: 'aaaaaaa' },
  { approvedCommit: 'b'.repeat(40) },
  { runRepository: 2 },
  { runStatus: 'in_progress' },
  { event: 'workflow_dispatch' },
  { conclusion: 'failure' },
  { path: '.github/workflows/another.yml' },
]) {
  test(`refuses an unauthorized or mismatched approval: ${JSON.stringify(options)}`, async () => {
    await expect(preview(options)).rejects.toThrow();
  });
}

test('automatically advertises same-repository PRs opened by current maintainers', async () => {
  for (const authorPermission of ['write', 'admin']) {
    const result = await preview({
      trigger: 'workflow_run',
      sameRepository: true,
      authorPermission,
    });
    expect(result.writes).toHaveLength(1);
  }
});

for (const options of [
  { sameRepository: false, authorPermission: 'write' },
  { sameRepository: true, authorPermission: 'read' },
  { sameRepository: true, authorPermission: 'none' },
  { sameRepository: true, authorPermission: 'write', head: 'b'.repeat(40) },
  { sameRepository: true, authorPermission: 'write', conclusion: 'failure' },
]) {
  test(`withholds automatic links without a current trusted PR: ${JSON.stringify(options)}`, async () => {
    expect(
      (await preview({ trigger: 'workflow_run', ...options })).writes,
    ).toEqual([]);
  });
}

// Execute the exact copyable command with controlled platform and download responses.
test('installs the matching executable and preserves the old file on failure', async () => {
  const { writes } = await preview();
  const command = String(writes[0]!.body).match(/```sh\n([^\n]+)\n```/)?.[1];
  expect(command).toBeDefined();
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
          writeFileSync(
            join(bin, 'gh'),
            `#!/bin/sh
set -eu
[ "$#" -eq 4 ]
[ "$1" = api ]
[ "$2" = --hostname ]
[ "$3" = github.com ]
[ "$4" = "/repos/fabricahq/code-rules/actions/artifacts/$FIXTURE_ARTIFACT/zip" ]
case "$FIXTURE_MODE" in
  partial) printf partial; exit 1 ;;
  empty) exit 0 ;;
esac
cat "$FIXTURE_PAYLOAD"
`,
            { mode: 0o755 },
          );
          writeFileSync(join(directory, 'payload'), executable);
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
          const result = spawnSync(shell, ['-c', command!], {
            cwd: directory,
            env: {
              ...process.env,
              PATH: `${bin}:${process.env.PATH}`,
              FIXTURE_PLATFORM:
                mode === 'unsupported' ? 'Linux riscv64' : platform,
              FIXTURE_ARTIFACT: artifact,
              FIXTURE_MODE: mode,
              FIXTURE_PAYLOAD: join(directory, 'payload'),
            },
            encoding: 'utf8',
          });
          if (['success', 'fresh', 'symlink'].includes(mode)) {
            expect(result.status).toBe(0);
            expect(readFileSync(destination, 'utf8')).toBe(executable);
            expect(statSync(destination).mode & 0o100).toBe(0o100);
            expect(execFileSync(destination, { encoding: 'utf8' })).toBe(
              'preview works\n',
            );
            if (mode === 'symlink')
              expect(readFileSync(other, 'utf8')).toBe('keep symlink target');
          } else {
            expect(result.status).not.toBe(0);
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
