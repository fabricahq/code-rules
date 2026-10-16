// Explain source selection consistently in command help and before interactive prompts.

package cli

import "fmt"

const sourceSelectionGuidance = `Choose rules from a library with rule selectors:
  techs/go/*                  Every rule in one group (example path).
  practices/testing/retries   One rule, without the rest of its group.
  *                           Every rule.
  practices/*, techs/*        Every practice or technology group.

Choose skills with skill IDs, such as skills/build-feature, recommended for
the skills a library recommends, or * for every skill the source offers. A
library's required skills install either way. A source can be a library, or
any Git repository with skill folders; a repository without library releases
needs --ref.
See syllabus project add source --help.

Example source:
  Repository: https://github.com/example/rules.git
  Rules: practices/testing/*, techs/go/*
  Skills: skills/build-feature`

func sourceSelectionIntroduction(name string) string {
	return fmt.Sprintf(`Adding a source as: %s

- The name %q identifies this source in your project configuration.

This command updates your project configuration. Run syllabus project sync afterward to fetch the source, build guidance, and install its skills.

%s

Enter the source's details below.

`, name, name, sourceSelectionGuidance)
}

const sourceSelectionHelp = `Add a source of rules and skills to the project configuration without fetching
it. A source is a library, or any other Git repository with skill folders; sync
detects which. Run from your project root after syllabus project init.

Interactive setup:
  syllabus project add source team

  Answer the prompts for the repository URL, rules, and skills.

For agents and scripts (no prompts):
  syllabus project add source team \
    --repository https://github.com/example/rules.git \
    --rules 'practices/testing/*' --rules 'techs/go/*' \
    --skills skills/build-feature \
    --non-interactive

Replace team with a name you choose for this source in your project.
Replace the example repository, selectors, and skill IDs with your source's.
Add --json for machine-readable output; --json also disables prompts.

Choosing rules (from a library only):
  One group:        --rules 'techs/go/*'
  One rule:         --rules practices/testing/verify-retry-limits
  Every rule:       --rules '*'
  All practices:    --rules 'practices/*'
  All technologies: --rules 'techs/*'

Quote * and selectors that end in /* so your shell doesn't expand them. Repeat
--rules for each selector. A group selector also takes rules the library adds
to the group later, on update; a rule ID takes only that rule.

Choosing skills:
  One skill:        --skills skills/build-feature
  Recommended:      --skills recommended
  Every skill:      --skills '*'

A library marks each of its skills required, recommended, or offered. Its
required skills install whatever you choose, so a source without --rules or
--skills installs only those. A library's skill IDs are its .syllabus/skills/
folders, such as skills/build-feature. In another repository, a skill is any
folder with a SKILL.md, named by its path from the repository root, such as
tools/review; * takes every one outside folders whose names start with a dot.

Taking one revision:
  Without --ref, the source follows the library's newest library release when
  the project updates. --ref takes the source exactly as it was at one tag, such
  as the library release tag release/5, or a full commit SHA. A repository
  without release/<number> tags, such as one that isn't a library, needs --ref.
  --ref can't be combined with pins. Branch names and abbreviated commits are
  not supported.

After adding the source, fetch it, build guidance, and install its skills:
  syllabus project sync`
