// Render common group orientation while leaving workflow instructions with the owning operation.

package rules

import (
	"fmt"
)

// GroupGuide identifies a group and appends the owner's authoring workflow without copying metadata.
func GroupGuide(id, instructions string) []byte {
	guide := fmt.Sprintf("# Group `%s`\n\n", id) + `This folder contains the source rules for one technology or engineering practice.
Read [_group.yaml](_group.yaml) first: its name and description define the group's scope, and whenToRead tells agents when to consider its rules.
Keep that metadata current when the group's scope changes.

## Instructions for agents

1. Read the group's metadata before adding or editing a rule. Put guidance here only when it fits that scope.
2. Keep each independently adoptable rule in its own Markdown file. Give it a clear title, reading cue, impact, and impact description.
3. Read each relevant or plausibly relevant rule completely, including its exceptions, before relying on it.
4. Complete unfinished drafts, validate the result, and inspect the changed files before reporting completion.

`
	return []byte(guide + instructions + "\nThis README explains authoring; it is not an engineering rule and is not included in generated guidance. Keep group descriptions in _group.yaml.\n")
}
