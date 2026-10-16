/** @fileoverview Scene 10: a skill becomes a playbook that cites the rules it applies. */
import { Panel } from '../components/panel';
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Headline, Muted } from '../components/text';
import { COLORS, FONTS } from '../theme';
import { cue, type SceneProps } from './scene-props';

const CITED_RULES = [
  'practices/verification/give-agents-a-way-to-check-their-work',
  'practices/verification/test-real-user-flows',
  'practices/verification/keep-test-output-short',
] as const;

/** Shows a playbook skill's front matter citing three of the new rules. */
export function SkillsAsPlaybooks(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Reveal at={0}>
        <Headline style={{ fontSize: 64 }}>Skills become playbooks that cite rules.</Headline>
      </Reveal>
      <Reveal at={cue(props, 0.15)}>
        <Panel title="skills/verify-changes/SKILL.md">
          <div style={{ fontFamily: FONTS.mono, fontSize: 28, lineHeight: 1.6, whiteSpace: 'pre' }}>
            <Muted>---</Muted>
            <div>
              <span style={{ color: COLORS.action }}>name</span>: verify-changes
            </div>
            <div>
              <span style={{ color: COLORS.action }}>trailguide-skill-type</span>: playbook
            </div>
            <div>
              <span style={{ color: COLORS.action }}>trailguide-rules-used</span>:
            </div>
            {CITED_RULES.map((rule, index) => (
              <Reveal key={rule} at={cue(props, 0.4 + index * 0.1)}>
                {'  - '}
                <span style={{ color: COLORS.success }}>{rule}</span>
              </Reveal>
            ))}
            <Muted>---</Muted>
            <div>1. Decide how you’ll check the change before you write it.</div>
          </div>
        </Panel>
      </Reveal>
    </SceneFrame>
  );
}
