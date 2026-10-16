/** @fileoverview Scene 8: the agent writes tests by following the payments skill and checking its work against their rules. */
import { AgentLine, AgentSession, UserMessage } from '../components/chat';
import { SceneFrame } from '../components/scene-frame';
import { Accent, Check, Muted } from '../components/text';
import { FONTS } from '../theme';
import { cue, type SceneProps } from './scene-props';

const RESULTS = [
  { text: 'Asserts the stored refund, not just a 200', rule: 'confirm-critical-writes' },
  { text: 'Retries the refund twice, charges once', rule: 'make-retries-idempotent' },
] as const;

/** Shows the installed skill and rules at work in a session. */
export function AgentAtWork(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <AgentSession>
        <UserMessage at={0} text="Add tests for refunds." />
        <AgentLine at={cue(props, 0.18)}>
          Following <Accent>write-tests</Accent>, with your critical-operations rules.
        </AgentLine>
        {RESULTS.map((result, index) => (
          <AgentLine key={result.rule} at={cue(props, 0.4 + index * 0.14)}>
            <Check /> {result.text}{' '}
            <Muted>
              <span style={{ fontFamily: FONTS.mono, fontSize: 26 }}>· {result.rule}</span>
            </Muted>
          </AgentLine>
        ))}
        <AgentLine at={cue(props, 0.72)}>
          <Check /> 6 tests passed
        </AgentLine>
      </AgentSession>
    </SceneFrame>
  );
}
