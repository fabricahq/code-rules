// Place a library's files in its repository: each file's library path, which rule IDs, release records, and vendor
// snapshots use, and where the file lives in the repository's Syllabus directory.

package librarytree

import (
	"iter"
	"strings"
)

// Directory is the Syllabus directory at the repository root. It holds a library's manifest, rules, shared
// assets, and change notes, beside the project files of the same repository.
const Directory = ".syllabus"

// LegacyDirectory is the Syllabus directory's name from before Code Rules was renamed Syllabus. Projects that
// haven't renamed it, and every library release published before the rename, keep their files there.
const LegacyDirectory = ".code-rules"

// UsesLegacyDirectory reports whether a tree, given by its repository paths, keeps its Syllabus directory under
// LegacyDirectory: it has a file there and none in Directory.
func UsesLegacyDirectory(paths iter.Seq[string]) bool {
	legacy := false
	for name := range paths {
		if strings.HasPrefix(name, Directory+"/") {
			return false
		}
		legacy = legacy || strings.HasPrefix(name, LegacyDirectory+"/")
	}
	return legacy
}

// Upgrade returns the repository path file of a tree that keeps its Syllabus directory under LegacyDirectory with
// that directory spelled as Directory, so the tree reads like one made after the rename.
func Upgrade(file string) string {
	if inside, found := strings.CutPrefix(file, LegacyDirectory+"/"); found {
		return Directory + "/" + inside
	}
	return file
}

// Manifest is the library path of the library's manifest: the repository's configuration, .syllabus/config.yaml,
// whose library section makes the repository a library.
const Manifest = "config.yaml"

// LegacyManifest is the manifest at the repository root of a library in the layout before version 2, which also
// kept its groups, shared assets, and change notes at the repository root.
const LegacyManifest = "rule-library.yaml"

// RepositoryPath returns where the library file at library path file lives in its repository. The manifest is
// .syllabus/config.yaml; groups, at techs/ and practices/, live in .syllabus/rules/; shared assets, at assets/,
// and change notes, at changes/, live in .syllabus/. Any other path is a license or notice file, at the same path
// relative to the repository root.
func RepositoryPath(file string) string {
	if file == Manifest {
		return Directory + "/" + Manifest
	}
	switch first, _, _ := strings.Cut(file, "/"); first {
	case "techs", "practices":
		return Directory + "/rules/" + file
	case "assets", "changes":
		return Directory + "/" + file
	}
	return file
}

// LibraryPath returns the library path of the file or directory at repository path file, the inverse of
// RepositoryPath. It reports false for a path that holds no library file: anything else in .syllabus/, such as
// vendor/, generated/, or rules/README.md, and a path outside .syllabus/ whose library path would be one of a
// file inside it, such as assets/logo.svg or config.yaml at the repository root.
func LibraryPath(file string) (string, bool) {
	inside, found := strings.CutPrefix(file, Directory+"/")
	if !found {
		if file == Directory || RepositoryPath(file) != file {
			return "", false
		}
		return file, true
	}
	if rule, found := strings.CutPrefix(inside, "rules/"); found {
		if first, _, _ := strings.Cut(rule, "/"); first == "techs" || first == "practices" {
			return rule, true
		}
		return "", false
	}
	if first, _, _ := strings.Cut(inside, "/"); first == "assets" || first == "changes" || inside == Manifest {
		return inside, true
	}
	return "", false
}
