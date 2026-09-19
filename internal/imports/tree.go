// Expose immutable Git tree metadata and bounded original blob bytes to the library loader.

package imports

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const maxTreeBytes = 8 << 20
const maxBlobBytes = 8 << 20
const maxRetainedBytes = 64 << 20
const maxTreeFiles = 10000

// treeEntry implements file metadata without materializing or following Git entries.
type treeEntry struct {
	name   string
	mode   fs.FileMode
	size   int64
	object string
}

// Name returns the final component of an immutable tree path.
func (e treeEntry) Name() string { return path.Base(e.name) }

// Size returns Git's declared blob size, verified again by the batch protocol on read.
func (e treeEntry) Size() int64 { return e.size }

// Mode preserves symlink and submodule distinction for the shared loader.
func (e treeEntry) Mode() fs.FileMode { return e.mode }

// ModTime is zero because Git trees do not contain per-file timestamps.
func (e treeEntry) ModTime() time.Time { return time.Time{} }

// IsDir identifies synthetic parent directories in the tree inventory.
func (e treeEntry) IsDir() bool { return e.mode.IsDir() }

// Sys returns no operating-system metadata for a Git object.
func (e treeEntry) Sys() any { return nil }

// gitFiles supplies metadata without I/O and fetches only files requested by shared catalog validation.
type gitFiles struct {
	ctx         context.Context
	revision    *revision
	entries     map[string]treeEntry
	directories map[string][]fs.DirEntry
	total       int
}

// openTree reads the bounded NUL-framed tree and constructs directories without a checkout.
func (r *revision) openTree(ctx context.Context) (*gitFiles, error) {
	if r == nil || r.directory == "" {
		return nil, fail("closed-revision", "Revision has been closed.", nil)
	}
	data, err := r.runner.command(ctx, r.directory, []string{"ls-tree", "-r", "-l", "-z", r.Commit}, maxTreeBytes)
	if err != nil {
		return nil, err
	}
	entries, err := parseTree(data)
	if err != nil {
		return nil, err
	}
	source := &gitFiles{ctx: ctx, revision: r, entries: entries, directories: map[string][]fs.DirEntry{}}
	for name, entry := range entries {
		if name == "." {
			continue
		}
		parent := path.Dir(name)
		source.directories[parent] = append(source.directories[parent], fs.FileInfoToDirEntry(entry))
	}
	for _, children := range source.directories {
		slices.SortFunc(children, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	}
	return source, nil
}

// parseTree rejects malformed framing and unsafe paths before exposing an immutable inventory.
func parseTree(data []byte) (map[string]treeEntry, error) {
	if len(data) > maxTreeBytes {
		return nil, fail("limit-exceeded", "Git tree listing exceeds 8 MiB.", nil)
	}
	if !utf8.Valid(data) || (len(data) > 0 && data[len(data)-1] != 0) {
		return nil, fail("unsupported-content", "Git tree must contain complete UTF-8 records.", nil)
	}
	entries := map[string]treeEntry{".": {name: ".", mode: fs.ModeDir | 0755}}
	count := 0
	for _, record := range bytes.Split(bytes.TrimSuffix(data, []byte{0}), []byte{0}) {
		if len(data) == 0 {
			break
		}
		if len(record) == 0 {
			return nil, fail("unsupported-content", "Empty Git tree record.", nil)
		}
		count++
		if count > maxTreeFiles {
			return nil, fail("limit-exceeded", "Library tree exceeds 10,000 files.", nil)
		}
		header, name, ok := strings.Cut(string(record), "\t")
		fields := strings.Fields(header)
		if !ok || len(fields) != 4 || !validObjectID(fields[2]) || !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, "\\\x00") || len(strings.Split(name, "/")) > 64 {
			return nil, fail("unsupported-content", "Unsupported Git tree entry or path.", nil)
		}
		entry := treeEntry{name: name, object: fields[2]}
		switch fields[0] + " " + fields[1] {
		case "100644 blob":
			entry.mode = 0644
		case "100755 blob":
			entry.mode = 0755
		case "120000 blob":
			entry.mode = fs.ModeSymlink | 0777
		case "160000 commit":
			entry.mode = fs.ModeIrregular
		default:
			return nil, fail("unsupported-content", "Unsupported Git tree object mode.", nil)
		}
		if fields[1] == "commit" {
			if fields[3] != "-" {
				return nil, fail("unsupported-content", "Invalid submodule size.", nil)
			}
		} else {
			size, err := strconv.ParseInt(fields[3], 10, 64)
			if err != nil || size < 0 {
				return nil, fail("unsupported-content", "Invalid Git blob size.", nil)
			}
			entry.size = size
		}
		if _, ok := entries[name]; ok {
			return nil, fail("unsupported-content", "Duplicate or conflicting Git path.", nil)
		}
		entries[name] = entry
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			existing, ok := entries[parent]
			if ok && !existing.IsDir() {
				return nil, fail("unsupported-content", "File and directory Git paths conflict.", nil)
			}
			entries[parent] = treeEntry{name: parent, mode: fs.ModeDir | 0755}
		}
	}
	return entries, nil
}

// Lstat returns immutable object metadata, preserving links as links rather than following them.
func (g *gitFiles) Lstat(name string) (fs.FileInfo, error) {
	if err := g.ctx.Err(); err != nil {
		return nil, contextFailure(err)
	}
	entry, ok := g.entries[name]
	if !ok {
		return nil, &fs.PathError{Op: "lstat", Path: name, Err: fs.ErrNotExist}
	}
	return entry, nil
}

// ReadDir returns a copy of the bounded sorted child inventory.
func (g *gitFiles) ReadDir(name string) ([]fs.DirEntry, error) {
	info, err := g.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	return slices.Clone(g.directories[name]), nil
}

// ReadFile verifies Git's batch framing before retaining a selected blob's original bytes.
func (g *gitFiles) ReadFile(name string) ([]byte, error) {
	info, err := g.Lstat(name)
	if err != nil {
		return nil, err
	}
	entry := g.entries[name]
	if !info.Mode().IsRegular() {
		return nil, fail("unsupported-content", name+": symlinks and submodules are unsupported.", nil)
	}
	if entry.size > maxBlobBytes || int64(g.total)+entry.size > maxRetainedBytes {
		return nil, fail("limit-exceeded", "Selected library content exceeds import byte limits.", nil)
	}
	result, err := g.revision.runner.run(g.ctx, g.revision.directory, []string{"cat-file", "--batch"}, int(entry.size)+4096, []byte(entry.object+"\n"))
	if err != nil {
		return nil, err
	}
	if result.status != 0 {
		return nil, fail("git-failed", "Could not read library blobs.", nil)
	}
	header, body, ok := bytes.Cut(result.output, []byte{'\n'})
	expected := fmt.Sprintf("%s blob %d", entry.object, entry.size)
	if !ok || string(header) != expected || int64(len(body)) != entry.size+1 || body[len(body)-1] != '\n' {
		return nil, fail("git-failed", "Git returned inconsistent or incomplete blob contents.", nil)
	}
	g.total += int(entry.size)
	return bytes.Clone(body[:len(body)-1]), nil
}
