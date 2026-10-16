/** @fileoverview The tour composition: plays each scene for as long as its narration, with its voiceover and captions. */
import type { ComponentType } from 'react';
import { AbsoluteFill, Audio, Sequence, Series, staticFile } from 'remotion';
import { getAudioDurationInSeconds } from '@remotion/media-utils';
import type { CalculateMetadataFunction } from 'remotion';
import { Caption } from './components/caption';
import { AddTheirLibrary } from './scenes/add-their-library';
import { AgentAtWork } from './scenes/agent-at-work';
import { CaptureWhatYouLearn } from './scenes/capture-what-you-learn';
import { Close } from './scenes/close';
import { RulesToo } from './scenes/rules-too';
import type { SceneProps } from './scenes/scene-props';
import { SkillsAndRules } from './scenes/skills-and-rules';
import { Sync } from './scenes/sync';
import { TheAsk } from './scenes/the-ask';
import { TheMess } from './scenes/the-mess';
import { WhatALibraryIs } from './scenes/what-a-library-is';
import { SCENES, type SceneId } from './script';
import { COLORS, VIDEO } from './theme';
import {
  estimateNarrationSeconds,
  estimateWords,
  MANIFEST_PATH,
  parseManifest,
  type TimedWord,
  type VoiceoverManifest,
} from './voiceover';

/** Silence before each scene's narration starts, so the scene can settle first. */
const LEAD_SECONDS = 0.5;
/** Silence after each scene's narration ends, before the next scene. */
const TAIL_SECONDS = 0.7;

const SCENE_COMPONENTS = {
  'the-mess': TheMess,
  'what-a-library-is': WhatALibraryIs,
  'the-ask': TheAsk,
  'add-their-library': AddTheirLibrary,
  sync: Sync,
  'rules-too': RulesToo,
  'skills-and-rules': SkillsAndRules,
  'agent-at-work': AgentAtWork,
  'capture-what-you-learn': CaptureWhatYouLearn,
  close: Close,
} as const satisfies Record<SceneId, ComponentType<SceneProps>>;

/** One scene as the composition plays it. */
export type TimedScene = {
  id: SceneId;
  durationInFrames: number;
  /** Path under `public/` of the narration clip, or null to play the scene silent. */
  audioFile: string | null;
  /** Caption words, timed from the start of the scene. */
  words: Array<TimedWord>;
};

/** The composition's props, which calculateTourMetadata fills in. */
export type TourProps = { scenes: Array<TimedScene> };

async function loadManifest(): Promise<VoiceoverManifest | null> {
  const response = await fetch(staticFile(MANIFEST_PATH));
  if (!response.ok) return null;
  const json: unknown = await response.json();
  return parseManifest(json);
}

/**
 * Times every scene from its voiceover clip, or from an estimate when the clip is
 * missing or was generated from different narration, so the video renders before
 * any voiceover exists.
 */
export const calculateTourMetadata: CalculateMetadataFunction<TourProps> = async () => {
  const manifest = await loadManifest();
  const scenes: Array<TimedScene> = [];
  for (const scene of SCENES) {
    const clip = manifest?.clips[scene.id];
    const isCurrent = clip !== undefined && clip.text === scene.narration;
    if (clip !== undefined && !isCurrent) {
      console.warn(`The voiceover for ${scene.id} is stale; run bun run voiceover. Rendering it silent.`);
    }
    const narrationSeconds = isCurrent
      ? await getAudioDurationInSeconds(staticFile(clip.file))
      : estimateNarrationSeconds(scene.narration);
    const words = isCurrent
      ? clip.words
      : estimateWords({ text: scene.narration, durationInSeconds: narrationSeconds });
    scenes.push({
      id: scene.id,
      durationInFrames: Math.ceil((LEAD_SECONDS + narrationSeconds + TAIL_SECONDS) * VIDEO.fps),
      audioFile: isCurrent ? clip.file : null,
      words: words.map((word) => ({ ...word, start: word.start + LEAD_SECONDS, end: word.end + LEAD_SECONDS })),
    });
  }
  return {
    durationInFrames: scenes.reduce((sum, scene) => sum + scene.durationInFrames, 0),
    props: { scenes },
  };
};

/** Plays the tour's scenes in order. */
export function Tour({ scenes }: TourProps) {
  return (
    <AbsoluteFill style={{ background: COLORS.page }}>
      <Series>
        {scenes.map((scene) => {
          const Scene = SCENE_COMPONENTS[scene.id];
          return (
            <Series.Sequence key={scene.id} durationInFrames={scene.durationInFrames} name={scene.id}>
              <Scene durationInFrames={scene.durationInFrames} />
              {scene.audioFile === null ? null : (
                <Sequence from={Math.round(LEAD_SECONDS * VIDEO.fps)}>
                  <Audio src={staticFile(scene.audioFile)} />
                </Sequence>
              )}
              <Caption words={scene.words} />
            </Series.Sequence>
          );
        })}
      </Series>
    </AbsoluteFill>
  );
}
