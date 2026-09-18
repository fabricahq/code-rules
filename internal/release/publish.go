// Stage verified assets on a GitHub draft and publish only after all remote checks pass.

package release

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v92/github"
)

// PublishOptions identifies the approved build and destination repository; credentials belong to the client.
type PublishOptions struct {
	Repository string
	Directory  string
	Commit     string
}

type publisher struct {
	client      *github.Client
	owner, repo string
}

// Publish returns the release URL after verifying and publishing the complete bundle.
// It preserves conflicting tags, notes, and assets; failed uploads leave a resumable draft.
// Matching published releases require no writes. Callers must serialize publication across versions.
func Publish(ctx context.Context, client *github.Client, options PublishOptions) (string, error) {
	parts := strings.Split(options.Repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.ContainsAny(options.Repository, "?#%\\ \t\r\n") {
		return "", fmt.Errorf("repository must be owner/name")
	}
	bundle, err := readBundle(options.Directory, options.Commit)
	if err != nil {
		return "", err
	}
	p := publisher{client: client, owner: parts[0], repo: parts[1]}
	if err := p.verifyTags(ctx, bundle.plan); err != nil {
		return "", err
	}
	if bundle.plan.Previous != "" {
		previous, _, err := client.Repositories.GetReleaseByTag(ctx, p.owner, p.repo, bundle.plan.Previous)
		if err != nil {
			return "", fmt.Errorf("read previous release %s: %w", bundle.plan.Previous, err)
		}
		if previous.Draft {
			return "", fmt.Errorf("the previous release must be published first")
		}
	}
	release, err := p.prepareDraft(ctx, bundle.plan)
	if err != nil {
		return "", err
	}
	if err := p.stageAssets(ctx, release, bundle.assets); err != nil {
		return "", err
	}
	if err := p.verifyTags(ctx, bundle.plan); err != nil {
		return "", err
	}
	if err := p.verifyCommit(ctx, bundle.plan); err != nil {
		return "", err
	}
	fresh, _, err := client.Repositories.GetRelease(ctx, p.owner, p.repo, release.ID)
	if err != nil {
		return "", fmt.Errorf("recheck release before publication: %w", err)
	}
	if !matchesPlan(fresh, bundle.plan) {
		return "", fmt.Errorf("release changed during upload; refusing publication")
	}
	if fresh.Draft {
		latest := "true"
		if bundle.plan.Prerelease {
			latest = "false"
		}
		fresh, _, err = client.Repositories.UpdateRelease(ctx, p.owner, p.repo, fresh.ID, github.UpdateReleaseRequest{Draft: new(false), MakeLatest: new(latest)})
		if err != nil {
			return "", fmt.Errorf("publish verified release: %w", err)
		}
	}
	return fresh.HTMLURL, nil
}

func matchesPlan(release *github.RepositoryRelease, plan Plan) bool {
	return release.TagName == plan.Tag && release.GetName() == plan.Tag && release.GetBody() == plan.Notes && release.Prerelease == plan.Prerelease
}
