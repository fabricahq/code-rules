# One config and folder layout for projects and libraries

Libraries had their own manifest file and kept their content at the repository root, while projects kept authored content in a separate local folder, so a repo that was both had two shapes and a library couldn't borrow skills the way a project takes them. Every repo now has one config, with a shared sources map and an optional library section, and one layout in which authored rules, skills, and assets sit beside per-source vendor snapshots and generated files; a library publishes exactly its authored content. A library borrows sources and installs skills for itself with the same machinery projects use, at the cost of moving the content of every existing library and changing Rulemart's importer.

## Consequences

- Sources aren't typed. Syllabus detects a source without a manifest, whose skills are all offered, and a source without library releases, which needs a `ref`; one command adds any kind of source.
- Vendor snapshots are kept per source, because two sources can both hold the same rule group ID.
- Rule IDs don't change, because they're relative to the authored content, not to the folder that holds it.
