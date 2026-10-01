// Report a project command that failed without writing files, so people know nothing changed.

package project

import (
	"context"
	"errors"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
)

// UnchangedError reports a failed command that left every file as it was; Err is why it failed.
type UnchangedError struct {
	Err error
}

// Error says that the command was cancelled, or why it failed, and that it wrote no files.
func (e *UnchangedError) Error() string {
	if errors.Is(e.Err, context.Canceled) {
		return "cancelled; no files were written"
	}
	message := strings.TrimSpace(e.Err.Error())
	if !strings.HasSuffix(message, ".") {
		message += "."
	}
	return message + " No files were written."
}

// Unwrap returns why the command failed, so callers still recognize its code, location, and cancellation.
func (e *UnchangedError) Unwrap() error { return e.Err }

// unchanged reports err, a failure of a command that changes files only in a transaction that rolls back, as one
// that wrote nothing, unless the rollback itself left files to recover.
func unchanged(err error) error {
	var domain *filetxn.Error
	if err == nil || errors.As(err, &domain) && domain.Code == "recovery-required" {
		return err
	}
	return &UnchangedError{Err: err}
}
