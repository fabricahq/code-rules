// Retain owned attachments and referenced shared assets through the same bounded reader.

package library

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/fabricahq/code-rules/internal/rules"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// registerPath prevents reserved paths and case/Unicode collisions in every retained prefix.
func (r *reader) registerPath(file string) error {
	if r.spellings == nil {
		r.spellings = map[string]string{}
	}
	parts := strings.Split(file, "/")
	for i, part := range parts {
		if strings.EqualFold(part, ".git") || (i == 0 && strings.EqualFold(part, "_source.json")) {
			return bad(file, "reserved library path")
		}
		prefix := strings.Join(parts[:i+1], "/")
		key := norm.NFC.String(cases.Fold().String(norm.NFC.String(prefix)))
		if previous, ok := r.spellings[key]; ok && previous != prefix {
			return bad(file, "case or Unicode path collision with "+previous)
		}
		r.spellings[key] = prefix
	}
	return nil
}

// ownedAssets retains complete rule-owned directories while exempting declared terms and term-only directories.
func (r *reader) ownedAssets(directory string, terms []string) error {
	entries, err := r.entries(directory, false)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		assetPath := directory + "/" + entry.Name()
		if slices.Contains(terms, assetPath) {
			continue
		}
		owner := path.Dir(directory) + "/" + entry.Name() + ".md"
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return bad(assetPath, "expected an owned assets directory")
		}
		onlyTerms, err := r.termDirectory(assetPath, terms)
		if err != nil {
			return err
		}
		if onlyTerms {
			continue
		}
		if _, err := rules.GroupFromPath(owner, assetPath); err != nil {
			return err
		}
		if slices.Contains(terms, owner) {
			return bad(assetPath, "license files cannot own rule assets")
		}
		info, err := r.input.Lstat(owner)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return bad(assetPath, "assets directory has no adjacent owning rule: "+owner)
			}
			return fmt.Errorf("inspect asset owner %s: %w", owner, err)
		}
		if !info.Mode().IsRegular() {
			return bad(assetPath, "assets directory has no adjacent owning rule: "+owner)
		}
		if err := r.assetTree(assetPath); err != nil {
			return err
		}
	}
	return nil
}

// assetTree preserves every regular file beneath an attachment directory, including binary data.
func (r *reader) assetTree(directory string) error {
	entries, err := r.entries(directory, false)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		file := directory + "/" + entry.Name()
		if entry.IsDir() {
			if err := r.assetTree(file); err != nil {
				return err
			}
		} else if _, err := r.read(file); err != nil {
			return err
		}
	}
	return nil
}

// supportingLinks checks retained Markdown and loads shared assets until no new files remain.
// Links to other rules fail before checking target existence, including inside attachments.
func (r *reader) supportingLinks(terms []string) error {
	checked := map[string]bool{}
	shared := false
	for {
		pending := []string{}
		for file := range r.files {
			if !checked[file] {
				pending = append(pending, file)
			}
		}
		if len(pending) == 0 {
			return r.ctx.Err()
		}
		slices.Sort(pending)
		for _, file := range pending {
			if err := r.ctx.Err(); err != nil {
				return err
			}
			checked[file] = true
			if !strings.HasSuffix(file, ".md") || slices.Contains(terms, file) {
				continue
			}
			data := r.files[file]
			if !utf8.Valid(data) {
				return bad(file, "expected UTF-8 Markdown")
			}
			targets, err := rules.MarkdownTargets(string(data), file)
			if cancelErr := r.ctx.Err(); cancelErr != nil {
				return cancelErr
			}
			if err != nil {
				return fmt.Errorf("inspect links in %s: %w", file, err)
			}
			for _, target := range targets {
				if err := r.ctx.Err(); err != nil {
					return err
				}
				if err := rules.RequireAllowedTarget(file, target, terms); err != nil {
					return err
				}
				if err := r.linkExists(target); err != nil {
					return fmt.Errorf("link from %s: %w", file, err)
				}
				if rules.AssetDirectory(target) == "assets/" && !shared {
					shared = true
					if err := r.assetTree("assets"); err != nil {
						return err
					}
				}
			}
		}
	}
}

// linkExists checks every component without reading the destination.
func (r *reader) linkExists(file string) error {
	if !fs.ValidPath(file) {
		return bad(file, "invalid link destination")
	}
	if err := r.registerPath(file); err != nil {
		return err
	}
	parts := strings.Split(file, "/")
	for i := range parts {
		prefix := strings.Join(parts[:i+1], "/")
		info, err := r.input.Lstat(prefix)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return bad(file, "missing link destination")
			}
			return fmt.Errorf("inspect link destination %s: %w", prefix, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return bad(prefix, "symlinks are unsupported")
		}
		if i < len(parts)-1 && !info.IsDir() {
			return bad(prefix, "link path component must be a directory")
		}
		if i == len(parts)-1 && !info.Mode().IsRegular() {
			return bad(file, "link destination must be an ordinary file")
		}
	}
	return nil
}
