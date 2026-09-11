/** @fileoverview Provides actionable Imports failures without exposing subprocess stderr or credentials. */

/** Stable failure categories for callers; inaccessible repositories cannot be distinguished from missing ones. */
export type ImportErrorCode =
  | 'invalid-configuration'
  | 'invalid-library'
  | 'git-unavailable'
  | 'not-found-or-no-access'
  | 'ref-not-found'
  | 'unsupported-content'
  | 'limit-exceeded'
  | 'cancelled'
  | 'timed-out'
  | 'git-failed'
  | 'io-error';

/** An Imports failure with a stable code and optional source alias, without partial output. */
export class ImportError extends Error {
  constructor(
    readonly code: ImportErrorCode,
    message: string,
    readonly source: string | null = null,
  ) {
    super(source === null ? message : `${source}: ${message}`);
    this.name = 'ImportError';
  }
}

/** Stop before starting more work when the caller or Imports deadline has cancelled the operation. */
export function requireActive(signal: AbortSignal): void {
  if (signal.aborted)
    throw new ImportError(
      signal.reason instanceof DOMException &&
        signal.reason.name === 'TimeoutError'
        ? 'timed-out'
        : 'cancelled',
      'Import stopped before completion.',
    );
}
