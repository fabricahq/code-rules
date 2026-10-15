// Place a library's files in its repository: each file's library path, which rule IDs, release records, and vendor
// snapshots use, and where the file lives in the repository's Code Rules directory.

package librarytree

import "strings"

// Directory is the Code Rules directory at the repository root. It holds a library's manifest, rules, shared
// assets, and change notes, beside the project files of the same repository.
const Directory = ".code-rules"

// Manifest is the library path of the library's manifest: the repository's configuration, .code-rules/config.yaml,
// whose library section makes the repository a library.
const Manifest = "config.yaml"

// LegacyManifest is the manifest at the repository root of a library in the layout before version 2, which also
// kept its groups, shared assets, and change notes at the repository root.
const LegacyManifest = "rule-library.yaml"

// RepositoryPath returns where the library file at library path file lives in its repository. The manifest is
// .code-rules/config.yaml; groups, at techs/ and practices/, live in .code-rules/rules/; shared assets, at assets/,
// and change notes, at changes/, live in .code-rules/. Any other path is a license or notice file, at the same path
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

// SnapshotPath returns where the library file at library path file is stored in a project's snapshot of the
// library, vendor/<source>/: its path relative to the library's Code Rules directory, as RepositoryPath places it,
// so a rule at rules/techs/go/x.md keeps its relative links to shared assets at assets/. A license or notice file
// outside the Code Rules directory keeps its path relative to the repository root.
func SnapshotPath(file string) string {
	return strings.TrimPrefix(RepositoryPath(file), Directory+"/")
}

// LibraryPath returns the library path of the file or directory at repository path file, the inverse of
// RepositoryPath. It reports false for a path that holds no library file: anything else in .code-rules/, such as
// vendor/, generated/, or rules/README.md, and a path outside .code-rules/ whose library path would be one of a
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
