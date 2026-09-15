# Library authoring and release candidate

The project-authoring CLI is merged in PR #7. Complete the approved launch sequence in two reviewable PRs; publication itself follows review.

## Library authoring

Reuse `src/authoring` templates, terminal collection, writer locking, and recoverable publication. Add library initialization, group/rule creation, and a read-only library check. Initialize a manifest and README, copy explicitly supplied license/notice bytes to conventional paths, and never infer terms. Check conventional rule/asset trees with shared format, link, and generation validation. Ignore unrelated repository files, including `.git`.

Test through the CLI: empty library, completed licensed library imported into a consumer, missing metadata, invalid license declarations, unsafe paths, collisions, draft detection, and cancellation. Update the library guide, CLI reference, status, and launch sequence.

## Release candidate

Package a Node.js CLI for macOS/Linux; keep Bun for development. Use an explicit executable entry point, package version in provenance, and a distributable canonical template. Choose the scoped package `@fabricahq/code-rules`; confirm the tool's license before setting distribution terms.

Build and pack from the checkout, install the tarball into an isolated prefix, then exercise the installed executable from a fresh project. Cover help/version, local-only setup, authored-library import, assets/licenses, selectors, constraints, exceptions, offline generation/check, stale output, and reinstall/upgrade preservation. Add CI checks and a versioned, trusted-publishing workflow. Document the first-publication account setup and a real-project pilot as release gates, not completed work.

## Interactive walkthrough

Use Gruntwork Runbooks with one visible real command per step. Install the packed artifact into a private prefix, author a real library and consumer project, and show generated files directly. Any network transport fixture must be explicit and isolated; no test-only executable or API wrapper may substitute for the CLI. Keep this local review artifact outside the product PR. Validate all executable steps and open the runbook for the user.
