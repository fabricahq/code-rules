---
title: Validate and render the complete result before the first write
whenToRead: Before planning, writing, changing, or reviewing Go code that writes project or library files, such as sync, update, build, init, fork, or authoring commands, or that adds a target, an input, or a check to such a write.
impact: HIGH
impactDescription: A check that fails partway through writing leaves imported, generated, and configuration files that disagree, which the next build or check rejects or agents read as valid guidance.
---

## Validate and render the complete result before the first write

Before an operation replaces any file, read every input, validate it, and render the complete contents of every file the operation will write, in memory.
Then hand all of them to one transaction: one `Writer.Apply` call, or one `filetxn.Edit`.

### Implementation

- Build each target's complete contents first, such as vendor snapshots, generated output, configuration edits, local group metadata, and replaced forks, as `install` does, and pass them to a single `w.Apply(targets, assertUnchanged)`.
- Finish everything that can fail before that call, including network reads, such as reading a fork's version from its library. Nothing that can fail runs between two writes.
- Pass an `assertUnchanged` callback that reads the inputs again and compares them, such as `requireUnchanged`, so an edit made while the operation ran fails with `concurrent-change` instead of being overwritten.
- Don't write a project or library file outside the transaction, such as with `os.WriteFile` or `root.WriteFile` next to an `Apply` call.

### Rationale

`Writer.Apply` stages every target, rechecks the inputs, writes a journal, and only then replaces live files, so an interrupted write can be recovered or rolled back as a whole.
A check that runs partway through writing defeats that: a failure leaves new vendor snapshots beside old generated output, or a configuration edit without the files it describes, which the next build or check rejects or, worse, agents read as valid guidance.

### Examples

**Incorrect (counterexample):**

```go
for name, files := range vendor {
	if err := writeVendorSource(root, name, files); err != nil {
		return FileChanges{}, err
	}
}
output, err := renderProject(ctx, state, libraries, in.options)
if err != nil {
	return FileChanges{}, err
}
```

An invalid local rule makes rendering fail after `vendor/` already holds the new snapshots, so `generated/` no longer matches them.

**Correct:**

```go
output, err := renderProject(ctx, state, libraries, in.options)
if err != nil {
	return FileChanges{}, err
}
targets := map[filetxn.Target]map[string][]byte{filetxn.Vendor: vendor, filetxn.Generated: output.Files}
// ...
err = w.Apply(targets, func() error {
	if err := requireUnchanged(ctx, root, before); err != nil {
		return err
	}
	return requireGuideUnchanged(ctx, root, in.guide)
})
```

Rendering fails before anything is written, and `Apply` installs every target together after rechecking the inputs.

### Validation

For an operation that a change adds or extends, check that every target it writes reaches the same `Apply` or `Edit` call, and that no validation, rendering, or network read happens after the first file changes.
Check that a test makes a late step fail, such as rendering an invalid local rule, and asserts that every managed file is byte-for-byte unchanged and that the command reports that no files were written.
