/** @fileoverview Scene 4: adding the payments team's Syllabus library as a source, with their testing skill. */
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Terminal } from '../components/terminal';
import { COLORS } from '../theme';
import { cue, type SceneProps } from './scene-props';

/** The command that records the payments library as a source. */
export const ADD_SKILL_COMMAND = [
  'syllabus project add source payments \\',
  '    --repository https://github.com/acme/payments-syllabus.git \\',
  '    --skills skills/write-tests',
].join('\n');

/** Shows the add-source command being typed. */
export function AddTheirLibrary(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Reveal at={0}>
        <div style={{ fontSize: 40, color: COLORS.muted }}>The payments team publishes a Syllabus library.</div>
      </Reveal>
      <Terminal lines={[{ kind: 'command', text: ADD_SKILL_COMMAND, at: cue(props, 0.12) }]} />
    </SceneFrame>
  );
}
