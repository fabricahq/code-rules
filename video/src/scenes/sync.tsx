/** @fileoverview Scene 5: sync installs the skill into every agent's folder and records its source and version. */
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Terminal, typingFrames } from '../components/terminal';
import { Check, Chip, Muted, Provenance } from '../components/text';
import { FONTS } from '../theme';
import { cue, type SceneProps } from './scene-props';

const SYNC_COMMAND = 'syllabus project sync';

/** Shows sync's output and the two folders it installed into. */
export function Sync(props: SceneProps) {
  const commandAt = cue(props, 0.05);
  const outputAt = commandAt + typingFrames(SYNC_COMMAND) + 12;
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Terminal
        lines={[
          { kind: 'command', text: SYNC_COMMAND, at: commandAt },
          {
            kind: 'output',
            at: outputAt,
            content: (
              <>
                <Check />
                {' skills/write-tests     '}
                <Provenance>payments · release 12</Provenance>
              </>
            ),
          },
          { kind: 'output', at: outputAt + 8, content: <Muted>{'    → .claude/skills/write-tests'}</Muted> },
          { kind: 'output', at: outputAt + 14, content: <Muted>{'    → .agents/skills/write-tests'}</Muted> },
        ]}
      />
      <div style={{ display: 'flex', gap: 24 }}>
        {['Claude Code reads .claude/skills', 'Other agents read .agents/skills'].map((label, index) => (
          <Reveal key={label} at={cue(props, 0.55 + index * 0.08)}>
            <Chip style={{ fontFamily: FONTS.mono, fontSize: 26 }}>{label}</Chip>
          </Reveal>
        ))}
      </div>
    </SceneFrame>
  );
}
