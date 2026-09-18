/** @fileoverview Computes a deterministic transcript frame from elapsed playback time, without browser state or timers. */

/** Counts of Unicode characters in a message and highlighted lines in its tool result. */
export type TranscriptEntry = { characters: number; lines: number };

/** The active entry's visible content; earlier entries are complete and later entries remain hidden. */
export type TranscriptFrame = {
  index: number;
  characters: number;
  lines: number;
  phase: 'typing' | 'running' | 'settled' | 'complete';
};

const CHARACTER_MS = 16;
const TOOL_START_MS = 350;
const OUTPUT_LINE_MS = 140;
const READING_HOLD_MS = 1300;

/** Return the visible message and output at elapsed milliseconds; Infinity reveals the complete session. */
export function transcriptFrame(
  entries: ReadonlyArray<TranscriptEntry>,
  elapsed: number,
): TranscriptFrame {
  let remaining = Math.max(0, elapsed);
  for (const [index, entry] of entries.entries()) {
    const typedAt = entry.characters * CHARACTER_MS;
    const toolAt = typedAt + TOOL_START_MS;
    const settledAt = toolAt + entry.lines * OUTPUT_LINE_MS;
    const endsAt = settledAt + READING_HOLD_MS;
    if (remaining < endsAt) {
      return {
        index,
        characters: Math.min(
          entry.characters,
          Math.floor(remaining / CHARACTER_MS),
        ),
        lines: Math.min(
          entry.lines,
          Math.max(0, Math.floor((remaining - toolAt) / OUTPUT_LINE_MS)),
        ),
        phase:
          remaining < toolAt
            ? 'typing'
            : remaining < settledAt
              ? 'running'
              : 'settled',
      };
    }
    remaining -= endsAt;
  }
  const last = entries.at(-1);
  return {
    index: Math.max(0, entries.length - 1),
    characters: last?.characters ?? 0,
    lines: last?.lines ?? 0,
    phase: 'complete',
  };
}
