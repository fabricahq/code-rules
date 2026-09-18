//go:build darwin

// Compare change timestamps using the platform's stat layout.

package project

import "syscall"

// changedTime detects metadata or data changes even when a writer restores modification time.
func changedTime(a, b *syscall.Stat_t) bool { return a.Ctimespec != b.Ctimespec }
