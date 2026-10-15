// Maintain the Syllabus section of the repository's AGENTS.md, leaving every line outside its markers to the project.

package project

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/filetxn"
)

// agentsFile is the instruction file Syllabus maintains a section in, at the repository root, the parent of the
// Syllabus directory. Agents that read another file, such as Claude Code's CLAUDE.md, import or link it.
const agentsFile = "AGENTS.md"

// agentsSectionTemplate is the section's fixed instructions, written for a project in .syllabus/; documentation
// pages quote it.
//
//go:embed agents-section.md
var agentsSectionTemplate string

// agentsSectionSkills follows the instructions when the project has skills, which generated/SKILLS.md lists.
const agentsSectionSkills = "\n`.syllabus/generated/SKILLS.md` lists the skills this project installs, what\neach one does, and where it's installed. When a task matches a skill's\ndescription, use that skill.\n"

// agentsSectionGuide ends every section, pointing agents that manage rules or skills to the project guide.
const agentsSectionGuide = "\nTo add, change, or update rules or skills, follow `.syllabus/README.md`.\n"

// The section is fenced by these markers. The begin marker records the version that wrote the section and the first
// 16 hexadecimal digits of the SHA-256 of the lines between the markers, so a refresh can tell an unedited section
// written by another version, which it replaces, from a hand edit, which it refuses to overwrite.
var (
	agentsBeginMarker = regexp.MustCompile(`^<!-- syllabus:begin (\S+) sha256:([0-9a-f]{16}) -->$`)
	agentsEndMarker   = regexp.MustCompile(`^<!-- syllabus:end -->$`)
)

// AgentsChange reports how a command changed the Syllabus section of AGENTS.md.
type AgentsChange struct {
	Path string `json:"path"` // Relative to the repository root.
	// Action is created when the command created AGENTS.md, added when it added the section to an existing file, and
	// updated when it replaced an older section.
	Action string `json:"action"`
}

// toolVersion is the version generated output and the AGENTS.md section record, with a placeholder for development
// builds that pass none.
func toolVersion(options Options) string {
	if options.ToolVersion == "" {
		return "0.0.0-development"
	}
	return options.ToolVersion
}

// renderAgentsSection returns the section, markers included, for a project whose Syllabus directory is named
// directory, such as .syllabus, mentioning SKILLS.md when skills is true.
func renderAgentsSection(directory, version string, skills bool) string {
	inner := agentsSectionTemplate
	if skills {
		inner += agentsSectionSkills
	}
	inner = strings.ReplaceAll(inner+agentsSectionGuide, ".syllabus/", directory+"/")
	return fmt.Sprintf("<!-- syllabus:begin %s sha256:%s -->\n%s<!-- syllabus:end -->\n", version, agentsDigest(inner), inner)
}

func agentsDigest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:8])
}

// agentsSection locates the Syllabus section in AGENTS.md's lines. begin and end are -1 when it has none.
type agentsSection struct {
	lines      []string
	begin, end int
	edited     bool
}

// inspectAgentsSection finds the section in existing, failing with agents-section-invalid when its markers are
// unpaired, out of order, or repeated.
func inspectAgentsSection(existing string) (agentsSection, error) {
	s := agentsSection{lines: strings.SplitAfter(existing, "\n"), begin: -1, end: -1}
	invalid := func(problem string) error {
		return failure("agents-section-invalid", agentsFile+": "+problem+"; fix or remove the syllabus markers, then run the command again", nil)
	}
	for i, line := range s.lines {
		trimmed := strings.TrimRight(line, "\r\n")
		switch {
		case agentsBeginMarker.MatchString(trimmed):
			if s.begin != -1 {
				return s, invalid("found more than one syllabus:begin marker")
			}
			s.begin = i
		case agentsEndMarker.MatchString(trimmed):
			if s.end != -1 {
				return s, invalid("found more than one syllabus:end marker")
			}
			s.end = i
		}
	}
	if (s.begin == -1) != (s.end == -1) || s.end < s.begin {
		return s, invalid("found a syllabus:begin or syllabus:end marker without its pair")
	}
	if s.begin != -1 {
		marker := agentsBeginMarker.FindStringSubmatch(strings.TrimRight(s.lines[s.begin], "\r\n"))
		// A checkout that converts line endings to CRLF keeps an unedited section.
		s.edited = agentsDigest(strings.ReplaceAll(strings.Join(s.lines[s.begin+1:s.end], ""), "\r\n", "\n")) != marker[2]
	}
	return s, nil
}

func (s agentsSection) found() bool { return s.begin != -1 }

// current returns the section, markers included, with LF line endings, ending in a newline.
func (s agentsSection) current() string {
	text := strings.ReplaceAll(strings.Join(s.lines[s.begin:s.end+1], ""), "\r\n", "\n")
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text
}

