// Retain original library bytes with the record of the revisions and rule versions that produced them.

package library

import (
	"bytes"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/fabricahq/code-rules/coderules"
	"github.com/fabricahq/code-rules/internal/rules"
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
	// Files maps each library path to its original bytes, excluding _source.json, except config.yaml, which holds
	// what the library publishes of its manifest, as rules.PublishedManifest gives it. The snapshot holds one version
	// of each imported rule, so every file is stored at its library path. Each skill's files are stored in the
	// folder its ID names, such as skills/build-feature/SKILL.md.
	Files map[string][]byte `json:"files"`
	// NonLibrary reports that the source's repository has no library manifest at Commit, so the snapshot holds only
	// skills, at their paths in the repository, and no rules, release, or catalog files.
	NonLibrary bool `json:"nonLibrary,omitempty"`
	// SkillSelection repeats the source's skills.include selectors when the snapshot was recorded, sorted; it is
	// empty, never nil, when there are none.
	SkillSelection []string `json:"skillSelection"`
	// Skills maps the ID of each skill the source installs to its record; it is empty, never nil, when there are
	// none. Every skill comes from Commit.
	Skills map[string]ImportedSkill `json:"skills"`
	// Executable lists, sorted, the paths in Files whose executable bit the repository records; it is empty, never
	// nil, when there are none.
	Executable []string `json:"executable"`
}

// ImportedSkill records one installed skill from a source.
type ImportedSkill struct {
	// Level is the skill's level in its library, which is "offered" for every skill a library publishes, or empty
	// for a skill from a repository that isn't a library.
	Level string `json:"level,omitempty"`
}

// SkillFiles returns the files of skill id, by path in its folder. The bytes are shared with the snapshot.
func (s Snapshot) SkillFiles(id string) map[string][]byte {
	files := map[string][]byte{}
	for name, data := range s.Files {
		if relative, ok := strings.CutPrefix(name, id+"/"); ok {
			files[relative] = data
		}
	}
	return files
}

// SkillExecutable returns, sorted, the executable files of skill id, by path in its folder.
func (s Snapshot) SkillExecutable(id string) []string {
	executable := []string{}
	for _, name := range s.Executable {
		if relative, ok := strings.CutPrefix(name, id+"/"); ok {
			executable = append(executable, relative)
		}
	}
	return executable
}

// IsSkillFile reports whether file, a path in Files, belongs to one of the snapshot's skills.
func (s Snapshot) IsSkillFile(file string) bool {
	for id := range s.Skills {
		if strings.HasPrefix(file, id+"/") {
			return true
		}
	}
	return false
}

// ImportedRule records one imported rule's version and the library release that published it.
type ImportedRule struct {
	// Version is nil, and Release is 0, when the rule's files aren't a published version, which only a ref other
	// than a library release can import.
	Version *coderules.RuleVersion `json:"version"`
	Release int                    `json:"release,omitempty"`
	// Commit is the full commit SHA of Release, or the snapshot's Commit when Version is nil.
	Commit string `json:"commit"`
	// LegacyLayout reports that Commit keeps the library in the layout before version 2, with the rule's files at
	// their library paths from the repository root rather than in .code-rules/rules/.
	LegacyLayout bool `json:"legacyLayout,omitempty"`
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
