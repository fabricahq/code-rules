// Explain library selection consistently in command help and before interactive prompts.

package cli

import "fmt"

const librarySelectionGuidance = `Choose groups from that library:
  practices/testing, techs/go  Only these groups (example paths).
  *                           All groups.
  practices/*                 All practice groups.
  techs/*                     All technology groups.

Find group paths in the library's documentation or its practices/ and techs/
directories. Use paths from that library, not this project. You can combine
a wildcard with paths, such as techs/*, practices/testing.

To import single rules without the rest of their group, use --rules. To import
the library as it was at one tag or commit, use --ref. See
code-rules project add library --help.

Example library selection:
  Repository: https://github.com/example/rules.git
  Groups: practices/testing, techs/go`

func librarySelectionIntroduction(alias string) string {
	return fmt.Sprintf(`Adding a library as: %s

- The alias %q identifies this library in your project configuration.

This command updates your project configuration. Run code-rules project sync afterward to fetch the rules and build guidance.

%s

Enter your library's details below.

`, alias, alias, librarySelectionGuidance)
}

const librarySelectionHelp = `Add a shared library to the project configuration without fetching its rules.
Run from your project root after code-rules project init.

Interactive setup:
  code-rules project add library team

  Answer the prompts for the repository URL and groups.

For agents and scripts (no prompts):
  code-rules project add library team \
    --repository https://github.com/example/rules.git \
    --groups practices/testing --groups techs/go \
    --non-interactive

Replace team with an alias you choose for this library in your project.
Replace the example repository and group paths with your library's values.
Add --json for machine-readable output; --json also disables prompts.

Choosing groups:
  Specific groups:  --groups practices/testing --groups techs/go
  All groups:       --groups '*'
  All practices:    --groups 'practices/*'
  All technologies: --groups 'techs/*'

Find group paths in the library's docs or its practices/ and techs/ directories.
Use paths that exist in the library. A wildcard can be combined with paths.
At the interactive prompt, enter paths and wildcards separated by commas.
On the command line, repeat --groups for each path, as shown above. The
configuration records each group as a selector in rules.include, such as
techs/go/*, so rules the library adds to the group later join on update.

Choosing individual rules:
  To import single rules without the rest of their group, repeat --rules:
    --rules practices/testing/verify-retry-limits --rules techs/go/wrap-errors
  Rules the library later adds to those groups don't join. Supply at least one
  --groups or --rules.

Importing one revision (optional and advanced):
  Without --ref, each rule follows its newest version when the project updates.
  --ref imports the library exactly as it was at one tag, such as the library
  release tag release/5, or a full commit SHA. It can't be combined with pins.
  Branch names and abbreviated commits are not supported.

After adding the library, fetch its rules and build guidance:
  code-rules project sync`
