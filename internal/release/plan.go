// Package release validates committed release requests before CI builds or publishes them.
package release

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
	version "github.com/hashicorp/go-version"
)

// Plan binds one new notes file to the exact commit CI must build. An empty Tag means no release was requested.
type Plan struct {
	// Tags captures the observed version-tag names, excluding this release, for publication freshness checks.
	Tags       []string `json:"tags"`
	Tag        string   `json:"tag"`
	Version    string   `json:"version"`
	Commit     string   `json:"commit"`
	Previous   string   `json:"previous"`
	Notes      string   `json:"notes"`
	Prerelease bool     `json:"prerelease"`
}

// Read accepts one committed releases/v<semver>.md request; tagged requests are immutable.
// Untagged requests can be corrected or withdrawn after a failed build.
// The first version must be v0.1.0; later versions must advance all existing version tags.
// An existing tag is accepted only at head, so retries never move a published version.
func Read(ctx context.Context, source, base, head string) (Plan, error) {
	git := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = source
		data, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(data)))
		}
		return string(data), nil
	}
	resolve := func(ref string) (string, error) {
		value, err := git("rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
		return strings.TrimSpace(value), err
	}
	base, err := resolve(base)
	if err != nil {
		return Plan{}, err
	}
	head, err = resolve(head)
	if err != nil {
		return Plan{}, err
	}
	if _, err = git("merge-base", "--is-ancestor", base, head); err != nil {
		return Plan{}, fmt.Errorf("release base must be an ancestor of head: %w", err)
	}
	changed, err := git("diff", "--name-status", "-z", "--no-renames", base, head, "--", "releases/")
	if err != nil {
		return Plan{}, err
	}
	fields := strings.Split(strings.TrimSuffix(changed, "\x00"), "\x00")
	var file string
	for i := 0; i+1 < len(fields); i += 2 {
		name := fields[i+1]
		if path.Dir(name) != "releases" || !strings.HasPrefix(path.Base(name), "v") || !strings.HasSuffix(name, ".md") {
			continue
		}
		if fields[i] != "A" {
			tag := strings.TrimSuffix(path.Base(name), ".md")
			existing, err := git("tag", "--list", tag)
			if err != nil {
				return Plan{}, err
			}
			if strings.TrimSpace(existing) != "" {
				return Plan{}, fmt.Errorf("tagged release notes are immutable: %s", name)
			}
			if fields[i] == "D" {
				continue
			}
			if fields[i] != "M" {
				return Plan{}, fmt.Errorf("unsupported release note change: %s", name)
			}
		}
		if file != "" {
			return Plan{}, fmt.Errorf("submit one release notes file per change")
		}
		file = name
	}
	if file == "" {
		return Plan{}, nil
	}
	tag := strings.TrimSuffix(path.Base(file), ".md")
	v, err := rules.TagVersion(tag, file)
	if err != nil {
		return Plan{}, err
	}
	if strings.Contains(v, "+") {
		return Plan{}, fmt.Errorf("release tags omit build metadata; the manifest records the source commit")
	}
	current, err := version.NewSemver(v)
	if err != nil {
		return Plan{}, err
	}
	notes, err := git("show", head+":"+file)
	if err != nil {
		return Plan{}, err
	}
	if strings.TrimSpace(notes) == "" {
		return Plan{}, fmt.Errorf("release notes must not be empty")
	}
	tags, err := git("tag", "--list", "v*")
	if err != nil {
		return Plan{}, err
	}
	observed := []string{}
	var previous string
	var latest *version.Version
	for _, existing := range strings.Fields(tags) {
		normalized, err := rules.TagVersion(existing, "tag")
		if err != nil {
			continue
		}
		other, err := version.NewSemver(normalized)
		if err != nil {
			return Plan{}, err
		}
		if existing == tag {
			target, err := resolve("refs/tags/" + tag)
			if err != nil {
				return Plan{}, err
			}
			if target != head {
				return Plan{}, fmt.Errorf("tag %s already points to another commit", tag)
			}
			continue
		}
		observed = append(observed, existing)
		if !current.GreaterThan(other) {
			return Plan{}, fmt.Errorf("%s must be newer than existing tag %s", tag, existing)
		}
		if latest == nil || other.GreaterThan(latest) {
			previous, latest = existing, other
		}
	}
	slices.Sort(observed)
	if previous == "" && tag != "v0.1.0" {
		return Plan{}, fmt.Errorf("the first release must be v0.1.0")
	}
	if previous != "" {
		if _, err := git("merge-base", "--is-ancestor", "refs/tags/"+previous, head); err != nil {
			return Plan{}, fmt.Errorf("previous release %s must be an ancestor: %w", previous, err)
		}
	}
	return Plan{Tags: observed, Tag: tag, Version: v, Commit: head, Previous: previous, Notes: notes, Prerelease: current.Prerelease() != ""}, nil
}
