// Retain original library bytes with the record of the revisions and rule versions that produced them.

package library

import (
	"bytes"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fabricahq/code-rules/internal/rules"
)

// Snapshot owns one source's original library bytes and the record of what was imported, which works like a lockfile.
// A verified digest establishes integrity against the record, not authenticity of its origin.
// Binary content remains bytes; required text is checked by the loader.
type Snapshot struct {
	Repository string `json:"repository"`
	// Pins repeats the source's pins when the snapshot was recorded; it is empty, never nil, when there are none.
	Pins map[string]rules.Pin `json:"pins"`
	// Ref repeats the source's ref when the snapshot was recorded, or is empty.
	Ref string `json:"ref,omitempty"`
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
	// Files maps each stored path to its original bytes. A rule from a library release other than Release is stored
	// under _releases/<number>/; see StoredPath. Files excludes _source.json.
	Files map[string][]byte `json:"files"`
}

// ImportedRule records one imported rule's version and the library release that published it.
type ImportedRule struct {
	// Version is nil, and Release is 0, when the rule's files aren't a published version, which only a ref other
	// than a library release can import.
	Version *rules.RuleVersion `json:"version"`
	Release int                `json:"release,omitempty"`
	// Commit is the full commit SHA of Release, or the snapshot's Commit when Version is nil.
	Commit string `json:"commit"`
}

// releasesDirectory holds the files of imported rules from library releases other than the snapshot's own.
const releasesDirectory = "_releases/"

// StoredPath returns where the snapshot stores the library file at file: the same path, or, for a file of an
// imported rule that a library release other than Release published, that path under _releases/<number>/.
func (s Snapshot) StoredPath(file string) string {
	id, ok := rules.VersionedRule(file)
	if !ok {
		return file
	}
	rule, ok := s.Rules[id]
	if !ok || rule.Release == s.Release || rule.Version == nil {
		return file
	}
	return releasesDirectory + strconv.Itoa(rule.Release) + "/" + file
}

// Store returns the stored files for catalog, which must be loaded from this snapshot's rules: each file's bytes,
// shared with the catalog, at its StoredPath.
func (s Snapshot) Store(catalog Catalog) map[string][]byte {
	stored := make(map[string][]byte, len(catalog.SupportingFiles))
	for file, data := range catalog.SupportingFiles {
		stored[s.StoredPath(file)] = data
	}
	for _, group := range catalog.Groups {
		for _, rule := range group.Rules {
			stored[s.StoredPath(rule.Path)] = []byte(rule.Document)
		}
	}
	return stored
}

// Source returns the snapshot's files at their library paths, as the catalog loader reads them: a file stored
// under _releases/<number>/ appears without that prefix. It fails when a stored path is malformed or two stored
// files share a library path. Whether each file is stored where StoredPath says is for the caller to compare.
func (s Snapshot) Source() (FileSource, error) {
	files := make(map[string][]byte, len(s.Files))
	for _, stored := range slices.Sorted(maps.Keys(s.Files)) {
		file := stored
		if rest, ok := strings.CutPrefix(stored, releasesDirectory); ok {
			number, inner, found := strings.Cut(rest, "/")
			if _, err := rules.ParseReleaseTag("release/" + number); err != nil || !found {
				return nil, bad(stored, "expected a rule file under _releases/<number>/")
			}
			file = inner
		}
		if _, exists := files[file]; exists {
			return nil, bad(stored, "the snapshot stores "+file+" more than once")
		}
		files[file] = s.Files[stored]
	}
	return newMemoryFiles(files), nil
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
