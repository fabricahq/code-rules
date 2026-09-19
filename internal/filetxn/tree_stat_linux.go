//go:build linux

// Compare change timestamps using the platform's stat layout.

package filetxn

import (
	"syscall"
)

// changedTime detects metadata or data changes even when a writer restores modification time.
func changedTime(a, b *syscall.Stat_t) bool { return a.Ctim != b.Ctim }
