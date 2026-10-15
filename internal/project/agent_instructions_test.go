// Keep every published copy of the AGENTS.md section identical to the section Syllabus writes.

package project

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// agentInstructions matches the fenced "Skills and Rules" section that pages quote from AGENTS.md.
var agentInstructions = regexp.MustCompile("(?ms)^```markdown\\n(## Skills and Rules\\n.*?)^```$")

// TestAgentInstructions_MatchTheSection fails when the docs drift from the section Syllabus writes.
func TestAgentInstructions_MatchTheSection(t *testing.T) {
	section := renderAgentsSection(".syllabus", "1.2.3")
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
			t.Errorf("%s: Skills and Rules section differs from the AGENTS.md section:\n%s", name, got)
		}
	}
}
