/** Paths are relative to the snapshot root or local/; values are UTF-8 text. */
export type FileContents = Readonly<Record<string, string>>;

/** Imports supplies identity; Workspace verifies bytes before calling Builds. */
export type LibrarySnapshot = {
  readonly repository: string;
  readonly ref: string;
  readonly resolvedCommit: string;
  readonly groups: ReadonlyArray<string>;
  readonly files: FileContents;
};

export type BuildInput = {
  readonly configuration: unknown;
  readonly snapshots: Readonly<Record<string, LibrarySnapshot>>;
  readonly localFiles: FileContents;
  readonly toolVersion: string;
};

export type BuildOutput = {
  /** Complete output, with paths relative to generated/. */
  readonly files: FileContents;
};

export type Replacement = { readonly file: string; readonly reason: string };
export type Source = {
  readonly name: string;
  readonly repository: string;
  readonly ref: string;
  readonly groups: ReadonlyArray<string>;
  readonly exclude: ReadonlyMap<string, string>;
  readonly replace: ReadonlyMap<string, Replacement>;
};
export type Configuration = {
  readonly sources: ReadonlyArray<Source>;
  readonly localGroups: ReadonlyArray<string>;
};
export type GroupMetadata = {
  readonly name: string;
  readonly description: string;
  readonly whenToRead: ReadonlyArray<string>;
};
export type Rule = {
  readonly id: string;
  readonly group: string;
  readonly path: string;
  readonly title: string;
  readonly metadata: string;
  readonly body: string;
};
export type Origin = {
  readonly source: string;
  readonly file: string;
  readonly repository: string | null;
  readonly ref: string | null;
  readonly resolvedCommit: string | null;
};
export type ActiveRule = {
  readonly rule: Rule;
  readonly origin: Origin;
  readonly upstream: Origin | null;
  readonly reason: string | null;
  readonly licenseFiles: ReadonlyArray<string>;
  readonly sourceFiles: ReadonlyMap<string, string>;
};
export type Group = {
  readonly id: string;
  readonly guidance: Array<{
    readonly source: string;
    readonly metadata: GroupMetadata;
  }>;
  readonly rules: Array<ActiveRule>;
};
