/** @fileoverview Scene 6: a project mixes rules and skills from its organization, public libraries, and itself, and excludes what doesn't fit. */
import { Panel } from '../components/panel';
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Headline, Muted } from '../components/text';
import { COLORS, FONTS } from '../theme';
import { cue, type SceneProps } from './scene-props';

/** One row of what a project takes from a source. */
type SourceRow = { kind: string; id: string; isExcluded: boolean };

const SOURCES: ReadonlyArray<{ name: string; label: string; color: string; rows: ReadonlyArray<SourceRow> }> = [
  {
    name: 'acme',
    label: 'your organization',
    color: COLORS.success,
    rows: [
      { kind: 'skill', id: 'write-tests', isExcluded: false },
      { kind: 'rules', id: 'practices/*', isExcluded: false },
      { kind: 'rule', id: 'api/pagination', isExcluded: true },
    ],
  },
  {
    name: 'fabrica',
    label: 'a public library',
    color: COLORS.action,
    rows: [{ kind: 'rules', id: 'techs/typescript/*', isExcluded: false }],
  },
  {
    name: 'this project',
    label: 'your own',
    color: COLORS.agent,
    rows: [
      { kind: 'skill', id: 'deploy', isExcluded: false },
      { kind: 'rule', id: 'checkout/refund-once', isExcluded: false },
    ],
  },
];

/** Shows three sources side by side, one rule excluded. */
export function MixAndMatch(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Reveal at={0}>
        <Headline style={{ fontSize: 64 }}>Mix and match rules and skills.</Headline>
      </Reveal>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 32 }}>
        {SOURCES.map((source, index) => (
          <Reveal key={source.name} at={cue(props, 0.18 + index * 0.12)}>
            <Panel
              title={
                <span>
                  <span style={{ color: source.color }}>{source.name}</span> · {source.label}
                </span>
              }
              style={{ height: '100%' }}
            >
              <div style={{ fontFamily: FONTS.mono, fontSize: 25, lineHeight: 1.8 }}>
                {source.rows.map((row) =>
                  row.isExcluded ? (
                    <Reveal key={row.id} at={cue(props, 0.78)}>
                      <span style={{ textDecoration: 'line-through', color: COLORS.sessionMuted }}>
                        {row.kind.padEnd(6)}
                        {row.id}
                      </span>
                      <span style={{ color: COLORS.error }}> excluded</span>
                    </Reveal>
                  ) : (
                    <div key={row.id}>
                      <Muted>{row.kind.padEnd(6)}</Muted>
                      {row.id}
                    </div>
                  ),
                )}
              </div>
            </Panel>
          </Reveal>
        ))}
      </div>
    </SceneFrame>
  );
}
