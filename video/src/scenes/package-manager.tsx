/** @fileoverview Scene 4: an organization curates skills in one library, sets their levels, and projects install them with one command. */
import { Panel } from '../components/panel';
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Terminal, typingFrames } from '../components/terminal';
import { Check, Muted } from '../components/text';
import { COLORS, FONTS } from '../theme';
import { cue, type SceneProps } from './scene-props';

const SKILLS = [
  { name: 'write-tests', level: 'required', note: null },
  { name: 'review-change', level: 'required', note: null },
  { name: 'release-checklist', level: 'recommended', note: null },
  { name: 'tdd', level: 'recommended', note: 'curated from mattpocock/skills' },
] as const;

const SYNC_COMMAND = 'trailguide project sync';

/** Shows an org library's skills with their levels, then one sync installing them. */
export function PackageManager(props: SceneProps) {
  const commandAt = cue(props, 0.68);
  const outputAt = commandAt + typingFrames(SYNC_COMMAND) + 10;
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Reveal at={0}>
        <Panel title="acme/.trailguide" tag="YOUR ORGANIZATION'S LIBRARY">
          <div style={{ fontFamily: FONTS.mono, fontSize: 30, lineHeight: 1.7 }}>
            {SKILLS.map((skill, index) => (
              <Reveal key={skill.name} at={cue(props, 0.12 + index * 0.06)}>
                <div style={{ display: 'flex', gap: 40, whiteSpace: 'pre' }}>
                  <span style={{ width: 360 }}>{skill.name}</span>
                  <Reveal at={cue(props, 0.42)}>
                    <span style={{ color: skill.level === 'required' ? COLORS.success : COLORS.action, width: 220 }}>
                      {skill.level}
                    </span>
                  </Reveal>
                  {skill.note === null ? null : <Muted>{skill.note}</Muted>}
                </div>
              </Reveal>
            ))}
          </div>
        </Panel>
      </Reveal>
      <Terminal
        lines={[
          { kind: 'command', text: SYNC_COMMAND, at: commandAt },
          {
            kind: 'output',
            at: outputAt,
            content: (
              <>
                <Check />
                {' 4 skills  → .claude/skills  .agents/skills'}
              </>
            ),
          },
        ]}
      />
    </SceneFrame>
  );
}
