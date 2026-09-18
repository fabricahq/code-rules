// Simulate GitHub over HTTP so publication tests exercise the real SDK and wire requests.

package release

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v92/github"
)

type publishFixture struct {
	t                 *testing.T
	directory         string
	plan              Plan
	manifest          map[string]any
	server            *httptest.Server
	client            *github.Client
	tag               string
	tags              []string
	release           *github.RepositoryRelease
	assets            []*github.ReleaseAsset
	writes            []string
	requests          int
	failUpload        bool
	badDigest         bool
	edited            bool
	newTagAfterUpload bool
	previousMissing   bool
	tagStatus         int
	paginate          bool
	makeLatest        string
}

func newPublishFixture(t *testing.T) *publishFixture {
	t.Helper()
	return versionFixture(t, "0.1.0")
}

func versionFixture(t *testing.T, version string) *publishFixture {
	t.Helper()
	f := &publishFixture{t: t, directory: t.TempDir(), plan: Plan{Tag: "v" + version, Version: version, Prerelease: strings.Contains(version, "-"), Commit: strings.Repeat("a", 40), Tags: []string{}, Notes: "Approved notes\n"}}
	artifacts := []map[string]any{}
	var sums strings.Builder
	for _, target := range []string{"darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64"} {
		data := []byte(target)
		name := "code-rules_" + version + "_" + strings.ReplaceAll(target, "/", "_") + ".tar.gz"
		f.writeFile(name, data)
		digest := fmt.Sprintf("%x", sha256.Sum256(data))
		artifacts = append(artifacts, map[string]any{"file": name, "target": target, "bytes": len(data), "sha256": digest})
		fmt.Fprintf(&sums, "%s  %s\n", digest, name)
	}
	f.manifest = map[string]any{"version": f.plan.Version, "sourceRevision": f.plan.Commit, "candidate": false, "sourceDirty": false, "license": "SEE LICENSE.md", "artifacts": artifacts}
	f.writeFile("SHA256SUMS", []byte(sums.String()))
	f.saveBundle()
	f.server = httptest.NewServer(http.HandlerFunc(f.serveHTTP))
	t.Cleanup(f.server.Close)
	base := f.server.URL + "/"
	client, err := github.NewClient(github.WithURLs(&base, &base), github.WithAuthToken("fixture-token"), github.WithTimeout(time.Second*5))
	if err != nil {
		t.Fatal(err)
	}
	f.client = client
	return f
}

func (f *publishFixture) options() PublishOptions {
	return PublishOptions{Repository: "test/test", Directory: f.directory, Commit: f.plan.Commit}
}

func (f *publishFixture) writeFile(name string, data []byte) {
	f.t.Helper()
	if err := os.WriteFile(filepath.Join(f.directory, name), data, 0600); err != nil {
		f.t.Fatal(err)
	}
}

func (f *publishFixture) saveBundle() {
	f.t.Helper()
	for name, value := range map[string]any{"release-plan.json": f.plan, "manifest.json": f.manifest} {
		data, err := json.Marshal(value)
		if err != nil {
			f.t.Fatal(err)
		}
		f.writeFile(name, data)
	}
}

