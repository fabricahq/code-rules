// Package filetxn protects authored files and managed trees with bounded reads and recoverable writes.
package filetxn

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"golang.org/x/text/unicode/norm"
)

// File describes a contained relative path and its intended bytes. Previous == nil means
// exclusive creation; non-nil Previous must match the existing bytes before replacement.
// An edit may replace at most one existing file and create any number of new files.
type File struct {
	Path              string
	Content, Previous []byte
}

// ReadOptional reads one bounded regular, unlinked file; only absence returns nil.
func ReadOptional(ctx context.Context, root *os.Root, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, failure("unsafe-file", name+": expected a regular file without links", nil)
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, actual) || !actual.Mode().IsRegular() || actual.Sys().(*syscall.Stat_t).Nlink != 1 {
		return nil, failure("unsafe-file", name+": file identity changed or has multiple links", nil)
	}
	const limit = 64 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, failure("input-limit", name+": file exceeds 64 MiB", nil)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

// rejectAlias prevents case or Unicode-normalization variants from overwriting another spelling.
func rejectAlias(root *os.Root, name string) error {
	dir, err := root.Open(path.Dir(name))
	if err != nil {
		return err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}
	key := strings.ToLower(norm.NFC.String(path.Base(name)))
	for _, entry := range names {
		if entry != path.Base(name) && strings.ToLower(norm.NFC.String(entry)) == key {
			return failure("path-collision", name+": conflicts with an existing path's spelling", nil)
		}
	}
	return nil
}

// ensureParents creates only missing contained directories and records them for empty-directory cleanup.
func ensureParents(root *os.Root, name string, created *[]string) error {
	if name == "." {
		return nil
	}
	if err := ensureParents(root, path.Dir(name), created); err != nil {
		return err
	}
	if err := rejectAlias(root, name); err != nil {
		return err
	}
	info, err := root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		if err = root.Mkdir(name, 0755); err == nil {
			*created = append(*created, name)
			return nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
		info, err = root.Lstat(name)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return failure("unsafe-path", name+": expected a directory without symlinks", nil)
	}
	return nil
}

// verifyTarget refuses existing new files and replacements whose original bytes changed.
func verifyTarget(ctx context.Context, root *os.Root, file File) error {
	if err := rejectAlias(root, file.Path); err != nil {
		return err
	}
	data, err := ReadOptional(ctx, root, file.Path)
	if err != nil {
		return err
	}
	if (file.Previous == nil && data != nil) || (file.Previous != nil && (data == nil || !bytes.Equal(data, file.Previous))) {
		return failure("concurrent-change", file.Path+": already exists or changed; no overwrite was performed", nil)
	}
	return nil
}

// restoreClaim restores a claimed file exclusively, preserving any editor-created replacement.
func restoreClaim(root *os.Root, claim, target string) error {
	if err := root.Link(claim, target); err != nil {
		return err
	}
	return root.Remove(claim)
}

// publication owns staged files and rollback state for one exclusive authoring operation.
type publication struct {
	ctx   context.Context
	root  *os.Root
	stage string
	// retain leaves uncertain user bytes available for manual recovery.
	retain    bool
	created   []string
	published []File
	// link is the exclusive installation boundary; tests can fail a specific publication.
	link func(string, string) error
	// removeStage is the post-publication cleanup boundary.
	removeStage func(string) error
}

// publishAuthored validates targets, prepares a private stage, and publishes under the caller's writer lock.
func publishAuthored(ctx context.Context, root *os.Root, files []File) (err error) {
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = validatePublication(root, files); err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	stage := ".code-rules-authoring-" + rand.Text()
	if err = root.Mkdir(stage, 0700); err != nil {
		return err
	}
	operation := &publication{ctx: ctx, root: root, stage: stage, link: root.Link, removeStage: root.RemoveAll}
	return operation.run(files)
}

