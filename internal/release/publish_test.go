// Exercise publication ordering, resumable drafts, immutable releases, and HTTP failures.

package release

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-github/v92/github"
)

func TestPublishAndRepeat(t *testing.T) {
	f := newPublishFixture(t)
	f.paginate = true
	url, err := Publish(context.Background(), f.client, f.options())
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://example.invalid/release" || f.release.Draft || len(f.assets) != 6 || f.makeLatest != "true" {
		t.Fatal(url, f.release, len(f.assets))
	}
	if len(f.writes) != 9 || f.writes[0] != "tag" || f.writes[1] != "draft" || f.writes[8] != "publish" {
		t.Fatal(f.writes)
	}
	f.writes = nil
	if _, err := Publish(context.Background(), f.client, f.options()); err != nil {
		t.Fatal(err)
	}
	if len(f.writes) != 0 {
		t.Fatal("published retry wrote", f.writes)
	}
}

func TestResumeInterruptedUpload(t *testing.T) {
	f := newPublishFixture(t)
	f.failUpload = true
	if _, err := Publish(context.Background(), f.client, f.options()); err == nil {
		t.Fatal("expected upload error")
	}
	if f.release == nil || !f.release.Draft {
		t.Fatal("failed upload must leave draft")
	}
	f.failUpload = false
	if _, err := Publish(context.Background(), f.client, f.options()); err != nil {
		t.Fatal(err)
	}
	if f.release.Draft {
		t.Fatal("retry did not publish")
	}
}

func TestPublicationRefusals(t *testing.T) {
	for _, tc := range []struct {
		name         string
		mutate       func(*publishFixture)
		want         string
		beforeWrites bool
	}{
		{"stale tags", func(f *publishFixture) { f.tags = []string{"v0.2.0"} }, "tags changed", true},
		{"tag collision", func(f *publishFixture) { f.tag = strings.Repeat("b", 40) }, "different commit", true},
		{"tag permission", func(f *publishFixture) { f.tagStatus = 403 }, "read release tag", true},
		{"missing predecessor", func(f *publishFixture) { f.plan.Previous = "v0.0.9"; f.previousMissing = true; f.saveBundle() }, "previous release", true},
		{"new tag during upload", func(f *publishFixture) { f.newTagAfterUpload = true }, "tags changed", false},
		{"bad server digest", func(f *publishFixture) { f.badDigest = true }, "stored asset differs", false},
		{"edited notes", func(f *publishFixture) { f.edited = true }, "changed during upload", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newPublishFixture(t)
			tc.mutate(f)
			_, err := Publish(context.Background(), f.client, f.options())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v; want %s", err, tc.want)
			}
			if slices.Contains(f.writes, "publish") {
				t.Fatal("published rejected release")
			}
			if tc.beforeWrites && len(f.writes) != 0 {
				t.Fatal("wrote before refusal", f.writes)
			}
		})
	}
}

func TestRetryRefusesChangedDraft(t *testing.T) {
	f := newPublishFixture(t)
	f.failUpload = true
	if _, err := Publish(context.Background(), f.client, f.options()); err == nil {
		t.Fatal("expected failed upload")
	}
	f.release.Body = new("Manual edit")
	f.writes = nil
	if _, err := Publish(context.Background(), f.client, f.options()); err == nil || !strings.Contains(err.Error(), "differs") {
		t.Fatal(err)
	}
	if len(f.writes) != 0 {
		t.Fatal(f.writes)
	}
}

func TestStoredAssetRefusals(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*publishFixture)
	}{
		{"duplicate", func(f *publishFixture) { f.assets = append(f.assets, f.assets[0]) }},
		{"extra", func(f *publishFixture) {
			f.assets = append(f.assets, &github.ReleaseAsset{Name: new("unexpected.zip")})
		}},
		{"missing published", func(f *publishFixture) { f.assets = f.assets[1:] }},
		{"changed", func(f *publishFixture) { f.assets[0].Digest = new("wrong") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newPublishFixture(t)
			if _, err := Publish(context.Background(), f.client, f.options()); err != nil {
				t.Fatal(err)
			}
			tc.mutate(f)
			f.writes = nil
			if _, err := Publish(context.Background(), f.client, f.options()); err == nil {
				t.Fatal("accepted mismatched assets")
			}
			if len(f.writes) != 0 {
				t.Fatal("modified published release", f.writes)
			}
		})
	}
}

func TestCancelledPublication(t *testing.T) {
	f := newPublishFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Publish(ctx, f.client, f.options()); err == nil {
		t.Fatal("ignored cancellation")
	}
	if len(f.writes) != 0 {
		t.Fatal(f.writes)
	}
}

func TestPrereleaseDoesNotBecomeLatest(t *testing.T) {
	f := versionFixture(t, "0.2.0-rc.1")
	f.plan.Previous = "v0.1.0"
	f.plan.Tags = []string{"v0.1.0"}
	f.tags = []string{"v0.1.0"}
	f.saveBundle()
	if _, err := Publish(context.Background(), f.client, f.options()); err != nil {
		t.Fatal(err)
	}
	if !f.release.Prerelease || f.makeLatest != "false" {
		t.Fatal("prerelease became latest", f.release, f.makeLatest)
	}
}
