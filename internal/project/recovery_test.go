// Report what recovering an interrupted earlier command kept rather than remove.

package project

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/filetxn"
)

// TestSyncAndBuild_WarnAboutADirectoryRecoveryKept: an interrupted fork update created local/techs/go/assets, and
// another process then put a file in it. Recovery keeps the directory and the file, and sync and build say so.
func TestSyncAndBuild_WarnAboutADirectoryRecoveryKept(t *testing.T) {
	for _, command := range []string{"sync", "build"} {
		t.Run(command, func(t *testing.T) {
			_, options, library := syncProject(t)
			root, err := openProject(context.Background(), options, false)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			writeFixture(t, root, "local/techs/go/mine.md", strings.Replace(projectRule, "# Return errors", "# Our errors", 1))
			if _, err := Sync(context.Background(), options, library); err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, "local/techs/go/assets/mine/notes.md", "Their notes.\n")
			after := &filetxn.Tree{Files: map[string][]byte{"notes.md": []byte("New notes.\n")}}
			journal, err := json.Marshal(map[string]any{
				"formatVersion": 6,
				"entries":       []map[string]any{{"name": "local/techs/go/assets/other", "before": (*filetxn.Tree)(nil).Digest(), "after": after.Digest(), "existed": false}},
				"parents":       []string{"local/techs/go/assets"},
			})
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, ".code-rules-transaction/journal.json", string(journal)+"\n")
			writeFixture(t, root, ".code-rules-transaction/parent-0.created", "\"local/techs/go/assets\"\n")
			var changes FileChanges
			if command == "sync" {
				changes, err = Sync(context.Background(), options, library)
			} else {
				changes, err = Build(context.Background(), options)
			}
			if err != nil {
				t.Fatal(err)
			}
			if data, err := root.ReadFile("local/techs/go/assets/mine/notes.md"); err != nil || string(data) != "Their notes.\n" {
				t.Fatalf("recovery lost another process's file: %q, %v", data, err)
			}
			if !changes.Recovered || len(changes.Warnings) == 0 || !strings.Contains(changes.Warnings[len(changes.Warnings)-1], "Kept local/techs/go/assets, which an interrupted earlier command created") {
				t.Fatalf("changes %+v", changes)
			}
		})
	}
}
