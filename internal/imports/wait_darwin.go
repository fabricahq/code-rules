// Observe child exit on macOS without releasing its process ID.

package imports

import (
	"errors"
	"syscall"

	"golang.org/x/sys/unix"
)

// waitForExit retains the child for cleanup; an already-exited child cannot be attached to kqueue.
func waitForExit(pid int) error {
	queue, err := unix.Kqueue()
	if err != nil {
		return err
	}
	defer unix.Close(queue)
	change := []unix.Kevent_t{{Ident: uint64(pid), Filter: unix.EVFILT_PROC, Flags: unix.EV_ADD | unix.EV_ONESHOT, Fflags: unix.NOTE_EXIT}}
	events := make([]unix.Kevent_t, 1)
	for {
		n, err := unix.Kevent(queue, change, events, nil)
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		if err != nil {
			return err
		}
		change = nil
		if n == 0 {
			continue
		}
		if events[0].Flags&unix.EV_ERROR != 0 {
			if events[0].Data == int64(syscall.ESRCH) {
				return nil
			}
			return syscall.Errno(events[0].Data)
		}
		return nil
	}
}
