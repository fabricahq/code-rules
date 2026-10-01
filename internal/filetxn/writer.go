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
	"strings"
	"syscall"

	"github.com/fabricahq/code-rules/internal/rules"
	"golang.org/x/sys/unix"
)

const transactionName = ".code-rules-transaction"

// backupPrefix begins the name of each backup of a target's previous output in the transaction directory.
const backupPrefix = "old-"

// maxJournalEntries is how many targets a journal of each format version can hold: format 2 holds the two managed
// trees, format 3 adds a project guide, format 4 adds the configuration, format 5 adds local group metadata, and
// format 6 adds local rules and their asset directories, which it alone can remove.
var maxJournalEntries = map[int]int{1: 2, 2: 2, 3: 3, 4: 4, 5: 4 + maxLocalGroupMetadata, 6: 4 + maxLocalGroupMetadata + 2*maxLocalRules}

// maxJournalFormat is the newest journal format; Apply writes the oldest one that can hold its targets.
const maxJournalFormat = 6

const lockName = ".code-rules-lock"
const cleanupName = ".code-rules-cleanup"

// Target is a managed directory, project guide, the project configuration, a local group's metadata, or a local
// rule or its asset directory, which only update replaces, when it replaces a fork.
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
	// rename and mkdir are the filesystem boundaries used by tests to stop a child at an exact rename or directory.
	rename func(string, string) error
	mkdir  func(string) error
	active bool
	// recovered reports that WithWriter first recovered an interrupted earlier operation, changing project files.
	recovered bool
	// kept lists the directories an interrupted transaction created that recovery kept, because they no longer are
	// empty directories.
	kept []string
}

// Recovered reports whether, before handing over the writer, WithWriter recovered an interrupted earlier operation,
// restoring or finishing its files, so the project changed even if the caller's own operation writes nothing.
func (w *Writer) Recovered() bool { return w.recovered }

