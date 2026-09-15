/** @fileoverview Defines command paths and option contracts shared by CLI parsing and help. */

/** One accepted long option; a null value denotes a switch rather than a value. */
export type Option = {
  readonly name: string;
  readonly value: string | null;
  readonly description: string;
  readonly repeat?: boolean;
};

/** Executable command contract, including its handler and command-specific guidance. */
export type Command = {
  readonly path: string;
  readonly summary: string;
  readonly argument: string | null;
  readonly options: readonly Option[];
  readonly details: readonly string[];
  readonly examples: readonly string[];
  readonly action:
    | { readonly type: 'project'; readonly kind: 'sync' | 'build' | 'check' }
    | {
        readonly type: 'authoring';
        readonly target: 'project' | 'library';
        readonly kind: 'init' | 'group' | 'rule' | 'source' | 'check';
      };
};

const config: Option = {
  name: 'config',
  value: 'path',
  description: 'Configuration file (default: .code-rules/config.json).',
};
const directory: Option = {
  name: 'directory',
  value: 'path',
  description: 'Library root (default: current directory).',
};
const nonInteractive: Option = {
  name: 'non-interactive',
  value: null,
  description: 'Fail on missing required inputs instead of prompting.',
};
const groupOptions: readonly Option[] = [
  {
    name: 'name',
    value: 'text',
    description: 'Group display name; prompted when omitted.',
  },
  {
    name: 'description',
    value: 'text',
    description: 'Group scope; prompted when omitted.',
  },
  {
    name: 'when-to-read',
    value: 'text',
    description:
      'Group reading cue; prompted when omitted. Repeat for more cues.',
    repeat: true,
  },
];
const ruleOptions: readonly Option[] = [
  {
    name: 'title',
    value: 'text',
    description: 'Action-oriented title; prompted when omitted.',
  },
  {
    name: 'when-to-read',
    value: 'text',
    description: 'Rule reading cue; prompted when omitted.',
  },
  {
    name: 'impact',
    value: 'level',
    description:
      'CRITICAL, HIGH, MEDIUM-HIGH, MEDIUM, LOW-MEDIUM, or LOW; prompted when omitted.',
  },
  {
    name: 'impact-description',
    value: 'text',
    description: 'Consequence the rule prevents; prompted when omitted.',
  },
  {
    name: 'body-file',
    value: 'path',
    description:
      'Read the rule body from a UTF-8 file; otherwise create a draft.',
  },
  {
    name: 'create-group',
    value: null,
    description: "Create the rule's group if missing.",
  },
  {
    name: 'group-name',
    value: 'text',
    description: 'New group name; requires --create-group.',
  },
  {
    name: 'group-description',
    value: 'text',
    description: 'New group scope; requires --create-group.',
  },
  {
    name: 'group-when-to-read',
    value: 'text',
    description: 'New group reading cue; repeatable; requires --create-group.',
    repeat: true,
  },
];
const prompts =
  'Omitted required metadata is prompted only on a terminal. Supply explicit flags in scripts or with --non-interactive.';
const groupExample =
  '--name Testing --description "Verify behavior" --when-to-read "Before changing behavior"';
const ruleExample =
  '--title "Limit retries" --when-to-read "When changing retries" --impact HIGH --impact-description "Avoid excess requests" --body-file retry-budget.md';

