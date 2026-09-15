# First-release sequence

The approved feature sequence separates project file handling, group resolution, project authoring, and library authoring. Each PR should deliver a complete workflow and keep planned commands clearly labeled until implemented.

1. **Imports, PR #2:** merged. Fetch versioned libraries, preserve selected rules and assets, and record provenance.
2. **Sync, PR #5:** use `.code-rules/config.json` by default, preserve custom `--config` locations, and generate resolved rules through recoverable file updates. Replacements use the complete local identity; configuration and provenance retain their history. Verify CLI defaults, custom paths, recovery, docs, and runbooks.
3. **Simplified groups, implemented in the PR stacked on #5:** remove `localGroups`; discover local groups from `_group.json`; let local metadata describe a project group even when imported libraries also supply it. Otherwise preserve source-labeled library descriptions. Include local rules automatically, reject missing metadata, and keep source-scoped rule exceptions explicit. Ignore only the root `local/README.md` as directory documentation. Test and demonstrate starting locally, adding a library, and removing it again without moving local files.
4. **Project setup:** implement `init`, `local add group`, `local add rule`, and `add source`. Scaffold `.code-rules/local/README.md`, link canonical authoring guidance, preserve existing files, and offer missing-group creation interactively. Noninteractive commands take explicit inputs. Share authoring helpers and templates. Validate the empty-repo to local-rules to imported-library journey.
5. **Library authoring:** implement `library init`, `library add group`, `library add rule`, and `library check`, reusing authoring helpers. Add library-wide manifest, license, and notice setup. Validate an authored library by importing it into a consuming project.

## Release recommendation

The feature PRs above do not publish an executable. A separate release pass must choose and document the supported runtime/platforms, configure package identity and executable entry points, include distributable templates, and automate versioned publication. Test the packed artifact from a fresh directory rather than relying on this checkout's dependencies. Validate installation, help, local-only setup, library import, offline build/check, and an upgrade against committed files. Make installation instructions and the docs site match the released package. Complete a real-project pilot before calling the first release ready.

The tool-update command, conflict-review prompt generator, and installable authoring skill remain proposed. Recommendation: defer them from the first release if documented package-manager upgrades, manual conflict-review instructions, and canonical authoring guidance cover those workflows. Do not advertise unimplemented commands as available.

Application-code enforcement is outside Code Rules. It is not missing launch functionality.
