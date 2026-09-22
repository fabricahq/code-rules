// Reject duplicate library authoring targets before prompts without acquiring writer ownership.

package library

import (
	"context"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
)

// CheckNewGroup requires both group files to be absent; AddGroup still enforces exclusive publication.
func CheckNewGroup(ctx context.Context, id string, options Options) error {
	if err := rules.ValidateGroupID(id, "group"); err != nil {
		return err
	}
	return checkNewFiles(ctx, options, id+"/_group.json", id+"/README.md")
}

// CheckNewRule rejects an occupied rule path; AddRule still enforces exclusive publication.
func CheckNewRule(ctx context.Context, id string, options Options) error {
	if _, err := rules.GroupFromPath(id+".md", "rule"); err != nil {
		return err
	}
	return checkNewFiles(ctx, options, id+".md")
}

func checkNewFiles(ctx context.Context, options Options, names ...string) error {
	root, err := openLibrary(ctx, options, false)
	if err != nil {
		return err
	}
	defer root.Close()
	if _, _, err := libraryManifest(ctx, root); err != nil {
		return err
	}
	return filetxn.RequireAbsent(ctx, root, names...)
}
