/** @fileoverview Resolves imported and local definitions into adopted groups while preserving source identity and exception policy. */

import type {
  ActiveRule,
  BuildInput,
  ProjectConfig,
  Group,
  GroupMetadata,
  LibrarySnapshot,
  RuleOrigin,
  RuleReplacement,
  ResolvedRules,
  Rule,
  LibrarySource,
  SourceRecord,
  LicenseDeclaration,
  ExpandedLibrarySource,
  GroupPattern,
} from './types';
import {
  BuildError,
  compare,
  files,
  groupMetadata,
  invalid,
  requiredFile,
  ruleGroup,
  groupId,
  strings,
} from './validation';
import { rule } from './rule-document';
import { readLibraryLicenses } from './library-licenses';
import { licensePaths, requireLicenseFiles } from './rule-licenses';

/** Mutable group storage owned by resolution; renderers receive read-only groups. */
type GroupAccumulator = {
  id: string;
  guidance: Array<Group['guidance'][number]>;
  rules: Array<ActiveRule>;
};
/** A validated source snapshot with selected definitions and guidance in selection order. */
type SelectedLibrary = {
  source: ExpandedLibrarySource;
  snapshot: LibrarySnapshot;
  files: ReadonlyMap<string, string>;
  licenses: ReadonlyArray<LicenseDeclaration>;
  licenseFiles: ReadonlyArray<string>;
  guidance: ReadonlyArray<{ id: string; metadata: GroupMetadata }>;
  rules: ReadonlyMap<string, Rule>;
};

/** Record an imported rule's source alias, definition path, and pinned repository revision. */
function origin(
  source: LibrarySource,
  snapshot: LibrarySnapshot,
  path: string,
): RuleOrigin {
  return {
    source: source.name,
    file: path,
    repository: source.repository,
    ref: source.ref,
    resolvedCommit: snapshot.resolvedCommit,
  };
}

/** Identify a local rule by its local-root-relative path, with no upstream repository or revision. */
function localOrigin(path: string): RuleOrigin {
  return {
    source: 'local',
    file: path,
    repository: null,
    ref: null,
    resolvedCommit: null,
  };
}

