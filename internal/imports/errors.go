// Share the Git runner's failure type, so every import failure carries one stable code.

package imports

import "github.com/fabricahq/code-rules/internal/gitexec"

// Error is the type of every import failure, including Git failures. Its Code is stable for callers.
type Error = gitexec.Error

// fail constructs an import failure with a stable category and safe explanation.
func fail(code, problem string, cause error) error {
	return gitexec.Fail(code, problem, cause)
}
