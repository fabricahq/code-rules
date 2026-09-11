/** @fileoverview Resolves adopted rules from in-memory inputs and renders their generated file set. */

import type {
  ActiveRule,
  BuildInput,
  BuildOutput,
  Group,
  LibrarySnapshot,
  Origin,
  Source,
} from './types';
import {
  BuildError,
  compare,
  configuration,
  field,
  files,
  groupMetadata,
  invalid,
  json,
  nonempty,
  object,
  relativePath,
  requiredFile,
  rule,
  ruleGroup,
} from './validation';
import { escapeText, renderRule } from './markdown';

function origin(
  source: Source,
  snapshot: LibrarySnapshot,
  path: string,
): Origin {
  return {
    source: source.name,
    file: path,
    repository: source.repository,
    ref: source.ref,
    resolvedCommit: snapshot.resolvedCommit,
  };
}

function localOrigin(path: string): Origin {
  return {
    source: 'local',
    file: path,
    repository: null,
    ref: null,
    resolvedCommit: null,
  };
}

function licenseFiles(
  input: ReadonlyMap<string, string>,
  location: string,
): ReadonlyArray<string> {
  const library = object(
    json(
      requiredFile(input, 'rule-library.json', location),
      `${location}/rule-library.json`,
    ),
    location,
  );
  if (field(library, 'formatVersion') !== 1)
    return invalid(location, 'only library formatVersion 1 is supported');
  const rawLicense = field(library, 'license');
  if (rawLicense === undefined) return [];
  const license = object(rawLicense, `${location}.license`);
  const file = relativePath(
    nonempty(field(license, 'file'), `${location}.license.file`),
    location,
  );
  const notices = field(license, 'notices');
  if (!Array.isArray(notices))
    return invalid(location, 'license.notices must be an array');
  const paths = [
    file,
    ...notices.map((path: unknown) =>
      relativePath(nonempty(path, `${location}.license.notices`), location),
    ),
  ];
  for (const path of paths) requiredFile(input, path, location);
  return [...new Set(paths)].sort(compare);
}

/** Return the matching source snapshot, or throw BuildError when it is missing, inconsistent, or has an invalid commit. */
function snapshotFor(input: BuildInput, source: Source): LibrarySnapshot {
  const snapshot = Object.hasOwn(input.snapshots, source.name)
    ? input.snapshots[source.name]
    : undefined;
  if (snapshot === undefined)
    throw new BuildError(
      'needs-sync',
      `${source.name}: missing snapshot; run sync`,
    );
  const sameGroups =
    [...snapshot.groups].sort(compare).join('\n') === source.groups.join('\n');
  if (
    snapshot.repository !== source.repository ||
    snapshot.ref !== source.ref ||
    !sameGroups
  ) {
    throw new BuildError(
      'needs-sync',
      `${source.name}: snapshot repository, ref, or groups differ from configuration; run sync`,
    );
  }
  if (!/^[a-f0-9]{40}$/iu.test(snapshot.resolvedCommit))
    return invalid(source.name, 'resolvedCommit must be a full commit SHA');
  if (
    /^[a-f0-9]{40}$/iu.test(source.ref) &&
    source.ref.toLowerCase() !== snapshot.resolvedCommit.toLowerCase()
  ) {
    throw new BuildError(
      'needs-sync',
      `${source.name}: resolvedCommit differs from configured commit; run sync`,
    );
  }
  return snapshot;
}

function addGroup(groups: Map<string, Group>, id: string): Group {
  const existing = groups.get(id);
  if (existing !== undefined) return existing;
  const group: Group = { id, guidance: [], rules: [] };
  groups.set(id, group);
  return group;
}

/**
 * Return the complete generated file map after applying source exclusions, replacements, and local rules.
 * Paths are relative to generated/; identical inputs yield identical bytes without I/O or input mutation.
 * Throws BuildError with invalid-input for malformed data or needs-sync for missing/mismatched snapshots.
 * Checks snapshot identity, while callers remain responsible for authenticating and verifying its contents.
 */
