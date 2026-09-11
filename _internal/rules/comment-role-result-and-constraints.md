---
title: Comment the Role, the Result, and the Hidden Constraint
impact: MEDIUM
impactDescription: keeps comments as a scannable index of file role, export contract, and hidden constraints so they stay true after the implementation moves
tags: typescript, comments, tsdoc, documentation, readability, purpose
---

## Comment the Role, the Result, and the Hidden Constraint

A comment is an index entry, not a narration of the next line.
A file comment lets a reader decide whether to open the file.
An export comment tells a caller what they get back, including rules the signature does not show.
A private-helper comment saves readers from tracing the implementation when its purpose or behavior is not obvious.
A body comment records a constraint that names and types cannot express.

Agent-authored code fails this rule by omission more than by narration: files land with no header, exports carry `@param` tags that repeat the signature, and a magic sleep ships with no reason.
When the reason is missing, do not invent one; a guessed rationale becomes a spec for the next agent.

### File role and exported contract

**Incorrect** - no file header, and the export comment repeats the name and signature:

```ts
import { splitRows } from "./split-rows";

/**
 * Parses transactions.
 * @param csv - The CSV string to parse
 * @returns The parsed transactions
 */
export function parseTransactions(csv: string): Transaction[] {
  return splitRows(csv).map(parseRow);
}
```

**Correct** - the file states its role, and the export describes the result and supported input:

```ts
/**
 * @fileoverview Ingests the ledger's `id,posted_at,amount` CSV export.
 * Not a general CSV parser: fields are never quoted or comma-embedded.
 */
import { splitRows } from "./split-rows";

/**
 * Parse the ledger export into transactions with amounts in integer cents.
 * Tolerates a leading UTF-8 BOM, CRLF line endings, and blank lines.
 * Throws on a row with the wrong field count, an unparsable timestamp,
 * or an amount that is not a two-decimal number.
 */
export function parseTransactions(csv: string): Transaction[] {
  return splitRows(csv).map(parseRow);
}
```

These contract details must be verified against `splitRows` and `parseRow`.
Do not infer supported input or failure modes from the function's name.

### Hidden constraints

**Incorrect** - comments narrate the implementation but omit the vendor constraint:

```ts
/** @fileoverview Retry wrapper for calls into the accounting vendor's API. */
import { sleep } from "../util/sleep";

// Wait 250 milliseconds.
const VENDOR_MIN_CALL_INTERVAL_MS = 250;

// Try three times.
const MAX_ATTEMPTS = 3;

/** Retries the function. */
export async function withVendorRetry<T>(fn: () => Promise<T>): Promise<T> {
  for (let attempt = 1; attempt <= MAX_ATTEMPTS; attempt++) {
    try {
      return await fn();
    } catch (error) {
      if (attempt === MAX_ATTEMPTS) throw error;
      await sleep(VENDOR_MIN_CALL_INTERVAL_MS);
    }
  }
  throw new Error("unreachable");
}
```

**Correct** - the comment explains the known constraint, and the export describes success and failure:

```ts
/** @fileoverview Retry wrapper for calls into the accounting vendor's API. */
import { sleep } from "../util/sleep";

// The vendor sandbox returns HTTP 429 for calls closer together than this.
const VENDOR_MIN_CALL_INTERVAL_MS = 250;

const MAX_ATTEMPTS = 3;

/** Run `fn` until it resolves or `MAX_ATTEMPTS` attempts fail, rethrowing the last error. */
export async function withVendorRetry<T>(fn: () => Promise<T>): Promise<T> {
  for (let attempt = 1; attempt <= MAX_ATTEMPTS; attempt++) {
    try {
      return await fn();
    } catch (error) {
      if (attempt === MAX_ATTEMPTS) throw error;
      await sleep(VENDOR_MIN_CALL_INTERVAL_MS);
    }
  }
  throw new Error("unreachable");
}
```

The attempt count remains named but unexplained because no rationale for choosing three has been recorded.

### Observable results

**Incorrect** - the comment repeats the function name:

