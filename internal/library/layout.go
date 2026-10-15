// Open a library repository in the version 2 layout, refuse the earlier layout, and lend its files to the catalog
// reader by library path.

package library

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/librarytree"
)

// legacyLayoutProblem tells a library still in the layout before version 2 where its manifest and files belong now.
const legacyLayoutProblem = `this version of Syllabus reads a library's manifest from the library section of .syllabus/config.yaml, and its groups, shared assets, and change notes from .syllabus/. It no longer reads rule-library.yaml or rule folders at the repository root, and doesn't move them. Move them yourself, keeping each path below its folder:

  rule-library.yaml                 ->  the library section of .syllabus/config.yaml
  techs/go/_group.yaml              ->  .syllabus/rules/techs/go/_group.yaml
  techs/go/errors.md                ->  .syllabus/rules/techs/go/errors.md
  techs/go/assets/errors/           ->  .syllabus/rules/techs/go/assets/errors/
  practices/testing/                ->  .syllabus/rules/practices/testing/
  assets/retry-lifecycle.svg        ->  .syllabus/assets/retry-lifecycle.svg
  changes/                          ->  .syllabus/changes/

Then delete rule-library.yaml. Write its license fields, without formatVersion, in .syllabus/config.yaml, beside the sources the repository uses as a project, if any:

  schemaVersion: 2
  sources: {}
  library:
    license:
      spdxExpression: MIT
      file: LICENSE.md
      notices: []

License and notice paths stay relative to the repository root. Rule IDs and versions don't change. A moved rule sits two folders deeper, so its relative links to shared assets need one more ../, such as ../../../assets/retry-lifecycle.svg, and its links to license files at the repository root need two more; links that start with /assets/ still work. Once the library has a library release, record each rule whose links change with syllabus library change --bump patch. Then run syllabus library check.`

// requireLibraryLayout fails, writing nothing, when the repository at root keeps a library in the layout before
// version 2: rule-library.yaml, or a techs or practices entry, at the repository root, whatever its kind. It
// explains where each kind of file belongs now.
func requireLibraryLayout(root *os.Root) error {
	for _, name := range []string{librarytree.LegacyManifest, "practices", "techs"} {
		info, err := root.Lstat(name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect the repository root's %s: %v", name, err)
		}
		location := name
		if info.IsDir() {
			location += "/"
		}
		return &ValidationError{Location: location, Problem: legacyLayoutProblem}
	}
	return nil
}

// openSyllabusDirectory opens the Syllabus directory of the library repository at root, whose writer the
// repository's project and library commands share. create makes a missing directory; otherwise a missing one fails
// with needs-init. The caller closes the returned root.
func openSyllabusDirectory(ctx context.Context, root *os.Root, create bool) (*os.Root, error) {
	directory := filepath.Join(root.Name(), librarytree.Directory)
	if create {
		return filetxn.Create(ctx, directory)
	}
	dir, err := filetxn.Open(ctx, directory)
	var missing *filetxn.Error
	if errors.As(err, &missing) && missing.Code == "missing-root" {
		return nil, failure("needs-init", "missing "+librarytree.Directory+"/config.yaml; run syllabus library init first", err)
	}
	return dir, err
}

// requireCurrentDirectory fails, writing nothing, when the repository at root has no Syllabus directory but has one
// under the name Code Rules used, explaining how to rename it.
func requireCurrentDirectory(root *os.Root) error {
	if _, err := root.Lstat(librarytree.Directory); !errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if _, err := root.Lstat(librarytree.LegacyDirectory); err != nil {
		return nil
	}
	return &ValidationError{Location: librarytree.LegacyDirectory + "/", Problem: legacyDirectoryProblem}
}

// legacyDirectoryProblem tells a library that still keeps its files in the directory Code Rules used to move them.
const legacyDirectoryProblem = `Code Rules is now Syllabus, and library commands read a library's files from .syllabus/. Rename the directory, keeping everything in it, then run the command again:

  git mv .code-rules .syllabus

Library releases published from .code-rules/ stay readable, so projects keep their versions. Projects that import this library need no changes. Update the line in AGENTS.md or CLAUDE.md that points agents to .code-rules/generated/RULES.md.`

// repositoryFiles lends a library repository's files, read through input by their repository paths, to the
// catalog reader by their library paths, as librarytree.RepositoryPath places them. The library root, ".", lists
// the library's top-level entries: its manifest and its assets, changes, practices, and techs folders, each spelled
// as the repository spells it.
type repositoryFiles struct{ input FileSource }

// Lstat inspects the file at library path name without following its final symbolic link.
func (r repositoryFiles) Lstat(name string) (fs.FileInfo, error) {
	if name == "." {
		return r.input.Lstat(name)
	}
	return r.input.Lstat(librarytree.RepositoryPath(name))
}

// ReadFile reads the file at library path name.
func (r repositoryFiles) ReadFile(name string) ([]byte, error) {
	return r.input.ReadFile(librarytree.RepositoryPath(name))
}

// ReadDir lists the directory at library path name, which keeps its entries' names.
func (r repositoryFiles) ReadDir(name string) ([]fs.DirEntry, error) {
	if name != "." {
		return r.input.ReadDir(librarytree.RepositoryPath(name))
	}
	entries := []fs.DirEntry{}
	for _, directory := range []string{librarytree.Directory, librarytree.Directory + "/rules"} {
		listed, err := r.input.ReadDir(directory)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range listed {
			if library, ok := librarytree.LibraryPath(directory + "/" + entry.Name()); ok && !strings.Contains(library, "/") {
				entries = append(entries, entry)
			}
		}
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, nil
}

// requireIdle fails with busy while a writer holds, or a pending recovery waits in, the Syllabus directory of the
// library repository at root, so a read-only command never reads a half-published edit.
func requireIdle(ctx context.Context, root *os.Root) error {
	dir, err := openSyllabusDirectory(ctx, root, false)
	if err != nil {
		return err
	}
	defer dir.Close()
	return filetxn.RequireIdle(dir)
}
