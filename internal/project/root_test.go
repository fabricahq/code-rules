// Check that a project still keeping authored files in local/ is refused with the moves of its own files.

package project

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestBuild_ListsTheMovesOfALocalFoldersOwnFiles refuses a project with a local/ entry and lists where each of its
// files belongs now, up to localFolderMoves of them, then how many more there are; an entry without files is only
// to be deleted.
func TestBuild_ListsTheMovesOfALocalFoldersOwnFiles(t *testing.T) {
	rules := func(count int) []string {
		var files []string
		for index := range count {
			files = append(files, fmt.Sprintf("local/practices/legacy/rule-%02d.md", index+1))
		}
		return files
	}
	moves := func(count int) string {
		var lines []string
		for index := range count {
			name := fmt.Sprintf("practices/legacy/rule-%02d.md", index+1)
			lines = append(lines, "  local/"+name+"  ->  rules/"+name)
		}
		return strings.Join(lines, "\n")
	}
	for _, test := range []struct {
		name  string
		files []string
		// dir makes local/ hold an empty folder when files is empty, and local a file otherwise.
		dir  bool
		want string
	}{
		{
			name:  "one rule",
			files: []string{"local/practices/legacy/old-rule.md"},
			want:  ":\n\n  local/practices/legacy/old-rule.md  ->  rules/practices/legacy/old-rule.md\n\nRule IDs",
		},
		{
			name:  "groups, rules, rule assets, shared assets, and the README",
			files: []string{"local/techs/go/errors.md", "local/README.md", "local/techs/go/_group.yaml", "local/assets/retry.svg", "local/techs/go/assets/errors/flow.svg"},
			want: ":\n\n" +
				"  local/README.md                        ->  rules/README.md\n" +
				"  local/assets/retry.svg                 ->  assets/retry.svg\n" +
				"  local/techs/go/_group.yaml             ->  rules/techs/go/_group.yaml\n" +
				"  local/techs/go/assets/errors/flow.svg  ->  rules/techs/go/assets/errors/flow.svg\n" +
				"  local/techs/go/errors.md               ->  rules/techs/go/errors.md\n\nRule IDs",
		},
		{name: "as many files as the list shows", files: rules(localFolderMoves), want: ":\n\n" + moves(localFolderMoves) + "\n\nRule IDs"},
		{name: "one file more than the list shows", files: rules(localFolderMoves + 1), want: ":\n\n" + moves(localFolderMoves) + "\n  and 1 more file, moved the same way\n\nRule IDs"},
		{name: "many files more than the list shows", files: rules(localFolderMoves + 7), want: ":\n\n" + moves(localFolderMoves) + "\n  and 7 more files, moved the same way\n\nRule IDs"},
		{name: "an empty folder", dir: true, want: "It no longer reads .syllabus/local/, which holds no files: delete it, then run syllabus project build."},
		{name: "a file", want: "It no longer reads .syllabus/local/, which holds no files: delete it, then run syllabus project build."},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, options := localProject(t)
			for _, name := range test.files {
				writeFixture(t, root, name, projectRule)
			}
			switch {
			case test.dir:
				if err := root.MkdirAll("local/practices", 0700); err != nil {
					t.Fatal(err)
				}
			case len(test.files) == 0:
				writeFixture(t, root, "local", "")
			}
			_, err := Build(context.Background(), options)
			var invalid *ValidationError
			if !errors.As(err, &invalid) || invalid.Location != ".syllabus/local/" || !strings.Contains(invalid.Problem, test.want) {
				t.Fatalf("got %v; want a problem containing\n%s", err, test.want)
			}
		})
	}
}
