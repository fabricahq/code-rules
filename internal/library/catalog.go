// Package library reads bounded rule catalogs from local files or immutable Git objects.
package library

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/fabricahq/code-rules/internal/rules"
)

const (
	maxFileBytes  = 8 * 1024 * 1024
	maxTotalBytes = 64 * 1024 * 1024
	maxFiles      = 10_000
)

// Catalog owns selected rules and supporting file bytes. Each rule owns its
// original Document; SupportingFiles holds only manifest, group metadata, and terms.
// SupportingFiles also holds complete owned assets and referenced shared assets.
type Catalog struct {
	Selection       rules.GroupSelection      `json:"groupSelection"`
	Groups          []Group                   `json:"groups"`
	License         *rules.LicenseDeclaration `json:"license"`
	SupportingFiles map[string][]byte         `json:"supportingFiles"`
}

// Group includes display metadata and path-sorted rules; empty groups are valid.
type Group struct {
	ID       string              `json:"id"`
	Metadata rules.GroupMetadata `json:"metadata"`
	Rules    []rules.Rule        `json:"rules"`
}

// reader binds resource limits and cancellation to one rooted catalog read.
type reader struct {
	ctx         context.Context
	input       FileSource
	files       map[string][]byte
	total       int
	visited     int
	spellings   map[string]string
	directories map[string][]fs.DirEntry
}

var sourceAlias = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Load expands the selection and validates every selected rule, returning no partial catalog.
// The caller owns and closes root. No files are written. Symlinks and special entries
// encountered in selected trees are rejected; os.Root confines concurrent path resolution.
// Cancellation remains inspectable with errors.Is. Other input failures are ValidationError.
func Load(ctx context.Context, root *os.Root, source string, selection rules.GroupSelection) (Catalog, error) {
	if root == nil {
		return Catalog{}, bad("library", "expected an open filesystem root")
	}
	return LoadSource(ctx, rootFiles{ctx: ctx, root: root}, source, selection)
}

// LoadSource applies the same catalog rules to bounded local or immutable Git bytes.
// The caller owns the source and must keep its identity stable for this operation.
func LoadSource(ctx context.Context, input FileSource, source string, selection rules.GroupSelection) (Catalog, error) {
	if err := ctx.Err(); err != nil {
		return Catalog{}, fmt.Errorf("load library: %w", err)
	}
	if input == nil {
		return Catalog{}, bad("library", "expected a file source")
	}
	if source == "local" {
		return Catalog{}, bad("source", "local is reserved for project rules; choose another source alias")
	}
	if !sourceAlias.MatchString(source) {
		return Catalog{}, bad("source", "expected a lowercase source alias")
	}
	if selection.Pattern != "" && selection.Groups != nil {
		return Catalog{}, bad("groups", "specify a pattern or explicit IDs, not both")
	}
	encoded, err := json.Marshal(selection)
	if err != nil {
		return Catalog{}, fmt.Errorf("encode group selection: %v", err)
	}
	selection, err = rules.ParseGroupSelection(encoded, "groups")
	if err != nil {
		return Catalog{}, err
	}
	r := reader{ctx: ctx, input: input, files: make(map[string][]byte)}
	if _, err := r.read("rule-library.json"); err != nil {
		return Catalog{}, err
	}
	license, err := r.license(source)
	if err != nil {
		return Catalog{}, err
	}
	terms := rules.LicensePaths(license)
	ids, err := r.groups(selection, terms)
	if err != nil {
		return Catalog{}, err
	}
	catalog := Catalog{Selection: selection, Groups: make([]Group, 0, len(ids)), License: license}
	for _, id := range ids {
		group, err := r.group(id, source, terms)
		if err != nil {
			return Catalog{}, err
		}
		catalog.Groups = append(catalog.Groups, group)
	}
	if err := r.supportingLinks(terms); err != nil {
		return Catalog{}, err
	}
	// Transfer each rule document to its Rule; retain only supporting files in the map.
	for _, group := range catalog.Groups {
		for _, rule := range group.Rules {
			delete(r.files, rule.Path)
		}
	}
	catalog.SupportingFiles = r.files
	return catalog, nil
}

// bad assigns validation context without logging authored file contents.
func bad(location, problem string) error {
	return &rules.ValidationError{Location: location, Problem: problem}
}

