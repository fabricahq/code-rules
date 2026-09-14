/** @fileoverview Parses rule documents while preserving raw frontmatter and body text for generated rule files. */

import { parseDocument } from 'yaml';
import type { Rule } from './types';
import {
  field,
  invalid,
  nonempty,
  object,
  ruleGroup,
  strings,
} from './validation';

/** Original frontmatter and body text; parsing must not replace either with reserialized content. */
type RuleText = { readonly metadata: string; readonly body: string };

/** Split required YAML frontmatter from Markdown without normalizing newlines or whitespace. */
function documentText(text: string, location: string): RuleText {
  const match = /^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)([\s\S]*)$/u.exec(text);
  const metadata = match?.[1];
  const body = match?.[2];
  if (metadata === undefined || body === undefined)
    return invalid(location, 'expected YAML frontmatter followed by Markdown');
  return { metadata, body };
}

/** Read a YAML object, rejecting duplicate keys, parse errors, and aliases at the rule location. */
function metadataObject(
  metadata: string,
  location: string,
): Record<string, unknown> {
  const parsed = parseDocument(metadata, { uniqueKeys: true });
  if (parsed.errors.length)
    return invalid(location, `invalid YAML: ${parsed.errors[0]?.message}`);
  let raw: unknown;
  try {
    raw = parsed.toJS({ maxAliasCount: 0 });
  } catch {
    return invalid(location, 'YAML aliases are unsupported');
  }
  return object(raw, location);
}

/** Validate required rule metadata and return its selection fields, leaving additional attribution fields untouched. */
function validateRuleMetadata(
  data: Record<string, unknown>,
  location: string,
): Pick<Rule, 'title' | 'impact' | 'impactDescription' | 'whenToRead'> {
  const title = nonempty(field(data, 'title'), `${location}.title`);
  const impact = nonempty(field(data, 'impact'), `${location}.impact`);
  if (
    ![
      'CRITICAL',
      'HIGH',
      'MEDIUM-HIGH',
      'MEDIUM',
      'LOW-MEDIUM',
      'LOW',
    ].includes(impact)
  )
    return invalid(location, `unknown impact ${impact}`);
  const impactDescription = nonempty(
    field(data, 'impactDescription'),
    `${location}.impactDescription`,
  );
  const tags = field(data, 'tags');
  if (tags !== undefined) {
    if (typeof tags === 'string') nonempty(tags, `${location}.tags`);
    else strings(tags, `${location}.tags`);
  }
  const whenToRead = nonempty(
    field(data, 'whenToRead'),
    `${location}.whenToRead`,
  );
  return { title, impact, impactDescription, whenToRead };
}

/**
 * Parse a rule into its source-qualified identity, metadata text, and Markdown body.
 * Preserves extra frontmatter fields and body whitespace; requires supported impact and non-blank content.
 * Throws an invalid-input BuildError for invalid paths/frontmatter, duplicate YAML keys, or aliases.
 */
export function rule(text: string, path: string, source: string): Rule {
  const location = `${source}:${path}`;
  const group = ruleGroup(path, location);
  const { metadata, body } = documentText(text, location);
  const data = metadataObject(metadata, location);
  const selection = validateRuleMetadata(data, location);
  nonempty(body, `${location}.body`);
  return {
    id: `${source}:${path.slice(0, -3)}`,
    group,
    path,
    ...selection,
    metadata,
    body,
  };
}
