// Explain library selection consistently in command help and before interactive prompts.

package cli

import "fmt"

const librarySelectionGuidance = `Choose a ref:
  v1.2.3                Use this exact tag.
  <full commit SHA>     Use this exact commit.
  >= 1.2.0, < 2.0.0     Let sync select the highest matching release tag.

Plain versions such as 1.2.3 are literal tags. Start a range with an operator
such as >=, =, or ~>. Branch names and abbreviated commits are not supported.

Choose groups from that library:
  practices/testing, techs/go  Only these groups (example paths).
  *                           All groups.
  practices/*                 All practice groups.
  techs/*                     All technology groups.

Find group paths in the library's documentation or its practices/ and techs/
directories at the chosen revision. Use paths from that library, not this project.
Choose explicit paths or one wildcard; do not combine them.

Example library selection:
  Repository: https://github.com/example/rules.git
  Ref: v1.2.3
  Groups: practices/testing, techs/go`

func librarySelectionIntroduction(alias string) string {
	return fmt.Sprintf(`Adding a library as: %s

- The alias %q identifies this library in your project's configuration.

This command updates your config. Run code-rules project sync afterward to fetch the rules and build guidance.

%s

Enter your library's details below.

`, alias, alias, librarySelectionGuidance)
}

const librarySelectionHelp = `Record a shared library in this project's configuration without fetching it.
ALIAS is your project's short name for the library (e.g. team), not its repository name.

` + librarySelectionGuidance + `

For scripts and agents, supply --repository, --ref, and --groups.
Repeat --groups for individual paths; quote wildcards and version ranges in the shell.
Use --non-interactive to require explicit flags, or --json for structured output without prompts.
Use refs/tags/<name> for a literal tag that looks like a range.
The example repository, tag, and groups are illustrative; replace them with your library's values.
Run code-rules project sync afterward to fetch the rules and build guidance.`
