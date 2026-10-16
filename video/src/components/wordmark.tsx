/** @fileoverview The Trail Guide name set as a wordmark, until the renamed product has a symbol of its own. */
import { COLORS } from '../theme';

/** Props for Wordmark. */
export type WordmarkProps = { fontSize: number };

/** Draws "Trail Guide" in the display style, at a font size in pixels. */
export function Wordmark({ fontSize }: WordmarkProps) {
  return (
    <span style={{ fontSize, fontWeight: 600, letterSpacing: '-0.05em', color: COLORS.ink }}>
      Trail <span style={{ color: COLORS.success }}>Guide</span>
    </span>
  );
}