function authorGroup(target: 'project' | 'library'): Command {
  const prefix = target === 'project' ? 'local' : 'library';
  return {
    path: prefix + ' add group',
    summary: 'Create a ' + prefix + ' rule group.',
    argument: 'group-id',
    options: [
      target === 'project' ? config : directory,
      nonInteractive,
      ...groupOptions,
    ],
    action: { type: 'authoring', target, kind: 'group' },
    details: [
      'Use practices/<name> or techs/<name>. Creates _group.json; existing groups are never overwritten.',
      prompts,
      target === 'project'
        ? 'Local groups need no source declaration. Add a rule, then run code-rules build.'
        : 'Add a rule, then run code-rules library check. This command does not initialize Git or publish files.',
    ],
    examples: [
      'code-rules ' + prefix + ' add group practices/testing',
      'code-rules ' +
        prefix +
        ' add group practices/testing ' +
        groupExample +
        ' --non-interactive',
    ],
  };
}
function authorRule(target: 'project' | 'library'): Command {
  const prefix = target === 'project' ? 'local' : 'library';
  return {
    path: prefix + ' add rule',
    summary: 'Create a ' + prefix + ' rule from the canonical template.',
    argument: 'rule-id',
    options: [
      target === 'project' ? config : directory,
      nonInteractive,
      ...ruleOptions,
    ],
    action: { type: 'authoring', target, kind: 'rule' },
    details: [
      'Use practices/<group>/<rule> or techs/<group>/<rule>, without .md. The group must exist' +
        (target === 'project'
          ? ' locally or in an imported library.'
          : ' in the library.'),
      prompts,
      'For a missing group, interactive mode offers creation. Scripts must supply --create-group and the group metadata flags.',
      'Without --body-file, complete the generated draft before validation. Existing rules are never overwritten.',
      target === 'project'
        ? 'Local rules join the project automatically. Configure exclusions or replacements in config.json, then run code-rules build.'
        : "Remove the completed draft's code-rules:draft marker, then run code-rules library check.",
    ],
    examples: [
      'code-rules ' + prefix + ' add rule practices/testing/retry-budget',
      'code-rules ' +
        prefix +
        ' add rule practices/testing/retry-budget ' +
        ruleExample +
        ' --non-interactive',
    ],
  };
}

