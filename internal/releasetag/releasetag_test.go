// Check listing and reading release tags against a real repository.

package releasetag_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fabricahq/code-rules/internal/gitexec"
	"github.com/fabricahq/code-rules/internal/releasetag"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// numbers returns the tags' numbers in order.
func numbers(tags []releasetag.Tag) []int {
	result := []int{}
	for _, tag := range tags {
		result = append(result, tag.Number)
	}
	return result
}

// TestListAndRead_ReleaseTagsInNumberOrder lists release/<number> tags numerically, leaves out other names and,
// when asked, tags off HEAD's history, and reads each annotated tag's notes and record.
func TestListAndRead_ReleaseTagsInNumberOrder(t *testing.T) {
	ctx := context.Background()
	f, err := gitfixture.New(ctx, map[string][]byte{"README.md": []byte("Library.\n")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	dir := f.Worktree()
	for _, number := range []string{"2", "10"} {
		if err := f.Tag(ctx, dir, "release/"+number, "Library release "+number+".\n\n---\nformatVersion: 1\nrelease: "+number+"\nrules: {}\n"); err != nil {
			t.Fatal(err)
		}
	}
	unreachable, err := f.Command(ctx, "commit-tree", "HEAD^{tree}", "-m", "Off the branch.")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"tag", "release/3", f.FirstCommit}, {"tag", "release/01"}, {"tag", "release/next"}, {"tag", "--annotate", "--message", "Notes only.", "release/4", unreachable}} {
		if _, err := f.Command(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	runner, err := gitexec.Isolated(gitexec.Options{GitPath: f.GitPath, Environment: f.Environment})
	if err != nil {
		t.Fatal(err)
	}

	all, err := releasetag.List(ctx, runner, dir, "")
	if err != nil || len(all) != 4 || all[0].Number != 2 || all[1].Number != 3 || all[2].Number != 4 || all[3].Number != 10 {
		t.Fatalf("listed %v, %v; want 2, 3, 4, 10", numbers(all), err)
	}
	if tag := all[0]; tag.Type != "tag" || tag.TargetType != "commit" || tag.Target != f.LatestCommit || tag.Size == 0 {
		t.Fatalf("annotated release/2: %+v", tag)
	}
	if tag := all[1]; tag.Type != "commit" || tag.Object != f.FirstCommit || tag.Target != "" || tag.TargetType != "" {
		t.Fatalf("lightweight release/3: %+v", tag)
	}
	reachable, err := releasetag.List(ctx, runner, dir, "HEAD")
	if err != nil || len(reachable) != 3 || reachable[2].Number != 10 {
		t.Fatalf("listed %v from HEAD, %v; want 2, 3, 10", numbers(reachable), err)
	}

	read, err := releasetag.Read(ctx, runner, dir, []releasetag.Tag{all[0], all[3]})
	if err != nil || len(read) != 2 || read[0].Notes != "Library release 2." || read[0].Record.Release != 2 || read[1].Record.Release != 10 {
		t.Fatalf("read %+v, %v", read, err)
	}
	_, err = releasetag.Read(ctx, runner, dir, []releasetag.Tag{all[0], all[2]})
	var invalid *releasetag.RecordError
	var validation *rules.ValidationError
	if !errors.As(err, &invalid) || invalid.Tag != "release/4" || !errors.As(err, &validation) || validation.Location != "release/4" {
		t.Fatalf("read a tag without a record: %v", err)
	}
}
