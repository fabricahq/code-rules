// Recognize macOS terminal streams without changing their terminal settings.

package cli

import (
	"golang.org/x/sys/unix"
	"os"
)

// terminalFile reports whether a stream supports terminal attributes.
func terminalFile(file *os.File) bool {
	_, err := unix.IoctlGetTermios(int(file.Fd()), unix.TIOCGETA)
	return err == nil
}
