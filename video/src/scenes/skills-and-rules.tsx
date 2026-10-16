/** @fileoverview Scene 7: rules are finer-grained than skills, and a project takes either from many sources or writes its own. */
import { Panel } from '../components/panel';
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Headline, Muted } from '../components/text';
import { COLORS, FONTS } from '../theme';
import { cue, type SceneProps } from './scene-props';

const SOURCES = [
  {
    name: 'payments',
    color: COLORS.agent,
    rows: [
      ['skill', 'write-tests'],
      ['rules', 'critical-operations/*'],
    ],
  },
  { name: 'fabrica', color: COLORS.action, rows: [['rules', 'techs/typescript/*']] },
  {
    name: 'this project',
    color: COLORS.ink,
    rows: [
      ['skill', 'release-checklist'],
      ['rule', 'checkout/refund-once'],
    ],
  },
] as const;

/** Contrasts skills and rules, and shows three sources side by side. */
export function SkillsAndRules(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Reveal at={0}>
        <Headline style={{ fontSize: 60 }}>
          Skills are playbooks.{' '}
          <span style={{ color: COLORS.muted }}>Rules are one best practice each, with their own version.</span>
        </Headline>
      </Reveal>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 32 }}>
        {SOURCES.map((source, index) => (
          <Reveal key={source.name} at={cue(props, 0.5 + index * 0.08)}>
            <Panel title={<span style={{ color: source.color }}>{source.name}</span>} style={{ height: '100%' }}>
              <div style={{ fontFamily: FONTS.mono, fontSize: 26, lineHeight: 1.8 }}>
                {source.rows.map(([kind, id]) => (
                  <div key={id}>
                    <Muted>{kind.padEnd(6)}</Muted>
                    {id}
                  </div>
                ))}
              </div>
            </Panel>
          </Reveal>
        ))}
      </div>
    </SceneFrame>
  );
}
