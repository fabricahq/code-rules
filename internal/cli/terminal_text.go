// Escape terminal control characters in human output, which carries text from libraries and other untrusted input.

package cli

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// terminalText returns text with every character a terminal could act on written as a visible escape instead:
// C0 controls other than newline, such as ESC as \x1b, DEL, C1 controls, such as U+009B as \u009b, and each byte
// of invalid UTF-8 as \xNN. Newlines stay, because human output uses them for its layout. Other text is unchanged.
func terminalText(text string) string {
	if !strings.ContainsFunc(text, unsafeForTerminal) && utf8.ValidString(text) {
		return text
	}
	var out strings.Builder
	for len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		switch {
		case r == utf8.RuneError && size == 1:
			fmt.Fprintf(&out, `\x%02x`, text[0])
		case r < 0x80 && unsafeForTerminal(r):
			fmt.Fprintf(&out, `\x%02x`, r)
		case unsafeForTerminal(r):
			fmt.Fprintf(&out, `\u%04x`, r)
		default:
			out.WriteString(text[:size])
		}
		text = text[size:]
	}
	return out.String()
}

// unsafeForTerminal reports whether r is a C0 control other than newline, DEL, or a C1 control.
func unsafeForTerminal(r rune) bool {
	return (r < 0x20 && r != '\n') || (r >= 0x7f && r <= 0x9f)
}
