// Validate all authored library content, including unreferenced assets, without generating or modifying files.

package library

import (
	"bytes"
	"context"
	"iter"
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

// CheckResult reports complete adoption counts, explicit licensing and change note caveats, and the next library release.
type CheckResult struct {
	Groups         int            `json:"groups"`
	Rules          int            `json:"rules"`
	Warnings       []string       `json:"warnings"`
	PendingRelease PendingRelease `json:"pendingRelease"`
}

// Check validates a captured library snapshot, ignoring unrelated files such as .git.
// In a Git repository, it also compares rules with the latest library release reachable from HEAD and
// requires a pending change note for each difference; a shallow clone fails. A library outside Git, or
// before its first library release, needs no notes, and its first library release gives every rule 1.0.0.
// A final comparison rejects observed changes; ordinary editors are not locked out.
func Check(ctx context.Context, options Options) (CheckResult, error) {
	result, _, err := checkLibrary(ctx, options)
	return result, err
}

// checkLibrary is Check, also returning the plan of the next library release its preview shows.
func checkLibrary(ctx context.Context, options Options) (CheckResult, releasePlan, error) {
	root, err := openLibrary(ctx, options, false)
	if err != nil {
		return CheckResult{}, releasePlan{}, err
	}
	defer root.Close()
	if err = filetxn.RequireIdle(root); err != nil {
		return CheckResult{}, releasePlan{}, err
	}
	input, err := libraryCheckInput(ctx, root)
	if err != nil {
		return CheckResult{}, releasePlan{}, err
	}
	if err = validateLibraryInventory(ctx, input.tree.Files, rules.LicensePaths(input.license)); err != nil {
		return CheckResult{}, releasePlan{}, err
	}
	catalog, err := LoadSource(ctx, capturedLibrary{input.tree}, "library", rules.GroupSelection{Pattern: "*"})
	if err != nil {
		return CheckResult{}, releasePlan{}, err
	}
	notes, err := parseChangeNotes(input.notes)
	if err != nil {
		return CheckResult{}, releasePlan{}, err
	}
	result := CheckResult{Groups: len(catalog.Groups), Warnings: []string{}}
	current := []string{}
	for _, group := range catalog.Groups {
		result.Rules += len(group.Rules)
		for _, rule := range group.Rules {
			current = append(current, strings.TrimSuffix(rule.Path, ".md"))
		}
	}
	slices.Sort(current)
	git, err := openLibraryGit(ctx, root.Name(), options.Git)
	if err != nil {
		return CheckResult{}, releasePlan{}, err
	}
	changes, warnings, err := git.compare(ctx, input.tree.Files, current, notes)
	if err != nil {
		return CheckResult{}, releasePlan{}, err
	}
	if problems := changes.review(); len(problems) > 0 {
		return CheckResult{}, releasePlan{}, failure("change-notes", "change notes don't match the rule changes since "+changes.history.latest.tagName()+":\n  - "+strings.Join(problems, "\n  - "), nil)
	}
	plan, err := changes.plan()
	if err != nil {
		return CheckResult{}, releasePlan{}, err
	}
	result.PendingRelease = plan.preview()
	if input.license == nil {
		result.Warnings = append(result.Warnings, "License is undeclared. Decide terms before sharing this library.")
	} else if input.license.SPDXExpression == nil {
		result.Warnings = append(result.Warnings, "The license has no SPDX expression. Declare the library terms explicitly.")
	}
	result.Warnings = append(result.Warnings, warnings...)
	if err = ctx.Err(); err != nil {
		return CheckResult{}, releasePlan{}, err
	}
	if err = requireLibraryUnchanged(ctx, root, input); err != nil {
		return CheckResult{}, releasePlan{}, err
	}
	if err = filetxn.RequireIdle(root); err != nil {
		return CheckResult{}, releasePlan{}, err
	}
	return result, plan, nil
}

// parseChangeNotes validates every note's format, including notes that library releases already published.
func parseChangeNotes(files map[string][]byte) (map[string]rules.ChangeNote, error) {
	notes := map[string]rules.ChangeNote{}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		note, err := rules.ParseChangeNote(files[name], name)
		if err != nil {
			return nil, err
		}
		notes[name] = note
	}
	return notes, nil
}

