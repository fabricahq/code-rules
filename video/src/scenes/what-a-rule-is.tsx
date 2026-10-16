/** @fileoverview Scene 5: a rule is one best practice in one versioned Markdown file. */
import { Panel } from '../components/panel';
import { Reveal } from '../components/reveal';
import { SceneFrame } from '../components/scene-frame';
import { Headline, Muted, Provenance } from '../components/text';
import { COLORS, FONTS } from '../theme';
import { cue, type SceneProps } from './scene-props';

/** Shows a rule file: its front matter, the practice, and incorrect and correct examples. */
export function WhatARuleIs(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <Reveal at={0}>
        <Headline style={{ fontSize: 60 }}>
          A rule: <span style={{ color: COLORS.muted }}>one best practice, one file, its own version.</span>
        </Headline>
      </Reveal>
      <Reveal at={cue(props, 0.18)}>
        <Panel
          title={
            <span style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span>practices/verification/test-real-user-flows.md</span>
              <Reveal at={cue(props, 0.8)}>
                <Provenance>1.2.0</Provenance>
              </Reveal>
            </span>
          }
        >
          <div style={{ fontFamily: FONTS.mono, fontSize: 29, lineHeight: 1.6, whiteSpace: 'pre' }}>
            <Muted>---</Muted>
            <div>
              <span style={{ color: COLORS.action }}>title</span>: Test real user flows end to end
            </div>
            <Reveal at={cue(props, 0.45)}>
              <span style={{ color: COLORS.action }}>whenToRead</span>: When adding or changing a user-facing flow.
            </Reveal>
            <Muted>---</Muted>
            <Reveal at={cue(props, 0.3)}>Cover each flow with a test that drives it the way a user would.</Reveal>
            <Reveal at={cue(props, 0.6)}>
              <span style={{ color: COLORS.error }}>Incorrect:</span> only unit tests for the checkout handler
            </Reveal>
            <Reveal at={cue(props, 0.66)}>
              <span style={{ color: COLORS.success }}>Correct:</span>
              {'   a browser test that completes checkout'}
            </Reveal>
          </div>
        </Panel>
      </Reveal>
    </SceneFrame>
  );
}
