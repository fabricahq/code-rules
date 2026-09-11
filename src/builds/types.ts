/** @fileoverview Defines input, output, and resolution contracts for the offline Builds subsystem. */

/** Paths are relative to the snapshot root or local/; values are UTF-8 text. */
export type FileContents = Readonly<Record<string, string>>;

/**
 * Text snapshot of selected library groups, with paths relative to the library root.
 * Imports supplies identity; Workspace verifies bytes before calling Builds.
 * This caller-supplied envelope does not authenticate its own contents.
 */
export type LibrarySnapshot = {
  readonly repository: string;
  readonly ref: string;
  readonly resolvedCommit: string;
  readonly groups: ReadonlyArray<string>;
  readonly files: FileContents;
};

/** In-memory configuration, source snapshots keyed by alias, and local files rooted at local/; the builder validates their relationships. */
export type BuildInput = {
  readonly configuration: unknown;
  readonly snapshots: Readonly<Record<string, LibrarySnapshot>>;
  readonly localFiles: FileContents;
  readonly toolVersion: string;
  /** Maximum UTF-8 bytes per generated index file; defaults to 24 KiB. Full rule bodies are never truncated. */
  readonly indexMaxBytes?: number;
  /** Inline a complete group page when it fits this UTF-8 byte budget and indexMaxBytes; defaults to 8 KiB. Zero forces summaries. */
  readonly groupInlineMaxBytes?: number;
};

/** Complete generated text files keyed by paths relative to generated/, ready for the caller to install. */
export type BuildOutput = {
  readonly files: FileContents;
};

/** Local replacement path including its local/ prefix, with the project decision explaining the override. */
export type RuleReplacement = {
  readonly file: string;
  readonly reason: string;
};
/** Validated source selection; exclusion/replacement keys are library-relative rule paths without .md. */
export type LibrarySource = {
  readonly name: string;
  readonly repository: string;
  readonly ref: string;
  readonly groups: ReadonlyArray<string>;
  readonly exclude: ReadonlyMap<string, string>;
  readonly replace: ReadonlyMap<string, RuleReplacement>;
};
/** Validated sources sorted by alias and local-only group IDs sorted by code-unit order. */
export type ProjectConfig = {
  readonly sources: ReadonlyArray<LibrarySource>;
  readonly localGroups: ReadonlyArray<string>;
};
/** Display name and selection guidance for deciding when agents should read a group. */
export type GroupMetadata = {
  readonly name: string;
  readonly description: string;
  readonly whenToRead: ReadonlyArray<string>;
};
/** Parsed rule retaining raw frontmatter/body text; its ID combines the source alias and extensionless path. */
export type Rule = {
  readonly id: string;
  readonly group: string;
  readonly path: string;
  readonly title: string;
  readonly impact: string;
  readonly impactDescription: string;
  readonly whenToRead: string;
  readonly metadata: string;
  readonly body: string;
};
/** Source identity and root-relative definition path; local origins have null repository, ref, and commit. */
export type RuleOrigin = {
  readonly source: string;
  readonly file: string;
  readonly repository: string | null;
  readonly ref: string | null;
  readonly resolvedCommit: string | null;
};
/** Effective definition with optional replaced upstream origin/reason; source files and licenses belong to the active origin. */
export type ActiveRule = {
  readonly rule: Rule;
  readonly origin: RuleOrigin;
  readonly upstream: RuleOrigin | null;
  readonly reason: string | null;
  readonly licenseFiles: ReadonlyArray<string>;
  readonly sourceFiles: ReadonlyMap<string, string>;
};
/** Resolved collection combining source-labeled selection guidance and active rules for one group ID. */
export type Group = {
  readonly id: string;
  readonly guidance: ReadonlyArray<{
    readonly source: string;
    readonly metadata: GroupMetadata;
  }>;
  readonly rules: ReadonlyArray<ActiveRule>;
};

/** Snapshot identity and library terms retained for provenance, even when no rules remain active. */
export type SourceRecord = {
  readonly name: string;
  readonly repository: string;
  readonly ref: string;
  readonly resolvedCommit: string;
  readonly groups: ReadonlyArray<string>;
  readonly licenseFiles: ReadonlyArray<string>;
};

/** Completed resolution with sources and groups sorted by ID; each active rule belongs to exactly one group. */
export type ResolvedRules = {
  readonly sources: ReadonlyArray<SourceRecord>;
  readonly groups: ReadonlyArray<Group>;
};
