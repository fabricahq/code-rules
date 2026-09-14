/** @fileoverview Selects the highest matching Git-tag version from bounded remote advertisements without trusting tag order. */

import { eq, maxSatisfying } from 'semver';
import { tagVersion } from '../versions';
import { gitProcess } from './git-process';
import { ImportError } from './errors';

/** Advertised tag identity and its semantic version; object identity detects movement during fetch. */
export type VersionSelection = {
  readonly tag: string;
  readonly version: string;
  readonly object: string;
};

/** Discover complete SemVer tags, reject ambiguous highest versions, and return a deterministic tag spelling. */
export async function selectVersion(
  cwd: string,
  repository: string,
  range: string,
  signal: AbortSignal,
): Promise<VersionSelection> {
  const result = await gitProcess(
    cwd,
    ['ls-remote', '--tags', repository],
    signal,
    8 * 1024 * 1024,
  );
  if (result.status !== 0)
    throw new ImportError(
      'not-found-or-no-access',
      'Repository not found or no access; check its name and Git credentials.',
    );
  const records = new Map<string, string>();
  for (const line of result.output.toString('utf8').split('\n')) {
    if (!line) continue;
    const match = /^([0-9a-f]{40})\trefs\/tags\/(.+)$/u.exec(line);
    if (!match?.[1] || !match[2])
      throw new ImportError(
        'git-failed',
        'Git returned an unsupported tag advertisement.',
      );
    records.set(match[2], match[1]);
    if (records.size > 20_000)
      throw new ImportError(
        'limit-exceeded',
        'The repository exceeds the version-discovery tag limit.',
      );
  }
  const versions: Array<VersionSelection> = [];
  for (const [tag, object] of records) {
    const version = tagVersion(tag);
    if (version !== null) versions.push({ tag, object, version });
  }
  const highest = maxSatisfying(
    versions.map((candidate) => candidate.version),
    range,
  );
  if (highest === null)
    throw new ImportError(
      'version-not-found',
      `No Git tag satisfies version constraint ${range}.`,
    );
  const candidates = versions
    .filter((candidate) => eq(candidate.version, highest))
    .sort((a, b) => (a.tag < b.tag ? -1 : a.tag > b.tag ? 1 : 0));
  const commits = new Set(
    candidates.map(
      (candidate) => records.get(`${candidate.tag}^{}`) ?? candidate.object,
    ),
  );
  if (commits.size !== 1)
    throw new ImportError(
      'ambiguous-version',
      `Tags for the highest matching version ${highest} point to different objects; pin an exact ref or fix the tags.`,
    );
  const selected = candidates[0];
  if (!selected)
    throw new ImportError(
      'git-failed',
      'Git tag selection produced no result.',
    );
  return selected;
}
