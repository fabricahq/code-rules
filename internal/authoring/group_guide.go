// Create a small orientation file alongside group metadata without copying mutable group descriptions.

package authoring

import (
	"fmt"
	"path"
)

// groupFiles publishes metadata and its guide together; existing files remain protected by the authoring writer.
func groupFiles(directory, id string, metadata []byte, library bool) []authoredFile {
	guide := fmt.Sprintf("# Group `%s`\n\n", id) + `This folder contains the source rules for one technology or engineering practice.
Read [_group.json](_group.json) first: its name and description define the group's scope, and whenToRead tells agents when to consider its rules.
Keep that metadata current when the group's scope changes.

## Instructions for agents

1. Read the group's metadata before adding or editing a rule. Put guidance here only when it fits that scope.
2. Keep each independently adoptable rule in its own Markdown file. Give it a clear title, reading cue, impact, and impact description.
3. Read each relevant or plausibly relevant rule completely, including its exceptions, before relying on it.
4. Complete unfinished drafts, validate the result, and inspect the changed files before reporting completion.

`
	if library {
		guide += fmt.Sprintf("## Add or edit rules\n\nRun commands from the library root, two directories above this folder.\nUse `code-rules library add rule %s/<rule-name>` to add a rule; run `code-rules library add rule --help` for metadata and body options.\nEdit existing rule files directly, then run `code-rules library check`. Resolve errors before committing or publishing the library.\n\nA consuming project selects this group in its configuration and runs sync. Its generated/RULES.md identifies the adopted rules after exclusions and replacements.\n", id)
	} else {
		guide += fmt.Sprintf("## Add or edit rules\n\nFollow [the project guide](../../../README.md) for complete commands and the correct configuration path.\nUse `code-rules local add rule %s/<rule-name>` to add a rule. Edit existing rule files directly.\nRun build and check with that configuration after local changes, then inspect [the resolved rules](../../../generated/RULES.md).\nUse the resolved rules when working on the project: they include imported guidance and apply exclusions and replacements.\n", id)
	}
	guide += "\nThis README explains authoring; it is not an engineering rule and is not included in generated guidance. Keep group descriptions in _group.json.\n"
	return []authoredFile{
		{name: path.Join(directory, "_group.json"), data: metadata},
		{name: path.Join(directory, "README.md"), data: []byte(guide)},
	}
}
