// Separate confined local filesystem reads from shared catalog validation.

package library

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"syscall"
)

// FileSource provides bounded, non-following reads for library validation.
// Lstat and ReadDir must preserve symlink/special-file types. ReadFile must reject
// special files and cap allocation at 8 MiB; ReadDir must cap entries at 10,001.
// Implementations must not mutate input or turn remote paths into local absolute paths.
type FileSource interface {
	Lstat(string) (fs.FileInfo, error)
	ReadFile(string) ([]byte, error)
	ReadDir(string) ([]fs.DirEntry, error)
}

// rootFiles retains the existing os.Root confinement and local file-kind checks.
type rootFiles struct {
	ctx  context.Context
	root *os.Root
}

// Lstat inspects a path without following its final symbolic link.
func (r rootFiles) Lstat(path string) (fs.FileInfo, error) { return r.root.Lstat(path) }

// ReadFile bounds actual bytes through a no-follow handle and rejects hard links and special files.
func (r rootFiles) ReadFile(path string) ([]byte, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	file, err := r.root.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, bad(path, "expected an ordinary file")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Nlink != 1 {
		return nil, bad(path, "hard links are unsupported")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxFileBytes {
		return nil, bad(path, "library exceeds file or total read limits")
	}
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

// ReadDir lists a confined directory without following a final link and bounds discovery allocation.
func (r rootFiles) ReadDir(path string) ([]fs.DirEntry, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, err
	}
	dir, err := r.root.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	entries, readErr := dir.ReadDir(maxFiles + 1)
	closeErr := dir.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return nil, fmt.Errorf("read directory: %w", readErr)
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return entries, nil
}
