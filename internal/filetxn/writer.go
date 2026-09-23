// Serialize project writers and recover interrupted managed-output replacements.

package filetxn

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"syscall"

	"github.com/fabricahq/code-rules/internal/rules"
)

const transactionName = ".code-rules-transaction"
const lockName = ".code-rules-lock"
const cleanupName = ".code-rules-cleanup"

// Target is a managed directory or project guide. Authored rules and configuration are never targets.
type Target string

const (
	Vendor    Target = "vendor"
	Generated Target = "generated"
)

// Writer holds exclusive project ownership only during WithWriter's callback.
// Callers must not retain it or call Apply concurrently.
type Writer struct {
	ctx  context.Context
	root *os.Root
	// rename is the filesystem boundary used by tests to stop a child at an exact rename.
	rename func(string, string) error
	active bool
}

// lockOwner records host, process, and invocation identity. Ambiguous or remote owners are never reclaimed.
type lockOwner struct {
	Host  string `json:"host"`
	PID   int    `json:"pid"`
	Token string `json:"token"`
}

// journalEntry identifies before/after output; Existed distinguishes absent from empty targets.
type journalEntry struct {
	Name    Target `json:"name"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Existed bool   `json:"existed"`
}
type journalRecord struct {
	FormatVersion int            `json:"formatVersion"`
	Entries       []journalEntry `json:"entries"`
}

// RequireIdle refuses read-only operations while writing or recovery could expose mixed output.
// It never creates a lock or performs recovery.
func RequireIdle(root *os.Root) error {
	for _, name := range []string{lockName, transactionName, cleanupName} {
		present, err := exists(root, name)
		if err != nil {
			return err
		}
		if present {
			return failure("busy", "a project writer or pending recovery exists; retry build or sync after the writer exits", nil)
		}
	}
	return nil
}

// WithWriter acquires ownership, recovers an interrupted write, and releases its own lock.
// Only a provably dead process on this host can be reclaimed. A failed recovery retains backups.
func WithWriter(ctx context.Context, root *os.Root, operation func(*Writer) error) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if root == nil || operation == nil {
		return failure("invalid-operation", "expected an open project root and writer operation", nil)
	}
	if err := acquireLock(ctx, root); err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.RemoveAll(lockName)) }()
	if err := recoverChanges(root); err != nil {
		return err
	}
	w := &Writer{ctx: ctx, root: root, active: true, rename: func(from, to string) error { return renameManaged(root, from, to) }}
	defer func() { w.active = false }()
	return operation(w)
}

// acquireLock exclusively creates an owner record or serializes reclamation of a dead local owner.
func acquireLock(ctx context.Context, root *os.Root) error {
	host, err := os.Hostname()
	if err != nil {
		return err
	}
	for attempts := 0; attempts < 3; attempts++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := root.Mkdir(lockName, 0700)
		if err == nil {
			token := make([]byte, 16)
			if _, err := rand.Read(token); err != nil {
				return errors.Join(err, root.RemoveAll(lockName))
			}
			if err := durableJSON(root, path.Join(lockName, "owner.json"), lockOwner{host, os.Getpid(), hex.EncodeToString(token)}); err != nil {
				return errors.Join(err, root.RemoveAll(lockName))
			}
			return nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("create project lock: %w", err)
		}
		tree, err := ReadTree(ctx, root, lockName)
		if err != nil {
			return err
		}
		var owner lockOwner
		if tree == nil || json.Unmarshal(tree.Files["owner.json"], &owner) != nil || owner.Host != host || owner.PID <= 0 || owner.Token == "" {
			return failure("busy", "project lock ownership is incomplete or belongs to another host; verify the owner before removing the lock", nil)
		}
		// ESRCH is the only evidence that permits reclamation; EPERM and all uncertainty fail closed.
		if err := syscall.Kill(owner.PID, 0); !errors.Is(err, syscall.ESRCH) {
			return failure("busy", "another writer is using this project or ownership cannot be verified", err)
		}
		claim := path.Join(lockName, "recovery-claim")
		if err := root.Mkdir(claim, 0700); err != nil {
			return failure("busy", "another process is recovering the project lock", err)
		}
		data, err := ReadFile(ctx, root, path.Join(lockName, "owner.json"))
		var current lockOwner
		if err != nil || json.Unmarshal(data, &current) != nil || current != owner {
			return failure("busy", "lock ownership changed during recovery; preserve its recovery claim", err)
		}
		if err := root.RemoveAll(lockName); err != nil {
			return err
		}
	}
	return failure("busy", "project lock contention; retry later", nil)
}

// Apply stages complete managed targets and rechecks caller inputs before replacing live output.
// It rolls back on failure when no later edits would be lost. Empty output is a no-op.
// This promises recoverability, not simultaneous visibility of two renames or power-loss durability.
func (w *Writer) Apply(output map[Target]map[string][]byte, assertUnchanged func() error) (err error) {
	if w == nil || !w.active {
		return failure("invalid-operation", "writer is outside its ownership callback", nil)
	}
	if err := w.ctx.Err(); err != nil {
		return err
	}
	if len(output) == 0 {
		return nil
	}
	if _, first := output[GuideReadme]; first {
		if _, second := output[GuideStandalone]; second {
			return failure("invalid-target", "only one project guide may be replaced", nil)
		}
	}
	for target, files := range output {
		if !validTarget(target) {
			return failure("invalid-target", fmt.Sprintf("%q: only managed trees and project guides may be replaced", target), nil)
		}
		if guideTarget(target) {
			if err := rejectAlias(w.root, string(target)); err != nil {
				return err
			}
		}
		if err := rules.ValidatePaths(files, nil); err != nil {
			return err
		}
	}
	if err := w.root.Mkdir(transactionName, 0700); err != nil {
		return err
	}
	prepared := false
	defer func() {
		if err == nil {
			return
		}
		var recoveryErr error
		if prepared {
			recoveryErr = recoverChanges(w.root)
		} else {
			recoveryErr = w.root.RemoveAll(transactionName)
		}
		if recoveryErr != nil {
			err = failure("recovery-required", "update failed and rollback could not finish; preserve .code-rules-transaction for recovery", errors.Join(err, recoveryErr))
		}
	}()
	entries := []journalEntry{}
	for _, target := range []Target{Vendor, Generated, GuideReadme, GuideStandalone} {
		files, ok := output[target]
		if !ok {
			continue
		}
		before, err := readTarget(w.ctx, w.root, target, string(target))
		if err != nil {
			return err
		}
		staged := path.Join(transactionName, "new-"+string(target))
		if err := writeTarget(w.ctx, w.root, target, staged, files); err != nil {
			return err
		}
		if guideTarget(target) {
			if err := w.probeGuideRename(target, staged); err != nil {
				return err
			}
		}
		after, err := readTarget(w.ctx, w.root, target, staged)
		if err != nil {
			return err
		}
		entries = append(entries, journalEntry{target, treeDigest(before), treeDigest(after), before != nil})
	}
	if assertUnchanged != nil {
		if err := assertUnchanged(); err != nil {
			return err
		}
	}
	// Recheck targets too, even when the caller only checks authored inputs.
	for _, entry := range entries {
		current, err := readTarget(w.ctx, w.root, entry.Name, string(entry.Name))
		if err != nil {
			return err
		}
		if treeDigest(current) != entry.Before {
			return failure("concurrent-change", string(entry.Name)+": output changed during staging", nil)
		}
	}
	formatVersion := 2
	for _, entry := range entries {
		if guideTarget(entry.Name) {
			formatVersion = 3
		}
	}
	if err := durableJSON(w.root, path.Join(transactionName, "journal.json"), journalRecord{formatVersion, entries}); err != nil {
		return err
	}
	prepared = true
	for _, entry := range entries {
		if err := w.ctx.Err(); err != nil {
			return err
		}
		if entry.Existed {
			backupPath := path.Join(transactionName, "old-"+string(entry.Name))
			if err := w.rename(string(entry.Name), backupPath); err != nil {
				return err
			}
			// Inspect the tree actually displaced, including edits made after the earlier check.
			backup, err := readTarget(w.ctx, w.root, entry.Name, backupPath)
			if err != nil {
				return err
			}
			if treeDigest(backup) != entry.Before {
				return failure("concurrent-change", string(entry.Name)+": output changed before replacement; preserve its backup", nil)
			}
		}
		if err := w.rename(path.Join(transactionName, "new-"+string(entry.Name)), string(entry.Name)); err != nil {
			return err
		}
	}
	if err := w.ctx.Err(); err != nil {
		return err
	}
	if err := durableJSON(w.root, path.Join(transactionName, "commit-ready.json"), true); err != nil {
		return err
	}
	if err := w.rename(path.Join(transactionName, "commit-ready.json"), path.Join(transactionName, "committed.json")); err != nil {
		return err
	}
	// A durable commit marker makes later cleanup failures cleanup-only, never rollback.
	_ = finishCommitted(w.root)
	return nil
}

// recoverChanges validates every restoration before mutating any target and ignores caller cancellation.
func recoverChanges(root *os.Root) error {
	return recoverWithRename(root, func(from, to string) error { return renameManaged(root, from, to) })
}

// recoverWithRename owns recovery renames; tests use this boundary to reproduce late edits and interruptions.
func recoverWithRename(root *os.Root, rename func(string, string) error) error {
	ctx := context.Background()
	garbage, err := readTree(ctx, root, cleanupName, recoveryLimits)
	if err != nil {
		return err
	}
	if garbage != nil {
		if err := root.RemoveAll(cleanupName); err != nil {
			return err
		}
	}
	tree, err := readTree(ctx, root, transactionName, recoveryLimits)
	if err != nil {
		return err
	}
	if tree == nil {
		return nil
	}
	data, hasJournal := tree.Files["journal.json"]
	if !hasJournal {
		for _, dir := range tree.Directories {
			if dir == "old-vendor" || dir == "old-generated" || dir == "old-README.md" || dir == "old-CODE_RULES.md" {
				return failure("recovery-required", "missing journal with retained output; preserve transaction files for manual recovery", nil)
			}
		}
		for _, name := range []string{"old-vendor", "old-generated", "old-README.md", "old-CODE_RULES.md"} {
			if _, ok := tree.Files[name]; ok {
				return failure("recovery-required", "missing journal with retained backup; preserve transaction files", nil)
			}
		}
		if _, ok := tree.Files["committed.json"]; ok {
			return failure("recovery-required", "missing journal with commit marker; preserve transaction files", nil)
		}
		return root.RemoveAll(transactionName)
	}
	var raw struct {
		Entries []map[string]json.RawMessage `json:"entries"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return failure("recovery-required", "invalid transaction journal", nil)
	}
	for _, entry := range raw.Entries {
		for _, key := range []string{"name", "before", "after", "existed"} {
			value, ok := entry[key]
			if !ok || string(value) == "null" {
				return failure("recovery-required", "incomplete transaction entry; preserve journal", nil)
			}
		}
	}
	var record journalRecord
	if json.Unmarshal(data, &record) != nil || (record.FormatVersion != 1 && record.FormatVersion != 2 && record.FormatVersion != 3) || len(record.Entries) == 0 || len(record.Entries) > 3 || (record.FormatVersion < 3 && len(record.Entries) > 2) {
		return failure("recovery-required", "invalid transaction journal; preserve it for manual recovery", nil)
	}
	seen := map[Target]bool{}
	for _, entry := range record.Entries {
		if (!validTarget(entry.Name) || (guideTarget(entry.Name) && record.FormatVersion < 3)) || seen[entry.Name] || !validDigest(entry.Before) || !validDigest(entry.After) || entry.Existed == (entry.Before == treeDigest(nil)) || entry.After == treeDigest(nil) {
			return failure("recovery-required", "invalid transaction entry; preserve it for manual recovery", nil)
		}
		seen[entry.Name] = true
	}
	if seen[GuideReadme] && seen[GuideStandalone] {
		return failure("recovery-required", "multiple project guides in transaction; preserve journal", nil)
	}
	marker, committed := tree.Files["committed.json"]
	if committed && string(marker) != "true\n" {
		return failure("recovery-required", "invalid commit marker; preserve transaction files", nil)
	}
	for _, entry := range record.Entries {
		current, err := readTarget(ctx, root, entry.Name, string(entry.Name))
		if err != nil {
			return err
		}
		backup, err := readTarget(ctx, root, entry.Name, path.Join(transactionName, "old-"+string(entry.Name)))
		if err != nil {
			return err
		}
		if backup != nil && !entry.Existed {
			return failure("recovery-required", "unexpected backup for an originally absent target; preserve transaction files", nil)
		}
		if committed {
			continue
		}
		discarded, err := readTarget(ctx, root, entry.Name, path.Join(transactionName, "discarded-"+string(entry.Name)))
		if err != nil {
			return err
		}
		if discarded != nil && treeDigest(discarded) != entry.After {
			return failure("recovery-required", string(entry.Name)+": displaced output changed; preserve transaction files", nil)
		}
		if backup != nil && treeDigest(backup) != entry.Before {
			return failure("recovery-required", string(entry.Name)+": backup changed; manual recovery required", nil)
		}
		currentHash := treeDigest(current)
		safe := currentHash == entry.Before || (!entry.Existed && currentHash == entry.After)
		if backup != nil {
			safe = current == nil || currentHash == entry.After
		}
		if !safe {
			return failure("recovery-required", string(entry.Name)+": changed after interruption; preserve current files and transaction backups", nil)
		}
	}
	if committed {
		return finishCommitted(root)
	}
	// Older TypeScript recovery cannot understand quarantined output. Upgrade before
	// introducing it so that either runtime refuses to discard unknown state.
	if record.FormatVersion == 1 {
		record.FormatVersion = 2
		next := path.Join(transactionName, "journal-next.json")
		if err := root.Remove(next); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := durableJSON(root, next, record); err != nil {
			return err
		}
		if err := root.Rename(next, path.Join(transactionName, "journal.json")); err != nil {
			return err
		}
	}
	for _, entry := range slices.Backward(record.Entries) {
		backup := path.Join(transactionName, "old-"+string(entry.Name))
		present, err := exists(root, backup)
		if err != nil {
			return err
		}
		if present || !entry.Existed {
			// Move live output into our transaction before inspecting or deleting it.
			// A path-based edit before this rename is retained on mismatch, never removed.
			current, err := exists(root, string(entry.Name))
			if err != nil {
				return err
			}
			if current {
				discardedPath := path.Join(transactionName, "discarded-"+string(entry.Name))
				already, err := exists(root, discardedPath)
				if err != nil {
					return err
				}
				if already {
					return failure("recovery-required", "both live and displaced output exist; preserve transaction files", nil)
				}
				if err := rename(string(entry.Name), discardedPath); err != nil {
					return err
				}
				discarded, err := readTarget(ctx, root, entry.Name, discardedPath)
				if err != nil {
					return err
				}
				if treeDigest(discarded) != entry.After {
					return failure("recovery-required", string(entry.Name)+": output changed during recovery; preserve displaced files and backup", nil)
				}
			}
		}
		if present {
			if err := rename(backup, string(entry.Name)); err != nil {
				return err
			}
		}
	}
	return root.RemoveAll(transactionName)
}

// finishCommitted retires the journal before deleting backups, preventing interrupted cleanup rollback.
func finishCommitted(root *os.Root) error {
	if err := root.Rename(transactionName, cleanupName); err != nil {
		return err
	}
	return root.RemoveAll(cleanupName)
}

// durableJSON exclusively creates a synced journal or lock record with a trailing newline.
func durableJSON(root *os.Root, name string, value any) error {
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	encodeErr := json.NewEncoder(f).Encode(value)
	var syncErr error
	if encodeErr == nil {
		syncErr = f.Sync()
	}
	return errors.Join(encodeErr, syncErr, f.Close())
}

// exists distinguishes absence from permission failures and includes symlinks as present.
func exists(root *os.Root, name string) (bool, error) {
	if root == nil {
		return false, failure("unsafe-path", "expected an open project root", nil)
	}
	_, err := root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// validDigest accepts only the SHA-256 representation written into version-1 journals.
func validDigest(value string) bool {
	data, err := hex.DecodeString(value)
	if err != nil || len(data) != 32 {
		return false
	}
	return hex.EncodeToString(data) == value
}
