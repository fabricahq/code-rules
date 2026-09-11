/** @fileoverview Exposes the offline builder and caller-facing input, output, and error contracts. */

export { buildRules } from './build';
export { BuildError } from './validation';
export type {
  BuildInput,
  BuildOutput,
  FileContents,
  LibrarySnapshot,
} from './types';
