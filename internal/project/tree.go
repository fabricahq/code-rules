// Read bounded project trees and compare their exact bytes, including empty directories.

package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
	"syscall"
	"unicode/utf16"
)

const maxProjectFile = 64 * 1024 * 1024

// Tree owns a directory inventory and original file bytes. A nil Tree means the directory is absent.
type Tree struct {
	Files       map[string][]byte `json:"files"`
	Directories []string          `json:"directories"`
}

// Error describes a failed project operation. Callers may branch on Code after errors.As.
// Stable codes are busy, concurrent-change, recovery-required, invalid-operation,
// invalid-target, unsafe-path, unsafe-file, and input-limit. New codes may be added;
// callers must handle unknown codes as failures. Problem is display text, not a stable identifier.
type Error struct {
	Code    string
	Problem string
	Cause   error
}

// Error returns actionable context without embedding authored file contents.
func (e *Error) Error() string { return e.Problem }

// Unwrap preserves cancellation and operating-system causes for callers.
func (e *Error) Unwrap() error { return e.Cause }

// projectError adds a stable category and optional underlying failure.
func projectError(code, problem string, cause error) error { return &Error{code, problem, cause} }

// treeLimits bounds discovery and allocated content for project and recovery trees.
type treeLimits struct{ entries, bytes, depth int }

var projectLimits = treeLimits{30_000, 256 * 1024 * 1024, 64}
var recoveryLimits = treeLimits{120_032, 1025 * 1024 * 1024, 65}

// ReadTree returns a contained, bounded tree without following observed links or accepting hard links.
// The caller owns root. Missing directories return nil; failures return no partial tree.
func ReadTree(ctx context.Context, root *os.Root, name string) (*Tree, error) {
	return readTree(ctx, root, name, projectLimits)
}

// readTree inventories directories and bytes under explicit limits, including empty directories.
func readTree(ctx context.Context, root *os.Root, name string, limits treeLimits) (*Tree, error) {
	if root == nil {
		return nil, projectError("unsafe-path", "expected an open project root", nil)
	}
	if name != "." {
		if err := validateFilePaths(map[string][]byte{name: nil}); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if name != "." {
		parts := strings.Split(name, "/")
		for i := range parts {
			info, err := root.Lstat(strings.Join(parts[:i+1], "/"))
			if errors.Is(err, fs.ErrNotExist) {
				return nil, nil
			}
			if err != nil {
				return nil, err
			}
			if !info.IsDir() {
				return nil, projectError("unsafe-path", name+": expected directories without symlinks", nil)
			}
		}
	}
	info, err := root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect project tree %s: %w", name, err)
	}
	if !info.IsDir() {
		return nil, projectError("unsafe-path", name+": expected a directory without symlinks", nil)
	}
	tree := &Tree{Files: map[string][]byte{}, Directories: []string{}}
	total, count := 0, 0
	var visit func(string) error
	visit = func(prefix string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		full := path.Join(name, prefix)
		dir, err := root.OpenFile(full, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return fmt.Errorf("open directory %s: %w", full, err)
		}
		entries, readErr := dir.ReadDir(limits.entries - count + 1)
		closeErr := dir.Close()
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return fmt.Errorf("list %s: %w", full, readErr)
		}
		if closeErr != nil {
			return closeErr
		}
		count += len(entries)
		if count > limits.entries {
			return projectError("input-limit", name+": too many filesystem entries", nil)
		}
		slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
		for _, entry := range entries {
			rel := path.Join(prefix, entry.Name())
			if len(strings.Split(rel, "/")) > limits.depth {
				return projectError("input-limit", rel+": directory depth exceeds limit", nil)
			}
			if entry.IsDir() {
				tree.Directories = append(tree.Directories, rel)
				if err := visit(rel); err != nil {
					return err
				}
			} else if entry.Type().IsRegular() {
				data, err := readFile(ctx, root, path.Join(name, rel))
				if err != nil {
					return err
				}
				total += len(data)
				if total > limits.bytes {
					return projectError("input-limit", name+": tree exceeds byte limit", nil)
				}
				tree.Files[rel] = data
			} else {
				return projectError("unsafe-path", rel+": links and special files are unsupported", nil)
			}
		}
		return nil
	}
	if err := visit(""); err != nil {
		return nil, err
	}
	if err := validatePaths(tree.Files, tree.Directories); err != nil {
		return nil, err
	}
	slices.Sort(tree.Directories)
	return tree, nil
}

