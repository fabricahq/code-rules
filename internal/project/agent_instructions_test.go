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

// TestRepositoryGuide_IsCurrent fails when this repository's own guide is edited by hand or lags the template, since init then refuses to refresh it.
func TestRepositoryGuide_IsCurrent(t *testing.T) {
	// This repository keeps its own files in .code-rules/, the directory Code Rules used, until it moves them.
	data, err := os.ReadFile("../../.code-rules/README.md")
	if err != nil {
		t.Fatal(err)
	}
	if !unmodifiedProjectGuide(data) {
		t.Fatal(".code-rules/README.md: checksum doesn't match its contents; move it aside, then regenerate it with syllabus project init")
	}
	if string(data) != string(renderProjectGuide()) {
		t.Fatal(".code-rules/README.md: differs from project-guide.md; regenerate it with syllabus project init")
	}
}
