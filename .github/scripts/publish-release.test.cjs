/** @fileoverview Exercises the shipped publisher through API fixtures, including interrupted and repeated publication. */

const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const crypto = require('node:crypto');
const { publish } = require('./publish-release.cjs');

function fixture(t, options = {}) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'release-test-'));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  const commit = 'a'.repeat(40);
  const plan = { tag: 'v0.1.0', version: '0.1.0', commit, previous: '', tags: [], notes: 'Approved notes\n', prerelease: false };
  const manifest = { version: plan.version, sourceRevision: commit, candidate: false, sourceDirty: false, license: 'SEE LICENSE.md', artifacts: [] };
  for (const target of ['darwin/amd64', 'darwin/arm64', 'linux/amd64', 'linux/arm64']) {
    const data = Buffer.from(target);
    const file = `code-rules_0.1.0_${target.replace('/', '_')}.tar.gz`;
    fs.writeFileSync(path.join(directory, file), data);
    manifest.artifacts.push({ file, target, bytes: data.length, sha256: crypto.createHash('sha256').update(data).digest('hex') });
  }
  Object.assign(manifest, options.manifest);
  Object.assign(plan, options.plan);
  fs.writeFileSync(path.join(directory, 'release-plan.json'), JSON.stringify(plan));
  fs.writeFileSync(path.join(directory, 'manifest.json'), JSON.stringify(manifest));
  fs.writeFileSync(path.join(directory, 'SHA256SUMS'), manifest.artifacts.map(a => `${a.sha256}  ${a.file}\n`).join(''));
  const writes = [];
  const state = { tag: options.tag, release: null, assets: [] };
  const repos = {
    listReleases: async () => state.release ? [state.release] : [],
    listReleaseAssets: async () => state.assets,
    getReleaseByTag: async () => { if (options.previousMissing) throw new Error('Previous release missing'); return { data: { draft: false } }; },
    getCommit: async () => ({ data: { sha: state.tag } }),
    getRelease: async () => ({ data: options.edited ? { ...state.release, body: 'Changed' } : state.release }),
    createRelease: async (args) => {
      writes.push('draft'); state.release = { ...args, id: 7, html_url: 'https://example.invalid/release' }; return { data: state.release };
    },
    uploadReleaseAsset: async ({ name, data }) => {
      if (options.failUpload) throw new Error('Upload failed');
      writes.push(`upload:${name}`);
      state.assets.push({ name, size: data.length, state: 'uploaded', digest: options.badDigest ? 'wrong' : `sha256:${crypto.createHash('sha256').update(data).digest('hex')}` });
    },
    updateRelease: async (args) => { writes.push('publish'); Object.assign(state.release, args); return { data: state.release }; },
  };
  const github = { paginate: async (fn, args) => fn(args), rest: { repos, git: {
    listMatchingRefs: async () => {
      const names = [...(options.tags || []), ...(state.tag ? [plan.tag] : [])];
      if (options.newTagAfterUpload && state.assets.length === 6) names.push('v0.2.0');
      return names.map(name => ({ ref: `refs/tags/${name}` }));
    },
    getRef: async () => { if (!state.tag) throw Object.assign(new Error('missing'), { status: 404 }); return {}; },
    createRef: async ({ sha }) => { writes.push('tag'); state.tag = sha; },
  } } };
  return { writes, state, options, args: { github, context: { repo: { owner: 'test', repo: 'test' } }, core: { info() {} }, directory, commit } };
}

test('uploads all six verified assets before publication; published retries write nothing', async t => {
  const f = fixture(t);
  await publish(f.args);
  assert.equal(f.writes[0], 'tag');
  assert.equal(f.writes[1], 'draft');
  assert.equal(f.writes.filter(w => w.startsWith('upload:')).length, 6);
  assert.equal(f.writes.at(-1), 'publish');
  assert.equal(f.state.release.draft, false);
  f.writes.length = 0;
  await publish(f.args);
  assert.deepEqual(f.writes, []);
});

test('a failed upload leaves a draft; retry completes it', async t => {
  const f = fixture(t, { failUpload: true });
  await assert.rejects(publish(f.args), /Upload failed/);
  assert.equal(f.state.release.draft, true);
  f.options.failUpload = false;
  await publish(f.args);
  assert.equal(f.state.release.draft, false);
});

for (const [name, options] of [
  ['stale version snapshot', { tags: ['v0.2.0'] }],
  ['candidate', { manifest: { candidate: true } }],
  ['dirty', { manifest: { sourceDirty: true } }],
  ['wrong commit', { manifest: { sourceRevision: 'b'.repeat(40) } }],
  ['wrong plan', { plan: { commit: 'b'.repeat(40) } }],
  ['wrong version', { manifest: { version: '1.0.0' } }],
  ['unlicensed', { manifest: { license: 'UNLICENSED' } }],
  ['missing target', { manifest: { artifacts: [] } }],
  ['tag collision', { tag: 'b'.repeat(40) }],
  ['unpublished predecessor', { plan: { previous: 'v0.0.9' }, previousMissing: true }],
]) {
  test(`refuses ${name} before writes`, async t => {
    const f = fixture(t, options);
    await assert.rejects(publish(f.args));
    assert.deepEqual(f.writes, []);
  });
}

for (const [name, options] of [['tag added during upload', { newTagAfterUpload: true }], ['bad server digest', { badDigest: true }], ['notes edited during upload', { edited: true }]]) {
  test(`refuses publication: ${name}`, async t => {
    const f = fixture(t, options);
    await assert.rejects(publish(f.args));
    assert.equal(f.state.release.draft, true);
    assert.ok(!f.writes.includes('publish'));
  });
}

test('rejects corrupt local archives before writes', async t => {
  const f = fixture(t);
  fs.writeFileSync(path.join(f.args.directory, 'code-rules_0.1.0_linux_amd64.tar.gz'), 'corrupt');
  await assert.rejects(publish(f.args), /checksum/);
  assert.deepEqual(f.writes, []);
});

test('does not overwrite a mismatching draft on retry', async t => {
  const f = fixture(t, { failUpload: true });
  await assert.rejects(publish(f.args));
  f.state.release.body = 'Manual edit';
  f.writes.length = 0;
  await assert.rejects(publish(f.args), /differs/);
  assert.deepEqual(f.writes, []);
});
