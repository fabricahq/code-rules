// Observe child exit on Linux without releasing its process ID.

package imports

import (
	"errors"
	"syscall"

	"golang.org/x/sys/unix"
)

// waitForExit leaves the exited child waitable so its process group cannot be reused before cleanup.
func waitForExit(pid int) error {
	var info unix.Siginfo
	for {
		err := unix.Waitid(unix.P_PID, pid, &info, unix.WEXITED|unix.WNOWAIT, nil)
		if !errors.Is(err, syscall.EINTR) {
			return err
		}
	}
}
