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
	root, err := openLibrary(ctx, options, false)
	if err != nil {
		return CheckResult{}, err
	}
	defer root.Close()
	if err = filetxn.RequireIdle(root); err != nil {
		return CheckResult{}, err
	}
	git, err := openLibraryGit(ctx, root.Name(), options.Git)
	if err != nil {
		return CheckResult{}, err
	}
	checked, err := checkLibrary(ctx, root, git)
	if err != nil {
		return CheckResult{}, err
	}
	if err = requireLibraryUnchanged(ctx, root, checked.input); err != nil {
		return CheckResult{}, err
	}
	if err = filetxn.RequireIdle(root); err != nil {
		return CheckResult{}, err
	}
	return checked.result, nil
}

// checkedLibrary is a library that passed library check: what check read, how its rules changed since the
// latest library release, and what the next library release would publish.
type checkedLibrary struct {
	result  CheckResult
	input   checkInput
	changes libraryChanges
	plan    releasePlan
}

// checkLibrary validates the library under root and its change notes against git's library releases; a nil
// git is a library outside Git. The caller confirms that the library didn't change while it was read.
func checkLibrary(ctx context.Context, root *os.Root, git *libraryGit) (checkedLibrary, error) {
	input, err := libraryCheckInput(ctx, root)
	if err != nil {
		return checkedLibrary{}, err
	}
	if err = validateLibraryInventory(ctx, input.tree.Files, rules.LicensePaths(input.license)); err != nil {
		return checkedLibrary{}, err
	}
	catalog, err := LoadSource(ctx, capturedLibrary{input.tree}, "library", rules.GroupSelection{Pattern: "*"}, nil)
	if err != nil {
		return checkedLibrary{}, err
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
	changes, warnings, err := git.compare(ctx, input.tree.Files, current, input.notes)
	if err != nil {
		return checkedLibrary{}, err
	}
	if problems := changes.review(); len(problems) > 0 {
		return checkedLibrary{}, failure("change-notes", "change notes don't match the rule changes since "+changes.history.latest.tagName()+":\n  - "+strings.Join(problems, "\n  - "), nil)
	}
	plan, err := changes.plan()
	if err != nil {
		return checkedLibrary{}, err
	}
	result.PendingRelease = plan.preview()
	if input.license == nil {
		result.Warnings = append(result.Warnings, "License is undeclared. Decide terms before sharing this library.")
	} else if input.license.SPDXExpression == nil {
		result.Warnings = append(result.Warnings, "The license has no SPDX expression. Declare the library terms explicitly.")
	}
	result.Warnings = append(result.Warnings, warnings...)
	if err = ctx.Err(); err != nil {
		return checkedLibrary{}, err
	}
	return checkedLibrary{result: result, input: input, changes: changes, plan: plan}, nil
}

// parseChangeNote validates one note's format.
func parseChangeNote(name string, data []byte) (pendingNote, error) {
	note, err := rules.ParseChangeNote(data, name)
	return pendingNote{path: name, note: note}, err
}

// compare finds the notes added since the latest library release and the rules whose versioned content
// differs from it, with a warning for each published note that was edited or deleted. It validates the format
// of pending notes only: a published note's edits have no effect, so an invalid edit only warns. Before the first
// library release every note is pending. A nil receiver has no history.
func (g *libraryGit) compare(ctx context.Context, files map[string][]byte, current []string, notes map[string][]byte) (libraryChanges, []string, error) {
	history, err := g.history(ctx)
	if err != nil {
		return libraryChanges{}, nil, err
	}
	changes := libraryChanges{history: history, current: current, changed: map[string]bool{}}
	latest := history.latest
	published := map[string]string{}
	if latest != nil {
		published = latest.files
	}
	var warnings []string
	var releasedNotes []string
	for _, name := range slices.Sorted(maps.Keys(notes)) {
		if _, ok := published[name]; ok {
			releasedNotes = append(releasedNotes, name)
			continue
		}
		pending, err := parseChangeNote(name, notes[name])
		if err != nil {
			return libraryChanges{}, nil, err
		}
		if latest != nil {
			changes.pending = append(changes.pending, pending)
		}
	}
	if latest == nil {
		return changes, nil, nil
	}
	working := map[string][]string{}
	for _, id := range current {
		if _, versioned := latest.record.Rules[id]; versioned {
			working[id] = ruleFiles(id, maps.Keys(files))
		}
	}
	if changes.changed, err = g.changedRules(ctx, latest, working); err != nil {
		return libraryChanges{}, nil, err
	}
	hashes, err := g.hashFiles(ctx, releasedNotes)
	if err != nil {
		return libraryChanges{}, nil, err
	}
	for _, name := range releasedNotes {
		if hashes[name] != published[name] {
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
