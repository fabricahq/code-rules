// Reject mismatched or incomplete local bundles before making GitHub requests.

package release

import (
	"context"
	"strings"
	"testing"
)

func TestBundleRefusals(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*publishFixture)
	}{
		{"candidate", func(f *publishFixture) { f.manifest["candidate"] = true }},
		{"missing candidate flag", func(f *publishFixture) { delete(f.manifest, "candidate") }},
		{"dirty", func(f *publishFixture) { f.manifest["sourceDirty"] = true }},
		{"missing dirty flag", func(f *publishFixture) { delete(f.manifest, "sourceDirty") }},
		{"wrong source", func(f *publishFixture) { f.manifest["sourceRevision"] = strings.Repeat("b", 40) }},
		{"wrong version", func(f *publishFixture) { f.manifest["version"] = "1.0.0" }},
		{"unlicensed", func(f *publishFixture) { f.manifest["license"] = "UNLICENSED" }},
		{"missing targets", func(f *publishFixture) { f.manifest["artifacts"] = []any{} }},
		{"nil tag snapshot", func(f *publishFixture) { f.plan.Tags = nil }},
		{"empty notes", func(f *publishFixture) { f.plan.Notes = " \n" }},
		{"wrong tag", func(f *publishFixture) { f.plan.Tag = "v0.2.0" }},
		{"unsafe version", func(f *publishFixture) { f.plan.Version = "../../arbitrary" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newPublishFixture(t)
			tc.mutate(f)
			f.saveBundle()
			if _, err := Publish(context.Background(), f.client, f.options()); err == nil {
				t.Fatal("accepted bad bundle")
			}
			if f.requests != 0 {
				t.Fatal("contacted GitHub before validating bundle")
			}
		})
	}
	for _, name := range []string{"code-rules_0.1.0_linux_amd64.tar.gz", "SHA256SUMS", "manifest.json", "release-plan.json"} {
		t.Run("corrupt "+name, func(t *testing.T) {
			f := newPublishFixture(t)
			f.writeFile(name, []byte("corrupt"))
			if _, err := Publish(context.Background(), f.client, f.options()); err == nil {
				t.Fatal("accepted corrupt file")
			}
			if f.requests != 0 {
				t.Fatal("contacted GitHub before validating bundle")
			}
		})
	}
}
