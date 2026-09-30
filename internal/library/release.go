// Publish a library release: tag the checked-out commit of the default branch, push the tag, and announce it.

package library

import (
	"context"
	"strconv"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
)

// ReleaseRequest selects what code-rules library release does.
type ReleaseRequest struct {
	Options
	// DryRun shows what would be published without creating a tag or a GitHub Release page.
	DryRun bool
	// NoGitHubRelease publishes the tag without a GitHub Release page.
	NoGitHubRelease bool
}

// ReleaseResult describes a library release that was published, finished, or previewed, or that there was
// nothing to publish.
type ReleaseResult struct {
	// Repository is where the library release is pushed, the upstream remote's repository, without credentials.
	Repository string `json:"repository"`
	Remote     string `json:"remote"`
	Branch     string `json:"branch"`
	Commit     string `json:"commit"`
	// GitHubRepository is OWNER/REPO when the repository is on GitHub.com, and empty for other hosts.
	GitHubRepository string `json:"githubRepository,omitempty"`
	DryRun           bool   `json:"dryRun"`
	// Release is the library release's number, and 0 when there's nothing to publish.
	Release int `json:"release"`
	// Tag is the release tag, such as release/4; it's empty when there's nothing to publish.
	Tag string `json:"tag,omitempty"`
	// Published reports that the remote already had the tag, on this commit, before this run.
	Published bool `json:"published"`
	// Rules lists each rule the library release changes, adds, or retires, sorted by ID.
	Rules        []PendingRule `json:"rules"`
	LibraryFiles []string      `json:"libraryFiles"`
	Notes        string        `json:"notes,omitempty"`
	// TagCreated reports that this run pushed the tag to the remote, creating it first unless an interrupted
	// run left it in the clone.
	TagCreated bool `json:"tagCreated"`
	// GitHubRelease is nil when no GitHub Release page was created or found: on a dry run, with
	// NoGitHubRelease, on other hosts, and when there's nothing to publish.
	GitHubRelease *GitHubReleasePage `json:"githubRelease,omitempty"`
	Warnings      []string           `json:"warnings"`
}

// GitHubReleasePage is a library release's announcement on GitHub.com.
type GitHubReleasePage struct {
	// Created reports that this run created the page; false means it already existed.
	Created bool   `json:"created"`
	URL     string `json:"url,omitempty"`
}

