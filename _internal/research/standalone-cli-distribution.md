# Standalone Code Rules distribution

Research date: 2026-09-14. Tested commit: `5e6ad4336758c1982cc2ce15565368545999ed8b` (PR #9, stacked on #8). Prototype compiler: Bun 1.4.2. The experiment does not change production code or release workflows.

## Recommendation

Keep TypeScript and qualify a Bun-compiled executable as a distribution option before deciding on a Go rewrite. The actual CLI already works as a standalone executable on the two ARM64 environments tested. Removing the user's Node installation requirement does not require changing languages.

This is a feasibility result, not release certification. Public distribution still needs native platform CI, downloaded-artifact signing/notarization checks, runtime update ownership, and review of bundled dependency notices. Go remains a serious alternative if the artifact size, JavaScript runtime policy, or required platform coverage makes this approach unsuitable. Prior implementation effort is not the reason to retain TypeScript.

## What the experiment established

| Check | Result |
| --- | --- |
| Compile the current CLI without assets | Version, init, and group creation work; rule creation fails because the canonical template is missing. |
| Embed the canonical template at its expected virtual path | Compilation and rule authoring work without any application-source changes. |
| Adapt the six installed-package tests to copy/run the standalone executable | All six pass on macOS ARM64 with the executable's PATH restricted to `/usr/bin:/bin`, where Node and Bun are unavailable. The test harness itself still runs under Bun. |
| Linux ARM64 smoke test | Passes in the existing `postgres:18` container, which has neither Node nor Bun on PATH. The container has networking disabled, a read-only root filesystem, an unprivileged user, and writable temporary directories only. |
| Compare generated output | All 10 generated files from the completed licensed-library walkthrough project are byte-identical between the npm/Node package and standalone/Bun executable. This is one fixture, not an exhaustive equivalence proof. |
| Missing Git | Sync exits 1 with `Cannot start Git; install Git 2.30 or later and check PATH.` |

The six macOS scenarios cover help/version without writes; canonical drafts and library validation; local-only generation and version provenance; licensed semantic-version imports, replacements, exclusions, and offline checks; stale output detection and repair; and reinstall preservation. Reinstallation in this adapted suite copies the executable again, rather than exercising an OS package manager. Git import tests use real committed fixture repositories with local Git transport routing, not live HTTPS/SSH authentication.

The Linux test covers library and project authoring, generation, offline checking, and stale-output repair. It does not cover Git imports. Native Intel macOS, Linux x64, Alpine/musl, interactive terminal prompts, signal handling, and a downloaded notarized artifact were not tested in this experiment.

## Size and timing

The macOS ARM64 executable is **62,969,970 bytes (60.1 MiB)**; gzip at level 9 produces **25,848,716 bytes (24.7 MiB)**. This contains the JavaScript runtime and dependencies. It is not comparable to the small npm tarball alone, which needs an installed Node runtime and dependency tree.

On this Mac, using 30 warm subprocess samples after three warmups:

| Command | Standalone Bun 1.4.2 | Existing npm package, Node 25.9.0 |
| --- | --- | --- |
| `--version`, median | 30.35 ms | 133.68 ms |
| Offline `check`, median | 57.61 ms | 167.11 ms |

These are local end-to-end timings, not cold-download startup benchmarks or general runtime comparisons. The standalone artifact bundles dependencies; the current npm package loads external dependencies. Both runtime and packaging differ. No equivalent Go implementation, Go performance measurement, or peak-memory comparison was produced.

## Packaging details that matter

The successful prototype command, run from the checkout, is:

```sh
bun build --compile \
  --no-compile-autoload-dotenv \
  --no-compile-autoload-bunfig \
  --reject-unresolved \
  --loader .md:file \
  --asset-naming '[name].[ext]' \
  src/cli.ts src/authoring/rule-template.md \
  --outfile /private/tmp/code-rules-standalone-research/code-rules
```

For the Linux build, add `--target=bun-linux-arm64` and choose a different output filename. The template entry point embeds `rule-template.md` at the virtual path used by the existing `readFile(new URL(..., import.meta.url))` call. No physical template or `node_modules` directory accompanies the executable. Future assets would need equally explicit inclusion and tests.

Bun documents automatic `.env` and `bunfig.toml` loading in compiled applications. Both are disabled here so merely running Code Rules in a repository does not introduce those additional configuration behaviors. This is not a sandbox: documented runtime environment options still exist. [Bun executable documentation](https://bun.com/docs/bundler/executables)

The executable still needs Git for importing libraries. The tested Linux glibc artifact is dynamically linked to the OS loader; "standalone" means no separately installed JavaScript runtime or application dependencies, not independence from the operating system.

## Alternatives and remaining release work

**Bun:** Current documented targets include macOS/Linux x64 and ARM64, with separate Linux musl targets. Requirements include macOS 13+, glibc 2.17+ for glibc builds, and SSE4.2 on x64. Current documentation treats baseline/modern x64 suffixes as aliases, unlike older indexed documentation. Test the exact pinned compiler and promised platform matrix instead of inferring support from cross-compilation success. [Targets and assets](https://bun.com/docs/bundler/executables), [system requirements](https://bun.com/docs/installation)

**Node single executable applications:** Node 24 LTS documents a CommonJS entry point, explicit assets, and blob injection. Current Node adds ESM and integrated building, but SEA remains under active development. Its documented tested-platform exclusions include macOS x64 and Alpine. For our current ESM CLI and macOS support, this does not look simpler than the working Bun prototype. No Node SEA prototype was built. [Node 24 SEA](https://nodejs.org/download/release/latest-krypton/docs/api/single-executable-applications.html), [current Node SEA](https://nodejs.org/api/single-executable-applications.html)

**Go:** Go compiles ahead of time and embeds its own runtime; it does not require a separately installed VM. It supports embedded files and a broad target matrix. A pure-Go implementation could offer attractive distribution characteristics, but an equivalent dependency graph and artifact size have not been established here. A port would need to preserve our npm-semver grammar, Markdown behavior, license bytes, rule identities, and filesystem transaction semantics. [Go runtime](https://go.dev/doc/faq#runtime), [embedding](https://pkg.go.dev/embed), [targets](https://go.dev/doc/install/source#environment)

Before shipping standalone artifacts:

1. Pin the compiler and run executable-level tests on native macOS ARM64/x64 and Linux ARM64/x64. Only promise musl if it is also tested. Extend current scenarios to prompts, cancellation, configuration isolation, and real Git authentication.
2. Establish macOS Developer ID signing and notarization, then verify the actual downloaded artifact. Local execution and ad-hoc signing do not establish Gatekeeper behavior. Bun's recommended JIT entitlements need a deliberate minimum-permission review. [Bun signing](https://bun.com/docs/bundler/executables#code-signing-on-macos), [Apple notarization](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution)
3. Publish checksums and versioned downloads; make rollback straightforward. Preserve the npm option if useful to teams already using Node.
4. Track the embedded compiler/runtime version and redistribute after relevant security fixes. Go embeds runtime code too, so a rewrite does not eliminate this responsibility. Go publishes a two-newer-major-release support window; Bun's security policy lists its supported line. [Go support policy](https://go.dev/doc/devel/release#policy), [Bun security policy](https://github.com/oven-sh/bun/security/policy)
5. Inventory bundled dependency notices and finish the existing tool-license/publication decisions.

## Reproducible local evidence

All prototype files are outside the application source tree at `/private/tmp/code-rules-standalone-research/`:

- `code-rules`: working macOS ARM64 executable.
- `code-rules-linux-arm64`: Linux ARM64 executable.
- `standalone.test.ts` and `tests.log`: adapted installed-package suite and six passing results.
- `linux-smoke.sh` and `linux-smoke.log`: isolated container scenario and passing result.
- `measurements.json`: sizes and local timing samples summarized above.
- `parity.py` and `parity.json`: output comparison and SHA-256 inventory.
- `missing-git.json`: observable missing-dependency error.

The current PRs, npm artifact, and interactive runbook remain unchanged by this experiment.
