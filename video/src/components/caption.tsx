/** @fileoverview Subtitles for the narration: the current phrase, with spoken words brightened as the voice reaches them. */
import { useCurrentFrame, useVideoConfig } from 'remotion';
import { COLORS, FONTS } from '../theme';
import type { TimedWord } from '../voiceover';

/** The most words a caption shows at once. */
const MAX_WORDS_PER_PHRASE = 11;

/** Splits words into phrases that end at sentence or clause breaks and stay short enough for one line. */
export function toPhrases(words: ReadonlyArray<TimedWord>): Array<Array<TimedWord>> {
  const phrases: Array<Array<TimedWord>> = [];
  let current: Array<TimedWord> = [];
  for (const word of words) {
    current.push(word);
    const endsSentence = /[.?!]$/.test(word.text);
    const endsClause = /[,:;]$/.test(word.text) && current.length >= 6;
    if (endsSentence || endsClause || current.length >= MAX_WORDS_PER_PHRASE) {
      phrases.push(current);
      current = [];
    }
  }
  if (current.length > 0) phrases.push(current);
  return phrases;
}

/** Props for Caption. */
export type CaptionProps = { words: ReadonlyArray<TimedWord> };

/** Shows the phrase being spoken at the current frame. */
export function Caption({ words }: CaptionProps) {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const seconds = frame / fps;
  const phrases = toPhrases(words);
  const phrase = phrases.find((candidate) => (candidate.at(-1)?.end ?? 0) >= seconds) ?? phrases.at(-1);
  if (phrase === undefined) return null;
  return (
    <div
      style={{
        position: 'absolute',
        left: 160,
        right: 160,
        bottom: 70,
        display: 'flex',
        justifyContent: 'center',
      }}
    >
      <div
        style={{
          padding: '18px 34px',
          borderRadius: 16,
          background: '#0d0d0dd9',
          border: `1.5px solid ${COLORS.border}`,
          fontFamily: FONTS.sans,
          fontSize: 40,
          lineHeight: 1.35,
          letterSpacing: '-0.01em',
          textAlign: 'center',
        }}
      >
        {phrase.map((word, index) => (
          <span key={index} style={{ color: word.start <= seconds ? COLORS.ink : COLORS.muted }}>
            {index > 0 ? ' ' : ''}
            {word.text}
          </span>
        ))}
      </div>
    </div>
  );
}
