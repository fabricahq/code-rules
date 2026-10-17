# All of a source's skills come from one revision

Rules have per-rule versions and pins, so a reader would expect skills to work the same way. Instead, every skill from a source comes from the one release or commit the project locked for that source, with no skill pins, no keeping one skill back on update, and no single-skill update; to hold a skill back, a project excludes or forks it or sets the source's `ref`. Skills use each other and share their source's library-wide files, and mixing revisions could install a skill beside dependencies it was never published with.

## Consequences

- `sync` never moves a source. When a project includes a skill that isn't in the locked revision, `sync` fails and points to updating that source, which moves all of its skills together.
