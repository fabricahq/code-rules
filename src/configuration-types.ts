/** @fileoverview Defines validated project selections shared by Imports and Builds. */

/** Supported discovery scopes: all groups or every group of one kind. */
export type GroupPattern = '*' | 'techs/*' | 'practices/*';
/** A supported discovery scope or concrete IDs; arbitrary globs and mixed wildcard lists are unsupported. */
export type GroupSelection = GroupPattern | ReadonlyArray<string>;

/** Local replacement path including its local/ prefix, with the project decision explaining the override. */
export type RuleReplacement = {
  readonly file: string;
  readonly reason: string;
};
/** Validated source selection; exclusion/replacement keys are library-relative rule paths without .md. */
export type LibrarySource = {
  readonly name: string;
  readonly repository: string;
  readonly parsedRef: LibraryRef;
  readonly groups: GroupSelection;
  readonly exclude: ReadonlyMap<string, string>;
  readonly replace: ReadonlyMap<string, RuleReplacement>;
} & (
  | { readonly ref: string; readonly version?: never }
  | { readonly version: string; readonly ref?: never }
);
/** Validated sources sorted by alias and local-only group IDs sorted by code-unit order. */
export type ProjectConfig = {
  readonly sources: ReadonlyArray<LibrarySource>;
  readonly localGroups: ReadonlyArray<string>;
};

/** A validated exact revision or npm version constraint; branches are never inferred. */
export type LibraryRef =
  | { readonly kind: 'commit'; readonly sha: string }
  | { readonly kind: 'tag'; readonly name: string }
  | { readonly kind: 'version'; readonly range: string };
