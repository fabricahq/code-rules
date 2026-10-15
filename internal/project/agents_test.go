// Verify the AGENTS.md section keeps the project's own text and refuses edits it would lose.

package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/filetxn"
)

// TestRenderAgentsSection records the version and a digest of the lines between the markers, names the project's
// directory, and mentions SKILLS.md only for a project with skills.
func TestRenderAgentsSection(t *testing.T) {
	section := renderAgentsSection(".syllabus", "1.2.3", false)
	lines := strings.SplitAfter(section, "\n")
	marker := agentsBeginMarker.FindStringSubmatch(strings.TrimSuffix(lines[0], "\n"))
	if marker == nil || marker[1] != "1.2.3" || marker[2] != agentsDigest(strings.Join(lines[1:len(lines)-2], "")) || lines[len(lines)-2] != "<!-- syllabus:end -->\n" {
		t.Fatalf("markers:\n%s", section)
	}
	if !strings.Contains(section, "Read `.syllabus/generated/RULES.md`") || !strings.Contains(section, "follow `.syllabus/README.md`") || strings.Contains(section, "SKILLS.md") {
		t.Fatalf("section without skills:\n%s", section)
	}
	if skills := renderAgentsSection(".syllabus", "1.2.3", true); !strings.Contains(skills, "`.syllabus/generated/SKILLS.md` lists the skills") {
		t.Fatalf("section with skills:\n%s", skills)
	}
	if legacy := renderAgentsSection(".code-rules", "1.2.3", false); strings.Contains(legacy, ".syllabus/") || !strings.Contains(legacy, "Read `.code-rules/generated/RULES.md`") {
		t.Fatalf("section for .code-rules/:\n%s", legacy)
	}
}

