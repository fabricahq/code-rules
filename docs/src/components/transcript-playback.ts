/** @fileoverview Owns tabbed transcript playback and frame timing without browser state or timers. */

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

/**
 * Owns one tabbed demo's playback decisions without DOM elements or timers.
 * Each session starts behind its own preview; changing tabs stops playback and
 * shows the selected transcript in full, retaining that session's preview state.
 */
export class TranscriptPlayback {
  private elapsed = Infinity;
  private playing = false;
  private revealed = new Set<string>();
  private session: string;

  constructor(
    private readonly sessions: ReadonlyMap<
      string,
      ReadonlyArray<TranscriptEntry>
    >,
    initialSession: string,
  ) {
    this.session = initialSession;
  }

  /** Current visible content and controls, without advancing playback. */
  get view() {
    const frame = transcriptFrame(
      this.sessions.get(this.session) ?? [],
      this.elapsed,
    );
    return {
      frame,
      isPlaying: this.playing,
      isPreview: !this.revealed.has(this.session),
      isComplete: frame.phase === 'complete',
    };
  }

  /** Select a different session without starting it or dismissing its preview. */
  select(session: string) {
    if (session === this.session || !this.sessions.has(session)) return;
    this.session = session;
    this.complete();
  }

  /** Start the selected session from the beginning and dismiss only its preview. */
  restart() {
    this.revealed.add(this.session);
    this.elapsed = 0;
    this.playing = this.view.frame.phase !== 'complete';
  }

  /** Toggle pause/resume; a completed session starts again from the beginning. */
  toggle() {
    if (this.view.isComplete) this.restart();
    else this.playing = !this.playing;
  }

  /** Pause without discarding progress. */
  pause() {
    this.playing = false;
  }

  /** Show all content; only an explicit full-session request dismisses the preview. */
  complete(reveal = false) {
    if (reveal) this.revealed.add(this.session);
    this.elapsed = Infinity;
    this.playing = false;
  }

  /** Advance by visible playback time only, stopping at the final frame. */
  advance(milliseconds: number) {
    if (!this.playing) return;
    this.elapsed += Math.max(0, milliseconds);
    if (this.view.isComplete) this.playing = false;
  }
}
