// Validate all authored library content, including unreferenced assets, without generating or modifying files.

package library

import (
	"bytes"
	"context"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
	"golang.org/x/text/unicode/norm"
)

// CheckResult reports complete adoption counts and explicit licensing caveats.
type CheckResult struct {
	Groups   int      `json:"groups"`
	Rules    int      `json:"rules"`
	Warnings []string `json:"warnings"`
}

// Check validates a captured library snapshot, ignoring unrelated files such as .git.
// A final comparison rejects observed changes; ordinary editors are not locked out.
func Check(ctx context.Context, options Options) (CheckResult, error) {
	root, err := openLibrary(ctx, options, false)
	if err != nil {
		return CheckResult{}, err
	}
	defer root.Close()
	if err = filetxn.RequireIdle(root); err != nil {
		return CheckResult{}, err
	}
	snapshot, license, err := libraryCheckInput(ctx, root)
	if err != nil {
		return CheckResult{}, err
	}
	if err = validateLibraryInventory(ctx, snapshot.Files, rules.LicensePaths(license)); err != nil {
		return CheckResult{}, err
	}
	catalog, err := LoadSource(ctx, capturedLibrary{snapshot}, "library", rules.GroupSelection{Pattern: "*"})
	if err != nil {
		return CheckResult{}, err
	}
	result := CheckResult{Groups: len(catalog.Groups), Warnings: []string{}}
	for _, group := range catalog.Groups {
		result.Rules += len(group.Rules)
	}
	if license == nil {
		result.Warnings = append(result.Warnings, "License is undeclared. Decide terms before sharing this library.")
	} else if license.SPDXExpression == nil {
		result.Warnings = append(result.Warnings, "The license has no SPDX expression. Declare the library terms explicitly.")
	}
	if err = ctx.Err(); err != nil {
		return CheckResult{}, err
	}
	if err = requireLibraryUnchanged(ctx, root, snapshot); err != nil {
		return CheckResult{}, err
	}
	if err = filetxn.RequireIdle(root); err != nil {
		return CheckResult{}, err
	}
	return result, nil
}

// validateLibraryInventory catches malformed unused assets, invalid files, draft markers, and missing local destinations.
func validateLibraryInventory(ctx context.Context, files map[string][]byte, terms []string) error {
	spellings := map[string]string{}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := ctx.Err(); err != nil {
			return err
		}
		parts := strings.Split(name, "/")
		for n := 1; n <= len(parts); n++ {
			prefix := strings.Join(parts[:n], "/")
			key := norm.NFC.String(strings.ToUpper(strings.ToLower(norm.NFC.String(prefix))))
			if old, ok := spellings[key]; ok && old != prefix {
				return failure("path-collision", name+": collides with "+old, nil)
			}
			spellings[key] = prefix
		}
		if name == "rule-library.yaml" || slices.Contains(terms, name) || rules.IsGroupReadme(name) {
			continue
		}
		asset := slices.Contains(parts, "assets")
		if asset {
			directory := rules.AssetDirectory(name)
			if directory == "" {
				return failure("invalid-asset", name+": invalid asset directory", nil)
			}
			if directory != "assets/" {
				ownerParts := strings.Split(strings.TrimSuffix(directory, "/"), "/")
				owner := strings.Join(ownerParts[:len(ownerParts)-2], "/") + "/" + ownerParts[len(ownerParts)-1] + ".md"
				if _, ok := files[owner]; !ok {
					return failure("invalid-asset", name+": missing rule owner "+owner, nil)
				}
			}
		} else {
			if len(parts) < 3 {
				return failure("invalid-library", name+": expected a group document", nil)
			}
			group := strings.Join(parts[:2], "/")
			if err := rules.ValidateGroupID(group, name); err != nil {
				return err
			}
			if name != group+"/_group.yaml" {
				if _, err := rules.GroupFromPath(name, name); err != nil {
					return failure("invalid-library", name+": use rule Markdown, _group.yaml, or a conventional assets directory", err)
				}
				if bytes.Contains(files[name], []byte("<!-- code-rules:draft -->")) {
					return failure("incomplete-rule", name+": complete the draft and remove its code-rules:draft marker", nil)
				}
			}
		}
		if strings.HasSuffix(name, ".md") {
			if !utf8.Valid(files[name]) {
				return failure("invalid-library", name+": expected UTF-8 Markdown", nil)
			}
			targets, err := rules.MarkdownTargets(string(files[name]), name)
			if err != nil {
				return err
			}
			for _, target := range targets {
				if err = rules.RequireAllowedTarget(name, target, terms); err != nil {
					return err
				}
				if _, ok := files[target]; !ok {
					return failure("missing-link", name+": missing link destination "+target, nil)
				}
			}
		}
		if strings.HasPrefix(strings.ReplaceAll(string(files[name][:min(len(files[name]), 128)]), "\r\n", "\n"), "version https://git-lfs.github.com/spec/v1\n") {
			return failure("invalid-library", path.Clean(name)+": Git LFS pointers are unsupported", nil)
		}
	}
	return nil
}

// libraryCheckInput captures every manifest, term, and library-owned file considered by a complete check.
func libraryCheckInput(ctx context.Context, root *os.Root) (*filetxn.Tree, *rules.LicenseDeclaration, error) {
	inventory, err := readLocalInventory(ctx, root)
	if err != nil {
		return nil, nil, err
	}
	return &filetxn.Tree{Files: inventory.Files, Directories: inventory.Directories}, inventory.License, nil
}

// requireLibraryUnchanged rejects differences observed after validating the captured snapshot.
func requireLibraryUnchanged(ctx context.Context, root *os.Root, before *filetxn.Tree) error {
	after, _, err := libraryCheckInput(ctx, root)
	if err != nil {
		return failure("changed-input", "library changed during validation; retry library check", err)
	}
	if !maps.EqualFunc(before.Files, after.Files, bytes.Equal) || !slices.Equal(before.Directories, after.Directories) {
		return failure("changed-input", "library changed during validation; retry library check", nil)
	}
	return nil
}
