// Preserve a newly appeared destination when publishing a managed guide on macOS.

package filetxn

import "golang.org/x/sys/unix"

func renameExclusive(fromFD int, from string, toFD int, to string) error {
	return unix.RenameatxNp(fromFD, from, toFD, to, unix.RENAME_EXCL)
}
