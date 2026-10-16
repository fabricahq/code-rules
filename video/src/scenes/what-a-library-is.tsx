/** @fileoverview Scene 2: a Syllabus library is a Git repository of skills that adds versions, change notes, recommendation levels, and rules. */
import { Panel } from '../components/panel';
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Chip, Headline, Muted, Provenance } from '../components/text';
import { COLORS, FONTS } from '../theme';
import { cue, type SceneProps } from './scene-props';

const TREE = [
  { branch: '├── ', name: '.syllabus/', note: null },
  { branch: '│   ├── ', name: 'config.yaml', note: <span style={{ color: COLORS.action }}>library: {'{}'}</span> },
  { branch: '│   ├── ', name: 'skills/', note: null },
  { branch: '│   │   └── ', name: 'write-tests/', note: <Provenance>2.3.0 · recommended</Provenance> },
  { branch: '│   └── ', name: 'rules/', note: <Muted>more on these soon</Muted> },
  { branch: '└── ', name: 'changes/', note: <Muted>change notes</Muted> },
] as const;

/** Introduces the Syllabus library before the story uses one. */
export function WhatALibraryIs(props: SceneProps) {
  const features = [
    { label: 'A version for every skill', at: 0.3 },
    { label: 'Change notes with each release', at: 0.4 },
    { label: 'A recommendation level: required, recommended, or available', at: 0.52 },
  ] as const;
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Reveal at={0}>
        <Headline style={{ fontSize: 64 }}>A Syllabus library</Headline>
      </Reveal>
      <div style={{ display: 'grid', gridTemplateColumns: '1.05fr 1fr', gap: 48, alignItems: 'center' }}>
        <Reveal at={cue(props, 0.08)}>
          <Panel title="acme/payments-syllabus">
            <div style={{ fontFamily: FONTS.mono, fontSize: 27, lineHeight: 1.75 }}>
              {TREE.map((row) => (
                <div
                  key={row.name}
                  style={{ display: 'flex', justifyContent: 'space-between', gap: 24, whiteSpace: 'pre' }}
                >
                  <span>
                    <Muted>{row.branch}</Muted>
                    {row.name}
                  </span>
                  {row.note}
                </div>
              ))}
            </div>
          </Panel>
        </Reveal>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 18 }}>
          <Reveal at={cue(props, 0.18)}>
            <div style={{ fontSize: 32, color: COLORS.muted }}>An ordinary Git repository of skills, plus:</div>
          </Reveal>
          {features.map((feature) => (
            <Reveal key={feature.label} at={cue(props, feature.at)}>
              <Chip>{feature.label}</Chip>
            </Reveal>
          ))}
          <Reveal at={cue(props, 0.82)}>
            <Chip isDimmed style={{ borderStyle: 'dashed' }}>
              Rules · more on these soon
            </Chip>
          </Reveal>
        </div>
      </div>
    </SceneFrame>
  );
}
