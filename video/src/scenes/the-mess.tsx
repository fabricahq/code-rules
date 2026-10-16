/** @fileoverview Scene 1: skills copied between repositories, with no record of source or version. */
import { Panel } from '../components/panel';
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Headline, Muted } from '../components/text';
import { COLORS, FONTS } from '../theme';
import { cue, type SceneProps } from './scene-props';

const FOLDERS = [
  { path: '.claude/skills/write-tests/', doubt: 'copied from ??? · edited?' },
  { path: '.claude/skills/write-tests-old/', doubt: 'which one is current?' },
  { path: '.agents/skills/testing/', doubt: 'a different copy' },
] as const;

/** Opens on the problem: skill folders with unknown sources and versions. */
export function TheMess(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Reveal at={0}>
        <Headline>Your agents already use skills.</Headline>
      </Reveal>
      <Reveal at={cue(props, 0.35)}>
        <Headline style={{ color: COLORS.muted, fontSize: 56 }}>
          Where did each one come from? Which version is it?
        </Headline>
      </Reveal>
      <Panel title="acme-checkout">
        <div style={{ fontFamily: FONTS.mono, fontSize: 32, lineHeight: 1.8 }}>
          {FOLDERS.map((folder, index) => (
            <div key={folder.path} style={{ display: 'flex', justifyContent: 'space-between', gap: 40 }}>
              <Muted>{folder.path}</Muted>
              <Reveal at={cue(props, 0.4 + index * 0.12)}>
                <span style={{ color: COLORS.error }}>{folder.doubt}</span>
              </Reveal>
            </div>
          ))}
        </div>
      </Panel>
    </SceneFrame>
  );
}
