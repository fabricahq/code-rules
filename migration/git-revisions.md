# Fetch verified Git revisions

`imports.FetchRevision(ctx, source, options)` fetches one exact commit/tag or the
highest release satisfying a HashiCorp version constraint. It owns a fresh bare
repository. The caller must call `Revision.Close` and handle cleanup failures.
Failure returns no revision and removes temporary state.

The transport accepts the existing HTTPS/SSH repository syntax. This candidate supports macOS and Linux, matching the TypeScript import boundary.
Git 2.30 or later
is required. Commands disable hooks, templates, recursive submodules, automatic
maintenance, replacement objects, prompting, and unsupported transport protocols.
Inherited repository/index/object-database state is removed while trusted caller
credentials and routing remain available. Git stderr is counted but never exposed.

Each fetch has a 120-second deadline by default. Cancellation and combined-output
overflow kill the process group, including helpers. Deadline expiry returns
`timed-out`; explicit cancellation returns `cancelled`. Both preserve the context
cause. Joined native cleanup failures remain visible; fixture cleanup failures use
the lab HTTP 500 boundary. Output limits do not bound
fetch traffic or temporary disk usage. There is no checkout or content adoption.

A version selection retains the advertised tag object and verifies it after fetch.
A moved tag fails. Exact commits must match the fetched commit. Tags of non-commit
objects fail. Failed refs never fall back to HEAD or a branch. Caller options are
trusted application process settings, not fields accepted from library content.

## Walkthrough and evidence

`/walkthrough/pr31` creates real commits, lightweight and annotated tags, then
serves them through an isolated local SSH helper running Git upload-pack. No
external repository, user Git configuration, or network listener is involved.
The returned object shows actual commit IDs and cleanup completion. Missing Git,
cancellation, timeout, and output-overflow scenarios are explicitly labelled
faults; timeout and overflow use controlled helper executables.

Tests exercise real fetches, moved tags, ambiguous aliases, missing refs,
non-commit tags, inherited-environment isolation, bounded output, and process
cancellation. The next slice reads validated library files from these objects;
project sync and Cobra command wiring remain later work.
