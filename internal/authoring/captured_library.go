// Adapt a bounded, already inspected filesystem snapshot to catalog validation without live rereads.
package authoring

import (
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/fabricahq/code-rules/internal/project"
)

// capturedLibrary lends immutable captured files and directories to the catalog reader.
type capturedLibrary struct{ tree *project.Tree }

// Lstat reports only paths that exist in the captured snapshot, including empty directories.
func (s capturedLibrary) Lstat(name string) (fs.FileInfo, error) {
	if name == "." || slices.Contains(s.tree.Directories, name) {
		return capturedInfo{name: name, mode: fs.ModeDir | 0755}, nil
	}
	if data, ok := s.tree.Files[name]; ok {
		return capturedInfo{name: name, size: int64(len(data)), mode: 0644}, nil
	}
	// Declared term files can introduce parent directories outside conventional trees.
	for file := range s.tree.Files {
		if strings.HasPrefix(file, name+"/") {
			return capturedInfo{name: name, mode: fs.ModeDir | 0755}, nil
		}
	}
	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

// ReadFile lends original bytes only; the catalog reader does not mutate its source.
func (s capturedLibrary) ReadFile(name string) ([]byte, error) {
	if data, ok := s.tree.Files[name]; ok {
		return data, nil
	}
	return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
}

// ReadDir returns sorted direct children, synthesizing ancestors of declared term files.
func (s capturedLibrary) ReadDir(name string) ([]fs.DirEntry, error) {
	info, err := s.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	prefix := name + "/"
	if name == "." {
		prefix = ""
	}
	children := map[string]bool{}
	add := func(candidate string) {
		if strings.HasPrefix(candidate, prefix) {
			rest := strings.TrimPrefix(candidate, prefix)
			if rest != "" {
				children[prefix+strings.SplitN(rest, "/", 2)[0]] = true
			}
		}
	}
	for file := range s.tree.Files {
		add(file)
	}
	for _, dir := range s.tree.Directories {
		add(dir)
	}
	entries := []fs.DirEntry{}
	for child := range children {
		info, err := s.Lstat(child)
		if err != nil {
			return nil, err
		}
		entries = append(entries, fs.FileInfoToDirEntry(info))
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, nil
}

// capturedInfo supplies stable filesystem metadata for captured regular files and directories.
type capturedInfo struct {
	name string
	size int64
	mode fs.FileMode
}

// Name returns the final component of the captured path.
func (i capturedInfo) Name() string { return path.Base(i.name) }

// Size reports the captured byte count.
func (i capturedInfo) Size() int64 { return i.size }

// Mode preserves whether the captured entry is a directory.
func (i capturedInfo) Mode() fs.FileMode { return i.mode }

// ModTime is unknown because validation depends on captured bytes.
func (i capturedInfo) ModTime() time.Time { return time.Time{} }

// IsDir distinguishes directories from captured regular files.
func (i capturedInfo) IsDir() bool { return i.mode.IsDir() }

// Sys has no live operating-system identity for an in-memory snapshot.
func (i capturedInfo) Sys() any { return nil }