// read rejects observed symlink components and bounds actual bytes through a confined handle.
func (r *reader) read(path string) ([]byte, error) {
	if data, ok := r.files[path]; ok {
		return data, nil
	}
	if err := r.ctx.Err(); err != nil {
		return nil, fmt.Errorf("read library file: %w", err)
	}
	if !fs.ValidPath(path) || strings.ContainsAny(path, "\\\x00") {
		return nil, bad(path, "expected a contained relative file path")
	}
	if err := r.registerPath(path); err != nil {
		return nil, err
	}
	parts := strings.Split(path, "/")
	for i := range parts {
		prefix := strings.Join(parts[:i+1], "/")
		info, err := r.input.Lstat(prefix)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				if path == "rule-library.json" {
					return nil, bad(path, "missing library manifest at the repository root; this required file declares the library format and optional license")
				}
				return nil, bad(path, "missing required file")
			}
			return nil, fmt.Errorf("inspect library path %s: %w", prefix, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, bad(prefix, "symlinks are unsupported")
		}
		if i < len(parts)-1 && !info.IsDir() {
			return nil, bad(prefix, "expected a directory")
		}
		if i == len(parts)-1 && !info.Mode().IsRegular() {
			return nil, bad(path, "expected an ordinary file")
		}
	}
	data, err := r.input.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read library file %s: %w", path, err)
	}
	if err := r.ctx.Err(); err != nil {
		return nil, fmt.Errorf("read library file: %w", err)
	}
	if len(data) > maxFileBytes || r.total+len(data) > maxTotalBytes || len(r.files) >= maxFiles {
		return nil, bad(path, "library exceeds file or total read limits")
	}
	if strings.HasPrefix(strings.ReplaceAll(string(data[:min(len(data), 128)]), "\r\n", "\n"), "version https://git-lfs.github.com/spec/v1\n") {
		return nil, bad(path, "Git LFS pointers are unsupported")
	}
	r.total += len(data)
	r.files[path] = data
	return data, nil
}

// license validates the declaration before reading its contained files; bytes stay unchanged.
func (r *reader) license(source string) (*rules.LicenseDeclaration, error) {
	manifest := r.files["rule-library.json"]
	inventory := map[string][]byte{"rule-library.json": manifest}
	var raw struct {
		License struct {
			File    string
			Notices []string
		}
	}
	// Invalid shapes are reported by the domain parser, not by this discovery pass.
	if json.Unmarshal(manifest, &raw) == nil {
		if raw.License.File != "rule-library.json" {
			inventory[raw.License.File] = nil
		}
		for _, path := range raw.License.Notices {
			if path != "rule-library.json" {
				inventory[path] = nil
			}
		}
	}
	declaration, err := rules.ReadLibraryLicense(inventory, source)
	if err != nil {
		return nil, err
	}
	for _, path := range rules.LicensePaths(declaration) {
		if _, err := r.read(path); err != nil {
			return nil, err
		}
	}
	return declaration, nil
}

// entries lists a real directory and applies cancellation and discovery-count limits.
func (r *reader) entries(path string, optional bool) ([]fs.DirEntry, error) {
	if err := r.ctx.Err(); err != nil {
		return nil, fmt.Errorf("discover library groups: %w", err)
	}
	if entries, ok := r.directories[path]; ok {
		return entries, nil
	}
	info, err := r.input.Lstat(path)
	if optional && errors.Is(err, fs.ErrNotExist) {
		return []fs.DirEntry{}, nil
	}
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, bad(path, "missing required group directory")
		}
		return nil, fmt.Errorf("inspect directory %s: %w", path, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, bad(path, "expected a directory without symlinks")
	}
	entries, err := r.input.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("list directory %s: %w", path, err)
	}
	r.visited += len(entries)
	if r.visited > maxFiles {
		return nil, bad(path, "library exceeds 10,000 discovered entries")
	}
	// File-system enumeration order must not affect selected groups or the first error.
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	if r.directories == nil {
		r.directories = make(map[string][]fs.DirEntry)
	}
	r.directories[path] = entries
	return entries, nil
}

