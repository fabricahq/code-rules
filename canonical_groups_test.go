// Keep the published canonical group list valid, since other tools read it at pinned commits.

package coderules_test

import (
	"os"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

func TestCanonicalGroupListIsValid(t *testing.T) {
	data, err := os.ReadFile("canonical-groups.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rules.ParseCanonicalGroups(data, "canonical-groups.yaml"); err != nil {
		t.Fatal(err)
	}
}
