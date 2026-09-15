/** @fileoverview Renders shared authoring templates without inventing engineering policy. */

import { readFile } from 'node:fs/promises';
import { stringify } from 'yaml';
import { rule } from '../builds/rule-document';
import { groupMetadata } from '../formats/validation';
import type { GroupMetadata } from '../formats/validation';

/** Required author-supplied discovery and consequence metadata for a rule. */
export type RuleMetadata = {
  readonly title: string;
  readonly whenToRead: string;
  readonly impact: string;
  readonly impactDescription: string;
};

/** Short project-owned orientation; repeated init preserves the author's existing version. */
export const localReadme = `# Local rules

Put project rules under techs/<group>/ or practices/<group>/.
Describe each local group in _group.json with its name, description, and whenToRead cues.
Local rules join your project's rule set automatically. Imported rules in the same group join them; local group metadata supplies the project's description.
Use explicit exclusions or replacements in config.json to override imported rules.

Follow the [canonical rule rubric and template](https://github.com/fabricahq/code-rules/blob/main/docs/src/content/docs/reference/rule-authoring.md).
Complete drafts before building. Run build after local edits, or sync after adding or updating a library source.
`;

/** Validate and serialize complete group metadata with readable, stable JSON formatting. */
export function renderGroup(metadata: GroupMetadata): string {
  const text = JSON.stringify(metadata, null, 2) + '\n';
  groupMetadata(text, '_group.json');
  if (metadata.whenToRead.length === 0)
    throw new Error('Provide at least one group when-to-read cue.');
  return text;
}

/** Render an explicit body or unfinished canonical draft, validating metadata and escaping YAML values. */
export async function renderRuleDraft(
  id: string,
  metadata: RuleMetadata,
  body: string | undefined,
): Promise<string> {
  const template = await readFile(
    new URL('./rule-template.md', import.meta.url),
    'utf8',
  );
  const draft = template
    .slice(template.indexOf('\n---\n', 4) + 5)
    .replace(
      '## <Short action-oriented title>',
      '## ' + metadata.title.replace(/[\r\n]/gu, ' '),
    );
  const text = `---\n${stringify(metadata)}---\n\n${body ?? draft}`;
  rule(text, `${id}.md`, 'local');
  return text.endsWith('\n') ? text : text + '\n';
}
