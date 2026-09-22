/** @fileoverview Verifies the demo reveals complete messages and tool results in order, including skipped time and static playback. */
import { describe, expect, test } from 'bun:test';
import {
  TranscriptPlayback,
  transcriptFrame,
} from '../src/components/transcript-playback';
import type { TranscriptFrame } from '../src/components/transcript-playback';

const entries = [
  { characters: 100, lines: 3 },
  { characters: 50, lines: 2 },
];

describe('agent transcript playback', () => {
  test('types the message before revealing tool output line by line', () => {
    expect(transcriptFrame(entries, 0)).toEqual({
      index: 0,
      characters: 0,
      lines: 0,
      phase: 'typing',
    });
    expect(transcriptFrame(entries, 800)).toMatchObject({
      index: 0,
      characters: 50,
      lines: 0,
    });
    expect(transcriptFrame(entries, 1950)).toMatchObject({
      characters: 100,
      lines: 0,
      phase: 'running',
    });
    expect(transcriptFrame(entries, 2090)).toMatchObject({
      lines: 1,
      phase: 'running',
    });
    expect(transcriptFrame(entries, 2370)).toMatchObject({
      characters: 100,
      lines: 3,
      phase: 'settled',
    });
  });

  test('holds completed output before starting the next message', () => {
    expect(transcriptFrame(entries, 3669)).toMatchObject({
      index: 0,
      lines: 3,
      phase: 'settled',
    });
    expect(transcriptFrame(entries, 3670)).toEqual({
      index: 1,
      characters: 0,
      lines: 0,
      phase: 'typing',
    });
    expect(transcriptFrame(entries, 4000)).toMatchObject({
      index: 1,
      characters: 20,
      lines: 0,
    });
  });

  test('completes after a large time jump and supports a full static transcript', () => {
    const complete: TranscriptFrame = {
      index: 1,
      characters: 50,
      lines: 2,
      phase: 'complete',
    };
    expect(transcriptFrame(entries, 100_000)).toEqual(complete);
    expect(transcriptFrame(entries, Infinity)).toEqual(complete);
    expect(transcriptFrame([], 0).phase).toBe('complete');
  });
});

const sessions = new Map([
  ['write', entries],
  ['review', [{ characters: 10, lines: 1 }]],
]);

describe('tabbed session controls', () => {
  test('resume preserves the paused frame and never replays completed sessions', () => {
    const playback = new TranscriptPlayback(sessions, 'write');
    playback.resume();
    expect(playback.view).toMatchObject({
      isPreview: true,
      isPlaying: false,
      isPaused: false,
    });
    playback.restart();
    playback.advance(2800);
    playback.pause();
    const pausedFrame = playback.view.frame;
    expect(playback.view.isPaused).toBe(true);
    playback.advance(5000);
    expect(playback.view.frame).toEqual(pausedFrame);
    playback.resume();
    expect(playback.view).toMatchObject({
      isPlaying: true,
      isPaused: false,
      frame: pausedFrame,
    });
    playback.advance(140);
    expect(playback.view.frame.lines).toBeGreaterThan(pausedFrame.lines);
    playback.complete(true);
    playback.resume();
    expect(playback.view).toMatchObject({
      isPlaying: false,
      isPaused: false,
      isComplete: true,
    });
  });

  test('shows the user message before the agent responds, including after replay', () => {
    const playback = new TranscriptPlayback(sessions, 'write');
    playback.restart();
    expect(playback.view.isPromptOnly).toBe(true);
    playback.advance(300);
    playback.pause();
    playback.advance(5000);
    expect(playback.view.isPromptOnly).toBe(true);
    playback.toggle();
    playback.advance(300);
    expect(playback.view).toMatchObject({
      isPromptOnly: false,
      frame: { characters: 0 },
    });
    playback.advance(160);
    expect(playback.view.frame.characters).toBe(10);
    playback.restart();
    expect(playback.view.isPromptOnly).toBe(true);
    playback.complete(true);
    expect(playback.view).toMatchObject({
      isPromptOnly: false,
      isComplete: true,
      isPlaying: false,
    });
    playback.select('review');
    playback.restart();
    expect(playback.view.isPromptOnly).toBe(true);
  });

  test('each tab keeps its own initial preview until explicitly played or revealed', () => {
    const playback = new TranscriptPlayback(sessions, 'write');
    expect(playback.view).toMatchObject({
      isPreview: true,
      isPlaying: false,
      isComplete: true,
    });
    playback.restart();
    playback.advance(1400);
    expect(playback.view).toMatchObject({
      isPreview: false,
      isPlaying: true,
      frame: { characters: 50 },
    });
    playback.select('review');
    expect(playback.view).toMatchObject({
      isPreview: true,
      isPlaying: false,
      isComplete: true,
    });
    playback.select('write');
    expect(playback.view).toMatchObject({
      isPreview: false,
      isPlaying: false,
      isComplete: true,
    });
    playback.select('review');
    playback.restart();
    expect(playback.view).toMatchObject({
      isPreview: false,
      isPlaying: true,
      frame: { index: 0, characters: 0 },
    });
  });

  test('pause preserves progress, resume continues, and replay starts over', () => {
    const playback = new TranscriptPlayback(sessions, 'write');
    playback.toggle();
    playback.advance(1400);
    playback.toggle();
    playback.advance(10_000);
    expect(playback.view).toMatchObject({
      isPlaying: false,
      frame: { characters: 50 },
    });
    playback.toggle();
    playback.advance(160);
    expect(playback.view.frame.characters).toBe(60);
    playback.pause();
    playback.advance(10_000);
    expect(playback.view.frame.characters).toBe(60);
    playback.restart();
    expect(playback.view).toMatchObject({
      isPlaying: true,
      frame: { index: 0, characters: 0 },
    });
    playback.advance(100_000);
    expect(playback.view).toMatchObject({ isPlaying: false, isComplete: true });
    playback.toggle();
    expect(playback.view).toMatchObject({
      isPlaying: true,
      frame: { index: 0, characters: 0 },
    });
  });

  test('full-session requests reveal only the selected tab; lifecycle completion keeps previews', () => {
    const playback = new TranscriptPlayback(sessions, 'write');
    playback.complete();
    expect(playback.view.isPreview).toBe(true);
    playback.complete(true);
    expect(playback.view).toMatchObject({
      isPreview: false,
      isComplete: true,
      isPlaying: false,
    });
    playback.select('review');
    expect(playback.view.isPreview).toBe(true);
    playback.restart();
    playback.advance(680);
    playback.select('review');
    expect(playback.view).toMatchObject({
      isPlaying: true,
      frame: { characters: 5 },
    });
    playback.complete();
    expect(playback.view).toMatchObject({
      isPlaying: false,
      isComplete: true,
      isPreview: false,
    });
  });

  test('an empty session finishes immediately and cannot leave a timer running', () => {
    const playback = new TranscriptPlayback(new Map([['empty', []]]), 'empty');
    playback.restart();
    expect(playback.view).toMatchObject({
      isPlaying: false,
      isComplete: true,
      isPreview: false,
    });
  });
});
