/** @fileoverview Validates explicit Git addresses and derives browser links only for recognized public hosts. */

import { invalid, nonempty } from './formats/validation';

/** Duplicate-detection identity and optional known-host web paths; neither changes the caller's Git transport address. */
type RepositoryAddress = {
  readonly identity: string;
  readonly web: {
    readonly root: string;
    readonly tree: string;
    readonly file: string;
    readonly raw: string;
  } | null;
};

/** Reject path normalization and encoded separators before URL parsing can erase their original meaning. */
function repositoryPath(path: string, location: string, uri: boolean): string {
  const trimmed = path.replace(/\/$/u, '');
  const parts = trimmed.split('/');
  const normalized: Array<string> = [];
  for (const part of parts) {
    let decoded: string;
    try {
      decoded = uri ? decodeURIComponent(part) : part;
    } catch {
      return invalid(
        location,
        'repository path contains invalid percent encoding',
      );
    }
    if (
      !decoded ||
      decoded === '.' ||
      decoded === '..' ||
      /[/\\\x00-\x1f\x7f]/u.test(decoded) ||
      (uri && decoded.includes('%'))
    ) {
      return invalid(
        location,
        'repository path contains an empty, dot, or unsafe segment',
      );
    }
    normalized.push(uri ? encodedSegment(decoded) : decoded);
  }
  return normalized.join('/');
}

/** Encode a browser path component, including punctuation that would terminate a Markdown link. */
function encodedSegment(value: string): string {
  return encodeURIComponent(value).replace(
    /[!'()*]/gu,
    (character) => `%${character.charCodeAt(0).toString(16).toUpperCase()}`,
  );
}

/**
 * Parse HTTPS, ssh://, or user@host:path Git addresses without fetching or rewriting the transport address.
 * Reject credentials in HTTPS, passwords, getter syntax, queries, fragments, and ambiguous path normalization.
 * Known public hosts share repository identities across standard transports; other hosts retain path case and transport semantics.
 */
export function repositoryAddress(
  value: unknown,
  location: string,
): RepositoryAddress {
  const address = nonempty(value, location);
  if (/[\s\\\x00-\x1f\x7f\uD800-\uDFFF?#]/u.test(address)) {
    return invalid(
      location,
      'repository must not contain whitespace, backslashes, query parameters, or fragments',
    );
  }
  const urlMatch = /^(https|ssh):\/\/([^/]+)\/(.+)$/u.exec(address);
  const scpMatch = urlMatch
    ? null
    : /^([A-Za-z0-9_][A-Za-z0-9_.-]*)@(\[[^\]]+\]|[^:/]+):(.+)$/u.exec(address);
  let urlText: string;
  let rawPath: string;
  if (urlMatch?.[3] !== undefined) {
    urlText = address;
    rawPath = repositoryPath(urlMatch[3], location, true);
  } else if (scpMatch?.[1] && scpMatch[2] && scpMatch[3]) {
    // SCP paths may be home-relative or absolute; parse only the authority as a URL.
    const absolute = scpMatch[3].startsWith('/');
    rawPath = `${absolute ? '/' : ''}${repositoryPath(absolute ? scpMatch[3].slice(1) : scpMatch[3], location, false)}`;
    urlText = `ssh://${scpMatch[1]}@${scpMatch[2]}/`;
  } else {
    return invalid(
      location,
      'repository must be an explicit HTTPS URL, ssh:// URL, or user@host:path address; owner/name shorthand is unsupported',
    );
  }
  let url: URL;
  try {
    url = new URL(urlText);
  } catch {
    return invalid(location, 'invalid Git repository URL');
  }
  if (
    !url.hostname ||
    url.password ||
    (urlMatch?.[2]?.includes('@') && !url.username) ||
    (urlMatch?.[2]?.split('@')[0]?.includes(':') &&
      urlMatch[2].includes('@')) ||
    (url.protocol === 'https:' && url.username) ||
    (url.username && !/^[A-Za-z0-9_][A-Za-z0-9_.-]*$/u.test(url.username))
  ) {
    return invalid(
      location,
      'repository requires a host and must not embed credentials; SSH may specify a username',
    );
  }
  const host = url.hostname.toLowerCase().replace(/\.$/u, '');
  if (
    !/^\[[a-f0-9:.]+\]$/u.test(host) &&
    host
      .split('.')
      .some((label) => !/^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/u.test(label))
  ) {
    return invalid(
      location,
      'repository hostname must be a DNS name or IP address',
    );
  }
  const standardTransport =
    url.protocol === 'https:'
      ? !url.port
      : (!url.port || url.port === '22') && url.username === 'git';
  const browserPath = scpMatch
    ? rawPath.replace(/^\//u, '').split('/').map(encodedSegment).join('/')
    : rawPath;
  const path = browserPath.replace(/\.git$/u, '');
  const segments = path.split('/');
  const knownHost =
    standardTransport &&
    segments.every(Boolean) &&
    ((host === 'github.com' && segments.length === 2) ||
      (host === 'gitlab.com' && segments.length >= 2));
  if (knownHost) {
    const identityPath = host === 'github.com' ? path.toLowerCase() : path;
    const webRoot = `https://${host}/${path}`;
    return {
      identity: `${host}/${identityPath}`,
      web:
        host === 'github.com'
          ? {
              root: webRoot,
              tree: `${webRoot}/tree`,
              file: `${webRoot}/blob`,
              raw: `https://raw.githubusercontent.com/${path}`,
            }
          : {
              root: webRoot,
              tree: `${webRoot}/-/tree`,
              file: `${webRoot}/-/blob`,
              raw: `${webRoot}/-/raw`,
            },
    };
  }
  const port = url.port ? `:${url.port}` : '';
  const user = url.username ? `${url.username}@` : '';
  return {
    identity: scpMatch
      ? `${user}${host}:${rawPath}`
      : `${url.protocol}//${user}${host}${port}/${rawPath}`,
    web: null,
  };
}

/** Return a pinned file/raw URL for a supported public host, or null when the host's browsing convention is unknown. */
export function repositoryFileUrl(
  repository: string,
  commit: string,
  path: string,
  image: boolean,
): string | null {
  const { web } = repositoryAddress(repository, 'repository');
  if (web === null) return null;
  const encodedPath = path.split('/').map(encodeURIComponent).join('/');
  return `${image ? web.raw : web.file}/${commit}/${encodedPath}`;
}
