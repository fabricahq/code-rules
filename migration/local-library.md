# Local library catalog

PR #21 builds on #20 (`codex/go-version-selection`) and completes the requested five-PR review collection. The root lab page links to `/walkthrough/pr17` through `/walkthrough/pr21`; each page covers one capability and all calls reach native Go.

`library.Load(ctx, root *os.Root, source, selection)` expands supported group patterns, reads selected metadata and rules, and returns a catalog plus the exact bytes read. The caller opens and closes the root. The loader does not write files. Empty groups and explicit empty selections are valid. Explicit missing groups, missing metadata, malformed selected rules, unsafe entries, invalid UTF-8 text, and Git LFS rule pointers return no partial catalog.

The filesystem boundary lives in `internal/library`; `internal/rules` remains pure. Reads use `os.Root` confinement, reject observed symlink components, and open final files with Unix no-follow and nonblocking flags before verifying their type. The supported targets are macOS and Linux. The loader bounds each file to 4 MiB, total retained bytes to 32 MiB, and discovered entries to 10,000. Cancellation is preserved for `errors.Is` checks. It is not an atomic snapshot of a concurrently edited directory, and root confinement does not prohibit mounted filesystems.

Library declarations are validated before their term files are read. License and notice bytes remain unchanged, including binary and CRLF data. Selected rules retain authored frontmatter/body through the existing parser. Assets and Markdown link closure are not part of this catalog slice; asset directories are skipped. Unselected groups are not parsed. Full import validation and snapshot provenance remain pending.

The browser supplies a map of fixture paths and text, never a host directory. Its adapter validates relative paths, writes into a new temporary directory, invokes the loader, and removes that directory on success or failure. Optional links remain fixture-owned and demonstrate symlink rejection. The displayed `filesRead` projects the catalog's byte slices as text for these text-only fixtures. Production code retains raw bytes.

Tests use real temporary files and cover patterns, explicit/empty selections, empty groups, missing groups and manifests, malformed metadata/rules, Unicode text failures, symlink files and directories, cancellation, no partial results, binary/CRLF term retention, the exact 4 MiB file boundary and its next byte, and unchanged fixture inventories. Prior 703 shared comparisons and the real-Git selection comparison remain regression checks. No claim is made that this slice completes all filesystem, asset, or import contracts.

The final stack also runs Go race/vet/Staticcheck, all repository checks, installed-package tests, browser examples, and independent PR reviews. Reference and private engineering corpus pins remain unchanged. Merging remains a separate human decision.