// readFile bounds actual bytes and verifies the open regular file did not change while read.
func readFile(ctx context.Context, root *os.Root, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// os.Root contains resolution; explicit component checks reject existing internal symlinks too.
	parts := strings.Split(name, "/")
	for i := range parts {
		info, err := root.Lstat(strings.Join(parts[:i+1], "/"))
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", name, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, projectError("unsafe-path", name+": symlinks are unsupported", nil)
		}
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		return nil, err
	}
	stat, ok := before.Sys().(*syscall.Stat_t)
	if !before.Mode().IsRegular() || !ok || stat.Nlink != 1 || before.Size() > maxProjectFile {
		return nil, projectError("unsafe-file", name+": expected a regular file of at most 64 MiB without hard links", nil)
	}
	data, err := io.ReadAll(io.LimitReader(f, maxProjectFile+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if len(data) > maxProjectFile {
		return nil, projectError("input-limit", name+": file exceeds 64 MiB", nil)
	}
	// Stat includes change time and link count. Access-time changes caused by this read are excluded.
	next, ok := after.Sys().(*syscall.Stat_t)
	if !ok || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) || next.Nlink != 1 || stat.Ino != next.Ino || changedTime(stat, next) {
		return nil, projectError("concurrent-change", name+": file changed while being read", nil)
	}
	return data, nil
}

// treeDigest preserves the TypeScript journal's version-1 hash representation.
func treeDigest(tree *Tree) string {
	if tree == nil {
		return digest([]byte("null"))
	}
	dirs := slices.Clone(tree.Directories)
	if dirs == nil {
		dirs = []string{}
	}
	slices.SortFunc(dirs, compareUTF16)
	files := make([][2]string, 0, len(tree.Files))
	for _, name := range slices.SortedFunc(maps.Keys(tree.Files), compareUTF16) {
		files = append(files, [2]string{name, digest(tree.Files[name])})
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(struct {
		Directories []string    `json:"directories"`
		Files       [][2]string `json:"files"`
	}{dirs, files})
	// Portable paths reject backslashes, so these cannot be authored literal escape sequences.
	encoded := strings.NewReplacer(`\u2028`, "\u2028", `\u2029`, "\u2029").Replace(strings.TrimSuffix(buf.String(), "\n"))
	return digest([]byte(encoded))
}

// writeTree creates a fresh staging directory and never overwrites an existing file.
func writeTree(ctx context.Context, root *os.Root, name string, files map[string][]byte) error {
	if err := validateFilePaths(files); err != nil {
		return err
	}
	if err := root.Mkdir(name, 0700); err != nil {
		return err
	}
	for _, file := range slices.Sorted(maps.Keys(files)) {
		if err := ctx.Err(); err != nil {
			return err
		}
		full := path.Join(name, file)
		if err := root.MkdirAll(path.Dir(full), 0700); err != nil {
			return err
		}
		f, err := root.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		_, writeErr := f.Write(files[file])
		closeErr := f.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			return fmt.Errorf("stage %s: %w", file, err)
		}
	}
	return nil
}

// compareUTF16 retains JavaScript's code-unit ordering in persisted version-1 journal hashes.
func compareUTF16(a, b string) int {
	return slices.Compare(utf16.Encode([]rune(a)), utf16.Encode([]rune(b)))
}

// Digest returns a stable identity including original bytes and empty directories; nil means absent.
// Callers may retain this value to detect changes before a managed update.
func (tree *Tree) Digest() string { return treeDigest(tree) }
