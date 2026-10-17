# Personal skills override repo skills, with a warning only

Skill decisions layer from library to org library to project to the user's personal skills, and a personal skill with the same name as a repo skill wins, even over a required skill. Trail Guide only warns, in `sync`, `update`, and `check`, naming what each agent does (Claude Code uses the personal skill, Codex shows both, and Grok Build and VS Code don't document which they use), and the generated skill index doesn't tell agents to prefer the repo's skill. Personal skill folders sit outside the repo, agents resolve the clash in different ways, and Trail Guide manages which skills a repo installs rather than enforcing how agents use them.

## Considered Options

- **Tell agents in the skill index to prefer the repo's skill:** dropped, because it's an instruction Trail Guide can't make agents follow, and it overrules a choice the user made deliberately.