// validatePublication rejects unsafe targets, multiple replacements, and interrupted prior stages.
func validatePublication(root *os.Root, files []File) error {
	seen := map[string]bool{}
	replacements := 0
	for _, f := range files {
		if !fs.ValidPath(f.Path) || strings.Contains(f.Path, "\\") {
			return failure("unsafe-path", "invalid authoring path", nil)
		}
		if seen[f.Path] {
			return failure("invalid-operation", "duplicate authoring target", nil)
		}
		seen[f.Path] = true
		if f.Previous != nil {
			replacements++
		}
	}
	if replacements > 1 {
		return failure("invalid-operation", "authoring may replace at most one file", nil)
	}
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := dir.Readdirnames(-1)
	err = errors.Join(readErr, dir.Close())
	if err != nil {
		return err
	}
	for _, name := range entries {
		if strings.HasPrefix(name, ".code-rules-authoring-") {
			return failure("recovery-required", "inspect retained authoring files before retrying: "+filepath.Join(root.Name(), name), nil)
		}
	}
	return nil
}

// run prepares all targets, publishes creations before the optional replacement, and rolls back failures.
func (p *publication) run(files []File) (err error) {
	defer func() {
		if err != nil {
			err = errors.Join(err, p.rollback(), p.cleanup())
			return
		}
		if cleanupErr := p.cleanup(); cleanupErr != nil {
			err = &committedCleanupError{Cause: cleanupErr}
		}
	}()
	for _, f := range files {
		if err = p.prepare(f); err != nil {
			return err
		}
	}
	ordered := slices.Clone(files)
	slices.SortStableFunc(ordered, func(a, b File) int {
		if a.Previous == nil && b.Previous != nil {
			return -1
		}
		if a.Previous != nil && b.Previous == nil {
			return 1
		}
		return 0
	})
	for i, f := range ordered {
		if err = p.publish(i, f); err != nil {
			return err
		}
	}
	return nil
}

// prepare creates contained parent directories and verifies a target before any file is installed.
func (p *publication) prepare(file File) error {
	if err := p.ctx.Err(); err != nil {
		return err
	}
	if err := ensureParents(p.root, path.Dir(file.Path), &p.created); err != nil {
		return err
	}
	return verifyTarget(p.ctx, p.root, file)
}

// publish stages bytes and installs one file exclusively, retaining a rollback record for creations.
func (p *publication) publish(index int, file File) error {
	if err := p.ctx.Err(); err != nil {
		return err
	}
	staged := path.Join(p.stage, fmt.Sprintf("new-%d", index))
	out, err := p.root.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := out.Write(file.Content)
	if err = errors.Join(writeErr, out.Close()); err != nil {
		return err
	}
	if err = verifyTarget(p.ctx, p.root, file); err != nil {
		return err
	}
	if file.Previous != nil {
		return p.replace(index, file, staged)
	}
	if err = p.link(staged, file.Path); err != nil {
		return err
	}
	p.published = append(p.published, file)
	if err = p.root.Remove(staged); err != nil {
		return err
	}
	return p.ctx.Err()
}

// replace claims old bytes before comparison and never overwrites a concurrently recreated target.
func (p *publication) replace(index int, file File, staged string) error {
	claim := path.Join(p.stage, fmt.Sprintf("previous-%d", index))
	record, err := jsonText(map[string]string{"target": file.Path, "previous": claim})
	if err != nil {
		return err
	}
	if err = p.root.WriteFile(path.Join(p.stage, "recovery.json"), record, 0600); err != nil {
		return err
	}
	if err = p.ctx.Err(); err != nil {
		return err
	}
	if err = p.root.Rename(file.Path, claim); err != nil {
		return err
	}
	actual, err := ReadOptional(p.ctx, p.root, claim)
	if err == nil && !bytes.Equal(actual, file.Previous) {
		err = failure("concurrent-change", file.Path+": changed during publication; preserving editor bytes", nil)
	}
	if err == nil {
		err = p.ctx.Err()
	}
	if err == nil {
		err = p.link(staged, file.Path)
	}
	if err != nil {
		if restoreErr := restoreClaim(p.root, claim, file.Path); restoreErr != nil {
			p.retain = true
			return failure("recovery-required", "target occupied; prior bytes retained at "+filepath.Join(p.root.Name(), claim), errors.Join(err, restoreErr))
		}
		return err
	}
	// The replacement link commits the operation. Stage cleanup follows without another cancellation point.
	return nil
}

