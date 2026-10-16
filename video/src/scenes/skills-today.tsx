/** @fileoverview Scene 2: skills are the common way to guide agents, but they're coarse-grained and loosely versioned. */
import { Panel } from '../components/panel';
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Chip, Muted } from '../components/text';
import { COLORS, FONTS } from '../theme';
import { cue, type SceneProps } from './scene-props';

const INSIGHTS = [
  'Prefer end-to-end tests for user flows',
  'Use property tests for invariants',
  'Never mock the database',
  'Quarantine flaky tests',
  'Keep test output short',
] as const;

/** Shows one skill bundling many insights, with no version anyone checks. */
export function SkillsToday(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Reveal at={0}>
        <Panel
          title={
            <span style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span>skills/testing/SKILL.md</span>
              <Reveal at={cue(props, 0.68)}>
                <span style={{ color: COLORS.warning }}>version: latest?</span>
              </Reveal>
            </span>
          }
        >
          <div style={{ fontFamily: FONTS.mono, fontSize: 30, lineHeight: 1.65 }}>
            <div># Testing</div>
            {INSIGHTS.map((insight, index) => (
              <Reveal key={insight} at={cue(props, 0.12 + index * 0.05)}>
                - {insight}
              </Reveal>
            ))}
            <Reveal at={cue(props, 0.38)}>
              <Muted>- … 40 more insights</Muted>
            </Reveal>
          </div>
        </Panel>
      </Reveal>
      <div style={{ display: 'flex', gap: 24 }}>
        <Reveal at={cue(props, 0.4)}>
          <Chip>Coarse-grained: all or nothing</Chip>
        </Reveal>
        <Reveal at={cue(props, 0.7)}>
          <Chip>Loosely versioned: take the latest</Chip>
        </Reveal>
      </div>
    </SceneFrame>
  );
}
