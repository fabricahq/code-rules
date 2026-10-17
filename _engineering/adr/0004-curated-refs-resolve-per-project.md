# Projects resolve a library's curated refs, not the library release

A library names a `ref`, such as `main` or `v1.2`, for each source it curates skills from, and a reader would expect a library release to lock those refs to commits. It doesn't: each project resolves the curated ref itself on `update` and records the commit in its own lock, so moving a curated skill forward needs no new library release, a branch ref gives each project the latest commit when that project updates, and `sync` stays reproducible per project. Because a skill's identity is its repository plus its path, a project that declares the same repository as its own source overrides the library's `ref` for it.

## Considered Options

- **Lock curated commits in each library release:** every upstream fix would have waited for the library to publish a release.
- **A separate per-project map of ref overrides:** replaced by declaring the repository as an ordinary source, which needs no new config shape.

## Consequences

- Two projects on the same library release can hold different commits of a curated skill.
