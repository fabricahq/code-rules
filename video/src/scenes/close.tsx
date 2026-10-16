/** @fileoverview Scene 14: the Trail Guide name, tagline, and site. */
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Headline } from '../components/text';
import { Wordmark } from '../components/wordmark';
import { COLORS, FONTS } from '../theme';
import type { SceneProps } from './scene-props';

/** Ends on the name and where to learn more. */
export function Close(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 40, textAlign: 'center' }}>
        <Reveal at={0}>
          <Wordmark fontSize={112} />
        </Reveal>
        <Reveal at={12}>
          <Headline style={{ fontSize: 64 }}>Agent guidance, one small insight at a time.</Headline>
        </Reveal>
        <Reveal at={24}>
          <div style={{ fontFamily: FONTS.mono, fontSize: 32, color: COLORS.muted }}>trailguide.fabricahq.com</div>
        </Reveal>
      </div>
    </SceneFrame>
  );
}
