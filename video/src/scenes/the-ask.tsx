/** @fileoverview Scene 3: a developer asks their agent for the payments team's testing skill. */
import { AgentSession, UserMessage } from '../components/chat';
import { SceneFrame } from '../components/scene-frame';
import { cue, type SceneProps } from './scene-props';

/** The request the rest of the story answers. */
export function TheAsk(props: SceneProps) {
  return (
    <SceneFrame durationInFrames={props.durationInFrames}>
      <AgentSession>
        <UserMessage
          at={cue(props, 0.08)}
          text="I like the way the payments team writes their automated tests. Can you install their testing skill for me?"
        />
      </AgentSession>
    </SceneFrame>
  );
}