/** Return the matching source snapshot, or throw BuildError when it is missing, inconsistent, or has an invalid commit. */
function snapshotFor(
  snapshots: BuildInput['snapshots'],
  source: LibrarySource,
): LibrarySnapshot {
  const snapshot = Object.hasOwn(snapshots, source.name)
    ? snapshots[source.name]
    : undefined;
  if (snapshot === undefined)
    throw new BuildError(
      'needs-sync',
      `${source.name}: missing snapshot; run sync`,
    );
  const snapshotGroups = strings(
    snapshot.groups,
    `${source.name}.snapshot.groups`,
  )
    .map((id) => groupId(id, `${source.name}.snapshot.groups`))
    .sort(compare);
  const recordedSelection = snapshot.groupSelection ?? snapshotGroups;
  const sameSelection =
    typeof source.groups === 'string' || typeof recordedSelection === 'string'
      ? source.groups === recordedSelection
      : [
          ...strings(
            recordedSelection,
            `${source.name}.snapshot.groupSelection`,
          ),
        ]
          .sort(compare)
          .join('\n') === source.groups.join('\n');
  const sameGroups =
    typeof source.groups === 'string' ||
    snapshotGroups.join('\n') === source.groups.join('\n');
  if (
    snapshot.repository !== source.repository ||
    snapshot.ref !== source.ref ||
    !sameGroups ||
    !sameSelection
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

/** Discover every group represented by metadata or rule files; later metadata loading rejects orphan rules instead of omitting them. */
function patternGroups(
  sourceFiles: ReadonlyMap<string, string>,
  rules: ReadonlyMap<string, Rule>,
  source: string,
  pattern: GroupPattern,
): ReadonlyArray<string> {
  const ids = new Set(
    [...rules.values()].map((definition) => definition.group),
  );
  for (const path of sourceFiles.keys()) {
    if (!path.endsWith('/_group.json')) continue;
    const parts = path.split('/');
    if (!matchesGroupPattern(path, pattern)) continue;
    const id = parts.slice(0, 2).join('/');
    ids.add(groupId(id, `${source}/${path}`));
    if (parts.length !== 3)
      return invalid(
        `${source}/${path}`,
        'group metadata must be directly under its group',
      );
  }
  return [...ids].sort(compare);
}

/** Match only the three supported scopes against library-relative paths, without interpreting arbitrary glob syntax. */
function matchesGroupPattern(path: string, pattern: GroupPattern): boolean {
  return pattern === '*'
    ? path.startsWith('techs/') || path.startsWith('practices/')
    : path.startsWith(pattern.slice(0, -1));
}

/** Return an existing group or register empty resolution storage for the selected ID. */
function addGroup(
  groups: Map<string, GroupAccumulator>,
  id: string,
): GroupAccumulator {
  const existing = groups.get(id);
  if (existing !== undefined) return existing;
  const group: GroupAccumulator = { id, guidance: [], rules: [] };
  groups.set(id, group);
  return group;
}

/** Reject extra snapshots before interpreting any source's contents. */
function requireDeclaredSnapshots(
  config: ProjectConfig,
  snapshots: BuildInput['snapshots'],
): void {
  for (const name of Object.keys(snapshots)) {
    if (!config.sources.some((source) => source.name === name))
      invalid(name, 'snapshot is not declared in configuration');
  }
}

/** Read metadata at its full source location while preserving missing-file diagnostics. */
function readGroupMetadata(
  sourceFiles: ReadonlyMap<string, string>,
  id: string,
  source: string,
): GroupMetadata {
  const path = `${id}/_group.json`;
  const text = requiredFile(sourceFiles, path, source);
  return groupMetadata(text, `${source}/${path}`);
}

/** Parse candidate rules before excluding their declared Markdown license assets; malformed rule candidates still fail. */
function readRuleFiles(
  sourceFiles: ReadonlyMap<string, string>,
  source: string,
  candidates: ReadonlyArray<string>,
): ReadonlyMap<string, Rule> {
  const parsed = new Map<string, Rule>();
  const failures = new Map<string, unknown>();
  for (const path of candidates) {
    try {
      parsed.set(
        path,
        rule(requiredFile(sourceFiles, path, source), path, source),
      );
    } catch (error) {
      if (!(error instanceof BuildError)) throw error;
      failures.set(path, error);
    }
  }
  const assets = new Set(
    [...parsed.values()].flatMap((definition) =>
      licensePaths(definition.licenses ?? []),
    ),
  );
  for (const [path, error] of failures) if (!assets.has(path)) throw error;
  for (const [path, definition] of parsed) {
    if (assets.has(path))
      return invalid(
        `${source}:${path}`,
        'a rule cannot also be a declared license or attribution file',
      );
    requireLicenseFiles(
      definition.licenses ?? [],
      sourceFiles,
      `${source}:${path}`,
    );
  }
  return parsed;
}

/** Load licensing, then each selected group's guidance and rules; excluded rules are still validated. */
function selectedLibrary(
  source: LibrarySource,
  snapshot: LibrarySnapshot,
): SelectedLibrary {
  const sourceFiles = files(snapshot.files, source.name);
  const licenses = readLibraryLicenses(sourceFiles, source.name);
  const licenseFiles = licensePaths(licenses);
  const guidance: Array<{ id: string; metadata: GroupMetadata }> = [];
  const candidates = [...sourceFiles.keys()].filter(
    (path) =>
      (typeof source.groups === 'string'
        ? matchesGroupPattern(path, source.groups)
        : source.groups.some((id) => path.startsWith(`${id}/`))) &&
      path.endsWith('.md') &&
      !path.split('/').at(-1)?.startsWith('_') &&
      !licenseFiles.includes(path),
  );
  const parsed = readRuleFiles(sourceFiles, source.name, candidates);
  const selectedGroups =
    typeof source.groups === 'string'
      ? patternGroups(sourceFiles, parsed, source.name, source.groups)
      : source.groups;
  if (
    typeof source.groups === 'string' &&
    [...snapshot.groups].sort(compare).join('\n') !== selectedGroups.join('\n')
  ) {
    throw new BuildError(
      'needs-sync',
      `${source.name}: wildcard snapshot groups differ from its library contents; run sync`,
    );
  }
  const expandedSource: ExpandedLibrarySource = {
    ...source,
    groups: selectedGroups,
  };
  const rules = new Map(
    [...parsed].map(([path, definition]) => [path.slice(0, -3), definition]),
  );
  for (const id of selectedGroups)
    guidance.push({
      id,
      metadata: readGroupMetadata(sourceFiles, id, source.name),
    });
  return {
    source: expandedSource,
    snapshot,
    files: sourceFiles,
    licenses,
    licenseFiles,
    guidance,
    rules,
  };
}

/** Reject exception targets outside the parsed selection before applying any exception. */
function requireExceptionTargets(library: SelectedLibrary): void {
  const { source, rules } = library;
  for (const id of [...source.exclude.keys(), ...source.replace.keys()]) {
    if (!rules.has(id))
      invalid(
        `${source.name}:${id}`,
        'exception target is missing from the selected groups',
      );
  }
}

/** Reserve a replacement file and return its local definition under the upstream ID; reject reuse or a different group first. */
function replacementRule(
  parsed: Rule,
  replacement: RuleReplacement,
  library: SelectedLibrary,
  localFiles: ReadonlyMap<string, string>,
  usedReplacements: Set<string>,
): ActiveRule {
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
  const text = requiredFile(localFiles, path, 'local');
  const definition = rule(text, path, 'local');
  requireLicenseFiles(definition.licenses ?? [], localFiles, `local:${path}`);
  return {
    rule: { ...definition, id: parsed.id },
    origin: localOrigin(path),
    upstream: origin(library.source, library.snapshot, parsed.path),
    reason: replacement.reason,
    licenses: definition.licenses ?? [],
    sourceFiles: localFiles,
  };
}

/** Apply source-scoped exceptions in parsed-rule order, reserving replacement paths across libraries. */
function importedRules(
  library: SelectedLibrary,
  localFiles: ReadonlyMap<string, string>,
  usedReplacements: Set<string>,
): ReadonlyArray<ActiveRule> {
  requireExceptionTargets(library);
  const active: Array<ActiveRule> = [];
  for (const [id, parsed] of library.rules) {
    if (library.source.exclude.has(id)) continue;
    const replacement = library.source.replace.get(id);
    active.push(
      replacement === undefined
        ? {
            rule: parsed,
            origin: origin(library.source, library.snapshot, parsed.path),
            upstream: null,
            reason: null,
            licenses: parsed.licenses ?? library.licenses,
            sourceFiles: library.files,
          }
        : replacementRule(
            parsed,
            replacement,
            library,
            localFiles,
            usedReplacements,
          ),
    );
  }
  return active;
}

/** Return non-replacement local rules in file order; reject rules in undeclared groups. */
function localRules(
  localFiles: ReadonlyMap<string, string>,
  groups: ReadonlyMap<string, Group>,
  usedReplacements: ReadonlySet<string>,
): ReadonlyArray<ActiveRule> {
  const active: Array<ActiveRule> = [];
  const candidates = [...localFiles.keys()].filter(
    (path) => path.endsWith('.md') && !path.split('/').at(-1)?.startsWith('_'),
  );
  const definitions = readRuleFiles(localFiles, 'local', candidates);
  for (const [path, parsed] of definitions) {
    if (usedReplacements.has(path)) continue;
    if (!groups.has(parsed.group))
      return invalid(
        `local/${path}`,
        'local rule belongs to an undeclared group',
      );
    active.push({
      rule: parsed,
      origin: localOrigin(path),
      upstream: null,
      reason: null,
      licenses: parsed.licenses ?? [],
      sourceFiles: localFiles,
    });
  }
  return active;
}

/** Reject local metadata outside explicitly declared local-only groups after checking local rules. */
function requireLocalMetadata(
  localFiles: ReadonlyMap<string, string>,
  localGroups: ReadonlyArray<string>,
): void {
  for (const path of localFiles.keys()) {
    if (!path.endsWith('/_group.json')) continue;
    const id = path.slice(0, -'/_group.json'.length);
    if (!localGroups.includes(id))
      invalid(
        `local/${path}`,
        'local metadata is only allowed for declared local-only groups',
      );
  }
}

/** Resolve sources then local rules into ID-sorted groups without mutating inputs; throw BuildError for invalid selections or stale snapshots. */
export function resolveRules(
  config: ProjectConfig,
  snapshots: BuildInput['snapshots'],
  localFiles: ReadonlyMap<string, string>,
): ResolvedRules {
  requireDeclaredSnapshots(config, snapshots);
  const groups = new Map<string, GroupAccumulator>();
  const usedReplacements = new Set<string>();
  const sources: Array<SourceRecord> = [];
  for (const source of config.sources) {
    const snapshot = snapshotFor(snapshots, source);
    const library = selectedLibrary(source, snapshot);
    sources.push({
      name: source.name,
      repository: source.repository,
      ref: source.ref,
      resolvedCommit: library.snapshot.resolvedCommit,
      groups: library.source.groups,
      groupSelection: source.groups,
      licenses: library.licenses,
      licenseFiles: library.licenseFiles,
    });
    for (const { id, metadata } of library.guidance)
      addGroup(groups, id).guidance.push({ source: source.name, metadata });
    for (const active of importedRules(library, localFiles, usedReplacements))
      addGroup(groups, active.rule.group).rules.push(active);
  }
  for (const id of config.localGroups) {
    if (groups.has(id))
      return invalid(
        id,
        'an imported group cannot also be declared in localGroups',
      );
    addGroup(groups, id).guidance.push({
      source: 'local',
      metadata: readGroupMetadata(localFiles, id, 'local'),
    });
  }
  for (const active of localRules(localFiles, groups, usedReplacements))
    addGroup(groups, active.rule.group).rules.push(active);
  requireLocalMetadata(localFiles, config.localGroups);
  return {
    sources,
    groups: [...groups.values()].sort((a, b) => compare(a.id, b.id)),
  };
}
