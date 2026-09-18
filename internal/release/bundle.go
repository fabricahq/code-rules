// Validate the complete publication bundle before contacting GitHub.

package release

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/distribution"
	"github.com/fabricahq/code-rules/internal/rules"
)

type releaseAsset struct {
	name   string
	data   []byte
	digest string
}

type releaseBundle struct {
	plan   Plan
	assets []releaseAsset
}

// readBundle keeps validated bytes in memory so uploads cannot pick up subsequent file edits.
func readBundle(directory, commit string) (releaseBundle, error) {
	var bundle releaseBundle
	if err := readJSON(filepath.Join(directory, "release-plan.json"), &bundle.plan); err != nil {
		return bundle, err
	}
	var manifest struct {
		distribution.Manifest
		Candidate   *bool `json:"candidate"`
		SourceDirty *bool `json:"sourceDirty"`
	}
	if err := readJSON(filepath.Join(directory, "manifest.json"), &manifest); err != nil {
		return bundle, err
	}
	plan := bundle.plan
	v, err := rules.TagVersion(plan.Version, "release version")
	if err != nil || v != plan.Version || plan.Tag != "v"+v ||
		commit == "" || plan.Commit != commit || manifest.SourceRevision != commit || plan.Version != manifest.Version ||
		manifest.Candidate == nil || *manifest.Candidate || manifest.SourceDirty == nil || *manifest.SourceDirty ||
		plan.Tags == nil || strings.TrimSpace(plan.Notes) == "" || manifest.License != "SEE LICENSE.md" {
		return bundle, fmt.Errorf("release plan and clean, licensed build must match the approved commit")
	}
	targets := []string{}
	for _, artifact := range manifest.Artifacts {
		targets = append(targets, artifact.Target)
	}
	slices.Sort(targets)
	if !slices.Equal(targets, []string{"darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64"}) {
		return bundle, fmt.Errorf("expected exactly four native targets")
	}
	var sums strings.Builder
	for _, artifact := range manifest.Artifacts {
		name := "code-rules_" + plan.Version + "_" + strings.ReplaceAll(artifact.Target, "/", "_") + ".tar.gz"
		if artifact.File != name {
			return bundle, fmt.Errorf("unexpected archive filename %q", artifact.File)
		}
		asset, err := readAsset(directory, name)
		if err != nil {
			return bundle, err
		}
		if asset.digest != "sha256:"+artifact.SHA256 || int64(len(asset.data)) != artifact.Bytes {
			return bundle, fmt.Errorf("archive checksum or size mismatch: %s", name)
		}
		bundle.assets = append(bundle.assets, asset)
		fmt.Fprintf(&sums, "%s  %s\n", artifact.SHA256, artifact.File)
	}
	for _, name := range []string{"manifest.json", "SHA256SUMS"} {
		asset, err := readAsset(directory, name)
		if err != nil {
			return bundle, err
		}
		if name == "SHA256SUMS" && string(asset.data) != sums.String() {
			return bundle, fmt.Errorf("SHA256SUMS does not match the manifest")
		}
		bundle.assets = append(bundle.assets, asset)
	}
	return bundle, nil
}

func readAsset(directory, name string) (releaseAsset, error) {
	data, err := os.ReadFile(filepath.Join(directory, name))
	if err != nil {
		return releaseAsset{}, fmt.Errorf("read asset %s: %w", name, err)
	}
	return releaseAsset{name: name, data: data, digest: fmt.Sprintf("sha256:%x", sha256.Sum256(data))}, nil
}

func readJSON(file string, value any) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.Base(file), err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("decode %s: %w", filepath.Base(file), err)
	}
	return nil
}