/** Available commands in discovery order; options here also govern accepted parser input. */
export const commands: readonly Command[] = [
  {
    path: 'init',
    summary: 'Set up .code-rules in your repository.',
    argument: null,
    action: { type: 'authoring', target: 'project', kind: 'init' },
    options: [config, nonInteractive],
    details: [
      'Creates an empty source configuration and local/README.md. Preserves existing files and runs without prompts.',
      'Add local groups and rules, then run code-rules build. Or add a library source and run code-rules sync.',
    ],
    examples: [
      'code-rules init',
      'code-rules init --config tools/rules/config.json',
    ],
  },
  {
    path: 'add source',
    summary: 'Add a library source to your project.',
    argument: 'alias',
    action: { type: 'authoring', target: 'project', kind: 'source' },
    options: [
      config,
      nonInteractive,
      {
        name: 'repository',
        value: 'url',
        description:
          'Git HTTPS or SSH repository address; prompted when omitted.',
      },
      {
        name: 'ref',
        value: 'tag-or-sha',
        description: 'Exact tag or commit; mutually exclusive with --version.',
      },
      {
        name: 'version',
        value: 'range',
        description:
          'npm semantic-version constraint; mutually exclusive with --ref.',
      },
      {
        name: 'groups',
        value: 'selector',
        description:
          'Group ID (repeatable), or one of *, practices/*, techs/*.',
        repeat: true,
      },
    ],
    details: [
      prompts,
      'Choose exactly one of --ref or --version. Quote wildcards and version ranges. Wildcard selectors must be used alone.',
      'Records configuration without fetching. Existing aliases are preserved; edit config.json explicitly to change a source. Run code-rules sync to fetch and generate rules.',
      'Use source exclusions and replacements in config.json for project exceptions.',
    ],
    examples: [
      'code-rules add source team',
      'code-rules add source team --repository https://github.com/example/rules.git --version "^1.2.0" --groups "*" --non-interactive',
      'code-rules add source team --repository https://gitlab.com/team/rules.git --ref v1.2.0 --groups practices/testing --groups techs/go --non-interactive',
    ],
  },
  authorGroup('project'),
  authorRule('project'),
  {
    path: 'sync',
    summary: 'Fetch selected libraries and generate resolved rules.',
    argument: null,
    action: { type: 'project', kind: 'sync' },
    options: [config],
    details: [
      'Resolves configured refs and version constraints again. Requires Git 2.30+ and access to each source through your existing Git credentials.',
      'Validates inputs before replacing vendor and generated files. Reports added, changed, and removed paths as JSON.',
    ],
    examples: [
      'code-rules sync',
      'code-rules sync --config tools/rules/config.json',
    ],
  },
  {
    path: 'build',
    summary: 'Generate resolved rules without fetching.',
    argument: null,
    action: { type: 'project', kind: 'build' },
    options: [config],
    details: [
      'Uses configuration, local rules, and the existing vendor snapshot. Works offline and preserves imported revisions.',
      'Run sync first if a source selection changed. Reports added, changed, and removed generated paths as JSON.',
    ],
    examples: [
      'code-rules build',
      'code-rules build --config tools/rules/config.json',
    ],
  },
  {
    path: 'check',
    summary: 'Validate rule inputs and check generated files.',
    argument: null,
    action: { type: 'project', kind: 'check' },
    options: [config],
    details: [
      'Works offline without writing files or checking newer remote versions. Reports stale, missing, or unexpected generated files as JSON.',
      'Exit 0: inputs and output agree. Exit 1: invalid inputs or stale output. Exit 2: invalid command usage.',
      'Use build to repair stale output, or sync after source selection changes. A successful check does not establish that application code follows the rules.',
    ],
    examples: [
      'code-rules check',
      'code-rules check --config tools/rules/config.json',
    ],
  },
  {
    path: 'library init',
    summary: 'Initialize a reusable rule library.',
    argument: null,
    action: { type: 'authoring', target: 'library', kind: 'init' },
    options: [
      directory,
      nonInteractive,
      {
        name: 'spdx',
        value: 'expression',
        description: 'Declared SPDX expression; requires --license-file.',
      },
      {
        name: 'license-file',
        value: 'path',
        description: 'Existing UTF-8 license text; requires --spdx.',
      },
      {
        name: 'notice-file',
        value: 'path',
        description: 'Optional UTF-8 notice; requires both license flags.',
      },
    ],
    details: [
      'Creates rule-library.json and README.md without prompts. With no license flags, leaves the license undeclared and reports a reminder.',
      'Copies supplied license and notice bytes to LICENSE.md and NOTICE.md. Existing files are never overwritten. To change an existing manifest, edit it explicitly.',
      'Relative input file paths are resolved from your working directory, independently of --directory. No Git repository is created or published.',
      'Add a group and rule, then run code-rules library check.',
    ],
    examples: [
      'code-rules library init',
      'code-rules library init --directory shared-rules',
      'code-rules library init --directory shared-rules --spdx MIT --license-file inputs/LICENSE.md --notice-file inputs/NOTICE.md',
    ],
  },
  authorGroup('library'),
  authorRule('library'),
  {
    path: 'library check',
    summary: 'Validate a reusable rule library without writing files.',
    argument: null,
    action: { type: 'authoring', target: 'library', kind: 'check' },
    options: [directory, nonInteractive],
    details: [
      'Checks the manifest, groups, rules, linked assets, and declared license files offline. Reports group and rule counts and warnings as JSON.',
      'Incomplete marked drafts and invalid files fail validation. Empty groups are valid; an undeclared license produces a warning.',
    ],
    examples: [
      'code-rules library check',
      'code-rules library check --directory shared-rules',
    ],
  },
];

/** Navigation-only command groups; executable children are defined in commands. */
export const commandGroups: Readonly<Record<string, string>> = {
  '': 'Manage versioned engineering rules for your codebase.',
  add: 'Add library sources to your project.',
  local: 'Create local groups and rules.',
  'local add': 'Create a local group or rule.',
  library: 'Create and validate a reusable rule library.',
  'library add': 'Create a library group or rule.',
};