export function buildRules(input: BuildInput): BuildOutput {
  const config = configuration(input.configuration);
  nonempty(input.toolVersion, 'toolVersion');
  const localFiles = files(input.localFiles, 'local');
  const groups = new Map<string, Group>();
  const usedReplacements = new Set<string>();
  const sourceRecords: Array<{
    name: string;
    repository: string;
    ref: string;
    resolvedCommit: string;
    groups: ReadonlyArray<string>;
    licenseFiles: ReadonlyArray<string>;
  }> = [];
  const activeRules: Array<ActiveRule> = [];
  for (const name of Object.keys(input.snapshots)) {
    if (!config.sources.some((source) => source.name === name))
      return invalid(name, 'snapshot is not declared in configuration');
  }
  for (const source of config.sources) {
    const snapshot = snapshotFor(input, source);
    const sourceFiles = files(snapshot.files, source.name);
    const licenses = licenseFiles(sourceFiles, source.name);
    sourceRecords.push({
      name: source.name,
      repository: source.repository,
      ref: source.ref,
      resolvedCommit: snapshot.resolvedCommit,
      groups: source.groups,
      licenseFiles: licenses,
    });
    const rulesById = new Map<string, ReturnType<typeof rule>>();
    for (const id of source.groups) {
      const group = addGroup(groups, id);
      group.guidance.push({
        source: source.name,
        metadata: groupMetadata(
          requiredFile(sourceFiles, `${id}/_group.json`, source.name),
          `${source.name}/${id}/_group.json`,
        ),
      });
      for (const [path, text] of sourceFiles) {
        if (
          !path.startsWith(`${id}/`) ||
          !path.endsWith('.md') ||
          path.split('/').at(-1)?.startsWith('_') ||
          licenses.includes(path)
        )
          continue;
        const parsed = rule(text, path, source.name);
        rulesById.set(path.slice(0, -3), parsed);
      }
    }
    for (const id of [...source.exclude.keys(), ...source.replace.keys()]) {
      if (!rulesById.has(id))
        return invalid(
          `${source.name}:${id}`,
          'exception target is missing from the selected groups',
        );
    }
    for (const [id, parsed] of rulesById) {
      const exclusion = source.exclude.get(id);
      if (exclusion !== undefined) {
        continue;
      }
      const replacement = source.replace.get(id);
      let active: ActiveRule;
      if (replacement !== undefined) {
        const path = replacement.file.slice('local/'.length);
        if (usedReplacements.has(path))
          return invalid(
            replacement.file,
            'replacement file is reused for multiple targets',
          );
        usedReplacements.add(path);
        if (ruleGroup(path, replacement.file) !== parsed.group)
          return invalid(
            replacement.file,
            'replacement must stay within the target group',
          );
        const definition = rule(
          requiredFile(localFiles, path, 'local'),
          path,
          'local',
        );
        active = {
          rule: { ...definition, id: parsed.id },
          origin: localOrigin(path),
          upstream: origin(source, snapshot, parsed.path),
          reason: replacement.reason,
          licenseFiles: [],
          sourceFiles: localFiles,
        };
      } else {
        active = {
          rule: parsed,
          origin: origin(source, snapshot, parsed.path),
          upstream: null,
          reason: null,
          licenseFiles: licenses,
          sourceFiles,
        };
      }
      addGroup(groups, parsed.group).rules.push(active);
      activeRules.push(active);
    }
  }
  for (const id of config.localGroups) {
    addGroup(groups, id).guidance.push({
      source: 'local',
      metadata: groupMetadata(
        requiredFile(localFiles, `${id}/_group.json`, 'local'),
        `local/${id}/_group.json`,
      ),
    });
  }
  for (const [path, text] of localFiles) {
    if (
      !path.endsWith('.md') ||
      path.split('/').at(-1)?.startsWith('_') ||
      usedReplacements.has(path)
    )
      continue;
    const parsed = rule(text, path, 'local');
    const group = groups.get(parsed.group);
    if (group === undefined)
      return invalid(
        `local/${path}`,
        'local rule belongs to an undeclared group',
      );
    const active: ActiveRule = {
      rule: parsed,
      origin: localOrigin(path),
      upstream: null,
      reason: null,
      licenseFiles: [],
      sourceFiles: localFiles,
    };
    group.rules.push(active);
    activeRules.push(active);
  }
  for (const path of localFiles.keys()) {
    if (path.endsWith('/_group.json')) {
      const id = path.slice(0, -'/_group.json'.length);
      if (!config.localGroups.includes(id))
        return invalid(
          `local/${path}`,
          'local metadata is only allowed for declared local-only groups',
        );
    }
  }
  const orderedGroups = [...groups.values()].sort((a, b) =>
    compare(a.id, b.id),
  );
  const index = [
    '# Code Rules',
    '',
    'This project uses [Fabrica Code Rules](https://github.com/fabricahq/code-rules) to declare the engineering best practices that people and agents should follow when writing or reviewing code.',
    'Each rule states an expectation, when it applies, and how to verify compliance.',
    '',
    "This file is the index to the project's adopted rules.",
    'The rules themselves are in the linked group files below, organized by technology or engineering practice.',
    "Each group file combines the active rules from shared libraries and local definitions, with the project's exclusions and replacements already applied.",
    '',
    '## How to use these rules',
    '',
    'Before writing code, use the descriptions below to find and read every group relevant to the task, technologies, and behavior involved.',
    'Apply each rule within its stated scope during implementation.',
    'During validation, check the code against those same rules and cite the rule IDs and evidence for any findings.',
    'Select practice groups by the behavior being changed: testing guidance can apply when adding production code, even if no test files have changed yet.',
    '',
    'This index and the linked group files are generated by Code Rules.',
    'To change the guidance, edit the source rules or project configuration and rebuild.',
  ];
  for (const [kind, title] of [
    ['techs', 'Technologies'],
    ['practices', 'Practices'],
  ]) {
    index.push('', `## ${title}`);
    const selected = orderedGroups.filter((group) =>
      group.id.startsWith(`${kind}/`),
    );
    if (selected.length === 0) index.push('', 'None selected.');
    for (const group of selected) {
      index.push('', `### [${group.id}](${group.id}.md)`);
      for (const guidance of group.guidance) {
        index.push(
          '',
          `**${guidance.source}: ${escapeText(guidance.metadata.name)}**`,
          '',
          escapeText(guidance.metadata.description),
        );
        if (guidance.metadata.whenToRead.length)
          index.push(
            '',
            ...guidance.metadata.whenToRead.map(
              (reason) => `- ${escapeText(reason)}`,
            ),
          );
      }
    }
  }
  index.push('', '[Source versions and rule origins](provenance.json).');
  const output = new Map<string, string>([
    ['RULES.md', `${index.join('\n')}\n`],
  ]);
  for (const group of orderedGroups) {
    const title = [
      ...new Set(group.guidance.map(({ metadata }) => metadata.name)),
    ]
      .sort(compare)
      .map(escapeText)
      .join(' / ');
    const sections = group.rules
      .sort((a, b) => compare(a.rule.id, b.rule.id))
      .map(renderRule);
    // Agents may open a group directly, so include its purpose and a route to project-wide guidance.
    const introduction = [
      `# ${title}`,
      '',
      `Group ID: \`${group.id}\``,
      '',
      "This file is a generated collection of the project's adopted engineering rules for this group, produced by [Fabrica Code Rules](https://github.com/fabricahq/code-rules).",
      "It aggregates rules from shared libraries and local definitions, with the project's exclusions and replacements already applied.",
      'Each rule below states an expectation, its scope, and how to verify compliance.',
      '',
      'Use these rules to guide implementation and to check the resulting work during validation.',
      'Apply every rule relevant to the task within its stated scope, and cite its source-qualified Rule ID when reporting findings.',
      'For project-wide instructions and help selecting other relevant technology or practice groups, read [RULES.md](../RULES.md).',
      'For source versions and rule origins, see [provenance.json](../provenance.json).',
      '',
      'To change this guidance, edit the source rules or project configuration and rebuild; edits to this generated file will be overwritten.',
    ];
    output.set(
      `${group.id}.md`,
      `${introduction.join('\n')}\n\n${sections.length ? sections.join('\n\n---\n\n') : 'No active rules in this group.'}\n`,
    );
  }
  const provenance = {
    toolVersion: input.toolVersion,
    sources: sourceRecords,
    rules: activeRules
      .sort((a, b) => compare(a.rule.id, b.rule.id))
      .map((active) => ({
        id: active.rule.id,
        group: active.rule.group,
        origin: active.origin,
        upstream: active.upstream,
        replacementReason: active.reason,
      })),
  };
  output.set('provenance.json', `${JSON.stringify(provenance, null, 2)}\n`);
  return {
    files: Object.fromEntries(
      [...output.entries()].sort(([a], [b]) => compare(a, b)),
    ),
  };
}
