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
A body comment records a constraint that names and types cannot express.

Agent-authored code fails this rule by omission more than by narration: files land with no header, exports carry `@param` tags that repeat the signature, and a magic sleep ships with no reason.
When the reason is missing, do not invent one; a guessed rationale becomes a spec for the next agent.

**Incorrect** - no file role, tags that repeat the signature, a summary that restates the name, and an unexplained constant:

```ts
import { splitRows } from "./split-rows";

/**
 * Parses transactions.
 *
 * @param csv - The CSV string to parse
 * @returns The parsed transactions
 */
export function parseTransactions(csv: string): Transaction[] {
  return splitRows(csv).map(parseRow);
}

async function withRetry<T>(fn: () => Promise<T>): Promise<T> {
  for (let attempt = 1; attempt <= 3; attempt++) {
    try {
      return await fn();
    } catch (error) {
      if (attempt === 3) throw error;
      await sleep(250);
    }
  }
  throw new Error("unreachable");
}
```

**Correct** - the file states its role under `@fileoverview` so the block cannot attach to the next declaration, and the export states its observable result:

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

**Correct** - the retry helper is its own file, the hidden constraint sits on the value it explains, and the attempt count is named but not explained because nobody recorded why it is three:

```ts
/**
 * @fileoverview Retry wrapper for calls into the accounting vendor's API.
 */
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

**Incorrect** - restates the name, or compresses role jargon that never says the result:

```ts
/** Sorts folders by name. */
export function sortFolders(folders: Folder[]): Folder[] {
  return [...folders].sort((a, b) =>
    a.name.localeCompare(b.name, "en-US", { sensitivity: "base" }),
  );
}

/** Locale-aware folder order for the explorer sidebar. */
export function sortFolders(folders: Folder[]): Folder[] {
  return [...folders].sort((a, b) =>
    a.name.localeCompare(b.name, "en-US", { sensitivity: "base" }),
  );
}
```

**Correct** - states the observable result in plain language, not the call that produces it:

```ts
/** Sort folders by name, case-insensitive, in en-US order. */
export function sortFolders(folders: Folder[]): Folder[] {
  return [...folders].sort((a, b) =>
    a.name.localeCompare(b.name, "en-US", { sensitivity: "base" }),
  );
}
```

**Guidelines:**

- Start every source file with a 1-3 line comment saying what the file is for, and what it is not for when that is surprising.
  Write it as a `/** @fileoverview */` block or `//` lines; a bare `/** */` before the first declaration documents that declaration instead.
  One file, one role: a helper the header cannot account for belongs elsewhere.
  This applies to every file a change touches, not only new ones: a change that edits a file without a header adds one.
  Generated files are exempt.
- Give every exported function, type, and component a TSDoc stating the observable result: what the caller gets back, in what order, and which rules the signature hides (case, locale, units, empty input, failure modes).
  Say `en-US` and case-insensitive, not `localeCompare`.
  A short export still gets one line, never a tag block.
  Drop `@param` and `@returns` that repeat names and types; keep `@throws` only when it names the condition.
- Comment a body line only for a hidden constraint: vendor footgun, legal requirement, workaround, or invariant the types cannot express.
  Name the value and put the one-sentence comment on the constant, so the fact outlives the call site.
- When the reason is not known, name the value and leave the comment out, or point at the issue that will settle it.
  Point workarounds and TODOs at an issue, PR, or upstream doc so the next reader can tell whether the comment is still live.
- Keep a comment only if it would still be true after the implementation changed.
  Prune scratchpad and change-history comments before the change lands; git owns the history.
- Lint presence, not content, with `eslint-plugin-jsdoc`: on `require-file-overview`, `require-jsdoc` with `publicOnly`, and `require-description`; off `require-param` and `require-returns`, which would regenerate the tags this rule removes.
  `require-file-overview` only sees the `@file` block form, so a `//`-header repo needs its own presence check.
  Review judges content against the durability test.

Reference: adapts the "explain why, not what" intent of [TypeScript Style Guide - Comments](https://mkosir.github.io/typescript-style-guide/#comments) for agent-authored code, under the MIT license; see [_rules/THIRD_PARTY.md](https://github.com/josh-padnick/code-rules/blob/9bc48e937db1af1c7a0174c9a6d5692951e4d3e7/THIRD_PARTY.md#typescript-style-guide).
