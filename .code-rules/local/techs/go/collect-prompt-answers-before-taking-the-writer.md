---
title: Collect prompt answers before taking the project writer
whenToRead: Before planning, writing, changing, or reviewing a Go command that asks a person for input and then writes project or library files, such as adding a group, rule, or library, forking a rule, or answering update questions.
impact: MEDIUM
impactDescription: Waiting for a person while holding the writer makes every other command on the project fail with busy for as long as the prompt stays open.
---

## Collect prompt answers before taking the project writer

When an operation needs a person's input, split it into a plan and a commit.
The plan reads and checks what it can and releases the writer before it returns, `internal/cli` collects every answer, and only then does the commit take writer ownership, recheck what the plan saw, and write.
Never wait for a person while holding the writer.

### Implementation

- Return a plan from the domain package, such as `project.PlanLocalRule`, `project.PlanSource`, or `project.PlanUpdate`.
  The plan checks the target before any question, so a person isn't asked about a command that would fail anyway.
  A plan may take the writer briefly, as `project.PlanUpdate` does to recover an interrupted command and read the project, but it releases the writer before it returns, so no question is ever asked while the writer is held.
- Ask the questions in `internal/cli`, between plan and commit. `--json` and `--non-interactive` never prompt, so every input comes from flags.
- Take the writer in the commit, such as `LocalRulePlan.Commit` or `UpdatePlan.Apply`, through `filetxn.WithWriter` or `filetxn.Edit`.
  Recheck what the plan relied on and fail, writing nothing, when it changed: `LocalRulePlan.Commit` checks the group and target again, and `UpdatePlan.Apply` fails with `concurrent-change`.
- Never call a prompt, a confirmation callback, or anything else that waits for a person inside a `filetxn.WithWriter` or `filetxn.Edit` callback.

### Rationale

The writer is an exclusive lock on the project: while one command holds it, every other command that writes the project fails with `busy`, and `project check` refuses to run.
A person can take minutes to answer, or leave a prompt open in another terminal.
Collecting answers first keeps the lock as short as the write, and rechecking under the writer catches edits made while the person was answering.

### Examples

**Incorrect (counterexample):**

```go
// Apply asks for confirmation while it holds the writer.
func (p *UpdatePlan) Apply(ctx context.Context, confirm func(UpdateResult) bool) (UpdateResult, error) {
	// ...
	err = filetxn.WithWriter(ctx, root, func(w *filetxn.Writer) error {
		before, err := readProject(ctx, root)
		if err != nil {
			return err
		}
		if !confirm(preview) {
			return nil
		}
		changes, err = install(ctx, root, w, before, in)
		return err
	})
	// ...
}
```

The project stays locked for as long as the preview sits unanswered.

**Correct:**

```go
// PlanUpdate reads the project under the writer and releases it before returning.
plan, err := project.PlanUpdate(cmd.Context(), project.Options{Directory: directory, ToolVersion: options.Version}, options.Git, targets, decisions)
// ...
answer, err := askChoice(f, "Apply the update?", [2]string{"yes", "no"})
// ...
// Apply takes the writer again and fails with concurrent-change if the project changed after the preview.
result, err := plan.Apply(cmd.Context(), decisions)
```

### Validation

For each `filetxn.WithWriter` or `filetxn.Edit` callback a change adds or edits, check that nothing inside it reads from standard input or calls back into `internal/cli`.
For a command that prompts, check that its commit rechecks the state the plan relied on, and that a test changes that state between plan and commit and asserts that nothing was written.
