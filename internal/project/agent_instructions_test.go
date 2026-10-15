// Keep every published copy of the AGENTS.md section identical to the section Syllabus writes.

package project

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// agentInstructions matches the fenced "Engineering rules" section that pages quote from AGENTS.md.
var agentInstructions = regexp.MustCompile("(?ms)^```markdown\\n(## Engineering rules\\n.*?)^```$")

// TestAgentInstructions_MatchTheSection fails when the docs drift from the section Syllabus writes for a project
// without skills.
func TestAgentInstructions_MatchTheSection(t *testing.T) {
	section := renderAgentsSection(".syllabus", "1.2.3", false)
	lines := strings.SplitAfter(section, "\n")
	want := strings.Join(lines[1:len(lines)-2], "")
	for _, name := range []string{
		"../../docs/src/content/docs/start-here/set-up-project.md",
		"../../docs/src/content/docs/for-agents/index.md",
	} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if got := agentInstructions.FindSubmatch(data); got == nil || string(got[1]) != want {
			t.Errorf("%s: Engineering rules section differs from the AGENTS.md section:\n%s", name, got)
		}
	}
}