// compare finds the notes added since the latest library release and the rules whose versioned content
// differs from it, with a warning for each published note that was edited or deleted. A nil receiver has no history.
func (g *libraryGit) compare(ctx context.Context, files map[string][]byte, current []string, notes map[string]rules.ChangeNote) (libraryChanges, []string, error) {
	history, err := g.history(ctx)
	if err != nil {
		return libraryChanges{}, nil, err
	}
	changes := libraryChanges{history: history, current: current, changed: map[string]bool{}}
	latest := history.latest
	if latest == nil {
		return changes, nil, nil
	}
	versioned := map[string][]string{}
	paths := []string{}
	for _, id := range current {
		if _, published := latest.record.Rules[id]; published {
			versioned[id] = ruleFiles(id, maps.Keys(files))
			paths = append(paths, versioned[id]...)
		}
	}
	for name := range notes {
		if _, published := latest.files[name]; published {
			paths = append(paths, name)
		}
	}
	hashes, err := g.hashFiles(ctx, paths)
	if err != nil {
		return libraryChanges{}, nil, err
	}
	for id, names := range versioned {
		released := ruleFiles(id, maps.Keys(latest.files))
		changes.changed[id] = !slices.Equal(names, released) || slices.ContainsFunc(names, func(name string) bool { return hashes[name] != latest.files[name] })
	}
	var warnings []string
	for _, name := range slices.Sorted(maps.Keys(notes)) {
		switch published, ok := latest.files[name]; {
		case !ok:
			changes.pending = append(changes.pending, pendingNote{path: name, note: notes[name]})
		case hashes[name] != published:
			warnings = append(warnings, name+" changed after a library release published it. Editing a published note has no effect.")
		}
	}
	for _, name := range slices.Sorted(maps.Keys(latest.files)) {
		if _, kept := notes[name]; strings.HasPrefix(name, changesDirectory+"/") && !kept {
			warnings = append(warnings, name+" was deleted after a library release published it. Notes are never deleted; restore it.")
		}
	}
	return changes, warnings, nil
}

// ruleFiles returns a rule's versioned files among names, in sorted order: its Markdown file and its asset directory's files.
func ruleFiles(id string, names iter.Seq[string]) []string {
	assets := rules.RuleAssetDirectory(id + ".md")
	var files []string
	for name := range names {
		if name == id+".md" || strings.HasPrefix(name, assets) {
			files = append(files, name)
		}
	}
	slices.Sort(files)
	return files
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
				if rules.HasDraftMarker(files[name]) {
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

// checkInput is a bounded snapshot of every library-owned file and change note a complete check considers.
type checkInput struct {
	tree    *filetxn.Tree
	license *rules.LicenseDeclaration
	// notes maps each file under changes/ to its bytes; notes stay out of catalogs and snapshots.
	notes map[string][]byte
}

// libraryCheckInput captures every manifest, term, library-owned file, and change note considered by a complete check.
func libraryCheckInput(ctx context.Context, root *os.Root) (checkInput, error) {
	inventory, err := readLocalInventory(ctx, root)
	if err != nil {
		return checkInput{}, err
	}
	notes, err := readChangeNotes(ctx, rootFiles{ctx: ctx, root: root})
	if err != nil {
		return checkInput{}, err
	}
	return checkInput{tree: &filetxn.Tree{Files: inventory.Files, Directories: inventory.Directories}, license: inventory.License, notes: notes}, nil
}

// requireLibraryUnchanged rejects differences observed after validating the captured snapshot.
func requireLibraryUnchanged(ctx context.Context, root *os.Root, before checkInput) error {
	after, err := libraryCheckInput(ctx, root)
	if err != nil {
		return failure("changed-input", "library changed during validation; retry library check", err)
	}
	if !maps.EqualFunc(before.tree.Files, after.tree.Files, bytes.Equal) || !slices.Equal(before.tree.Directories, after.tree.Directories) || !maps.EqualFunc(before.notes, after.notes, bytes.Equal) {
		return failure("changed-input", "library changed during validation; retry library check", nil)
	}
	return nil
}
