/** @fileoverview Publishes verified archives to a draft first; retries preserve published releases and matching assets. */

const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');

/** Verify a complete build bundle before making any GitHub writes. */
function readBundle(directory, commit) {
  const plan = JSON.parse(fs.readFileSync(path.join(directory, 'release-plan.json'), 'utf8'));
  const manifest = JSON.parse(fs.readFileSync(path.join(directory, 'manifest.json'), 'utf8'));
  if (plan.commit !== commit || manifest.sourceRevision !== commit ||
      plan.tag !== `v${manifest.version}` || plan.version !== manifest.version ||
      manifest.candidate !== false || manifest.sourceDirty !== false ||
      !Array.isArray(plan.tags) || !plan.notes?.trim() || manifest.license !== 'SEE LICENSE.md') {
    throw new Error('Release plan and clean, licensed build must match the approved commit');
  }
  const targets = ['darwin/amd64', 'darwin/arm64', 'linux/amd64', 'linux/arm64'];
  if (JSON.stringify(manifest.artifacts.map(a => a.target).sort()) !== JSON.stringify(targets)) {
    throw new Error('Expected exactly four native targets');
  }
  const files = manifest.artifacts.map(a => {
    const name = `code-rules_${plan.version}_${a.target.replace('/', '_')}.tar.gz`;
    if (a.file !== name) throw new Error('Unexpected archive filename');
    const file = asset(directory, name);
    if (file.digest !== `sha256:${a.sha256}` || file.data.length !== a.bytes) {
      throw new Error(`Archive checksum or size mismatch: ${name}`);
    }
    return file;
  });
  const sums = manifest.artifacts.map(a => `${a.sha256}  ${a.file}\n`).join('');
  if (fs.readFileSync(path.join(directory, 'SHA256SUMS'), 'utf8') !== sums) {
    throw new Error('SHA256SUMS does not match the manifest');
  }
  files.push(asset(directory, 'manifest.json'), asset(directory, 'SHA256SUMS'));
  return { plan, files };
}

function asset(directory, name) {
  const data = fs.readFileSync(path.join(directory, name));
  return { name, data, digest: `sha256:${crypto.createHash('sha256').update(data).digest('hex')}` };
}

/** Stage every expected asset, verify GitHub's stored digests, then publish exactly once. */
async function publish({ github, context, core, directory, commit }) {
  const { plan, files } = readBundle(directory, commit);
  const repo = context.repo;
  // Publication is globally serialized; a plan made before another release must be rebuilt.
  async function verifyTagSnapshot() {
    const refs = await github.paginate(github.rest.git.listMatchingRefs, { ...repo, ref: 'tags/v' });
    const names = refs.map(r => r.ref.replace(/^refs\/tags\//, '')).filter(t => t !== plan.tag).sort();
    if (JSON.stringify(names) !== JSON.stringify(plan.tags)) {
      throw new Error('Version tags changed since planning; review the new predecessor before retrying');
    }
  }
  await verifyTagSnapshot();
  if (plan.previous) {
    const { data: previous } = await github.rest.repos.getReleaseByTag({ ...repo, tag: plan.previous });
    if (previous.draft) throw new Error('The previous release must be published first');
  }
  const releases = await github.paginate(github.rest.repos.listReleases, { ...repo, per_page: 100 });
  const matches = releases.filter(r => r.tag_name === plan.tag);
  if (matches.length > 1) throw new Error('Multiple releases have this tag');
  let release = matches[0];
  let tagExists = true;
  try {
    await github.rest.git.getRef({ ...repo, ref: `tags/${plan.tag}` });
  } catch (error) {
    if (error.status !== 404) throw error;
    tagExists = false;
  }
  async function verifyTag() {
    const { data: target } = await github.rest.repos.getCommit({ ...repo, ref: `refs/tags/${plan.tag}` });
    if (target.sha !== commit) throw new Error('Release tag points to a different commit');
  }
  if (tagExists) await verifyTag();
  if (release && (!tagExists || release.body !== plan.notes || release.name !== plan.tag ||
                  release.prerelease !== plan.prerelease)) {
    throw new Error('Existing release differs from the approved request; inspect it before retrying');
  }
  if (!tagExists) {
    await github.rest.git.createRef({ ...repo, ref: `refs/tags/${plan.tag}`, sha: commit });
  }
  if (!release) {
    const { data } = await github.rest.repos.createRelease({
      ...repo, tag_name: plan.tag, target_commitish: commit,
      name: plan.tag, body: plan.notes, draft: true, prerelease: plan.prerelease,
    });
    release = data;
  }
  const listAssets = () => github.paginate(github.rest.repos.listReleaseAssets, {
    ...repo, release_id: release.id, per_page: 100,
  });
  const verifyAsset = (remote, file) => {
    if (remote.state !== 'uploaded' || remote.size !== file.data.length || remote.digest !== file.digest) {
      throw new Error(`Stored asset differs: ${file.name}`);
    }
  };
  const existing = await listAssets();
  if (existing.some(a => !files.some(f => f.name === a.name))) {
    throw new Error('Release contains unexpected assets');
  }
  for (const file of files) {
    const matching = existing.filter(a => a.name === file.name);
    if (matching.length > 1) throw new Error(`Duplicate release asset: ${file.name}`);
    if (matching[0]) {
      verifyAsset(matching[0], file);
    } else {
      if (!release.draft) throw new Error('Published release is missing assets; refusing to modify it');
      await github.rest.repos.uploadReleaseAsset({
        ...repo, release_id: release.id, name: file.name, data: file.data,
        headers: { 'content-type': 'application/octet-stream', 'content-length': file.data.length },
      });
    }
  }
  const stored = await listAssets();
  if (stored.length !== files.length) throw new Error('Release asset set is incomplete');
  for (const file of files) {
    const matching = stored.filter(a => a.name === file.name);
    if (matching.length !== 1) throw new Error(`Missing or duplicate asset: ${file.name}`);
    verifyAsset(matching[0], file);
  }
  await verifyTagSnapshot();
  await verifyTag();
  const { data: fresh } = await github.rest.repos.getRelease({ ...repo, release_id: release.id });
  if (fresh.body !== plan.notes || fresh.tag_name !== plan.tag || fresh.name !== plan.tag || fresh.prerelease !== plan.prerelease) {
    throw new Error('Release changed during the build; refusing publication');
  }
  if (fresh.draft) {
    await github.rest.repos.updateRelease({ ...repo, release_id: release.id, draft: false,
      make_latest: plan.prerelease ? 'false' : 'true' });
  }
  core.info(`Release ready: ${fresh.html_url}`);
}

module.exports = { readBundle, publish };