// TestPlanAgentsSection covers creating, appending, refreshing, and leaving alone, and keeps every line outside the
// markers byte for byte.
func TestPlanAgentsSection(t *testing.T) {
	current := renderAgentsSection(".syllabus", "1.2.3", false)
	older := renderAgentsSection(".syllabus", "1.0.0", false)
	for name, test := range map[string]struct {
		existing *string
		want     string
		action   string
	}{
		"missing":          {nil, current, "created"},
		"empty":            {ptr(""), current, "added"},
		"own text":         {ptr("# Guide\n\nRun make test."), "# Guide\n\nRun make test.\n\n" + current, "added"},
		"blank line":       {ptr("# Guide\n\n"), "# Guide\n\n" + current, "added"},
		"older version":    {ptr("# Guide\n\n" + older + "\n## Releases\n"), "# Guide\n\n" + current + "\n## Releases\n", "updated"},
		"other directory":  {ptr(renderAgentsSection(".code-rules", "1.2.3", false)), current, "updated"},
		"current":          {ptr("# Guide\n\n" + current + "Footer\n"), "", ""},
		"pasted":           {ptr("## Engineering rules\n\n1. Read `.syllabus/generated/RULES.md`.\n"), "", ""},
		"crlf current":     {ptr(strings.ReplaceAll(current, "\n", "\r\n")), "", ""},
		"pasted elsewhere": {ptr("Read `.code-rules/generated/RULES.md`.\n"), "Read `.code-rules/generated/RULES.md`.\n\n" + current, "added"},
	} {
		t.Run(name, func(t *testing.T) {
			root := agentsProject(t, test.existing)
			plan, err := planAgentsSection(context.Background(), root, Options{ToolVersion: "1.2.3"}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer plan.close()
			if test.action == "" {
				if plan.file != nil {
					t.Fatalf("planned a change:\n%s", plan.file.Content)
				}
				return
			}
			if plan.file == nil || string(plan.file.Content) != test.want || plan.change.Action != test.action || plan.change.Path != "AGENTS.md" {
				t.Fatalf("got %+v:\n%s", plan.change, plan.file.Content)
			}
		})
	}
}

// TestPlanAgentsSectionRefuses an edited section and broken markers, so no command overwrites what someone wrote.
func TestPlanAgentsSectionRefuses(t *testing.T) {
	current := renderAgentsSection(".syllabus", "1.2.3", false)
	for name, test := range map[string]struct{ existing, code string }{
		"edited":     {strings.Replace(current, "Briefly report", "Always report", 1), "agents-section-edited"},
		"begin only": {"# Guide\n<!-- syllabus:begin 1.2.3 sha256:0123456789abcdef -->\n", "agents-section-invalid"},
		"end first":  {"<!-- syllabus:end -->\n" + current, "agents-section-invalid"},
		"two begins": {"<!-- syllabus:begin 1.2.3 sha256:0123456789abcdef -->\n" + current, "agents-section-invalid"},
		"two ends":   {current + "<!-- syllabus:end -->\n", "agents-section-invalid"},
		"link":       {current, "unsafe-file"},
	} {
		t.Run(name, func(t *testing.T) {
			root := agentsProject(t, &test.existing)
			if name == "link" {
				agents := filepath.Join(filepath.Dir(root.Name()), "AGENTS.md")
				if err := os.Rename(agents, agents+".real"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("AGENTS.md.real", agents); err != nil {
					t.Fatal(err)
				}
			}
			_, err := planAgentsSection(context.Background(), root, Options{ToolVersion: "1.2.3"}, false)
			var failure *filetxn.Error
			if !errors.As(err, &failure) || failure.Code != test.code {
				t.Fatalf("got %v, want %s", err, test.code)
			}
		})
	}
}

// TestBuildAgentsSection writes the section through build, mentions an authored skill, keeps the project's own text,
// and makes check fail, writing nothing, for an edited section.
func TestBuildAgentsSection(t *testing.T) {
	ctx := context.Background()
	root, options := localProject(t)
	agents := filepath.Join(options.Directory, "AGENTS.md")
	own := "# Project\n\nUse tabs.\n\n"
	data, err := os.ReadFile(agents)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agents, append([]byte(own), data...), 0644); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "skills/check-release/SKILL.md", "---\nname: check-release\ndescription: Check a release.\n---\n# Check\n")
	changes, err := Build(ctx, options)
	if err != nil || changes.Agents == nil || changes.Agents.Action != "updated" {
		t.Fatalf("%+v %v", changes, err)
	}
	if got, err := os.ReadFile(agents); err != nil || string(got) != own+renderAgentsSection(".syllabus", "1.2.3", true) {
		t.Fatalf("%s %v", got, err)
	}
	if result, err := Check(ctx, options); err != nil || !result.Current() {
		t.Fatalf("%+v %v", result, err)
	}
	edited := own + strings.Replace(renderAgentsSection(".syllabus", "1.2.3", true), "use that skill", "use it", 1)
	if err := os.WriteFile(agents, []byte(edited), 0644); err != nil {
		t.Fatal(err)
	}
	var failure *filetxn.Error
	if _, err := Check(ctx, options); !errors.As(err, &failure) || failure.Code != "agents-section-edited" {
		t.Fatalf("check: %v", err)
	}
	if err := root.RemoveAll("skills/check-release"); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(ctx, options); !errors.As(err, &failure) || failure.Code != "agents-section-edited" {
		t.Fatalf("build: %v", err)
	}
	if got, err := os.ReadFile(agents); err != nil || string(got) != edited {
		t.Fatalf("build changed the edited section: %s %v", got, err)
	}
	if generated, err := filetxn.ReadOptional(ctx, root, "generated/SKILLS.md"); err != nil || generated == nil {
		t.Fatalf("a refused build removed the skill's output: %v", err)
	}
}

// TestCheckAcceptsAPastedSection leaves a pointer to RULES.md outside the markers to the project, as earlier versions
// asked users to paste it.
func TestCheckAcceptsAPastedSection(t *testing.T) {
	ctx := context.Background()
	_, options := localProject(t)
	pasted := "## Engineering rules\n\n1. Read `.syllabus/generated/RULES.md` and follow its instructions.\n"
	agents := filepath.Join(options.Directory, "AGENTS.md")
	if err := os.WriteFile(agents, []byte(pasted), 0644); err != nil {
		t.Fatal(err)
	}
	if changes, err := Build(ctx, options); err != nil || changes.Agents != nil {
		t.Fatalf("%+v %v", changes, err)
	}
	if result, err := Check(ctx, options); err != nil || !result.Current() {
		t.Fatalf("%+v %v", result, err)
	}
	if got, err := os.ReadFile(agents); err != nil || string(got) != pasted {
		t.Fatalf("%s %v", got, err)
	}
}

// agentsProject returns the Syllabus directory of a repository whose AGENTS.md holds existing, or has none when nil.
func agentsProject(t *testing.T, existing *string) *os.Root {
	t.Helper()
	repository := t.TempDir()
	if err := os.Mkdir(filepath.Join(repository, ".syllabus"), 0755); err != nil {
		t.Fatal(err)
	}
	if existing != nil {
		if err := os.WriteFile(filepath.Join(repository, "AGENTS.md"), []byte(*existing), 0644); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(filepath.Join(repository, ".syllabus"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	return root
}

func ptr(s string) *string { return &s }
