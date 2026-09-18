// Own GitHub tag checks, pagination, draft reuse, and upload verification for publication.

package release

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/google/go-github/v92/github"
)

func (p publisher) verifyTags(ctx context.Context, plan Plan) error {
	refs, _, err := p.client.Git.ListMatchingRefs(ctx, p.owner, p.repo, "tags/v")
	if err != nil {
		return fmt.Errorf("read version tags: %w", err)
	}
	names := []string{}
	for _, ref := range refs {
		name := strings.TrimPrefix(ref.GetRef(), "refs/tags/")
		if name != plan.Tag {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	if !slices.Equal(names, plan.Tags) {
		return fmt.Errorf("version tags changed since planning; review the new predecessor before retrying")
	}
	return nil
}

func (p publisher) verifyCommit(ctx context.Context, plan Plan) error {
	target, _, err := p.client.Repositories.GetCommit(ctx, p.owner, p.repo, "refs/tags/"+plan.Tag, nil)
	if err != nil {
		return fmt.Errorf("resolve release tag %s: %w", plan.Tag, err)
	}
	if target.GetSHA() != plan.Commit {
		return fmt.Errorf("release tag points to a different commit")
	}
	return nil
}

func (p publisher) prepareDraft(ctx context.Context, plan Plan) (*github.RepositoryRelease, error) {
	var selected *github.RepositoryRelease
	opts := &github.ListOptions{PerPage: 100}
	for {
		releases, response, err := p.client.Repositories.ListReleases(ctx, p.owner, p.repo, opts)
		if err != nil {
			return nil, fmt.Errorf("list releases: %w", err)
		}
		for _, release := range releases {
			if release.TagName == plan.Tag {
				if selected != nil {
					return nil, fmt.Errorf("multiple releases have tag %s", plan.Tag)
				}
				selected = release
			}
		}
		if response.NextPage == 0 {
			break
		}
		opts.Page = response.NextPage
	}
	_, _, err := p.client.Git.GetRef(ctx, p.owner, p.repo, "tags/"+plan.Tag)
	tagExists := err == nil
	var apiError *github.ErrorResponse
	if err != nil && (!errors.As(err, &apiError) || apiError.Response.StatusCode != http.StatusNotFound) {
		return nil, fmt.Errorf("read release tag: %w", err)
	}
	if tagExists {
		if err := p.verifyCommit(ctx, plan); err != nil {
			return nil, err
		}
	}
	if selected != nil && (!tagExists || !matchesPlan(selected, plan)) {
		return nil, fmt.Errorf("existing release differs from the approved request; inspect it before retrying")
	}
	if !tagExists {
		if _, _, err := p.client.Git.CreateRef(ctx, p.owner, p.repo, github.CreateRef{Ref: "refs/tags/" + plan.Tag, SHA: plan.Commit}); err != nil {
			return nil, fmt.Errorf("create release tag: %w", err)
		}
	}
	if selected == nil {
		selected, _, err = p.client.Repositories.CreateRelease(ctx, p.owner, p.repo, github.CreateReleaseRequest{
			TagName:         plan.Tag,
			TargetCommitish: new(plan.Commit),
			Name:            new(plan.Tag),
			Body:            new(plan.Notes),
			Draft:           new(true),
			Prerelease:      new(plan.Prerelease),
		})
		if err != nil {
			return nil, fmt.Errorf("create release draft: %w", err)
		}
	}
	return selected, nil
}

func (p publisher) listAssets(ctx context.Context, id int64) ([]*github.ReleaseAsset, error) {
	var result []*github.ReleaseAsset
	opts := &github.ListOptions{PerPage: 100}
	for {
		assets, response, err := p.client.Repositories.ListReleaseAssets(ctx, p.owner, p.repo, id, opts)
		if err != nil {
			return nil, fmt.Errorf("list release assets: %w", err)
		}
		result = append(result, assets...)
		if response.NextPage == 0 {
			return result, nil
		}
		opts.Page = response.NextPage
	}
}

func (p publisher) stageAssets(ctx context.Context, release *github.RepositoryRelease, files []releaseAsset) error {
	existing, err := p.listAssets(ctx, release.ID)
	if err != nil {
		return err
	}
	missing, err := missingAssets(existing, files)
	if err != nil {
		return err
	}
	if !release.Draft && len(missing) > 0 {
		return fmt.Errorf("published release is missing assets; refusing to modify it")
	}
	for _, file := range missing {
		// Use the client's configured upload origin, and upload the bytes already validated locally.
		endpoint := fmt.Sprintf("repos/%s/%s/releases/%d/assets?name=%s", p.owner, p.repo, release.ID, url.QueryEscape(file.name))
		req, err := p.client.NewUploadRequest(ctx, endpoint, bytes.NewReader(file.data), int64(len(file.data)), "application/octet-stream")
		if err != nil {
			return fmt.Errorf("prepare upload %s: %w", file.name, err)
		}
		if _, err := p.client.Do(req, nil); err != nil {
			return fmt.Errorf("upload %s: %w", file.name, err)
		}
	}
	stored, err := p.listAssets(ctx, release.ID)
	if err != nil {
		return err
	}
	missing, err = missingAssets(stored, files)
	if err != nil {
		return err
	}
	if len(missing) != 0 {
		return fmt.Errorf("release asset set is incomplete")
	}
	return nil
}

// missingAssets refuses unexpected, duplicate, or mismatched assets before returning missing uploads.
func missingAssets(remote []*github.ReleaseAsset, files []releaseAsset) ([]releaseAsset, error) {
	expected := make(map[string]releaseAsset, len(files))
	for _, file := range files {
		expected[file.name] = file
	}
	seen := make(map[string]bool, len(remote))
	for _, asset := range remote {
		name := asset.GetName()
		file, ok := expected[name]
		if !ok || seen[name] {
			return nil, fmt.Errorf("unexpected or duplicate release asset: %s", name)
		}
		if asset.GetState() != "uploaded" || asset.GetSize() != len(file.data) || asset.GetDigest() != file.digest {
			return nil, fmt.Errorf("stored asset differs: %s", name)
		}
		seen[name] = true
	}
	var missing []releaseAsset
	for _, file := range files {
		if !seen[file.name] {
			missing = append(missing, file)
		}
	}
	return missing, nil
}