// rollback removes only proven unchanged creations, restoring any intervening editor changes exclusively.
func (p *publication) rollback() (err error) {
	for i, file := range slices.Backward(p.published) {
		claim := path.Join(p.stage, fmt.Sprintf("rollback-%d", i))
		if e := p.root.Rename(file.Path, claim); e != nil {
			if errors.Is(e, fs.ErrNotExist) {
				continue
			}
			p.retain = true
			err = errors.Join(err, failure("recovery-required", "could not claim "+file.Path+"; inspect "+p.stage, e))
			continue
		}
		data, e := ReadOptional(context.Background(), p.root, claim)
		if e == nil && !bytes.Equal(data, file.Content) {
			e = restoreClaim(p.root, claim, file.Path)
		}
		if e != nil {
			p.retain = true
			err = errors.Join(err, failure("recovery-required", "preserved uncertain rollback content at "+claim, e))
		}
	}
	return err
}

// cleanup removes a completed stage and empty created parents, preserving every uncertain recovery stage.
func (p *publication) cleanup() (err error) {
	if !p.retain {
		if p.removeStage != nil {
			err = p.removeStage(p.stage)
		} else {
			err = p.root.RemoveAll(p.stage)
		}
	}
	for _, name := range slices.Backward(p.created) {
		e := p.root.Remove(name)
		if e != nil && !errors.Is(e, syscall.ENOTEMPTY) && !errors.Is(e, fs.ErrExist) && !errors.Is(e, fs.ErrNotExist) {
			err = errors.Join(err, e)
		}
	}
	return err
}

// committedCleanupError distinguishes a fully published result from failed best-effort cleanup.
type committedCleanupError struct{ Cause error }

// Error explains that retrying publication is unnecessary even though cleanup needs attention.
func (e *committedCleanupError) Error() string {
	return "all authored files were committed; cleanup needs attention: " + e.Cause.Error()
}

// Unwrap preserves the filesystem failure for callers inspecting the warning's cause.
func (e *committedCleanupError) Unwrap() error { return e.Cause }

// publicationComplete recognizes successful publication even when its later cleanup failed.
func publicationComplete(err error) bool {
	var cleanup *committedCleanupError
	return err == nil || errors.As(err, &cleanup)
}

// Changes reports absolute paths in publication order and cleanup warnings after a committed edit.
// An error returns no changes; a warning means every requested file was committed.
type Changes struct {
	Files    []string `json:"files"`
	Warnings []string `json:"warnings,omitempty"`
}

// Edit acquires writer ownership, prepares files against current state, and publishes them safely.
// Preparation must not write; the owned callback revalidates input before file creation or replacement.
func Edit(ctx context.Context, root *os.Root, prepare func() ([]File, error)) (Changes, error) {
	if prepare == nil {
		return Changes{}, failure("invalid-operation", "expected an edit preparation operation", nil)
	}
	var files []File
	committed := false
	err := WithWriter(ctx, root, func(_ *Writer) error {
		var err error
		files, err = prepare()
		if err != nil {
			return err
		}
		err = publishAuthored(ctx, root, files)
		committed = publicationComplete(err)
		return err
	})
	return finishPublication(root, files, committed, err)
}

func finishPublication(root *os.Root, files []File, committed bool, err error) (Changes, error) {
	if err != nil && !committed {
		return Changes{}, err
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, filepath.Join(root.Name(), filepath.FromSlash(file.Path)))
	}
	result := Changes{Files: paths}
	if err != nil {
		result.Warnings = []string{"All requested files were committed. Cleanup needs attention before another authoring operation: " + err.Error()}
	}
	return result, nil
}

func jsonText(value any) ([]byte, error) {
	var out bytes.Buffer
	e := json.NewEncoder(&out)
	e.SetEscapeHTML(false)
	e.SetIndent("", "  ")
	err := e.Encode(value)
	return out.Bytes(), err
}
