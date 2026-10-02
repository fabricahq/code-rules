// The validation error this package reports for invalid input, and its translation of internal packages' errors.

package coderules

import (
	"errors"

	"github.com/fabricahq/code-rules/internal/errs"
)

// ValidationError reports invalid input at a location, such as a field of a release record or a rule file. Every
// exported function reports invalid input as a *ValidationError; inspect it with errors.As.
type ValidationError struct {
	// Location names the input and, when the problem is inside it, the field, such as
	// release/4.changes.techs/go/handle-errors.from or fabrica:techs/go/handle-errors.md.title. It is diagnostic
	// text, never a path to access.
	Location string
	// Problem explains the failure to a human. Callers must not branch on its text.
	Problem string
}

var _ errs.ValidationError = (*ValidationError)(nil)

// Error returns the location and the problem.
func (e *ValidationError) Error() string { return e.Location + ": " + e.Problem }

// ValidationLocation returns Location.
func (e *ValidationError) ValidationLocation() string { return e.Location }

// ValidationProblem returns Problem.
func (e *ValidationError) ValidationProblem() string { return e.Problem }

// invalid returns a *ValidationError reporting problem at location.
func invalid(location, problem string) error {
	return &ValidationError{Location: location, Problem: problem}
}

// translate returns err with the validation error of an internal package, such as decode or librarytree, replaced
// by a *ValidationError with the same location and problem, keeping err's text, so callers outside Code Rules only
// ever see this package's error types. It returns other errors, and nil, as they are.
func translate(err error) error {
	var internal errs.ValidationError
	if !errors.As(err, &internal) {
		return err
	}
	if _, own := internal.(*ValidationError); own {
		return err
	}
	translated := &ValidationError{Location: internal.ValidationLocation(), Problem: internal.ValidationProblem()}
	if err == error(internal) {
		return translated
	}
	return &contextError{text: err.Error(), invalid: translated}
}

// contextError keeps the text of an error that adds context around an internal package's validation error, such
// as the rule path around an invalid group ID, while carrying the translated *ValidationError instead.
type contextError struct {
	text    string
	invalid *ValidationError
}

func (e *contextError) Error() string { return e.text }

func (e *contextError) Unwrap() error { return e.invalid }
