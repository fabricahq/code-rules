/** @fileoverview Preserves Builds errors while sharing format validation with Imports. */

/** Identifies invalid input or a snapshot requiring sync; the message includes the failing location. */
export class BuildError extends Error {
  constructor(
    readonly code: 'invalid-input' | 'needs-sync',
    message: string,
  ) {
    super(message);
    this.name = 'BuildError';
  }
}

export * from '../formats/validation';