// Release publishes the pending change notes and library-wide changes as the next library release: an
// annotated release/<number> tag on the checked-out commit, pushed to the upstream remote, and, for a
// repository on GitHub.com, a GitHub Release page created with gh. It changes no files in the library.
//
// It fetches first, and refuses unless the checked-out branch is the remote's default branch, matches the
// remote exactly, and has no uncommitted library changes, and library check passes. When a release tag
// already tags the commit, it creates only what's missing, such as the GitHub Release page after gh failed.
// A dry run fetches, checks, and describes the library release without creating anything.
func Release(ctx context.Context, request ReleaseRequest) (ReleaseResult, error) {
	root, err := openLibrary(ctx, request.Options, false)
	if err != nil {
		return ReleaseResult{}, err
	}
	defer root.Close()
	if err = filetxn.RequireIdle(root); err != nil {
		return ReleaseResult{}, err
	}
	git, err := openLibraryGit(ctx, root.Name(), request.Git)
	if err != nil {
		return ReleaseResult{}, err
	}
	if git == nil {
		return ReleaseResult{}, failure("not-a-repository", "a library release is a Git tag, so the library must be the root of a Git repository.", nil)
	}
	if err = git.requireFullHistory(ctx); err != nil {
		return ReleaseResult{}, err
	}
	branch, err := git.upstream(ctx)
	if err != nil {
		return ReleaseResult{}, err
	}
	result := ReleaseResult{Repository: displayRepository(branch.pushURL), Remote: branch.remote, Branch: branch.branch, GitHubRepository: gitHubRepository(branch.pushURL), DryRun: request.DryRun, Rules: []PendingRule{}, LibraryFiles: []string{}}
	var gh *gitHubCLI
	if !request.DryRun {
		if err = git.requireCommitterIdentity(ctx); err != nil {
			return ReleaseResult{}, err
		}
		if result.GitHubRepository != "" && !request.NoGitHubRelease {
			cli, err := findGitHubCLI(ctx, request.Git.Environment, root.Name())
			if err != nil {
				return ReleaseResult{}, err
			}
			gh = &cli
		}
	}
	remote, err := git.readRemote(ctx, branch)
	if err != nil {
		return ReleaseResult{}, err
	}
	if result.Commit, err = git.headCommit(ctx); err != nil {
		return ReleaseResult{}, err
	}
	unpushed, err := git.syncReleaseTags(ctx, branch, remote, result.Commit)
	if err != nil {
		return ReleaseResult{}, err
	}
	if err = git.requireCurrent(ctx, branch, remote.head, result.Commit); err != nil {
		return ReleaseResult{}, err
	}
	checked, err := checkLibrary(ctx, root, git)
	if err != nil {
		return ReleaseResult{}, err
	}
	result.Warnings = checked.result.Warnings
	committed, err := git.requireCommitted(ctx, result.Commit, checked.input)
	if err != nil {
		return ReleaseResult{}, err
	}
	var object string
	if latest := checked.changes.history.latest; latest != nil && latest.commit == result.Commit {
		describe(&result, latest.record, latest.notes)
		result.Published = unpushed != latest.number
		object = latest.object
	} else {
		if checked.plan.release != remote.latest()+1 {
			return ReleaseResult{}, failure("release-tag-mismatch", branch.remote+"'s newest library release is release/"+strconv.Itoa(remote.latest())+", but "+branch.branch+"'s history reaches only release/"+strconv.Itoa(checked.plan.release-1)+". Release tags belong on the default branch; don't create them by hand.", nil)
		}
		released := map[string]string{}
		if latest != nil {
			if released, err = git.treeFiles(ctx, latest.object, libraryPaths(checked.input.license)); err != nil {
				return ReleaseResult{}, err
			}
		}
		libraryFiles := changedLibraryFiles(committed, released, rules.LicensePaths(checked.input.license))
		if checked.plan.empty() && len(libraryFiles) == 0 {
			return result, nil
		}
		record := checked.plan.record(libraryFiles)
		notes := renderReleaseNotes(record)
		message, err := releaseMessage(notes, record)
		if err != nil {
			return ReleaseResult{}, err
		}
		describe(&result, record, notes)
		if request.DryRun {
			return result, nil
		}
		if err = requireLibraryUnchanged(ctx, root, checked.input); err != nil {
			return ReleaseResult{}, err
		}
		if err = filetxn.RequireIdle(root); err != nil {
			return ReleaseResult{}, err
		}
		if object, err = git.createTag(ctx, result.Tag, result.Commit, message); err != nil {
			return ReleaseResult{}, err
		}
		unpushed = record.Release
	}
	if request.DryRun {
		return result, nil
	}
	if unpushed == result.Release {
		if err = git.pushTag(ctx, branch, result.Tag, object); err != nil {
			return ReleaseResult{}, err
		}
		result.TagCreated = true
	}
	if gh != nil {
		page, err := publishReleasePage(ctx, *gh, result.GitHubRepository, result.Tag, result.Notes)
		if err != nil {
			return ReleaseResult{}, err
		}
		result.GitHubRelease = &page
	}
	return result, nil
}

// describe fills in what a library release publishes.
func describe(result *ReleaseResult, record rules.ReleaseRecord, notes string) {
	result.Release = record.Release
	result.Tag = "release/" + strconv.Itoa(record.Release)
	result.Rules = releaseRules(record)
	result.LibraryFiles = record.LibraryFiles
	result.Notes = notes
}

// publishReleasePage finds tag's GitHub Release page, creating it with notes as its body when it's missing.
func publishReleasePage(ctx context.Context, gh gitHubCLI, repository, tag, notes string) (GitHubReleasePage, error) {
	url, exists, err := gh.releasePage(ctx, repository, tag)
	if err != nil {
		return GitHubReleasePage{}, err
	}
	if exists {
		return GitHubReleasePage{URL: url}, nil
	}
	url, err = gh.createReleasePage(ctx, repository, tag, notes)
	if err != nil {
		return GitHubReleasePage{}, err
	}
	return GitHubReleasePage{Created: true, URL: url}, nil
}
