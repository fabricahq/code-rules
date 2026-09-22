// Validate terminal answers before advancing to another field or acquiring writer ownership.

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
)

// askValidated retries only rejected answers; cancellation and terminal failures end the command.
func (f *authoringFlags) askValidated(label string, validate func(string) error) (string, error) {
	for {
		answer, err := f.ask(label)
		if err != nil {
			return "", err
		}
		if err := validate(answer); err != nil {
			if _, writeErr := io.WriteString(f.command.ErrOrStderr(), humanError(f.command.ErrOrStderr(), err)+"\nPlease try again.\n\n"); writeErr != nil {
				return "", writeErr
			}
			continue
		}
		return answer, nil
	}
}

// validateAnswer reuses domain parsers for constrained fields and rejects blank required text.
// Explicit flags remain subject to operation validation and are never replaced by prompts.
func validateAnswer(name, value string) error {
	if value == "" {
		return fmt.Errorf("provide a value for --%s", name)
	}
	switch name {
	case "impact":
		_, err := rules.ParseImpact(value, "--impact")
		return err
	case "repository":
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		_, err = rules.ParseRepository(data, "--repository")
		return err
	case "ref":
		_, err := rules.ParseGitRef(value, "--ref")
		return err
	case "version":
		_, err := rules.ParseVersionConstraint(value, "--version")
		return err
	}
	return nil
}

// sourceGroupSelection keeps prompt validation and the saved declaration on the same wildcard representation.
func sourceGroupSelection(groups []string) any {
	if len(groups) == 1 && (groups[0] == "*" || groups[0] == "techs/*" || groups[0] == "practices/*") {
		return groups[0]
	}
	return groups
}

// splitPromptGroups trims each comma-separated path before domain validation.
func splitPromptGroups(answer string) []string {
	groups := strings.Split(answer, ",")
	for i := range groups {
		groups[i] = strings.TrimSpace(groups[i])
	}
	return groups
}
