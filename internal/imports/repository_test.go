// Exercise actual Git transport, exact revision fetches, and bounded process failures.

package imports

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// fixtureRepository creates a real repository served by an isolated local upload-pack helper.
func fixtureRepository(t *testing.T) *gitfixture.Fixture {
	t.Helper()
	f, err := gitfixture.New(context.Background(), map[string][]byte{"README.md": []byte("fixture\n")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	})
	return f
}

// fetchRevision opens a partial repository for source and fetches ref, which it parses as the source's ref, unless
// ref is empty, returning the repository, which the caller closes, and the ref's commit. Failure closes the
// repository.
func fetchRevision(ctx context.Context, source rules.Source, ref string, options Options) (*repository, string, error) {
	if ref != "" {
		parsed, err := rules.ParseGitRef(ref, "ref")
		if err != nil {
			return nil, "", err
		}
		source.Ref = parsed
	}
	repo, err := openRepository(ctx, source, options)
	if err != nil {
		return nil, "", err
	}
	commit, err := repo.fetchRef(ctx, source)
	if err != nil {
		return nil, "", errors.Join(err, repo.Close())
	}
	return repo, commit, nil
}

// gitRef parses text as a ref, failing the test when it isn't one.
func gitRef(t *testing.T, text string) rules.GitRef {
	t.Helper()
	ref, err := rules.ParseGitRef(text, "ref")
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

// requireCode checks the stable failure category without matching Git's private diagnostics.
func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	var failure *Error
	if !errors.As(err, &failure) || failure.Code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
}

// TestFetchRevision exercises exact commits, lightweight and annotated tags, and missing refs.
func TestFetchRevision(t *testing.T) {
	f := fixtureRepository(t)
	options := Options{GitPath: f.GitPath, Environment: f.Environment}
	for _, tc := range []struct{ name, ref, want string }{
		{"commit", f.FirstCommit, f.FirstCommit}, {"lightweight", "v1.0.0", f.FirstCommit}, {"annotated", "v1.2.0", f.LatestCommit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			revision, commit, err := fetchRevision(context.Background(), rules.Source{Name: "team", Repository: f.Repository}, tc.ref, options)
			if err != nil {
				t.Fatal(err)
			}
			dir := revision.directory
			if commit != tc.want {
				t.Fatalf("commit %s", commit)
			}
			if _, err := os.Stat(filepath.Join(dir, "README.md")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("working files were checked out")
			}
			if err := revision.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("temporary repository retained")
			}
		})
	}
	revision, _, err := fetchRevision(context.Background(), rules.Source{Name: "team", Repository: f.Repository}, "missing", options)
	requireCode(t, err, "version-not-found")
	if revision != nil {
		t.Fatal("partial result")
	}
}

// TestFetchRejectsNonCommitTags refuses a tag that names a blob instead of a commit.
func TestFetchRejectsNonCommitTags(t *testing.T) {
	f := fixtureRepository(t)
	ctx := context.Background()
	options := Options{GitPath: f.GitPath, Environment: f.Environment}
	blob, err := f.Command(ctx, "rev-parse", "HEAD:README.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Command(ctx, "tag", "blob", blob); err != nil {
		t.Fatal(err)
	}
	_, _, err = fetchRevision(ctx, rules.Source{Repository: f.Repository}, "blob", options)
	requireCode(t, err, "unsupported-content")
}

// TestEnvironmentIsolation verifies inherited repository state cannot alter the fetched commit.
func TestEnvironmentIsolation(t *testing.T) {
	f := fixtureRepository(t)
	environment := append(append([]string{}, f.Environment...), "GIT_DIR=/missing", "GIT_WORK_TREE=/missing", "GIT_INDEX_FILE=/missing", "GIT_OBJECT_DIRECTORY=/missing", "GIT_ALTERNATE_OBJECT_DIRECTORIES=/missing", "GIT_NAMESPACE=hidden", "GIT_SHALLOW_FILE=/missing")
	r, commit, err := fetchRevision(context.Background(), rules.Source{Repository: f.Repository}, "v1.0.0", Options{GitPath: f.GitPath, Environment: environment})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if commit != f.FirstCommit {
		t.Fatal("inherited repository state affected fetch")
	}
}

// TestExpiredContextCodes preserves separate timeout and cancellation categories before any Git process starts.
func TestExpiredContextCodes(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		want := "cancelled"
		cause := context.Canceled
		if deadline {
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			want = "timed-out"
			cause = context.DeadlineExceeded
		} else {
			cancel()
		}
		_, _, err := fetchRevision(ctx, rules.Source{}, "", Options{})
		requireCode(t, err, want)
		if !errors.Is(err, cause) {
			t.Fatal("lost context cause")
		}
		cancel()
	}
}

// TestFetchUsesEnvironmentPath resolves Git from the same environment supplied to its children.
func TestFetchUsesEnvironmentPath(t *testing.T) {
	f := fixtureRepository(t)
	bin := t.TempDir()
	if err := os.Symlink(f.GitPath, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "/missing-parent-path")
	env := append(append([]string{}, f.Environment...), "PATH="+bin+":/usr/bin:/bin")
	revision, commit, err := fetchRevision(context.Background(), rules.Source{Repository: f.Repository}, "v1.0.0", Options{Environment: env})
	if err != nil {
		t.Fatal(err)
	}
	defer revision.Close()
	if commit != f.FirstCommit {
		t.Fatal("wrong fetched revision")
	}
}

// TestFetchDoesNotFallBackToParentPath rejects missing Git in an explicitly replaced environment.
func TestFetchDoesNotFallBackToParentPath(t *testing.T) {
	f := fixtureRepository(t)
	_, _, err := fetchRevision(context.Background(), rules.Source{Repository: f.Repository}, "v1.0.0", Options{Environment: []string{"PATH=/missing-custom-path"}})
	requireCode(t, err, "git-unavailable")
}

// TestFetchSkipsRelativePathEntries finds trusted Git after an excluded current-directory executable.
func TestFetchSkipsRelativePathEntries(t *testing.T) {
	f := fixtureRepository(t)
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte("#!/bin/sh\nexit 42\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(bin)
	env := append(append([]string{}, f.Environment...), "PATH=.:"+filepath.Dir(f.GitPath)+":/usr/bin:/bin")
	revision, commit, err := fetchRevision(context.Background(), rules.Source{Repository: f.Repository}, "v1.0.0", Options{Environment: env})
	if err != nil {
		t.Fatal(err)
	}
	defer revision.Close()
	if commit != f.FirstCommit {
		t.Fatal("wrong fetched revision")
	}
}

// TestFetchRevisionFailsBeforeStartingGit reports cancellation and a missing Git executable without partial state.
func TestFetchRevisionFailsBeforeStartingGit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := fetchRevision(ctx, rules.Source{}, "", Options{})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	_, _, err = fetchRevision(context.Background(), rules.Source{Repository: "https://example.invalid/rules"}, "v1.0.0", Options{GitPath: "/missing/git"})
	requireCode(t, err, "git-unavailable")
}
