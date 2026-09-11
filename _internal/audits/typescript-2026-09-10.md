<!-- Records the standards audit of Code Rules before further subsystem work. -->

# Code Rules standards audit

Reviewed the working tree on September 10, 2026, including uncommitted Builds and documentation-site changes.
The base commit is `28bcf6d74328a5214ace999dcaec3d3ae97940ee`.
This is an audit, not a remediation change.
The comment findings were re-audited after the generated group introductions changed, using the replacement rule supplied by the user on September 10, 2026.
Other findings retain the scope and evidence of the original audit.

## Comment remediation status

The comment findings below are historical, recorded before remediation.
The final [comment rule](../rules/comment-role-result-and-constraints.md) now governs this repository and permits useful private-helper documentation.
All authored TypeScript, JavaScript, and Astro files have role headers, and exported functions/types have contract descriptions.
The Markdown contract was corrected, experiment-history comments were removed, and useful private-helper and invariant comments were retained or added.
`bun run lint` enforces file overviews and export descriptions; focused lint tests cover omissions and optional private-helper comments.
Astro components use their frontmatter overview for role and rendered-result documentation because their export is implicit.
The remaining non-comment findings are still follow-up work; this remediation does not claim to fix them.

## Rules and scope

- [Fabrica app rules](https://github.com/fabricahq/app/tree/bd48a1daabfa691bfa9777d2db0223fad68e8555/_rules), at `bd48a1daabfa691bfa9777d2db0223fad68e8555`.
- [Standalone corpus](https://github.com/josh-padnick/code-rules/tree/9bc48e937db1af1c7a0174c9a6d5692951e4d3e7), at `9bc48e937db1af1c7a0174c9a6d5692951e4d3e7`.
- [Comment the Role, the Result, and the Hidden Constraint](rules/comment-role-result-constraint.md), supplied by the user as the replacement comment standard.
  This audit copy preserves the supplied text, with its attribution link made absolute so it resolves from this directory.

Git blob comparison confirmed that all 38 TypeScript rule bodies in the app match the standalone corpus.
The standalone corpus separates two of those rules into its Fabrica overlay.
The user has deprecated the app's `_docs/code-comments.md` in favor of the supplied rule.
The replacement rule controls this audit's comment findings; the old policy is not an additional obligation.
In particular, private helpers no longer require comments simply because they are non-trivial, and the old blanket requirement for headers on authored Markdown documents no longer applies.
The new rule requires source-file role headers, observable contracts on every exported function/type/component, comments for hidden constraints, durable wording, and automated presence checks.

Inspected Builds, its tests and example, documentation components and client scripts, the aside transformation, Astro configuration, and utility scripts.
Generated output, dependencies, and image assets are outside the authored-code audit.
No Go, React runtime, Playwright, Goose, TanStack, or Zustand implementation exists here to assess against those technology groups.
The Python scripts and CSS were inspected for scope, but the supplied TypeScript rules do not establish a Python or CSS style standard.
The app-specific logger, Wails bindings, directory paths, and browser-testing setup are not automatic requirements for this CLI.

## Findings before comment remediation

### 1. Source files lack unambiguous role headers

**Confirmed against the replacement rule; MEDIUM rule impact.**

Inspected all 24 TypeScript, JavaScript, and Astro source/configuration files under `src/`, `_tools/`, and `docs/`, including tests and prototypes.
Only `SeededLogoStudies.ts:1` and `VectorLogoStudies.ts:1` already have top-of-file `//` role comments.
The other 22 files lack a header in the required form.
This is an inventory of current source, not a requirement to add headers to generated output, strict JSON, or prose documents.

| Files | Current state |
| --- | --- |
| All six `src/builds/` files | Five start directly with imports/exports; `types.ts:1` documents `FileContents`, not the file. |
| `_tools/builds-example.ts:1` | Has a useful role description in a bare `/** */` block; change it to `@fileoverview` or `//` to make its file scope explicit. |
| `docs/astro.config.mjs:1`, `docs/src/content.config.ts:1` | Start directly with imports. |
| `SmallLogoStudies.ts:1`, `SocketLogoStudies.ts:1`, `TinkerLogoStudies.ts:1` | Start directly with imports; no generated-file marker or checked-in generator for these modules was found. |
| `docs/src/plugins/accessible-aside-titles.mjs:1` | Its existing TSDoc belongs to the following exported function, so it cannot also serve as a file header. |
| All nine Astro components/pages | No opening role header: `Footer`, `HomeHero`, `LogoPrototypeSwitcher`, `NavLinks`, `PageTitle`, `PrototypeFabricaMark`, `SiteTitle`, `404`, and `logo-study/[study]`. |

Use a concise role comment at the start of each authored module, including existing files when they are edited.
For example, `build.ts` owns resolving adopted rules and emitting their in-memory aggregates; filesystem installation belongs to Workspace.
The replacement rule allows `//` headers, so lack of a blank line after the two existing metadata headers is no longer a finding.
Generated images do not by themselves make the TypeScript modules importing them generated code.
If a module is actually generated, establish that ownership and change its generator instead of editing its output.

For Astro, place the role comment at the start of frontmatter using `@fileoverview` or `//`, rather than adding visible documentation to the rendered page.
Astro's implicit component export needs an explicit documentation/checker convention; there is no authored `export function` to which ordinary TSDoc can attach.
The missing role/result documentation is clear, but the exact component lint adaptation is a repository decision.

### 2. Export contracts, hidden constraints, and comment durability

**Confirmed against the replacement rule; MEDIUM rule impact.**

#### Exported functions and types

The rule covers module exports, including helpers exported only for another Builds file.
Being short or internal to the subsystem is not an exemption.
The barrel can reuse the defining declaration's documentation rather than duplicating its contracts.

- `validation.ts` has 15 exported functions without TSDoc, from `invalid` at line 21 through `rule` at line 279.
  Describe observable behavior where signatures hide it: `nonempty` rejects whitespace-only text but returns accepted text untrimmed; `field` ignores inherited properties; `compare` uses case-sensitive JavaScript string ordering, not locale-aware collation; `files` validates paths and returns a sorted map.
  Parsing and validation contracts should identify their `BuildError` conditions.
- `markdown.ts` has four undocumented exports: `escapeText:8`, `encodedPath:14`, `sourceLink:18`, and `renderRule:153`.
  Explain escaping/newline handling, preservation of path separators, local versus commit-pinned destinations, and the rule section produced, respectively.
- `build.ts:128` has useful guarantees about determinism, I/O, and input mutation, but omits what callers receive and when the operation fails.
  Document the complete generated file map, paths relative to `generated/`, application of exclusions/replacements, and the `invalid-input` versus `needs-sync` error conditions.
- `types.ts` has 12 exported types; ten lack declaration-level TSDoc.
  The comment on `BuildOutput.files:21` does not document the `BuildOutput` declaration at line 20.
  `FileContents:1` already describes path roots and text values well; retain it as type documentation and add a separate file header.
  `LibrarySnapshot:4` names subsystem ownership, but should also describe the snapshot supplied and the fact that it does not authenticate its own contents.
  Explain identity, path roots, null semantics, and provenance relationships on the remaining types where their signatures do not make those contracts clear.
- `docs/src/pages/logo-study/[study].astro:10` exports `getStaticPaths` without a comment describing its five development routes and empty production result.
- `accessible-aside-titles.mjs:1` already states a useful user-visible result.
  Expand it to identify the returned transformer and its in-place mutation of the supplied tree, including preservation of existing title IDs and avoidance of collisions when assigning new ones.

For the Astro components, describe rendered results and relevant behavior, such as the design-preview notice in `PageTitle`, the branding links in `SiteTitle`, and the opt-in preview controls in `LogoPrototypeSwitcher`.
Do not require repetitive `@param`/`@returns` tags or add TSDoc to exported data constants solely because they are exported: the supplied rule explicitly enumerates functions, types, and components.
Document the exported `BuildError` class as the caller-visible error contract as well.

#### Correct or prune existing comments

`markdown.ts:68` promises that unrelated Markdown stays byte-for-byte intact.
The function also removes a leading title matching the rule title, namespaces reference labels, and trims the result at line 150.
Rewrite the export comment around the actual returned body and rejection conditions instead of the position-editing technique or an overbroad preservation promise.

`PrototypeFabricaMark.astro:17-22` contains experiment history and scratchpad seed/geometry notes, including “remain available as source history.”
Remove that history from the implementation comments; retain any still-useful design record in the study documentation.
The neighboring statement about CSS masks inheriting theme color at line 16 describes a current rendering constraint and is useful.

#### Hidden constraints: retain evidence, do not invent rationale

- Keep `markdown.ts:102`: the whole-link capture comment explains the invariant that prevents overlapping edits for nested images.
- Keep the purpose of `build.ts:345`: agents can open group files without first reading the index, which explains why the generated introduction must stand on its own.
- `docs/astro.config.mjs:8` explains why `satteri` is externalized, but lacks the issue, PR, or upstream documentation pointer required for a workaround.
  Find evidence for that dependency constraint before adding a reference.
- Ref restrictions at `validation.ts:181-198` and YAML alias rejection at line 292 are observable policies, but their full rationale is not recorded next to the code.
  State the accepted/rejected behavior in the export contract; add a body comment only when a hidden constraint is supported by a real decision or source.
  In particular, do not invent a security rationale for alias rejection or legitimize the ref defects in finding 4 with a guessed explanation.
- Reference-label namespacing and choosing a metadata fence longer than embedded backtick runs encode aggregation invariants.
  A short comment can explain prevention of cross-rule reference collisions or premature fence termination, without narrating the string operations.

Withdraw the previous requirement to add intent comments above private `licenseFiles`, `snapshotFor`, and `relocatedUrl` merely because they are non-trivial.
Likewise, the private test helpers and nested AST visitor do not need ceremonial summaries.
Comment their bodies only for hidden constraints as specified by the new rule.

#### Automated presence checks are absent

Neither package declares ESLint or `eslint-plugin-jsdoc`, and the root check runs Prettier, TypeScript, tests, and documentation checks.
None of those currently enforces the replacement rule's header/export-description requirements.
Add the requested presence checks with deliberate coverage for exported TypeScript types and Astro components, while leaving prose quality to review.
A `//` header needs the alternative presence check allowed by the supplied rule because `require-file-overview` recognizes the JSDoc file block form.
Do not enable repetitive parameter/return-tag requirements.

### 3. Footnotes are silently changed into ordinary links

**Confirmed behavior defect; related to [boundary-case testing](https://github.com/fabricahq/app/blob/bd48a1daabfa691bfa9777d2db0223fad68e8555/_rules/typescript/testing-cover-degenerate-and-boundary-cases.md).**

`markdown.ts:70` parses CommonMark without a footnote extension.
The reference rewrite at `:89` then treats a GitHub-style footnote as an ordinary reference link.

A public-interface probe used:

```markdown
Text with a note.[^same]

[^same]: Explanation.
```

The output points the note at a fabricated repository path ending in `practices/testing/Explanation.`.
The existing tests cover ordinary references and nested images but not this input.
Support footnotes with namespaced identifiers, or reject unsupported syntax explicitly before altering its meaning.
Add a failing-first regression through `buildRules`.

### 4. Ref validation accepts invalid names and rejects a valid numeric tag

**Confirmed behavior defect and missing boundary cases.**

At `validation.ts:181`, the approximation of Git ref syntax accepts `release//v1` and `release/.hidden`.
Git's `check-ref-format` rejects both when placed under `refs/tags/`.
At `:194`, the hexadecimal-length heuristic rejects `2026` as an abbreviated commit, although `refs/tags/2026` is valid.
The configuration docs promise plain exact tag names, with name resolution restricted to tags.

Centralize the ref policy, explain the ambiguity rule, and cover invalid path components and numeric or hexadecimal tag names.
Do not infer that a valid tag name is an abbreviated commit solely from its characters.
The audit probes exercised the public builder with matching snapshot envelopes; no network resolution was involved.

### 5. Documentation prototypes use unchecked type assertions

**Confirmed against [Avoid Type and Non-null Assertions](https://github.com/fabricahq/app/blob/bd48a1daabfa691bfa9777d2db0223fad68e8555/_rules/typescript/avoid-type-and-non-null-assertions.md).**

`docs/src/components/LogoPrototypeSwitcher.astro:89`, `:94`, `:97`, and `:104` cast event targets without narrowing or an explained exception.
`docs/src/pages/logo-study/[study].astro:18`, `:19`, and `:41` assert that array entries and a `find()` result exist.

Use element guards for event targets and explicit lookup checks for required logo entries.
Temporary prototype status does not itself satisfy the rule's requirement for a justified exception.
No corresponding unchecked assertions were found in Builds.

### 6. Most test descriptions omit the triggering condition

**Confirmed against [Use Clear should-when Test Descriptions](https://github.com/fabricahq/app/blob/bd48a1daabfa691bfa9777d2db0223fad68e8555/_rules/typescript/use-clear-should-when-test-descriptions.md).**

In `src/builds/build.test.ts`, 20 of 21 test-title declarations omit `when`.
Parameterized declarations expand to the 32 passing tests, so declaration count and runtime test count differ.
Examples are at `:80`, `:90`, and `:111`.
Only the deterministic-order title at `:202` follows the requested pattern.

Name the behavior and condition explicitly, including parameterized failure cases.
For example: `should preserve both rule IDs when two libraries share a group`.

### 7. Import ordering has no automated enforcement

**Confirmed against [Use Relative Imports Within a Feature](https://github.com/fabricahq/app/blob/bd48a1daabfa691bfa9777d2db0223fad68e8555/_rules/typescript/use-relative-imports-within-feature.md).**

That rule requires imports to be sorted by tooling.
`package.json` runs plain Prettier without an import-sorting plugin and has no ESLint configuration.
Prettier's formatting check does not establish this requirement.

Relative imports within Builds comply.
The rule's cross-feature alias convention needs an explicit CLI adaptation before flagging the example script's relative import as a violation.

### 8. Naming and immutable-constant conventions are inconsistent

**Confirmed naming departures; constant immutability is a stated preference.**

[Use Consistent Naming](https://github.com/fabricahq/app/blob/bd48a1daabfa691bfa9777d2db0223fad68e8555/_rules/typescript/use-consistent-naming.md) calls for boolean prefixes and capitalized constants.
Examples include `sameGroups` at `build.ts:94`, `image` at `markdown.ts:25`, `captured` and `changed` at `:73`, and `dark` in the logo study's theme handler.
The exported logo tables use camelCase names and mutable inferred arrays; see `docs/src/components/SeededLogoStudies.ts:14` and its sibling metadata modules.
The relevant companion rules are `prefer-immutable-data.md` and `use-const-assertions-for-constants.md`.

The plain TypeScript metadata files `SeededLogoStudies.ts`, `SmallLogoStudies.ts`, `SocketLogoStudies.ts`, `TinkerLogoStudies.ts`, and `VectorLogoStudies.ts` also depart from `use-predictable-file-names.md`.
Their source files are not marked as generated.
Use kebab-case, or document actual generator ownership if they are intended to be generated artifacts.
Astro component filename conventions should be addressed as an explicit framework adaptation rather than bundled into that rename.

### 9. The reusable aside plugin uses a default export

**Confirmed against [Use Named Exports](https://github.com/fabricahq/app/blob/bd48a1daabfa691bfa9777d2db0223fad68e8555/_rules/typescript/use-named-exports.md), applying the portable module convention to JavaScript.**

`docs/src/plugins/accessible-aside-titles.mjs:2` exports its reusable function as default, and `docs/astro.config.mjs:4` imports it that way.
Use a named export and import for the plugin.
Astro's configuration default export is a framework requirement and should remain an explicit exception.

The plugin also contains non-trivial ID collision handling and tree transformation without focused tests.
This is a testing gap under the portable intent of `testing-extract-logic-for-unit-tests.md`; its Fabrica-specific test directories and Wails examples do not transfer.

### 10. Correlated states and large functions need a design pass

**Rule-backed design recommendations, not demonstrated runtime failures.**

[Model Complex Variants as Discriminated Unions](https://github.com/fabricahq/app/blob/bd48a1daabfa691bfa9777d2db0223fad68e8555/_rules/typescript/model-complex-variants-as-discriminated-unions.md) applies to `Origin` and `ActiveRule` at `types.ts:51`.
They permit contradictory combinations of local source, upstream repository, null commit, and replacement reason.
Separate local/imported origins and replacement/non-replacement states so required fields follow the state.
The current construction paths produce valid combinations; this is a gap in what the type system guarantees.

[Keep Functions Pure and Focused](https://github.com/fabricahq/app/blob/bd48a1daabfa691bfa9777d2db0223fad68e8555/_rules/typescript/keep-functions-pure-and-focused.md) supports separating collection/resolution from rendering in `buildRules`.
The function at `build.ts:129` contains roughly 240 lines spanning several independently changeable responsibilities.
`configuration` at `validation.ts:141` similarly mixes source parsing, ref policy, exceptions, and cross-source checks.
Keep the public builder interface small while extracting named internal operations.

`prefer-single-object-function-args.md` also favors named arguments for helpers such as `rule(text, path, source)` and `relocatedUrl(url, active, image)`.
Treat this as a readability preference, especially for adjacent string parameters, rather than requiring options objects for comparator callbacks.

## Coverage of the 38 TypeScript rules

| Rule or related rules | Assessment |
| --- | --- |
| `write-comments-for-why`, superseded for this audit by the supplied replacement rule | Findings 1 and 2; old linked policy deprecated. |
| `avoid-type-and-non-null-assertions` | Finding 5; Builds complies. |
| `annotate-public-return-types` | Builds annotates returns; the exported Astro `getStaticPaths` needs an explicit contract/type annotation. |
| `testing-cover-degenerate-and-boundary-cases` | Finding 3 and 4; several existing empty/single/identity cases already covered. |
| `testing-regression-test-every-bug-fix` | Duplicate-heading regression exists; historical failing-first execution is not inferable for every change from the tree alone. |
| `write-focused-user-oriented-tests` | Builder tests use its public interface and no internal mocks; some tests could reuse one result instead of rebuilding per assertion. |
| `avoid-snapshot-tests` | Complies: focused assertions, no snapshot assertions. |
| `use-clear-should-when-test-descriptions` | Finding 6. |
| `use-fast-focused-test-tooling` | Bun supports focused runs; editor extensions are recommendations, not a release blocker. |
| `testing-extract-logic-for-unit-tests` | Builds follows the portable intent; aside plugin and prototype logic lack focused tests. App-specific setup excluded. |
| `use-relative-imports-within-feature` | Finding 7; same-feature imports comply. |
| `use-consistent-naming`, `use-predictable-file-names` | Finding 8; framework and generated-file exceptions need explicit scope. |
| `prefer-immutable-data`, `use-const-assertions-for-constants`, `use-as-const-satisfies-for-typed-constants` | Public Builds input uses readonly types; mutable logo tables need attention. Locally owned compiler accumulators and AST edits are not caller-input mutation. |
| `use-named-exports` | Finding 9; framework config excluded. |
| `model-complex-variants-as-discriminated-unions`, `use-discriminated-unions-for-function-args`, `prefer-unions-over-boolean-flags` | Finding 10; logo selection also spreads mutually related variants across several flags. |
| `keep-functions-pure-and-focused`, `prefer-single-object-function-args`, `keep-function-args-mostly-required` | Finding 10; most production parameters are required. |
| `colocate-code-by-feature`, `organize-projects-by-feature` | Builds is colocated; logo study logic/data are spread between the shared component folder and route. Lower-priority organization refinement. |
| `prefer-required-object-properties`, `distinguish-null-from-undefined` | Mostly follows the conventions; correlated nulls need the stronger state model in finding 10. |
| `prefer-type-aliases`, `separate-type-imports`, `use-generic-array-types` | Complies in Builds. |
| `prefer-inference-unless-annotation-narrows`, `prefer-literal-unions-over-enums` | No material finding; useful explicit annotations and no enums. |
| `use-boolean-for-explicit-boolean-coercion`, `use-ts-expect-error-with-description` | No double-negation coercions or suppression directives found. |
| `use-unknown-instead-of-any` | Builds validates external metadata through unknown; no explicit any found. The untyped JavaScript plugin is outside strict TypeScript checks. |
| `use-template-literal-types-for-patterned-strings` | Recommended follow-up: narrow validated group/rule IDs instead of retaining unrestricted strings internally. Runtime validation already exists. |
| `generate-service-types-from-contracts` | Not applicable: no external service client or generated service contract. |
| `logging-use-the-logger-facade` | App-specific and not applicable to the example CLI's deliberate stdout messages. |

## Validation and remediation order

All 32 existing tests pass, including with the audit's identified defects still present.
The two additional probes ran against the public builder without changing implementation files.
The reproduction script is `/private/tmp/code-rules-audit-probes.ts`.

First add unambiguous source-file headers and truthful export contracts, prune stale comments, and add failing-first cases for the two behavior defects.
Then address assertion narrowing and correlated state types.
Add automated checks for the enforceable naming, import, export, and test-title rules, with explicit framework exceptions.
Keep semantic review for comment usefulness, applicability, and design judgments.
A passing build and formatter are not a standards audit.

The comment re-audit is a static review of all 24 relevant source files and the configured checks.
No implementation files were changed or tests rerun for this re-audit; the passing-test result above belongs to the earlier validation.
The replacement rule and this revised report are the only files written by the re-audit.
