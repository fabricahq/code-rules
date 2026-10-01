// Report a project command that failed without writing files of its own, so people know what changed.

package project

import (
	"context"
	"errors"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
)

// UnchangedError reports a failed command that wrote no files of its own; Err is why it failed. Recovered reports
// that it first recovered an interrupted earlier command, restoring or finishing that command's files, so the
// project may have changed anyway.
type UnchangedError struct {
	Err       error
	Recovered bool
}

// Outcome says what the failed command did to the project's files.
func (e *UnchangedError) Outcome() string { return WriteOutcome(e.Recovered) }

// WriteOutcome says, for a command that wrote no files of its own, what happened to the project's files: nothing,
// or, when recovered is true, the recovery of an interrupted earlier command, which restored or finished that
// command's files.
func WriteOutcome(recovered bool) string {
	if recovered {
		return "It wrote no files of its own, but first recovered an interrupted earlier command, which restored or finished that command's files."
	}
	return "No files were written."
}

// Error says that the command was cancelled, or why it failed, and what it did to the project's files.
func (e *UnchangedError) Error() string {
	if errors.Is(e.Err, context.Canceled) && !e.Recovered {
		return "cancelled; no files were written"
	}
	message := strings.TrimSpace(e.Err.Error())
	if errors.Is(e.Err, context.Canceled) {
		message = "cancelled"
	}
	if !strings.HasSuffix(message, ".") {
		message += "."
	}
	return message + " " + e.Outcome()
}

// Unwrap returns why the command failed, so callers still recognize its code, location, and cancellation.
func (e *UnchangedError) Unwrap() error { return e.Err }

// keptWarnings explains each directory, in kept, that an interrupted earlier command created and that recovering it
// kept, because another process has since changed it, so it no longer is that command's to remove.
func keptWarnings(kept []string) []string {
	warnings := []string{}
	for _, directory := range kept {
		warnings = append(warnings, "Kept "+directory+", which an interrupted earlier command created, because another process has since put files in it or replaced it; check what it holds, and remove it if nothing needs it.")
	}
	return warnings
}

// unchanged reports err, a failure of a command that changes files only in a transaction that rolls back, as one
// that wrote no files of its own, noting when it first recovered an interrupted earlier command, unless the
// rollback itself left files to recover.
func unchanged(err error, recovered bool) error {
	var domain *filetxn.Error
	if err == nil || errors.As(err, &domain) && domain.Code == "recovery-required" {
		return err
	}
	return &UnchangedError{Err: err, Recovered: recovered}
}
