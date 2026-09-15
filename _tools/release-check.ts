/** @fileoverview Rejects mismatched release tags, channel metadata, or undeclared tool licensing before publication. */

import { readFile } from 'node:fs/promises';
import { valid } from 'semver';
import metadata from '../package.json';

if (
  valid(metadata.version) !== metadata.version ||
  process.env.RELEASE_TAG !== `v${metadata.version}`
)
  throw new Error(
    'The GitHub release tag must exactly match v<package.json version>.',
  );
if (
  (process.env.RELEASE_PRERELEASE === 'true') !==
  metadata.version.includes('-')
)
  throw new Error('GitHub prerelease status must match the package version.');
if (
  metadata.license === 'UNLICENSED' ||
  !(await readFile('LICENSE.md', 'utf8')).trim()
)
  throw new Error(
    'Declare the approved tool license and include LICENSE.md before publishing.',
  );