// Kept returns the directories an interrupted earlier operation created that recovering it kept, because another
// process had put files in them or replaced them; they are no longer the operation's to remove. It is empty when
// nothing was kept.
func (w *Writer) Kept() []string { return slices.Clone(w.kept) }

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
	// Parents lists, outermost first, the missing parent directories of local rule targets that the transaction
	// creates after writing the journal; recovery removes those that are empty when it rolls back. Format 6 only.
	Parents []string `json:"parents,omitempty"`
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
	// Joining only a failed removal keeps err itself, so callers still see its identity, such as a validation error.
	defer func() {
		if removeErr := root.RemoveAll(lockName); removeErr != nil {
			err = errors.Join(err, removeErr)
		}
	}()
	recovered := pendingTransaction(root)
	kept, err := recoverChanges(root)
	if err != nil {
		return err
	}
	w := &Writer{ctx: ctx, root: root, active: true, recovered: recovered, kept: kept, rename: func(from, to string) error { return renameManaged(root, from, to) }, mkdir: func(name string) error { return root.Mkdir(name, 0755) }}
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
// It rolls back on failure when no later edits would be lost, and recovery restores or finishes every target in
// the transaction together. Empty output is a no-op. A nil file map removes a local rule's asset directory, and
// does nothing when it is absent. It changes nothing outside its transaction directory until assertUnchanged has
// passed and the journal is written; only then does it create the missing parent directories of local rule targets,
// which the journal lists, so recovery removes them again when it rolls back.
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
	order := []Target{Vendor, Generated, GuideReadme, GuideStandalone, Config}
	metadata, localRules := 0, 0
	for target, files := range output {
		if !validTarget(target) {
			return failure("invalid-target", fmt.Sprintf("%q: only managed trees, project guides, the configuration, local group metadata, and local rules and their asset directories may be replaced", target), nil)
		}
		if files == nil && !localRuleAssetsTarget(target) {
			return failure("invalid-target", fmt.Sprintf("%q: only a local rule's asset directory may be removed", target), nil)
		}
		switch {
		case localGroupMetadataTarget(target):
			metadata++
		case localRuleTarget(target):
			localRules++
		}
		if localTarget(target) {
			order = append(order, target)
			if err := requireRealParents(w.root, target); err != nil {
				return err
			}
		}
		if fileTarget(target) && !localRuleTarget(target) {
			if err := rejectAlias(w.root, string(target)); err != nil {
				return err
			}
		}
		if err := rules.ValidatePaths(files, nil); err != nil {
			return err
		}
	}
	// Local targets follow the fixed targets in path order, so journals are deterministic.
	fixed := 5
	if metadata > maxLocalGroupMetadata || localRules > maxLocalRules || len(order)-fixed-metadata-localRules > maxLocalRules {
		return failure("invalid-target", fmt.Sprintf("one transaction can hold at most %d local group metadata files and %d local rules with their asset directories", maxLocalGroupMetadata, maxLocalRules), nil)
	}
	slices.Sort(order[fixed:])
	if err := w.root.Mkdir(transactionName, 0700); err != nil {
		return err
	}
	prepared := false
	// parents lists, outermost first, the missing parent directories of local rule targets to create once the
	// journal holds them.
	parents := []string{}
	defer func() {
		if err == nil {
			return
		}
		var recoveryErr error
		var kept []string
		if prepared {
			kept, recoveryErr = recoverChanges(w.root)
		} else {
			recoveryErr = w.root.RemoveAll(transactionName)
		}
		if recoveryErr != nil {
			err = failure("recovery-required", "update failed and rollback could not finish; preserve .code-rules-transaction for recovery", errors.Join(err, recoveryErr))
			return
		}
		if len(kept) > 0 {
			w.kept = append(w.kept, kept...)
			err = fmt.Errorf("%w; it kept %s, which it created but which another process has since changed", err, strings.Join(kept, ", "))
		}
	}()
	entries := []journalEntry{}
	for _, target := range order {
		files, ok := output[target]
		if !ok {
			continue
		}
		before, err := readTarget(w.ctx, w.root, target, string(target))
		if err != nil {
			return err
		}
		if files == nil {
			if before != nil {
				entries = append(entries, journalEntry{target, treeDigest(before), treeDigest(nil), true})
			}
			continue
		}
		if localRuleTarget(target) || localRuleAssetsTarget(target) {
			missing, err := missingParents(w.root, path.Dir(string(target)))
			if err != nil {
				return err
			}
			for _, directory := range missing {
				if !slices.Contains(parents, directory) {
					parents = append(parents, directory)
				}
			}
			// A target whose parent is still missing can't collide with an existing spelling.
			if len(missing) == 0 {
				if err := rejectAlias(w.root, string(target)); err != nil {
					return err
				}
			}
		}
		staged := path.Join(transactionName, "new-"+entryName(target))
		if err := writeTarget(w.ctx, w.root, target, staged, files); err != nil {
			return err
		}
		if fileTarget(target) {
			if err := w.probeFileRename(target, staged); err != nil {
				return err
			}
		}
		after, err := readTarget(w.ctx, w.root, target, staged)
		if err != nil {
			return err
		}
		entries = append(entries, journalEntry{target, treeDigest(before), treeDigest(after), before != nil})
	}
	// Only removals of absent asset directories leave nothing to replace.
	if len(entries) == 0 {
		return w.root.RemoveAll(transactionName)
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
		switch {
		case guideTarget(entry.Name):
			formatVersion = max(formatVersion, 3)
		case entry.Name == Config:
			formatVersion = max(formatVersion, 4)
		case localGroupMetadataTarget(entry.Name):
			formatVersion = max(formatVersion, 5)
		case localRuleTarget(entry.Name), localRuleAssetsTarget(entry.Name):
			formatVersion = 6
		}
	}
	if err := durableJSON(w.root, path.Join(transactionName, "journal.json"), journalRecord{formatVersion, entries, parents}); err != nil {
		return err
	}
	prepared = true
	for i, directory := range parents {
		if err := w.createParent(i, directory); err != nil {
			return err
		}
	}
	for _, entry := range entries {
		if err := w.ctx.Err(); err != nil {
			return err
		}
		if entry.Existed {
			backupPath := path.Join(transactionName, backupPrefix+entryName(entry.Name))
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
		if removes(entry) {
			continue
		}
		if err := w.rename(path.Join(transactionName, "new-"+entryName(entry.Name)), string(entry.Name)); err != nil {
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

// missingParents returns, outermost first, the directories from directory up to the first existing one that don't
// exist yet. An existing one must be a directory, not a symbolic link.
func missingParents(root *os.Root, directory string) ([]string, error) {
	missing := []string{}
	for ; directory != "." && directory != ""; directory = path.Dir(directory) {
		info, err := root.Lstat(directory)
		if errors.Is(err, fs.ErrNotExist) {
			missing = append(missing, directory)
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, failure("unsafe-path", directory+": expected a directory without symlinks", nil)
		}
		break
	}
	slices.Reverse(missing)
	return missing, nil
}

// createParent creates directory, the missing parent of a local rule target at index i of the journal's list, and
// then durably records that it created it, which is recovery's only proof that the directory is the transaction's
// to remove. It refuses a name that collides with an existing spelling, and, with concurrent-change, a directory or
// file that appeared after the inputs were checked, which belongs to another process and is never claimed. A crash
// between creating and recording leaves an empty directory behind, rather than risk removing another's.
func (w *Writer) createParent(i int, directory string) error {
	if err := rejectAlias(w.root, directory); err != nil {
		return err
	}
	err := w.mkdir(directory)
	if errors.Is(err, fs.ErrExist) {
		return failure("concurrent-change", directory+": appeared after the operation checked its inputs; retry the command", nil)
	}
	if err != nil {
		return err
	}
	return durableJSON(w.root, path.Join(transactionName, parentRecord(i)), directory)
}

// parentRecord names the record, in the transaction directory, that the parent at index i of the journal's list
// was created by the transaction.
func parentRecord(i int) string { return fmt.Sprintf("parent-%d.created", i) }

// parentRecordContent is what the record of creating directory holds, as durableJSON writes it, so recovery
// accepts only a record of the directory the journal lists at that index.
func parentRecordContent(directory string) string {
	data, _ := json.Marshal(directory)
	return string(data) + "\n"
}

// removeCreatedParent removes directory, which the transaction created, only while it is an empty directory, with
// an operation that can't remove anything else. It reports whether something stays at the path: a directory another
// process put files in, or a file or link that replaced it. An absent directory is already gone.
func removeCreatedParent(root *os.Root, directory string) (bool, error) {
	if err := requireRealParents(root, Target(directory)); err != nil {
		var unsafe *Error
		if errors.As(err, &unsafe) {
			return true, nil
		}
		return false, err
	}
	parent, err := root.Open(path.Dir(directory))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer parent.Close()
	err = unix.Unlinkat(int(parent.Fd()), path.Base(directory), unix.AT_REMOVEDIR)
	switch {
	case err == nil, errors.Is(err, unix.ENOENT):
		return false, nil
	case errors.Is(err, unix.ENOTEMPTY), errors.Is(err, unix.EEXIST), errors.Is(err, unix.ENOTDIR):
		return true, nil
	}
	return false, err
}

// validParents reports whether parents, from a journal of format version, are the outermost-first missing parents
// of its local rule entries: each under local/ and an ancestor of an entry, and, of its own ancestors, those listed
// are listed before it and form a contiguous chain up from an ancestor that existed, since a missing directory's
// subdirectories are missing too.
func validParents(version int, parents []string, entries []journalEntry) bool {
	if len(parents) > 0 && version < 6 {
		return false
	}
	for i, directory := range parents {
		if !strings.HasPrefix(directory, "local/") || path.Clean(directory) != directory || slices.Contains(parents[:i], directory) {
			return false
		}
		// Walking up from directory, listed ancestors come first, each listed earlier, then only unlisted ones.
		existed := false
		for ancestor := path.Dir(directory); ancestor != "local" && ancestor != "."; ancestor = path.Dir(ancestor) {
			listed := slices.Contains(parents, ancestor)
			if listed && (existed || !slices.Contains(parents[:i], ancestor)) {
				return false
			}
			existed = existed || !listed
		}
		if !slices.ContainsFunc(entries, func(entry journalEntry) bool {
			return (localRuleTarget(entry.Name) || localRuleAssetsTarget(entry.Name)) && strings.HasPrefix(string(entry.Name), directory+"/")
		}) {
			return false
		}
	}
	return true
}

// removes reports whether entry removes its target rather than replacing it, which only a local rule's asset
// directory can.
func removes(entry journalEntry) bool { return entry.After == treeDigest(nil) }

// recoverChanges validates every restoration before mutating any target and ignores caller cancellation. It returns
// the directories the interrupted transaction created that it kept, because they no longer are empty directories.
func recoverChanges(root *os.Root) ([]string, error) {
	return recoverWithRename(root, func(from, to string) error { return renameManaged(root, from, to) })
}

// pendingTransaction reports whether an interrupted operation left a transaction for recovery to restore or finish.
func pendingTransaction(root *os.Root) bool {
	_, err := root.Lstat(transactionName)
	return err == nil
}

// recoverWithRename owns recovery renames; tests use this boundary to reproduce late edits and interruptions.
func recoverWithRename(root *os.Root, rename func(string, string) error) ([]string, error) {
	ctx := context.Background()
	garbage, err := readTree(ctx, root, cleanupName, recoveryLimits)
	if err != nil {
		return nil, err
	}
	if garbage != nil {
		if err := root.RemoveAll(cleanupName); err != nil {
			return nil, err
		}
	}
	tree, err := readTree(ctx, root, transactionName, recoveryLimits)
	if err != nil {
		return nil, err
	}
	if tree == nil {
		return nil, nil
	}
	data, hasJournal := tree.Files["journal.json"]
	if !hasJournal {
		for _, dir := range tree.Directories {
			if strings.HasPrefix(dir, backupPrefix) {
				return nil, failure("recovery-required", "missing journal with retained output; preserve transaction files for manual recovery", nil)
			}
		}
		for name := range tree.Files {
			if strings.HasPrefix(name, backupPrefix) && !strings.Contains(name, "/") {
				return nil, failure("recovery-required", "missing journal with retained backup; preserve transaction files", nil)
			}
		}
		if _, ok := tree.Files["committed.json"]; ok {
			return nil, failure("recovery-required", "missing journal with commit marker; preserve transaction files", nil)
		}
		return nil, root.RemoveAll(transactionName)
	}
	var raw struct {
		Entries []map[string]json.RawMessage `json:"entries"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return nil, failure("recovery-required", "invalid transaction journal", nil)
	}
	for _, entry := range raw.Entries {
		for _, key := range []string{"name", "before", "after", "existed"} {
			value, ok := entry[key]
			if !ok || string(value) == "null" {
				return nil, failure("recovery-required", "incomplete transaction entry; preserve journal", nil)
			}
		}
	}
	var record journalRecord
	if json.Unmarshal(data, &record) != nil || record.FormatVersion < 1 || record.FormatVersion > maxJournalFormat || len(record.Entries) == 0 || len(record.Entries) > maxJournalEntries[record.FormatVersion] {
		return nil, failure("recovery-required", "invalid transaction journal; preserve it for manual recovery", nil)
	}
	seen := map[Target]bool{}
	for _, entry := range record.Entries {
		localRule := localRuleTarget(entry.Name) || localRuleAssetsTarget(entry.Name)
		if (!validTarget(entry.Name) || (guideTarget(entry.Name) && record.FormatVersion < 3) || (entry.Name == Config && record.FormatVersion < 4) || (localGroupMetadataTarget(entry.Name) && record.FormatVersion < 5) || (localRule && record.FormatVersion < 6)) || seen[entry.Name] || !validDigest(entry.Before) || !validDigest(entry.After) || entry.Existed == (entry.Before == treeDigest(nil)) || (removes(entry) && (!localRuleAssetsTarget(entry.Name) || !entry.Existed)) {
			return nil, failure("recovery-required", "invalid transaction entry; preserve it for manual recovery", nil)
		}
		seen[entry.Name] = true
		// A symbolic link in a parent could make recovery inspect, discard, or replace another file.
		if err := requireRealParents(root, entry.Name); err != nil {
			return nil, failure("recovery-required", string(entry.Name)+": a parent directory changed into a symbolic link or file after interruption; preserve the transaction and restore the directory", err)
		}
	}
	if seen[GuideReadme] && seen[GuideStandalone] {
		return nil, failure("recovery-required", "multiple project guides in transaction; preserve journal", nil)
	}
	if !validParents(record.FormatVersion, record.Parents, record.Entries) {
		return nil, failure("recovery-required", "invalid parent directories in transaction journal; preserve it for manual recovery", nil)
	}
	for _, directory := range record.Parents {
		if err := requireRealParents(root, Target(directory+"/x")); err != nil {
			return nil, failure("recovery-required", directory+": a directory changed into a symbolic link or file after interruption; preserve the transaction and restore the directory", err)
		}
	}
	marker, committed := tree.Files["committed.json"]
	if committed && string(marker) != "true\n" {
		return nil, failure("recovery-required", "invalid commit marker; preserve transaction files", nil)
	}
	for _, entry := range record.Entries {
		current, err := readTarget(ctx, root, entry.Name, string(entry.Name))
		if err != nil {
			return nil, err
		}
		backup, err := readTarget(ctx, root, entry.Name, path.Join(transactionName, backupPrefix+entryName(entry.Name)))
		if err != nil {
			return nil, err
		}
		if backup != nil && !entry.Existed {
			return nil, failure("recovery-required", "unexpected backup for an originally absent target; preserve transaction files", nil)
		}
		if committed {
			continue
		}
		discarded, err := readTarget(ctx, root, entry.Name, path.Join(transactionName, "discarded-"+entryName(entry.Name)))
		if err != nil {
			return nil, err
		}
		if discarded != nil && treeDigest(discarded) != entry.After {
			return nil, failure("recovery-required", string(entry.Name)+": displaced output changed; preserve transaction files", nil)
		}
		if backup != nil && treeDigest(backup) != entry.Before {
			return nil, failure("recovery-required", string(entry.Name)+": backup changed; manual recovery required", nil)
		}
		currentHash := treeDigest(current)
		safe := currentHash == entry.Before || (!entry.Existed && currentHash == entry.After)
		if backup != nil {
			safe = current == nil || currentHash == entry.After
		}
		if !safe {
			return nil, failure("recovery-required", string(entry.Name)+": changed after interruption; preserve current files and transaction backups", nil)
		}
	}
	if committed {
		return nil, finishCommitted(root)
	}
	// Older TypeScript recovery cannot understand quarantined output. Upgrade before
	// introducing it so that either runtime refuses to discard unknown state.
	if record.FormatVersion == 1 {
		record.FormatVersion = 2
		next := path.Join(transactionName, "journal-next.json")
		if err := root.Remove(next); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		if err := durableJSON(root, next, record); err != nil {
			return nil, err
		}
		if err := root.Rename(next, path.Join(transactionName, "journal.json")); err != nil {
			return nil, err
		}
	}
	for _, entry := range slices.Backward(record.Entries) {
		backup := path.Join(transactionName, backupPrefix+entryName(entry.Name))
		present, err := exists(root, backup)
		if err != nil {
			return nil, err
		}
		if present || !entry.Existed {
			// Move live output into our transaction before inspecting or deleting it.
			// A path-based edit before this rename is retained on mismatch, never removed.
			current, err := exists(root, string(entry.Name))
			if err != nil {
				return nil, err
			}
			if current {
				discardedPath := path.Join(transactionName, "discarded-"+entryName(entry.Name))
				already, err := exists(root, discardedPath)
				if err != nil {
					return nil, err
				}
				if already {
					return nil, failure("recovery-required", "both live and displaced output exist; preserve transaction files", nil)
				}
				if err := rename(string(entry.Name), discardedPath); err != nil {
					return nil, err
				}
				discarded, err := readTarget(ctx, root, entry.Name, discardedPath)
				if err != nil {
					return nil, err
				}
				if treeDigest(discarded) != entry.After {
					return nil, failure("recovery-required", string(entry.Name)+": output changed during recovery; preserve displaced files and backup", nil)
				}
			}
		}
		if present {
			if err := rename(backup, string(entry.Name)); err != nil {
				return nil, err
			}
		}
	}
	// Only a directory the transaction recorded creating is its own to remove, and only while it is still an empty
	// directory; anything else at its path is kept and reported.
	kept := []string{}
	for i, directory := range slices.Backward(record.Parents) {
		if string(tree.Files[parentRecord(i)]) != parentRecordContent(directory) {
			continue
		}
		stays, err := removeCreatedParent(root, directory)
		if err != nil {
			return nil, err
		}
		if stays {
			kept = append(kept, directory)
		}
	}
	return kept, root.RemoveAll(transactionName)
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
