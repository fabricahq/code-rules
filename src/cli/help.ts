/** @fileoverview Renders plain-text command discovery, contextual usage, and copyable examples. */

import { commandGroups, commands } from './commands';
import type { Command } from './commands';

function wrap(text: string, width: number): string[] {
  const lines: string[] = [];
  let line = '';
  for (const word of text.split(/\s+/u)) {
    if (line && line.length + word.length + 1 > width) {
      lines.push(line);
      line = word;
    } else line += (line ? ' ' : '') + word;
  }
  if (line) lines.push(line);
  return lines;
}

function rows(entries: readonly (readonly [string, string])[]): string {
  const column = Math.max(...entries.map(([name]) => name.length)) + 4;
  return entries
    .map(([name, description]) => {
      const lines = wrap(description, 80 - column);
      return lines
        .map((line, index) =>
          index === 0
            ? '  ' + name.padEnd(column - 2) + line
            : ' '.repeat(column) + line,
        )
        .join('\n');
    })
    .join('\n');
}

function children(path: string): readonly (readonly [string, string])[] {
  const prefix = path ? path + ' ' : '';
  const entries = new Map<string, string>();
  for (const command of commands) {
    if (!command.path.startsWith(prefix)) continue;
    const suffix = command.path.slice(prefix.length);
    const first = suffix.split(' ')[0] ?? '';
    const child = prefix + first;
    const summary = commandGroups[child];
    if (summary === undefined) entries.set(suffix, command.summary);
    else if (
      commands.filter((entry) => entry.path.startsWith(child + ' ')).length ===
      1
    )
      entries.set(suffix, command.summary);
    else entries.set(first, summary);
  }
  return [...entries];
}

function example(command: string): string {
  if (command.length <= 76) return '  ' + command;
  return command
    .split(/ (?=--)/u)
    .map(
      (part, index, parts) =>
        (index === 0 ? '  ' : '    ') +
        part +
        (index < parts.length - 1 ? ' \\' : ''),
    )
    .join('\n');
}

/** Show discovery for a command group or usage for one command; formatting is independent of color support. */
export function renderHelp(path: string, command: Command | null): string {
  const invocation = 'code-rules' + (path ? ' ' + path : '');
  const title = command?.summary ?? commandGroups[path] ?? '';
  const usage = command
    ? invocation +
      (command.argument ? ' <' + command.argument + '>' : '') +
      ' [options]'
    : invocation + ' <command> [options]';
  const sections = [title, 'Usage:\n  ' + usage];
  if (command === null) {
    const entries = children(path);
    if (path === '') {
      sections.push(
        'Project commands:\n' +
          rows(entries.filter(([name]) => name !== 'library')),
      );
      sections.push(
        'Library commands:\n' +
          rows(entries.filter(([name]) => name === 'library')),
      );
    } else sections.push('Commands:\n' + rows(entries));
  }
  const options: Array<readonly [string, string]> =
    command?.options.map((option) => [
      '--' + option.name + (option.value ? ' <' + option.value + '>' : ''),
      option.description,
    ]) ?? [];
  options.push(['-h, --help', 'Show help.']);
  if (path === '')
    options.push(['-v, --version', 'Show the installed version.']);
  sections.push('Options:\n' + rows(options));
  if (command) {
    sections.push(
      command.details.map((text) => wrap(text, 80).join('\n')).join('\n\n'),
    );
    sections.push('Examples:\n' + command.examples.map(example).join('\n\n'));
  } else {
    sections.push(
      'Run "' + invocation + ' <command> --help" for usage and examples.',
    );
  }
  return sections.join('\n\n') + '\n';
}

/** A copyable command that narrows a usage error to the relevant help page. */
export function helpHint(path: string): string {
  return (
    'Run "code-rules' +
    (path ? ' ' + path : '') +
    ' --help" for usage and examples.'
  );
}
