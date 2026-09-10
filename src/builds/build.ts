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
import { encodedPath, escapeText, renderRule } from './markdown';

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

/** Deterministic offline compilation; performs no IO and never mutates its inputs. */
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
  const exceptions: Array<{
    id: string;
    kind: 'exclude' | 'replace';
    reason: string;
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
        exceptions.push({ id: parsed.id, kind: 'exclude', reason: exclusion });
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
        exceptions.push({
          id: parsed.id,
          kind: 'replace',
          reason: replacement.reason,
        });
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
    'Generated by Code Rules. Edit source rules and configuration, then rebuild.',
    '',
    'Before implementation or review, select relevant groups from the task, technologies, and behavior involved.',
    'Read the selected files and apply each active rule within its scope.',
    'Practice groups can apply even when their technology or test files are not being changed.',
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
  index.push('', '## Sources and licenses');
  if (sourceRecords.length === 0) index.push('', 'Local rules only.');
  for (const source of sourceRecords) {
    index.push(
      '',
      `### ${source.name}`,
      '',
      `Repository: ${source.repository}`,
      '',
      `Requested ref: ${escapeText(source.ref)}`,
      '',
      `Resolved commit: \`${source.resolvedCommit}\``,
    );
    if (source.licenseFiles.length === 0)
      index.push('', 'Library license: unspecified.');
    else
      index.push(
        '',
        ...source.licenseFiles.map(
          (path) =>
            `- [${escapeText(path)}](../vendor/${source.name}/${encodedPath(path)})`,
        ),
      );
  }
  index.push(
    '',
    'Sources retain their own terms. Rule-specific terms and attribution remain with each definition.',
    '',
    '## Exceptions',
    '',
    'These are project decisions about inactive or replaced upstream rules, not additional obligations.',
  );
  if (exceptions.length === 0) index.push('', 'None.');
  for (const exception of exceptions.sort((a, b) => compare(a.id, b.id)))
    index.push(
      '',
      `- ${exception.kind === 'exclude' ? 'Excluded' : 'Replaced'} \`${exception.id}\`: ${escapeText(exception.reason)}`,
    );
  const output = new Map<string, string>([
    ['RULES.md', `${index.join('\n')}\n`],
  ]);
  for (const group of orderedGroups) {
    const sections = group.rules
      .sort((a, b) => compare(a.rule.id, b.rule.id))
      .map(renderRule);
    output.set(
      `${group.id}.md`,
      `# ${group.id}\n\nGenerated by Code Rules. Apply each rule within its stated scope.\n\n${sections.length ? sections.join('\n\n---\n\n') : 'No active rules in this group.'}\n`,
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
