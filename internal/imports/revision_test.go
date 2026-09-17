// Exercise actual Git transport, immutable revision selection, and bounded process failures.

package imports

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fabricahq/code-rules/internal/gitfixture"
	"github.com/fabricahq/code-rules/internal/rules"
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

// requireCode checks the stable failure category without matching Git's private diagnostics.
func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	var failure *Error
	if !errors.As(err, &failure) || failure.Code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
}

// TestFetchRevision exercises exact commits, lightweight/annotated tags, constraints, and missing refs.
func TestFetchRevision(t *testing.T) {
	f := fixtureRepository(t)
	options := Options{GitPath: f.GitPath, Environment: f.Environment}
	for _, tc := range []struct{ name, ref, version, want string }{
		{"commit", f.FirstCommit, "", f.FirstCommit}, {"lightweight", "v1.0.0", "", f.FirstCommit}, {"annotated", "v1.2.0", "", f.LatestCommit}, {"constraint", "", ">= 1.0.0", f.LatestCommit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			revision, err := FetchRevision(context.Background(), rules.Source{Name: "team", Repository: f.Repository, Ref: tc.ref, Version: tc.version}, options)
			if err != nil {
				t.Fatal(err)
			}
			dir := revision.directory
			if revision.Commit != tc.want {
				t.Fatalf("commit %s", revision.Commit)
			}
			if tc.version != "" && (revision.Tag != "v1.2.0" || revision.Version != "1.2.0") {
				t.Fatal("lost selected tag identity")
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
	revision, err := FetchRevision(context.Background(), rules.Source{Repository: f.Repository, Ref: "missing"}, options)
	requireCode(t, err, "ref-not-found")
	if revision != nil {
		t.Fatal("partial result")
	}
	_, err = FetchRevision(context.Background(), rules.Source{Repository: f.Repository, Version: ">= 9.0.0"}, options)
	var selection *rules.VersionSelectionError
	if !errors.As(err, &selection) || selection.Kind != rules.VersionNotFound {
		t.Fatal(err)
	}
}

// TestFetchRejectsMovedAnnotatedTag moves the tag after discovery but before the actual fetch.
func TestFetchRejectsMovedAnnotatedTag(t *testing.T) {
	f := fixtureRepository(t)
	transport := filepath.Join(f.Directory, "ssh")
	marker := filepath.Join(f.Directory, "served")
	repo := filepath.Join(f.Directory, "repository")
	script := "#!/bin/sh\nif [ -f " + gitfixture.Quote(marker) + " ]; then " + gitfixture.Quote(f.GitPath) + " -C " + gitfixture.Quote(repo) + " tag -f v1.2.0 " + f.FirstCommit + " >/dev/null; fi\ntouch " + gitfixture.Quote(marker) + "\nexec " + gitfixture.Quote(f.GitPath) + " upload-pack " + gitfixture.Quote(repo) + "\n"
	if err := os.WriteFile(transport, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	revision, err := FetchRevision(context.Background(), rules.Source{Repository: f.Repository, Version: ">= 1.0.0"}, Options{GitPath: f.GitPath, Environment: f.Environment})
	requireCode(t, err, "ref-changed")
	if revision != nil {
		t.Fatal("partial result")
	}
}

// TestFetchRejectsAmbiguousAndNonCommitTags covers ambiguous release aliases and tags of blob objects.
func TestFetchRejectsAmbiguousAndNonCommitTags(t *testing.T) {
	f := fixtureRepository(t)
	ctx := context.Background()
	options := Options{GitPath: f.GitPath, Environment: f.Environment}
	if _, err := f.Command(ctx, "tag", "1.2.0", f.FirstCommit); err != nil {
		t.Fatal(err)
	}
	_, err := FetchRevision(ctx, rules.Source{Repository: f.Repository, Version: ">= 1.0.0"}, options)
	var selection *rules.VersionSelectionError
	if !errors.As(err, &selection) || selection.Kind != rules.VersionAmbiguous {
		t.Fatal(err)
	}
	blob, err := f.Command(ctx, "rev-parse", "HEAD:README.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Command(ctx, "tag", "blob", blob); err != nil {
		t.Fatal(err)
	}
	_, err = FetchRevision(ctx, rules.Source{Repository: f.Repository, Ref: "blob"}, options)
	requireCode(t, err, "unsupported-content")
}

// TestProcessLimitsAndCancellation verifies combined output limits, deadlines, and secret-free diagnostics.
func TestProcessLimitsAndCancellation(t *testing.T) {
	for _, stream := range []string{"stdout", "stderr"} {
		t.Run(stream, func(t *testing.T) {
			script := filepath.Join(t.TempDir(), "git")
			redirect := ""
			if stream == "stderr" {
				redirect = " >&2"
			}
			if err := os.WriteFile(script, []byte("#!/bin/sh\nwhile :; do printf secret-token"+redirect+"; done\n"), 0700); err != nil {
				t.Fatal(err)
			}
			_, err := (gitRunner{script, gitEnvironment(os.Environ())}).run(context.Background(), t.TempDir(), nil, 100, nil)
			requireCode(t, err, "limit-exceeded")
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("stderr leaked")
			}
		})
	}
	script := filepath.Join(t.TempDir(), "git")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30 &\nwait\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := (gitRunner{script, gitEnvironment(os.Environ())}).run(ctx, t.TempDir(), nil, 100, nil)
	requireCode(t, err, "timed-out")
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 3*time.Second {
		t.Fatal("deadline or child-pipe cleanup failed", err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	_, err = FetchRevision(ctx, rules.Source{}, Options{})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	_, err = FetchRevision(context.Background(), rules.Source{Repository: "https://example.invalid/rules", Ref: "v1.0.0"}, Options{GitPath: "/missing/git"})
	requireCode(t, err, "git-unavailable")
}

// TestEnvironmentIsolation verifies inherited repository state cannot alter the fetched commit.
func TestEnvironmentIsolation(t *testing.T) {
	f := fixtureRepository(t)
	environment := append(append([]string{}, f.Environment...), "GIT_DIR=/missing", "GIT_WORK_TREE=/missing", "GIT_INDEX_FILE=/missing", "GIT_OBJECT_DIRECTORY=/missing", "GIT_ALTERNATE_OBJECT_DIRECTORIES=/missing", "GIT_NAMESPACE=hidden", "GIT_SHALLOW_FILE=/missing")
	r, err := FetchRevision(context.Background(), rules.Source{Repository: f.Repository, Ref: "v1.0.0"}, Options{GitPath: f.GitPath, Environment: environment})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if r.Commit != f.FirstCommit {
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
		_, err := FetchRevision(ctx, rules.Source{}, Options{})
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
	revision, err := FetchRevision(context.Background(), rules.Source{Repository: f.Repository, Ref: "v1.0.0"}, Options{Environment: env})
	if err != nil {
		t.Fatal(err)
	}
	defer revision.Close()
	if revision.Commit != f.FirstCommit {
		t.Fatal("wrong fetched revision")
	}
}

// TestFetchDoesNotFallBackToParentPath rejects missing Git in an explicitly replaced environment.
func TestFetchDoesNotFallBackToParentPath(t *testing.T) {
	f := fixtureRepository(t)
	_, err := FetchRevision(context.Background(), rules.Source{Repository: f.Repository, Ref: "v1.0.0"}, Options{Environment: []string{"PATH=/missing-custom-path"}})
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
	revision, err := FetchRevision(context.Background(), rules.Source{Repository: f.Repository, Ref: "v1.0.0"}, Options{Environment: env})
	if err != nil {
		t.Fatal(err)
	}
	defer revision.Close()
	if revision.Commit != f.FirstCommit {
		t.Fatal("wrong fetched revision")
	}
}