```ts
/** @fileoverview Folder ordering for the explorer sidebar. */

/** Sorts folders by name. */
export function sortFolders(folders: Folder[]): Folder[] {
  return [...folders].sort((a, b) =>
    a.name.localeCompare(b.name, "en-US", { sensitivity: "base" }),
  );
}
```

**Correct** - the comment describes ordering and mutation behavior:

```ts
/** @fileoverview Folder ordering for the explorer sidebar. */

/** Return a new array ordered by name using en-US collation, ignoring case and accents. */
export function sortFolders(folders: Folder[]): Folder[] {
  return [...folders].sort((a, b) =>
    a.name.localeCompare(b.name, "en-US", { sensitivity: "base" }),
  );
}
```

### Private helpers

These excerpts show a private helper inside a module.

**Incorrect** - the comment adds nothing to the name:

```ts
/** Gets the required value. */
function requiredValue<T>(value: T | undefined, name: string): T {
  if (value === undefined) throw new Error(`Missing ${name}`);
  return value;
}
```

**Correct** - the comment explains the failure condition and treatment of null:

```ts
/** Return the value, or throw an error identifying `name` when it is undefined; null is accepted. */
function requiredValue<T>(value: T | undefined, name: string): T {
  if (value === undefined) throw new Error(`Missing ${name}`);
  return value;
}
```

### Guidelines

- Start every source file with a 1-3 line comment saying what the file is for, and what it is not for when that is surprising.
  Write it as a `/** @fileoverview */` block or `//` lines; a bare `/** */` before the first declaration documents that declaration instead.
  One file, one role: a helper the header cannot account for belongs elsewhere.
  This applies to every file a change touches, not only new ones: a change that edits a file without a header adds one.
  Generated files are exempt.

- Give every exported function, type, and component a TSDoc stating its observable contract: what callers receive or can rely on, including rules the signature hides such as ordering, case, locale, units, mutation, empty input, and failure modes.
  Describe behavior in plain language rather than naming the implementation mechanism.
  A short export still gets one line, never a repetitive tag block.
  Drop `@param` and `@returns` that repeat names and types; keep `@throws` only when it names the condition.

- For private helpers, prefer names and signatures that make their purpose clear.
  Add a short doc comment when it helps readers understand behavior, purpose, or constraints that would otherwise require reading the implementation.
  Omit comments that merely restate the name.
  If a comment must explain several unrelated responsibilities, consider splitting the helper.

- Comment a body line only for a hidden constraint: vendor footgun, legal requirement, workaround, or invariant the types cannot express.
  Keep the comment next to the value or operation it explains.
  When the constraint belongs to a constant, name the constant and place the one-sentence comment there so the fact outlives the call site.

- When the reason is not known, name the value and leave the rationale out, or point at the issue that will settle it.
  Document observable behavior without guessing why it was chosen.
  Point workarounds and TODOs at an issue, PR, or upstream doc so the next reader can tell whether the comment is still live.

- Prefer comments that would remain true if the implementation changed while preserving its contract.
  Prune scratchpad and change-history comments before the change lands; git owns the history.
  Comments should save readers work without becoming a second implementation to maintain.

- Lint presence, not content, with `eslint-plugin-jsdoc`: on `require-file-overview`, `require-jsdoc` with `publicOnly`, and `require-description`; off `require-param` and `require-returns`, which would regenerate the tags this rule removes.
  `require-file-overview` only sees the `@file` block form, so a `//`-header repo needs its own presence check.
  Private-helper doc comments are optional; do not require or prohibit them mechanically.
  Review judges whether comments are useful, accurate, and durable.

Reference: adapts the "explain why, not what" intent of [TypeScript Style Guide - Comments](https://mkosir.github.io/typescript-style-guide/#comments) for agent-authored code, under the MIT license; see [the source corpus's THIRD_PARTY.md](https://github.com/josh-padnick/code-rules/blob/9bc48e937db1af1c7a0174c9a6d5692951e4d3e7/THIRD_PARTY.md#typescript-style-guide).
