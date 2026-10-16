/** @fileoverview Scene 9: the new rules can live in the project, the organization's library, or a public library. */
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Chip, Headline, Muted } from '../components/text';
import { COLORS, FONTS } from '../theme';
import { cue, type SceneProps } from './scene-props';

const PLACES = [
  { title: 'This project', detail: 'Just for this repository', where: '.trailguide/rules/', at: 0.05 },
  { title: 'Your org’s library', detail: 'Every project gets them', where: 'acme/.trailguide', at: 0.3 },
  { title: 'A public library', detail: 'Versioned, for anyone to use', where: 'github.com/you/rules', at: 0.6 },
] as const;

/** Shows the three places a rule can live. */
export function WhereTheyLive(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Reveal at={0}>
        <Headline style={{ fontSize: 64 }}>Put them where they belong.</Headline>
      </Reveal>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 32 }}>
        {PLACES.map((place) => (
          <Reveal key={place.title} at={cue(props, place.at)}>
            <Chip style={{ padding: '36px 36px', fontSize: 40, height: '100%' }}>
              {place.title}
              <div style={{ fontSize: 30, marginTop: 12 }}>
                <Muted>{place.detail}</Muted>
              </div>
              <div style={{ fontFamily: FONTS.mono, fontSize: 24, marginTop: 24, color: COLORS.action }}>
                {place.where}
              </div>
            </Chip>
          </Reveal>
        ))}
      </div>
    </SceneFrame>
  );
}
