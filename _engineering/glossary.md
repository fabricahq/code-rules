# Syllabus

Syllabus manages which engineering rules and agent skills a repo takes from shared sources and from its own authors, and delivers them where coding agents read them. A repo can be a project, a library, or both.

## Language

### Content

**Rule**:
One independently adoptable engineering expectation, stating what to do, when it applies, and what evidence shows compliance.
_Avoid_: Guideline, policy, standard

**Rule ID**:
A rule's rule group ID plus its own name, unique within its source; the source's name tells apart rules with the same ID from different sources.
_Avoid_: Rule name, slug, path

**Rule group**:
A named collection of related rules that tells agents when to read them. There are two kinds of rule groups: technology groups and practice groups.
_Avoid_: Group on its own, category, rule set, package

**Technology group**:
A rule group about one named language, framework, tool, platform, or protocol.
_Avoid_: Tech stack, language group

**Practice group**:
A rule group about one concern that cuts across technologies, such as testing or error handling.
_Avoid_: Discipline, topic

**Skill**:
An Agent Skills folder whose instructions tell an agent how to carry out one kind of task, step by step, citing the rules each step applies.
_Avoid_: Prompt, command, workflow, playbook (that is a skill type)

**Skill ID**:
A skill's path within the repository that holds it. A skill's identity is its repository plus that path, so two sources that reach the same repository name the same skill.
_Avoid_: Skill name (that is the install name)

**Skill type**:
An optional label, playbook or review, saying whether a skill changes work or judges work that already exists; it never changes how Syllabus treats the skill.
_Avoid_: Skill kind, category

**Cited rule**:
A rule that a skill names as one it applies. A cited rule is reference material for that skill only and never joins the rules that govern all of a repo's work.
_Avoid_: Rule dependency, imported rule, required rule

**Skill dependency**:
Another skill that a skill uses, which installs alongside it.
_Avoid_: Sub-skill, child skill

**Asset**:
A supporting file, such as a diagram or sample data, that rules link to and that travels with them.
_Avoid_: Attachment, resource

### Repos, sources, and libraries

**Project**:
A repo that takes rules and skills for its own work and records exactly what it took.
_Avoid_: Consumer, client, workspace

**Library**:
A repo whose manifest publishes its authored rules and skills for other repos. A repo can be a library and a project at once.
_Avoid_: Rule library, catalog, preset, package

**Manifest**:
The part of a repo's configuration that makes it a library: what it publishes and the level of each skill it offers.
_Avoid_: Library metadata, preset

**Org library**:
A library an organization publishes as the shared default for its projects.
_Avoid_: Preset, org defaults, base config

**Source**:
A Git repository that a repo takes rules or skills from, declared once and shared by its rule and skill selections. A source has no declared type; Syllabus detects whether it has a manifest and library releases.
_Avoid_: Dependency, import, remote, plain repository

**Non-library source**:
A source without a manifest, such as a third party's skill collection; every skill in it counts as offered.
_Avoid_: Plain repository, raw repository, bare repository, skill repository

**Borrowed source**:
A source a library declares so that it can place that source's skills in its own levels.
_Avoid_: Upstream, transitive source

**Borrowed skill**:
A skill a library lists in one of its levels but takes from a borrowed source instead of authoring it. A library must list every skill it borrows.
_Avoid_: Endorsed skill, endorsement, re-exported skill

**Authored content**:
Rules, rule groups, skills, and assets written in the repo itself rather than taken from a source; a library publishes exactly its authored content.
_Avoid_: Local rules, local skills, custom rules

**Vendor snapshot**:
The managed copy of what a repo takes from one source, kept in the repo with a record of the release or commit it came from.
_Avoid_: Imported content, import, cache

**Personal skill**:
A skill a user installs for themselves outside any repo, which their agents can load in every repo they work in.
_Avoid_: User skill, global skill, local skill

### Selection

**Selector**:
An expression that names one rule or skill by ID, or everything under an ID prefix: a rule group, every rule group of one type, or a whole source.
_Avoid_: Glob, pattern, filter

**Include**:
The selectors and skill levels that name what a repo takes from a source for its own use.
_Avoid_: Adopt, use, select, import

