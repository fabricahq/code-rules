---
title: Parse and render in memory; read and write files only in the operation
whenToRead: Before planning, writing, changing, or reviewing Go code that parses Code Rules formats, such as configuration, rules, group metadata, release records, or change notes, that renders generated output, or that reads the files those steps need.
impact: MEDIUM-HIGH
impactDescription: A parser or renderer that reads files itself escapes the operation's containment, size limits, cancellation, and concurrent-change check, and can't run offline or on bytes a caller already holds.
---

## Parse and render in memory; read and write files only in the operation

Parsers and renderers take bytes or parsed values and return values or errors.
They don't open files, list directories, run Git, or use the network.
The operation that calls them, such as `project.Sync`, `project.Build`, or a library command, does all file and network access, with the containment, limits, cancellation, and write ownership that operation already holds.

### Implementation

- Give a parser the content and a diagnostic location, such as `ParseGroupMetadataYAML(input []byte, location string)`, rather than a path to open.
- Give a renderer every input it needs and let it return the files, as `build.Generate` takes local rules as `map[string][]byte` and returns `build.Output`.
- Read inputs once, in the operation, through its `*os.Root` and the bounded readers in `internal/filetxn`, such as `filetxn.ReadTree`, which refuse links and enforce entry, byte, and depth limits. Pass the bytes down.
- Check `ctx` in the operation, around its blocking steps. A parser or renderer that does no I/O needs no context.
- Write only through the operation's writer, with `filetxn.WithWriter` and `Writer.Apply`, or `filetxn.Edit`.
- Pure standard-library helpers, such as `net/url` parsing or embedded data, are fine.

### Rationale

Code Rules parses and renders the same formats offline in `project build` and `project check`, after fetching in `project sync`, and in library commands, so parsing and rendering must not depend on where the bytes came from.
The operation that reads the files also records them: `project.Sync` and `project.Update` compare that record before they write, and fail with `concurrent-change` when a file changed.
A renderer that reads a file itself escapes that check, the operation's `*os.Root` containment, and its size limits.

### Examples

**Incorrect (counterexample):**

```go
// Generate reads the project's local rules itself.
func Generate(config rules.Configuration, libraries map[string]Library, directory string, options Options) (Output, error) {
	localFiles, err := readLocalRules(filepath.Join(directory, ".code-rules", "local"))
	if err != nil {
		return Output{}, err
	}
	// ...
}
```

The renderer follows paths outside the project root, has no size limits, can't render files a caller already holds in memory, and reads local rules that the sync's concurrent-change check never recorded.

**Correct:**

```go
// In internal/build: inputs arrive as bytes, and output is returned, not written.
func Generate(config rules.Configuration, libraries map[string]Library, localFiles map[string][]byte, options Options) (Output, error)

// In internal/project: the operation reads once, under its root and limits, then renders.
tree, err := filetxn.ReadTree(ctx, root, item.name)
// ...
output, err := build.Generate(state.config, libraries, treeFiles(state.local), build.Options{ToolVersion: version})
```

### Validation

Run `go list -f '{{.ImportPath}}: {{join .Imports " "}}'` on the parsing and rendering packages, such as `./internal/build`.
They must not import `os`, `os/exec`, `net/http`, `internal/filetxn`, or `internal/gitexec`.
For a new parser or renderer, check that its signature takes bytes or values, not paths, and that a test can call it with plain values.

Not a violation: a project or library operation that reads and writes files, or `io/fs` used only for path validation, such as `fs.ValidPath`.