// replace swaps the section, markers included, keeping every line outside it.
func (s agentsSection) replace(section string) string {
	return strings.Join(s.lines[:s.begin], "") + section + strings.Join(s.lines[s.end+1:], "")
}

// appendAgentsSection adds section after the file's own text, separated by one blank line.
func appendAgentsSection(existing, section string) string {
	if existing == "" {
		return section
	}
	if !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	if !strings.HasSuffix(existing, "\n\n") {
		existing += "\n"
	}
	return existing + section
}

// agentsPlan is the AGENTS.md publication a command makes, in the repository root: file is nil when AGENTS.md needs
// no change. close releases the repository root.
type agentsPlan struct {
	repository *os.Root
	file       *filetxn.File
	change     *AgentsChange
}

// planAgentsSection plans the AGENTS.md beside the Syllabus directory at root so it holds the current section for
// options' version, mentioning SKILLS.md when skills is true. It plans no change when the section is current, or when
// the file has no section but already points to the project's generated/RULES.md, as the block Syllabus once asked
// users to paste does: that file stays the project's until its pointer is removed. It fails with
// agents-section-edited when the section's lines changed since Syllabus wrote them, and with agents-section-invalid
// when its markers are broken.
func planAgentsSection(ctx context.Context, root *os.Root, options Options, skills bool) (plan agentsPlan, err error) {
	plan.repository, err = os.OpenRoot(filepath.Dir(root.Name()))
	if err != nil {
		return agentsPlan{}, fmt.Errorf("open the repository root for %s: %w", agentsFile, err)
	}
	defer func() {
		if err != nil {
			plan.repository.Close()
			plan = agentsPlan{}
		}
	}()
	data, err := filetxn.ReadOptional(ctx, plan.repository, agentsFile)
	var unsafe *filetxn.Error
	if errors.As(err, &unsafe) && unsafe.Code == "unsafe-file" {
		return plan, failure("unsafe-file", agentsFile+": Syllabus keeps a section in this file and doesn't write through a link; make AGENTS.md the file, and link or import it from the file your agent reads, such as CLAUDE.md, then run the command again", err)
	}
	if err != nil {
		return plan, err
	}
	directory := filepath.Base(root.Name())
	want := renderAgentsSection(directory, toolVersion(options), skills)
	existing := string(data)
	s, err := inspectAgentsSection(existing)
	if err != nil {
		return plan, err
	}
	var content, action string
	switch {
	case !s.found() && strings.Contains(existing, directory+"/generated/RULES.md"):
		return plan, nil
	case !s.found():
		content, action = appendAgentsSection(existing, want), "added"
		if data == nil {
			action = "created"
		}
	case s.edited:
		return plan, failure("agents-section-edited", agentsFile+": the Syllabus section was edited after Syllabus wrote it; move your text outside the syllabus:begin and syllabus:end markers and delete the section, then run the command again to restore it", nil)
	case s.current() == want:
		return plan, nil
	default:
		content, action = s.replace(want), "updated"
	}
	plan.file = &filetxn.File{Path: agentsFile, Content: []byte(content), Previous: data}
	plan.change = &AgentsChange{Path: agentsFile, Action: action}
	return plan, nil
}

// publish writes the planned AGENTS.md under w, refusing with concurrent-change when the file changed since
// planning, and returns the change and any cleanup warnings. It runs after the command's own transaction, which its
// publication isn't part of: the section depends only on the version and the directory's name, so a failure here
// leaves the project's other files consistent, and running the command again finishes it.
func (p agentsPlan) publish(w *filetxn.Writer) (*AgentsChange, []string, error) {
	if p.file == nil {
		return nil, nil, nil
	}
	changes, err := w.Publish(p.repository, []filetxn.File{*p.file})
	if err != nil {
		return nil, nil, err
	}
	return p.change, changes.Warnings, nil
}

func (p agentsPlan) close() {
	if p.repository != nil {
		p.repository.Close()
	}
}

// includeAgents publishes plan under w and reports it in changes.
func includeAgents(w *filetxn.Writer, plan agentsPlan, changes *FileChanges) error {
	change, warnings, err := plan.publish(w)
	if err != nil {
		return err
	}
	changes.Agents = change
	changes.Warnings = append(nonNil(changes.Warnings), warnings...)
	return nil
}

// listsSkills reports whether output lists skills in generated/SKILLS.md, which the AGENTS.md section then mentions.
func listsSkills(output build.Output) bool {
	_, ok := output.Files[skillsIndexFile]
	return ok
}

// skillsIndexFile is the generated file, by path in generated/, that lists a project's skills.
const skillsIndexFile = "SKILLS.md"
