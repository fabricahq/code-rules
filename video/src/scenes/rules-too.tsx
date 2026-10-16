/** @fileoverview Scene 6: the payments library also publishes rules, so the project adds them to the same source. */
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Terminal, typingFrames, type TerminalLine } from '../components/terminal';
import { Check, Provenance } from '../components/text';
import { COLORS } from '../theme';
import { ADD_SKILL_COMMAND } from './add-their-library';
import { cue, type SceneProps } from './scene-props';

const RULES_FLAG = "--rules 'practices/critical-operations/*'";
const ADD_RULES_COMMAND = `${ADD_SKILL_COMMAND} \\\n    ${RULES_FLAG}`;
const SYNC_COMMAND = 'syllabus project sync';

const RULES = [
  { id: 'confirm-critical-writes', version: '2.1.0' },
  { id: 'make-retries-idempotent', version: '1.3.0' },
  { id: 'record-audit-events', version: '1.0.2' },
  { id: 'alert-on-missing-confirmation', version: '1.1.0' },
] as const;

/** Adds the critical-operations rules and shows each one sync imports with its version. */
export function RulesToo(props: SceneProps) {
  const addAt = cue(props, 0.32);
  const syncAt = addAt + typingFrames(ADD_RULES_COMMAND) + 10;
  const outputAt = syncAt + typingFrames(SYNC_COMMAND) + 10;
  const lines: Array<TerminalLine> = [
    { kind: 'command', text: ADD_RULES_COMMAND, at: addAt },
    { kind: 'command', text: SYNC_COMMAND, at: syncAt },
    ...RULES.map((rule, index): TerminalLine => ({
      kind: 'output',
      at: outputAt + index * 6,
      content: (
        <>
          <Check /> {rule.id.padEnd(31)}
          <Provenance>payments · {rule.version}</Provenance>
        </>
      ),
    })),
  ];
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Reveal at={cue(props, 0.12)}>
        <div style={{ fontSize: 40, color: COLORS.muted }}>
          Their library also publishes rules: <span style={{ color: COLORS.ink }}>practices/critical-operations</span>
        </div>
      </Reveal>
      <Terminal lines={lines} highlight={RULES_FLAG} />
    </SceneFrame>
  );
}
