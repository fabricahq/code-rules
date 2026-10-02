// Package errs holds error contracts that several packages share, so callers can recognize an error by its
// behavior without importing every package that produces one. Each package still defines its own error types.
package errs

// ValidationError is invalid input at a location, such as a configuration key, a file, or a flag. Recognize it
// with errors.As into a variable of this interface type.
type ValidationError interface {
	error
	// ValidationLocation returns the field path, file, or flag the problem is at. It is diagnostic text, never a
	// path to access.
	ValidationLocation() string
	// ValidationProblem explains the failure to a human. Callers must not branch on its text.
	ValidationProblem() string
}