// groups expands supported patterns over metadata-bearing directories, including empty groups.
func (r *reader) groups(selection rules.GroupSelection, terms []string) ([]string, error) {
	if selection.Pattern == "" {
		return slices.Clone(selection.Groups), nil
	}
	roots := []string{"practices", "techs"}
	if selection.Pattern != "*" {
		roots = []string{strings.TrimSuffix(selection.Pattern, "/*")}
	}
	ids := []string{}
	for _, root := range roots {
		entries, err := r.entries(root, true)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			id := root + "/" + entry.Name()
			if slices.Contains(terms, id) {
				continue
			}
			if entry.IsDir() {
				onlyTerms, err := r.termDirectory(id, terms)
				if err != nil {
					return nil, err
				}
				if onlyTerms {
					continue
				}
			}
			if err := rules.ValidateGroupID(id, id); err != nil {
				return nil, err
			}
			if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return nil, bad(id, "expected a group directory without symlinks")
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// group validates metadata and every rule under one selected group, retaining original bytes.
func (r *reader) group(id, source string, terms []string) (Group, error) {
	metadataPath := id + "/_group.json"
	data, err := r.read(metadataPath)
	if err != nil {
		return Group{}, err
	}
	metadata, err := rules.ParseGroupMetadata(data, source+"/"+metadataPath)
	if err != nil {
		return Group{}, err
	}
	paths := []string{}
	if err := r.rulePaths(id, metadataPath, terms, &paths); err != nil {
		return Group{}, err
	}
	slices.Sort(paths)
	group := Group{ID: id, Metadata: metadata, Rules: []rules.Rule{}}
	for _, path := range paths {
		data, err := r.read(path)
		if err != nil {
			return Group{}, err
		}
		if !utf8.Valid(data) {
			return Group{}, bad(path, "expected UTF-8 text")
		}
		if strings.HasPrefix(strings.ReplaceAll(string(data[:min(len(data), 128)]), "\r\n", "\n"), "version https://git-lfs.github.com/spec/v1\n") {
			return Group{}, bad(path, "Git LFS pointers are unsupported")
		}
		rule, err := rules.Parse(string(data), path, source)
		if err != nil {
			return Group{}, err
		}
		group.Rules = append(group.Rules, rule)
	}
	return group, nil
}

// rulePaths discovers selected rules and their complete owned assets; unsafe entries fail closed.
func (r *reader) rulePaths(directory, metadata string, terms []string, paths *[]string) error {
	entries, err := r.entries(directory, false)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := directory + "/" + entry.Name()
		if entry.Type()&os.ModeSymlink != 0 {
			return bad(path, "symlinks are unsupported")
		}
		if path == metadata || slices.Contains(terms, path) {
			continue
		}
		if entry.IsDir() {
			if entry.Name() == "assets" {
				if err := r.ownedAssets(path, terms); err != nil {
					return err
				}
				continue
			}
			if err := r.rulePaths(path, metadata, terms, paths); err != nil {
				return err
			}
			continue
		}
		if !entry.Type().IsRegular() {
			return bad(path, "expected an ordinary file")
		}
		if _, err := rules.GroupFromPath(path, path); err != nil {
			return err
		}
		*paths = append(*paths, path)
	}
	return nil
}

// Paths returns the exact read inventory in sorted order without exposing map iteration order.
func (c Catalog) Paths() []string {
	paths := slices.Collect(maps.Keys(c.SupportingFiles))
	for _, group := range c.Groups {
		for _, rule := range group.Rules {
			paths = append(paths, rule.Path)
		}
	}
	slices.Sort(paths)
	return paths
}

// termDirectory identifies directories containing only declared terms, without hiding actual groups.
func (r *reader) termDirectory(directory string, terms []string) (bool, error) {
	hasTerm := false
	for _, term := range terms {
		if strings.HasPrefix(term, directory+"/") {
			hasTerm = true
			break
		}
	}
	if !hasTerm {
		return false, nil
	}
	entries, err := r.entries(directory, false)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		file := directory + "/" + entry.Name()
		// Group metadata retains its group role even when it also supplies terms.
		if strings.Count(directory, "/") == 1 && entry.Name() == "_group.json" {
			return false, nil
		}
		if slices.Contains(terms, file) {
			continue
		}
		if !entry.IsDir() {
			return false, nil
		}
		onlyTerms, err := r.termDirectory(file, terms)
		if err != nil {
			return false, err
		}
		if !onlyTerms {
			return false, nil
		}
	}
	return true, nil
}
