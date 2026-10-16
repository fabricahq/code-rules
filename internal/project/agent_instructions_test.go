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

// TestRepositoryGuide_IsCurrent fails when this repository's own guide is edited by hand or lags the template, since init then refuses to refresh it.
func TestRepositoryGuide_IsCurrent(t *testing.T) {
	data, err := os.ReadFile("../../.code-rules/README.md")
	if err != nil {
		t.Fatal(err)
	}
	if !unmodifiedProjectGuide(data) {
		t.Fatal(".code-rules/README.md: checksum doesn't match its contents; move it aside, then regenerate it with code-rules project init")
	}
	if string(data) != string(renderProjectGuide()) {
		t.Fatal(".code-rules/README.md: differs from project-guide.md; regenerate it with code-rules project init")
	}
}
