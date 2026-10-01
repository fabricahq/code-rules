// Retain original library bytes with the record of the revisions and rule versions that produced them.

package library

import (
	"bytes"
	"io/fs"
	"maps"
	"path"
	"slices"
	"time"

	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/fabricahq/code-rules/libraryformat"
)

// Snapshot owns one source's original library bytes and the record of what was imported, which works like a lockfile.
// A verified digest establishes integrity against the record, not authenticity of its origin.
// Binary content remains bytes; required text is checked by the loader.
type Snapshot struct {
	Repository string `json:"repository"`
	// RetiredRules lists, sorted, the rules the library had retired, as far as the snapshot's record knows, that the
	// source's groups or rules list selects; it is empty, never nil, when there are none. It is a fact about the
	// library, not the source's configuration, so offline checks can tell a pin or exclusion of a retired rule from
	// a typo. Configuration alone owns pins: each imported rule's version is what a pin must match.
	RetiredRules []string `json:"retiredRules"`
	// Ref repeats the source's ref when the snapshot was recorded, or is the zero GitRef.
	Ref rules.GitRef `json:"ref,omitzero"`
	// Release is the library release that supplied the library-wide files, or 0 when Ref names a revision other
	// than a library release.
	Release int `json:"release,omitempty"`
	// Commit is the full commit SHA of Release, or of the revision Ref names.
	Commit string `json:"resolvedCommit"`
	// Selection retains the requested group IDs or wildcard; Groups lists the groups imported in full.
	Selection rules.GroupSelection `json:"groupSelection"`
	Groups    []string             `json:"groups"`
	// RuleSelection lists the individually selected rule IDs, sorted; it is empty, never nil, when there are none.
	RuleSelection []string `json:"ruleSelection"`
	// Rules maps each imported rule's library rule ID, however it was selected, to its version record.
	Rules map[string]ImportedRule `json:"rules"`
	// Files maps each library path to its original bytes, excluding _source.json. The snapshot holds one version of
	// each imported rule, so every file is stored at its library path.
	Files map[string][]byte `json:"files"`
}

// ImportedRule records one imported rule's version and the library release that published it.
type ImportedRule struct {
	// Version is nil, and Release is 0, when the rule's files aren't a published version, which only a ref other
	// than a library release can import.
	Version *libraryformat.RuleVersion `json:"version"`
	Release int                        `json:"release,omitempty"`
	// Commit is the full commit SHA of Release, or the snapshot's Commit when Version is nil.
	Commit string `json:"commit"`
}

// Source returns the snapshot's files for the catalog loader to read.
func (s Snapshot) Source() FileSource {
	return newMemoryFiles(s.Files)
}

// memoryFiles lends in-memory files and the directories their paths imply to the catalog reader.
type memoryFiles struct {
	files       map[string][]byte
	directories map[string][]fs.DirEntry
}

// newMemoryFiles indexes the directories of files, which must be valid, contained paths.
func newMemoryFiles(files map[string][]byte) memoryFiles {
	children := map[string]map[string]fs.FileInfo{".": {}}
	for file, data := range files {
		name := file
		var info fs.FileInfo = memoryInfo{name: name, size: int64(len(data)), mode: 0644}
		for {
			parent := path.Dir(name)
			if children[parent] == nil {
				children[parent] = map[string]fs.FileInfo{}
			}
			children[parent][path.Base(name)] = info
			if parent == "." {
				break
			}
			name, info = parent, memoryInfo{name: parent, mode: fs.ModeDir | 0755}
		}
	}
	result := memoryFiles{files: files, directories: map[string][]fs.DirEntry{}}
	for directory, entries := range children {
		list := make([]fs.DirEntry, 0, len(entries))
		for _, name := range slices.Sorted(maps.Keys(entries)) {
			list = append(list, fs.FileInfoToDirEntry(entries[name]))
		}
		result.directories[directory] = list
	}
	return result
}

// Lstat reports files and the directories their paths imply.
func (m memoryFiles) Lstat(name string) (fs.FileInfo, error) {
	if data, ok := m.files[name]; ok {
		return memoryInfo{name: name, size: int64(len(data)), mode: 0644}, nil
	}
	if _, ok := m.directories[name]; ok {
		return memoryInfo{name: name, mode: fs.ModeDir | 0755}, nil
	}
	return nil, &fs.PathError{Op: "lstat", Path: name, Err: fs.ErrNotExist}
}

// ReadFile lends original bytes; the catalog reader does not mutate its source.
func (m memoryFiles) ReadFile(name string) ([]byte, error) {
	if data, ok := m.files[name]; ok {
		return bytes.Clone(data), nil
	}
	return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
}

// ReadDir returns a copy of a directory's sorted entries.
func (m memoryFiles) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, ok := m.directories[name]
	if !ok {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	return slices.Clone(entries), nil
}

// memoryInfo supplies stable metadata for in-memory files and directories.
type memoryInfo struct {
	name string
	size int64
	mode fs.FileMode
}

// Name returns the final path component.
func (i memoryInfo) Name() string { return path.Base(i.name) }

// Size reports a file's byte count, or zero for a directory.
func (i memoryInfo) Size() int64 { return i.size }

// Mode distinguishes regular files from directories.
func (i memoryInfo) Mode() fs.FileMode { return i.mode }

// ModTime is zero because validation depends only on bytes.
func (i memoryInfo) ModTime() time.Time { return time.Time{} }

// IsDir reports whether the entry is a directory.
func (i memoryInfo) IsDir() bool { return i.mode.IsDir() }

// Sys has no operating-system identity.
func (i memoryInfo) Sys() any { return nil }