func (f *publishFixture) serveHTTP(w http.ResponseWriter, r *http.Request) {
	f.requests++
	if r.Header.Get("Authorization") != "Bearer fixture-token" {
		f.t.Error("missing API authentication")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	endpoint := strings.TrimPrefix(r.URL.Path, "/repos/test/test/")
	var result any
	switch {
	case r.Method == "GET" && endpoint == "git/matching-refs/tags/v":
		refs := []map[string]string{}
		names := append([]string{}, f.tags...)
		if f.tag != "" {
			names = append(names, f.plan.Tag)
		}
		if f.newTagAfterUpload && len(f.assets) == 6 {
			names = append(names, "v0.2.0")
		}
		for _, name := range names {
			refs = append(refs, map[string]string{"ref": "refs/tags/" + name})
		}
		result = refs
	case r.Method == "GET" && endpoint == "git/ref/tags/"+f.plan.Tag:
		if f.tagStatus != 0 {
			w.WriteHeader(f.tagStatus)
			result = map[string]string{"message": "fixture tag failure"}
		} else if f.tag == "" {
			w.WriteHeader(http.StatusNotFound)
			result = map[string]string{"message": "Not Found"}
		} else {
			result = map[string]any{"ref": "refs/tags/" + f.plan.Tag, "object": map[string]string{"sha": f.tag}}
		}
	case r.Method == "GET" && endpoint == "commits/refs/tags/"+f.plan.Tag:
		result = map[string]string{"sha": f.tag}
	case r.Method == "GET" && strings.HasPrefix(endpoint, "releases/tags/"):
		if f.previousMissing {
			w.WriteHeader(http.StatusNotFound)
			result = map[string]string{"message": "Not Found"}
		} else {
			result = github.RepositoryRelease{Draft: false}
		}
	case r.Method == "GET" && endpoint == "releases":
		releases := []*github.RepositoryRelease{}
		if f.release != nil {
			releases = append(releases, f.release)
		}
		if f.paginate && r.URL.Query().Get("page") != "2" {
			w.Header().Set("Link", fmt.Sprintf("<%s%s?page=2>; rel=\"next\"", f.server.URL, r.URL.Path))
			releases = []*github.RepositoryRelease{{TagName: "unrelated"}}
		}
		result = releases
	case r.Method == "GET" && endpoint == "releases/7/assets":
		assets := append([]*github.ReleaseAsset{}, f.assets...)
		if f.paginate && len(assets) > 1 {
			if r.URL.Query().Get("page") != "2" {
				w.Header().Set("Link", fmt.Sprintf("<%s%s?page=2>; rel=\"next\"", f.server.URL, r.URL.Path))
				assets = assets[:1]
			} else {
				assets = assets[1:]
			}
		}
		result = assets
	case r.Method == "GET" && endpoint == "releases/7":
		copy := *f.release
		if f.edited {
			copy.Body = new("Manual edit")
		}
		result = copy
	case r.Method == "POST" && endpoint == "git/refs":
		var request github.CreateRef
		if !f.decode(w, r, &request) {
			return
		}
		if request.Ref != "refs/tags/"+f.plan.Tag || request.SHA != f.plan.Commit {
			f.t.Error("unexpected tag request", request)
		}
		f.writes = append(f.writes, "tag")
		f.tag = request.SHA
		result = map[string]string{"ref": request.Ref}
	case r.Method == "POST" && endpoint == "releases":
		var request github.CreateReleaseRequest
		if !f.decode(w, r, &request) {
			return
		}
		if request.Draft == nil || !*request.Draft || request.GetTargetCommitish() != f.plan.Commit {
			f.t.Error("release must start as draft at approved commit")
		}
		f.writes = append(f.writes, "draft")
		f.release = &github.RepositoryRelease{ID: 7, TagName: request.TagName, Name: request.Name, Body: request.Body, Draft: request.GetDraft(), Prerelease: request.GetPrerelease(), HTMLURL: "https://example.invalid/release"}
		result = f.release
	case r.Method == "POST" && endpoint == "releases/7/assets":
		if f.failUpload {
			w.WriteHeader(http.StatusInternalServerError)
			result = map[string]string{"message": "upload failed"}
			break
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			f.t.Error(err)
			w.WriteHeader(400)
			return
		}
		if r.ContentLength != int64(len(data)) || r.Header.Get("Content-Type") != "application/octet-stream" {
			f.t.Error("incorrect binary upload headers")
		}
		name := r.URL.Query().Get("name")
		f.writes = append(f.writes, "upload:"+name)
		digest := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
		if f.badDigest {
			digest = "wrong"
		}
		asset := &github.ReleaseAsset{Name: new(name), Size: new(len(data)), State: new("uploaded"), Digest: new(digest)}
		f.assets = append(f.assets, asset)
		result = asset
	case r.Method == "PATCH" && endpoint == "releases/7":
		var request github.UpdateReleaseRequest
		if !f.decode(w, r, &request) {
			return
		}
		if request.Draft == nil || *request.Draft || len(f.assets) != 6 {
			f.t.Error("published before all six assets were uploaded")
		}
		f.makeLatest = request.GetMakeLatest()
		f.writes = append(f.writes, "publish")
		f.release.Draft = false
		result = f.release
	default:
		f.t.Errorf("unexpected API request: %s %s", r.Method, r.URL)
		w.WriteHeader(500)
		result = map[string]string{"message": "unexpected request"}
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		f.t.Error(err)
	}
}

func (f *publishFixture) decode(w http.ResponseWriter, r *http.Request, value any) bool {
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		f.t.Error(err)
		w.WriteHeader(400)
		return false
	}
	return true
}
