// Exercise release selection against committed notes, real tags, and retry histories.

package release

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/gitfixture"
)

func TestReleaseRequests(t *testing.T) {
	for _, tc := range []struct{ name, tag, notes, existing, want string }{
		{"first", "v0.1.0", "## Features\nFirst release.\n", "", ""},
		{"wrong-first", "v1.0.0", "Notes", "", "first release"},
		{"empty", "v0.1.0", " \n", "", "empty"},
		{"invalid", "v01.1.0", "Notes", "", "semantic version"},
		{"partial", "v0.1", "Notes", "", "semantic version"},
		{"metadata", "v0.1.0+build.1", "Notes", "", "build metadata"},
		{"patch", "v0.1.1", "Notes", "v0.1.0", ""},
		{"minor", "v0.2.0", "Notes", "v0.1.0", ""},
		{"prerelease", "v0.2.0-rc.1", "Notes", "v0.1.0", ""},
		{"backwards", "v0.1.0", "Notes", "v0.2.0", "newer"},
		{"reused", "v0.1.0", "Notes", "v0.1.0", "another commit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, source := fixture(t)
			if tc.existing != "" {
				command(t, f, "tag", tc.existing)
			}
			write(t, source, "releases/"+tc.tag+".md", tc.notes)
			command(t, f, "add", ".")
			command(t, f, "-c", "commit.gpgsign=false", "commit", "-m", "Release")
			head := command(t, f, "rev-parse", "HEAD")
			plan, err := Read(context.Background(), source, f.LatestCommit, head)
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("got %v, want %s", err, tc.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if plan.Tag != tc.tag || plan.Notes != tc.notes || plan.Commit != head || plan.Previous != tc.existing || plan.Prerelease != strings.Contains(tc.tag, "-") {
				t.Fatalf("unexpected plan: %+v", plan)
			}
			command(t, f, "tag", tc.tag)
			retry, err := Read(context.Background(), source, f.LatestCommit, head)
			if err != nil || !reflect.DeepEqual(retry, plan) {
				t.Fatalf("retry: %+v %v", retry, err)
			}
		})
	}
}

func TestReleaseNotesOwnership(t *testing.T) {
	f, source := fixture(t)
	plan, err := Read(context.Background(), source, f.FirstCommit, f.LatestCommit)
	if err != nil || plan.Tag != "" {
		t.Fatal(plan, err)
	}
	write(t, source, "releases/v0.1.0.md", "Original notes")
	command(t, f, "add", ".")
	command(t, f, "-c", "commit.gpgsign=false", "commit", "-m", "Notes")
	first := command(t, f, "rev-parse", "HEAD")
	command(t, f, "tag", "v0.1.0")
	write(t, source, "releases/v0.1.0.md", "Uncommitted edit")
	plan, err = Read(context.Background(), source, f.LatestCommit, first)
	if err != nil || plan.Notes != "Original notes" {
		t.Fatal(plan, err)
	}
	command(t, f, "add", ".")
	command(t, f, "-c", "commit.gpgsign=false", "commit", "-m", "Edit")
	if _, err := Read(context.Background(), source, first, "HEAD"); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatal(err)
	}
	command(t, f, "tag", "-d", "v0.1.0")
	write(t, source, "releases/v0.2.0.md", "Second")
	command(t, f, "add", ".")
	command(t, f, "-c", "commit.gpgsign=false", "commit", "-m", "More notes")
	if _, err := Read(context.Background(), source, f.LatestCommit, "HEAD"); err == nil || !strings.Contains(err.Error(), "one release") {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) (*gitfixture.Fixture, string) {
	t.Helper()
	f, err := gitfixture.New(context.Background(), map[string][]byte{"README.md": []byte("Fixture")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	})
	command(t, f, "tag", "-d", "v1.0.0", "v1.2.0")
	return f, filepath.Join(f.Directory, "repository")
}

func command(t *testing.T, f *gitfixture.Fixture, args ...string) string {
	t.Helper()
	out, err := f.Command(context.Background(), args...)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(out)
}

func write(t *testing.T, source, name, body string) {
	t.Helper()
	file := filepath.Join(source, name)
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCorrectOrWithdrawUntaggedRequest(t *testing.T) {
	f, source := fixture(t)
	write(t, source, "releases/v0.1.0.md", "Original")
	command(t, f, "add", ".")
	command(t, f, "-c", "commit.gpgsign=false", "commit", "-m", "Request")
	original := command(t, f, "rev-parse", "HEAD")
	write(t, source, "releases/v0.1.0.md", "Corrected")
	command(t, f, "add", ".")
	command(t, f, "-c", "commit.gpgsign=false", "commit", "-m", "Correct")
	corrected := command(t, f, "rev-parse", "HEAD")
	plan, err := Read(context.Background(), source, original, corrected)
	if err != nil || plan.Tag != "v0.1.0" || plan.Notes != "Corrected" {
		t.Fatal(plan, err)
	}
	command(t, f, "rm", "releases/v0.1.0.md")
	command(t, f, "-c", "commit.gpgsign=false", "commit", "-m", "Withdraw")
	plan, err = Read(context.Background(), source, corrected, "HEAD")
	if err != nil || plan.Tag != "" {
		t.Fatal(plan, err)
	}
}
