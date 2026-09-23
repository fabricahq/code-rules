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

  Answer the prompts for the repository URL, ref, and groups.

For agents and scripts (no prompts):
  code-rules project add library team \
    --repository https://github.com/example/rules.git \
    --ref '>= 1.2.0, < 2.0.0' \
    --groups practices/testing --groups techs/go \
    --non-interactive

Replace team with an alias you choose for this library in your project.
Replace the example repository, ref, and group paths with your library's values.
Add --json for machine-readable output; --json also disables prompts.

Choosing a ref:
  Use an exact tag (v1.2.3), a full commit SHA, or a range (>= 1.2.0, < 2.0.0).
  Plain versions are literal tags. For ranges, sync selects the highest matching release tag.
  Quote ranges in the shell. Branch names and abbreviated commits are not supported.
  Use refs/tags/<name> for a literal tag that looks like a range.

Choosing groups:
  Specific groups:  --groups practices/testing --groups techs/go
  All groups:       --groups '*'
  All practices:    --groups 'practices/*'
  All technologies: --groups 'techs/*'

Find group paths in the library's docs or its practices/ and techs/ directories.
Use paths that exist at your chosen ref. Do not mix a wildcard with specific paths.
At the interactive prompt, enter paths separated by commas, or one wildcard.
On the command line, repeat --groups for each path, as shown above.

After adding the library, fetch its rules and build guidance:
  code-rules project sync`
