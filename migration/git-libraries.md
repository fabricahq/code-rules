# Complete Git library imports

`imports.ImportLibraries(ctx, configuration, options)` accepts a parsed
configuration and returns each source's `Catalog` and `Snapshot`. A failure in
any source returns no map. It never writes into a consuming project.

Each source uses the verified revision fetch from PR31. A bounded, NUL-framed Git
tree inventory supplies paths, modes, object IDs, and sizes. Selected blobs are
read by object ID through `git cat-file --batch`, with exact header, size, and
terminator verification. No checkout is created. Temporary repositories are
closed on success and failure; cleanup failures are returned.

`library.LoadSource` shares catalog validation between immutable Git objects and
confined local files. Both expand selections, parse rules and metadata, retain
complete owned attachments and referenced shared attachments, validate Markdown
links, and retain license/notice bytes. A malformed rule in an unselected group
is not parsed. Selected symlinks, submodules, LFS pointers, cross-rule links,
unsafe paths, and case collisions are rejected. Local reads retain their
`os.Root`, no-follow, hard-link, and special-file protections.

The catalog owns parsed fields and original documents. Its snapshot includes the
original retained bytes, resolved commit, configured selector, resolved tag and
version when selected by constraint, and expanded groups. `project.EncodeSnapshots`
can produce the vendor tree without rereading or reserializing authored files.

## Bounds and limitations

- 8 MiB tree listing, 10,000 tree files, and 64 path components.
- 8 MiB per selected blob and 64 MiB selected content per source.
- One 120-second deadline per source across fetch, reads, and validation.
- macOS and Linux, with Git 2.30 or newer.

The current adapter invokes one bounded cat-file process per selected file. This
keeps unselected blob contents unread and ownership simple; high file counts may
reach the deadline sooner than a persistent batch process would. Fetch network
traffic and temporary disk usage are not bounded by retained-content limits.
Installing snapshots, sync orchestration, and the production Go CLI are later
slices. The TypeScript CLI remains the production entry point.

## Interactive review

Run `go run ./cmd/rules-lab -serve -port 4391`, then open
`/walkthrough/pr32`. The lab creates real tagged repositories from editable files
and routes Git through a private local SSH helper. Repository addresses must be
`git@fixture.invalid:<source-alias>`; it never contacts an external Git host.

Presets cover exact tags, constraints, original CRLF and binary bytes, shared
attachment cycles, independent sources, empty selection, invalid unselected
rules, wildcard validation, unsupported content, and cancellation. Vendor files
are readable text where possible and explicitly labeled base64 otherwise. Every
fixture and temporary fetch repository is removed after the invocation.

## Validation

The Go suite includes real Git imports, byte-preserving snapshot round trips,
all-or-nothing failures, malformed tree records, unsupported objects, source
isolation, and cancellation identity. Existing local-library tests exercise the
shared reader refactor. Independent review also compares local and Git catalogs
and a pinned TypeScript import, including exact retained license bytes.
