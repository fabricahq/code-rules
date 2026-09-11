# Meaningful-steps audit

Date: 2026-09-10.
Scope: Fabrica Code Rules implementation, developer tooling, and executable documentation code in the current working tree.
Baseline: commit `59dad62`, including the uncommitted library-license extraction and tests.
Concurrent logo work was inspected as a moving snapshot and was not changed.

Rule: [Express Operations as Meaningful Steps](https://github.com/josh-padnick/code-rules/pull/3), in the `practices/code-design` group.
This review evaluates abstraction levels, ownership of invariants, diagnostic context, and behavior-preserving test boundaries.
Function length alone is not a finding.

## Implementation follow-up

The seven findings below have been addressed in the working tree.
Builds now separates configuration parsing, rule resolution, document parsing, and output rendering.
Markdown rewriting names its edit-ownership operations; configuration errors retain group indices and replacement field locations.
The docs link checker separates resolution from diagnostics and has command-level tests.
The retained logo-study pages resolve presentation data once, and the switcher separates preview updates from navigation updates.
The private visitors and cycling helper now have contract comments.

The public Builds exports are unchanged.
All 60 recorded calls matched before the diagnostic improvements; afterward, the only difference was the intended replacement-field location in one error.
The full repository check passed with 72 tests, and live HTML comparisons preserved content, links, and article styles across all five older study routes.
Browser accessibility inspection confirmed the vector page rendered; automated browser interaction checks were blocked by a chrome-devtools-axi/MCP pageId incompatibility.
The original audit findings and suggested verification below remain as the rationale for these changes.

## Assessment

The public Builds interface is already appropriately small: callers supply an in-memory project and receive generated files.
Keep that interface and its offline, deterministic contract.
The main opportunity is to give the implementation equally clear internal operations.

Three refactors offer the strongest improvement: separate rule resolution from generated output, expose configuration parsing steps, and clarify Markdown edit ownership.
The licensing extraction is a useful model for these changes.
The remaining findings are narrower improvements rather than reasons to redesign the subsystem.

This audit does not change implementation code.
Its priorities express maintainability value, not newly established runtime defect severity.

## Primary findings

### 1. buildRules combines policy resolution with document construction

Location: `src/builds/build.ts:104`, especially source processing at line 123, replacements at line 171, and output construction at line 255.

The operation validates snapshots, discovers rules, applies exceptions, incorporates local definitions, constructs agent guidance, formats group documents, and serializes provenance.
Understanding any one of those operations requires reading through details belonging to the others.
For example, a wording change to the generated index lives in the same function as the policy that prevents one replacement file from serving two upstream rules.

There is also an ownership constraint hidden in the function: each active rule is added to both its group and the flat provenance list.
Imported, replaced, and local paths must keep those collections consistent.
They currently do, but the correspondence is maintained at several call sites.

Recommendation:

- First extract generated-file rendering into a cohesive internal module with a single entry point and private index, group, and provenance renderers.
- Then extract rule resolution, with one owner for the selected groups, origins, source records, and replacement bookkeeping.
- Have the resolver return a completed result that rendering consumes.
  Either derive the flat rule list from the groups or centralize registration so the two representations cannot drift.
- Keep `buildRules` as the public operation coordinating validation, resolution, and rendering.
  Do not export every new helper from `src/builds/index.ts`.

Avoid moving the existing body into helpers that all mutate a large shared state object.
That would reduce visible length without clarifying ownership.

Verification: retain tests through `buildRules` for overlapping sources, source-scoped exclusions, upstream IDs on replacements, local definitions appearing once, empty groups, stable ordering, and provenance.
Capture representative complete generated output before extraction and compare bytes afterward.
Preserve the current order of validation and the guarantee that caller inputs are not mutated.

### 2. Configuration policies are hidden inside one source-mapping callback

Location: `src/builds/validation.ts:162`, particularly lines 176-274.

The callback parses a source name, repository, ref, groups, exclusions, and replacements while also updating a cross-source repository registry.
Several substantial domain steps are buried inside nested `Object.entries`, `Map`, and validation expressions.
A reviewer looking for replacement policy must mentally skip unrelated Git-ref syntax and cross-source uniqueness checks.

Recommendation: give configuration interpretation its own internal module.
Use private operations for source parsing, repository/ref validation, and exception declarations.
Keep the generic validation primitives reusable, but keep configuration-specific policy with the configuration parser.
A source's main parsing operation should read in field-validation order without exposing every regex and nested object schema.

Treat validation order as part of the contract.
Currently, duplicate-repository detection happens before validating that source's ref.
Parsing all sources completely and only then checking duplicates would change which error appears when both are invalid.
Preserve that sequence during extraction or make a deliberate diagnostic change separately.

Verification: malformed individual fields, duplicate repositories, excluded-and-replaced targets, reserved aliases, local/imported group overlap, deterministic ordering, and inputs containing more than one error.
Test observable diagnostics through `buildRules`, not each new private parser.

### 3. Markdown rewriting hides the non-overlapping-edit invariant

Location: `src/builds/markdown.ts:79`, especially the nested visitor at line 84.

The visitor traverses the tree, relocates URLs, namespaces references, rejects unsupported HTML, chooses which nodes own source edits, and serializes replacements.
The `captured` flag means an ancestor already owns the replacement span, while `changed` actually identifies node kinds that require rewriting.
Those names make the most important constraint less apparent than it should be.
The same node-kind list appears again when constructing serialized content.

Recommendation: make the outer operation express parsing, collecting rewrites, removing a redundant leading title, and applying source edits.
Within the rewrite collector, keep traversal and edit ownership together, with explicit contracts for node rewriting and serialization.
Use names such as `ancestorOwnsEdit` that state the constraint directly.
A focused source-edit type and a private edit-application operation would make offset handling easier to inspect.

Preserve the current algorithm's useful property: rewrite children before serializing their owning outer link, then apply non-overlapping edits from right to left.
Separate passes for links, images, and references could accidentally introduce overlapping edits or double transformations.
Do not replace the whole body with reserialized Markdown, which would lose the existing preservation of untouched text.

Verification: nested linked images, inline and reference links, reference-label collisions across rules, fenced code, surrounding formatting, title removal, encoded paths, suffixes, unsupported relative HTML links, and missing local targets.
Keep these checks at the public Builds boundary.

## Smaller findings

### 4. Configuration validation drops useful field locations

Location: `src/builds/validation.ts:220` and `src/builds/validation.ts:244`.

Group values pass from a field-specific string-array check into `groupId(id, where)`, which reports only the source location for a malformed group ID.
Replacement paths pass their precise `.replace.<id>.file` location into `nonempty`, then use the broader source location for containment and local-prefix checks.
Consequently, different failures in the same field provide different levels of diagnostic precision.

Carry the complete field location through every validation step, as the new `DeclaredPath` design does for license declarations.
Where array position matters, retain it before deduplication or sorting.
This is an intentional diagnostic improvement, so add assertions for exact field paths separately from structural extraction.

### 5. Rule-document interpretation shares a module with unrelated configuration policy

Location: `src/builds/validation.ts:306`.

The `rule` operation combines frontmatter boundary detection, YAML conversion safeguards, rule-field validation, and construction of the domain value.
Its parsing safeguards are harder to scan alongside the much larger configuration parser.

After the primary configuration extraction, give rule-document parsing a cohesive owner.
Private frontmatter-reading and metadata-validation operations would let the main function show the document-processing steps clearly.
Keep the original metadata text and body alongside validated values.
Do not regenerate frontmatter from parsed YAML, which could discard formatting and attribution details.

Verification: CRLF input, duplicate YAML keys, aliases, malformed metadata, blank body, extra attribution fields, and preserved source text.
The smaller `groupMetadata` operation does not need equivalent decomposition simply for symmetry.

### 6. The documentation link checker interleaves resolution and reporting

Location: `_tools/check-doc-links.py:26`, particularly the nested conditional at line 32.

One loop decodes a URL, chooses its filesystem base, resolves directory indexes, checks containment and existence, validates fragments, and constructs diagnostics.
The nested target expression obscures the distinct meanings of root-relative, page-relative, and fragment-only links.

Introduce a small local-target resolution operation and a link-checking operation that returns a diagnostic when appropriate.
Keep site discovery and final reporting in a straightforward command-level sequence.
A generic filesystem adapter or link-checking framework would add no useful abstraction here.

Before refactoring, exercise the script as a command against a temporary built-site fixture covering directory indexes, encoded paths, same-page fragments, missing anchors, missing destinations, and links outside the output root.
The current Bun suite does not exercise this Python script.

### 7. Logo-study selection repeats conditional policy across the template

Location: `docs/src/pages/logo-study/[study].astro:16` and `docs/src/components/LogoPrototypeSwitcher.astro:74`.

The study page derives overlapping flags and then repeats their precedence in expressions for titles, concepts, descriptions, navigation, and sample presentation.
Adding a study requires updating several parallel interpretations of the same route.
Resolve the route into one study description before rendering, with the content and presentation choices the template needs.
Keep that data local to the study feature rather than introducing a general page framework.

The switcher's `selectLogo` also mixes preview updates with current-URL and same-origin-link rewriting.
A private navigation-update operation would let the selection handler express those two responsibilities directly.

These are lower-priority changes because the code serves temporary design exploration.
If the studies are retired, remove them rather than refining them into a durable subsystem.
Coordinate with the ongoing logo work before editing these files.
If retained, verify route selection, unknown and retired variants, keyboard cycling, unrelated URL parameters, local-link propagation, and unchanged external links in the browser.

## What should stay simple

- `library-licenses.ts` already exposes meaningful stages, keeps field context with paths, and limits its export surface.
  Keep it as the reference implementation rather than splitting it further.
- Snapshot matching, origin construction, low-level validation guards, and the small group-metadata parser are cohesive.
  Their brevity is not a reason to wrap each expression.
- The Builds tests exercise the caller-facing interface and survived the licensing extraction unchanged.
  Keep that boundary even if the implementation gains internal modules.
- `_tools/builds-example.ts` is mostly fixture data followed by a clear build-and-write sequence.
  It does not justify implementing a Workspace abstraction ahead of the actual subsystem.
- Static logo inventories, configuration objects, CSS, and straightforward Astro templates do not need helper extraction for this rule.
- The aside-accessibility plugin already separates whole-tree discovery from mutation.
  Preserve that order because generated IDs must avoid IDs encountered later in the document.
  Its private visitor, like the Markdown visitor and logo cycling helper, still needs a concise contract to satisfy this rule's helper-documentation guidance.

## Recommended sequence and acceptance criteria

1. Extract output rendering, then rule resolution.
2. Extract configuration interpretation and improve diagnostic paths in a distinct change.
3. Isolate rule-document parsing.
4. Clarify Markdown rewriting after strengthening its behavior coverage where needed.
5. Improve the documentation checker and retained prototype code.

Each increment should retain the public Builds interface and remain independently reviewable.
Success means a reader can understand the main operation from its steps, locate a policy in one owner, and rename private helpers without rewriting tests.
A smaller maximum function length is not an acceptance criterion.

Validation during this audit: `bun run test` passed all 56 tests; `bun run typecheck` passed.
Those results establish a baseline, not proof that future extractions preserve every behavior.
No implementation changes, new tests, full documentation build, or browser validation were performed as part of this audit.
