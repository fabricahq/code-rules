// Keep every published copy of the agent instructions identical to the managed project guide's.

package project

import (
	"os"
	"regexp"
	"testing"
)

// agentInstructions matches the fenced "Engineering rules" section users paste into AGENTS.md.
var agentInstructions = regexp.MustCompile("(?ms)^```markdown\\n## Engineering rules\\n.*?^```$")

// TestAgentInstructions_MatchTheManagedGuide fails when the README or docs drift from the section init ships.
func TestAgentInstructions_MatchTheManagedGuide(t *testing.T) {
	want := agentInstructions.Find([]byte(projectGuideTemplate))
	if want == nil {
		t.Fatal("project-guide.md has no Engineering rules section")
	}
	for _, name := range []string{
		"../../README.md",
		"../../docs/src/content/docs/start-here/set-up-project.md",
		"../../docs/src/content/docs/for-agents/index.md",
	} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if got := agentInstructions.Find(data); string(got) != string(want) {
			t.Errorf("%s: Engineering rules section differs from project-guide.md:\n%s", name, got)
		}
	}
}
