# Version 2 config has no backward compatibility

Version 2 of the config replaces the version 1 project config and the separate library manifest, even though the CLI is already released, because few projects and libraries use it yet. Code Rules reads neither old shape and has no upgrade command: a version 1 config or an old manifest fails with a message that shows the new shape, and its owner rewrites it by hand. This keeps one reader and one set of meanings, rather than translating presets, `extends`, typed sources, and the old layout into a model that no longer has them, at the cost of a one-time manual migration for every existing project and library.

## Considered Options

- **A `project upgrade` command that rewrites version 1 configs:** proposed during the design and dropped.
