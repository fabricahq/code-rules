/** @fileoverview Verifies the demo reveals complete messages and tool results in order, including skipped time and static playback. */
import { describe, expect, test } from 'bun:test';
import { transcriptFrame } from '../src/components/transcript-playback';
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

  test('restarting begins at the first character without carrying over the previous session', () => {
    transcriptFrame(entries, Infinity);
    expect(transcriptFrame(entries, 0)).toEqual({
      index: 0,
      characters: 0,
      lines: 0,
      phase: 'typing',
    });
    expect(transcriptFrame([{ characters: 10, lines: 1 }], 0).index).toBe(0);
  });
});