**Exclude**:
A recorded decision, with a reason, to leave out a rule or skill that an include or a level would otherwise bring in.
_Avoid_: Ignore, skip, disable, opt out

**Replacement**:
An authored rule or skill that takes the place of an excluded one.
_Avoid_: Override, substitute

**Fork**:
A replacement that began as a copy of the excluded rule or skill and records the version it was based on.
_Avoid_: Copy, patch, override

**Skill level**:
How strongly a library asks its projects to install one of its skills: required, recommended, or offered. A project sees only the levels of the libraries it takes from directly; levels don't chain through borrowed sources.
_Avoid_: Tier, endorsement, policy, preset, priority

**Required skill**:
A skill that every repo taking anything from its library installs, whatever that repo includes; no project can exclude or fork it.
_Avoid_: Mandatory skill, enforced skill, default skill

**Recommended skill**:
A skill a library suggests to every project, installed when a project includes the library's recommendations.
_Avoid_: Default skill, suggested skill, endorsed skill

**Offered skill**:
A skill a library makes available without suggesting it, installed only when a project names it or includes everything offered. A library skill listed in no level is offered.
_Avoid_: Optional skill, available skill, catalog skill

**Skill precedence**:
The order in which skill decisions layer: library, then org library, then project, then personal skills, each overriding the ones before it. Required skills are a floor that the project cannot remove.
_Avoid_: Inheritance, extends, cascade

**Install name**:
The name an installed skill goes by in an agent's skill folder, which is the skill's own declared name unless the project renames it.
_Avoid_: Skill ID, alias

**Name clash**:
Two included skills with the same install name, which the project resolves by renaming one or excluding one.
_Avoid_: Collision, conflict, duplicate

**Rename**:
A project's choice of a different install name for one skill. The skill ID stays the same, and mentions of the old name in skill text are not rewritten.
_Avoid_: Alias, remap

**Shadowed skill**:
A repo skill that shares its install name with one of the user's personal skills; each agent resolves the pair its own way, and Syllabus only warns.
_Avoid_: Overridden skill, conflicting skill

### Versions and locking

**Version**:
The semantic version of one published rule or skill, whose bump says how much its obligation changed. Rules and skills that no library release published have no version.
_Avoid_: Library version, release number

**Change note**:
An author's record of one rule or skill change, with its size and a one-line summary, which the next library release turns into a new version.
_Avoid_: Changelog entry, commit message

**Library release**:
A numbered publication of a library that turns all its pending change notes into new versions at once. Its number is a reference, not a version of the library.
_Avoid_: Library version, tag

**Retired rule**:
A rule a library has stopped publishing, because another rule replaces it or the practice is no longer recommended.
_Avoid_: Deprecated rule, deleted rule, removed rule

**Ref**:
The branch, tag, commit, or library release a repo names for a source, which each project resolves to a commit when it updates. A project that declares a repository as its own source overrides the ref any library names for that repository.
_Avoid_: Version, revision, pin

**Pin**:
A project's hold on one rule at a chosen version, with a reason. Skills are never pinned.
_Avoid_: Lock, freeze, keep

**Lock**:
The record of exactly which library release or commit, and which rule versions, a project took from each source at its last update. All of a source's skills come from its one locked revision.
_Avoid_: Lockfile, snapshot, pin

**Update**:
The deliberate step that moves a project's sources forward along their refs and records a new lock, after a preview of every change.
_Avoid_: Upgrade, refresh, sync

**Sync**:
The step that rebuilds a repo's vendor snapshots, generated guidance, and installed skills from its configuration and lock, without ever moving a source.
_Avoid_: Install, update, build, pull

### Installation

**Skill target**:
An agent's skill folder that Syllabus writes installed skills into, chosen by each repo.
_Avoid_: Install target, output directory, harness

**Installed skill**:
The copy of an included skill that Syllabus writes into each skill target under its install name, linked to the repo's versions of the rules it cites.
_Avoid_: Deployed skill, skill copy

**Generated guidance**:
The files Syllabus writes for agents from the selected rules and installed skills: the rule index, rule group pages, and skill index. Nobody edits them by hand.
_Avoid_: Output, build artifacts, compiled rules
