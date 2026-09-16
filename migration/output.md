# Licensed output and provenance

`build.Prepare` combines native standalone rule rendering and summary indexes with source READMEs, exact license/notice bytes, and `provenance.json`. It accepts a resolved rule set and explicit tool version/index budget. Any error returns no partial output, and inputs remain unchanged.

Declared terms are copied even when all rules from their source are excluded. Replacement definitions keep a local origin plus upstream identity and the authored reason; an upstream declaration is not silently assigned to a local replacement. Provenance identifies the requested selector, chosen release when applicable, supplied commit, effective guidance, attribution, and original/generated term paths. Raw rule documents are not duplicated there.

Output paths are portable, contained, and free of file/directory conflicts. The returned byte slices belong to the output. No filesystem writes occur. Git authenticity and snapshot freshness remain the import boundary's responsibility; inline combined group delivery and production CLI integration are still unimplemented.

`/walkthrough/pr26` calls the real pipeline on editable fixtures and lets reviewers browse each generated file. Go tests additionally cover binary terms, empty terms, unchanged CRLF bytes, all-excluded sources, replacement provenance, repeatability, and missing-term failure.

## Provenance compatibility

Source declarations keep library-relative `files` and `attributionFiles`; rule declarations use `vendor/<source>/` paths relative to the configuration directory. Both include corresponding generated-root-relative paths. Source records retain the sorted `licenseFiles` compatibility inventory, including an explicit empty array when undeclared. Absent rule replacement reasons and unavailable local origin repository/ref/commit fields are explicit JSON nulls. See the existing [provenance format](../docs/src/content/docs/reference/files.mdx#license-declarations-in-provenance).
