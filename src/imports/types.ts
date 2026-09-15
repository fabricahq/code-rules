/** @fileoverview Defines the byte-preserving result and cancellation contract of Imports. */

import type { LibrarySnapshot } from '../builds';

/** Original library-relative file bytes plus a text projection for offline Builds. */
export type ImportedLibrary = {
  readonly snapshot: LibrarySnapshot;
  readonly files: ReadonlyMap<string, Uint8Array>;
};

/** Complete source-alias mapping, returned only after all requested libraries succeed. */
export type ImportOutput = Readonly<Record<string, ImportedLibrary>>;

/** Optional caller cancellation; Imports also applies its own bounded operation deadline. */
export type ImportOptions = { readonly signal?: AbortSignal };
