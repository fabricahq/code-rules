---
title: Comment the role, the result, and the hidden constraint in Go
whenToRead: 'Before writing, changing, or reviewing Go files in this repository, including tests: their header comments, doc comments on exported and unexported declarations, and comments on values or lines whose reason the code doesn''t show.'
impact: MEDIUM
impactDescription: Missing headers and contract comments force readers and agents to trace implementations to learn what a file is for, who owns a result, and what nil means, and guessed rationales become false specifications.
---

## Comment the role, the result, and the hidden constraint in Go

Start every Go file, including tests, with a comment that states the file's role.
Give every exported declaration a doc comment that states its observable contract.
Comment a line only for a constraint that the code can't show.
Don't narrate the code, and don't invent reasons you don't know.

### Implementation

- **File role:** write one to three lines saying what the file is for, and what it isn't for when that's surprising.
  In the file that holds the `Package <name> ...` comment, that comment states the role. In every other file, separate the header from the `package` clause with a blank line.
  One file, one role: a declaration the header can't account for belongs in another file.
  Generated files are exempt.
- **Exported contract:** say what callers get and can rely on, including what the signature hides: who owns a returned value or must close it, what a nil or empty result means, which failures it reports and with which codes, and what it leaves unchanged when it fails.
  Start with the declared name, as Go doc comments do, and describe behavior rather than the mechanism.
- **Unexported helpers:** add a comment when the purpose or behavior isn't clear from the name and signature. Skip comments that only restate the name.
- **Hidden constraints:** put the reason next to the constant or line it explains, such as a server or protocol quirk, a security boundary, or an invariant the types can't express.
  When the reason isn't known, name the value and leave the reason out, or point to the issue that will settle it.
- **Durability:** prefer comments that stay true while the contract holds. Remove scratch and change-history comments before merging; Git owns history.

Where to attach a package comment, and when to comment a struct field, have their own rules in the Go group.

### Rationale

A file header lets a reader decide whether to open the file, and an export's doc comment tells a caller what they get without reading the body.
In this codebase the contracts that matter most are invisible in signatures: who owns an `*os.Root` or a temporary repository, whether a missing directory is nil or an error, and whether a failure leaves files unchanged.
A guessed rationale is worse than none, because the next reader or agent treats it as a requirement.

### Examples

#### Application: File role and exported contract

**Incorrect (counterexample):**

```go
package filetxn

// ReadTree reads a tree.
func ReadTree(ctx context.Context, root *os.Root, name string) (*Tree, error) {
```

The file has no header, and the doc comment repeats the name. It hides that the function refuses links, that the caller still owns `root`, that a missing directory returns nil rather than an error, and that a failure returns no partial tree.

**Correct:**

```go
// Read bounded project trees and compare their exact bytes, including empty directories.

package filetxn

// ReadTree returns a contained, bounded tree without following observed links or accepting hard links.
// The caller owns root. Missing directories return nil; failures return no partial tree.
func ReadTree(ctx context.Context, root *os.Root, name string) (*Tree, error) {
```

#### Application: Hidden constraint

**Incorrect (counterexample):**

```go
// Strings to look for.
var objectRefusals = []string{"server does not allow request for unadvertised object", "not our ref"}
```

**Correct:**

```go
// objectRefusals are the texts Git prints when a server refuses to send an object by its ID, which only servers
// that speak Git protocol version 0 or 1 without uploadpack.allowAnySHA1InWant do.
var objectRefusals = []string{"server does not allow request for unadvertised object", "not our ref"}
```

The comment records which servers produce the texts, which the strings alone can't show.

### Validation

Check that every Go file a change adds or edits starts with a comment line, for example with `head -n 1` on each changed file.
Run `go doc` on a changed package and check that it shows the intended package description and no file-specific text.
For each exported declaration a change adds or edits, check that its doc comment states ownership, nil or empty meanings, and failure behavior wherever the signature leaves them open.

Not a violation: an unexported helper without a comment whose name and signature make its behavior clear.
