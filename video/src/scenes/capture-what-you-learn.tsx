/** @fileoverview Scene 9: record each insight as a new or updated rule or skill, then share it or keep it local. */
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Terminal } from '../components/terminal';
import { Chip } from '../components/text';
import { COLORS } from '../theme';
import { cue, type SceneProps } from './scene-props';

const ADD_RULE_COMMAND = [
  'syllabus project add rule practices/checkout/retry-declined-cards \\',
  '    --title "Retry declined cards only for soft declines" ...',
].join('\n');

/** Shows a new rule being created, then where it can go. */
export function CaptureWhatYouLearn(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Terminal lines={[{ kind: 'command', text: ADD_RULE_COMMAND, at: cue(props, 0.05) }]} />
      <div style={{ display: 'flex', gap: 24 }}>
        {['New rule', 'New skill', 'Update a rule or skill'].map((label, index) => (
          <Reveal key={label} at={cue(props, 0.3 + index * 0.08)}>
            <Chip>{label}</Chip>
          </Reveal>
        ))}
      </div>
      <Reveal at={cue(props, 0.62)}>
        <div style={{ fontSize: 44, letterSpacing: '-0.02em' }}>
          Share it in a library, <span style={{ color: COLORS.muted }}>or keep it in this project.</span>
        </div>
      </Reveal>
    </SceneFrame>
  );
}
