/** @fileoverview Scene 10: the Syllabus name, tagline, and site. */
import { BrandMark } from '../components/brand-mark';
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Headline } from '../components/text';
import { COLORS, FONTS } from '../theme';
import type { SceneProps } from './scene-props';

/** Ends on the brand and where to learn more. */
export function Close(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 40, textAlign: 'center' }}>
        <Reveal at={0} style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
          <BrandMark size={110} />
          <span style={{ fontSize: 96, fontWeight: 600, letterSpacing: '-0.05em' }}>Syllabus</span>
        </Reveal>
        <Reveal at={12}>
          <Headline style={{ fontSize: 64 }}>The package manager for your agent skills and rules</Headline>
        </Reveal>
        <Reveal at={24}>
          <div style={{ fontFamily: FONTS.mono, fontSize: 32, color: COLORS.muted }}>syllabus.fabricahq.com</div>
        </Reveal>
      </div>
    </SceneFrame>
  );
}
