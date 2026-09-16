// Select a release from a bounded Git tag advertisement without running Git.

package rules

import (
	"errors"
	"fmt"
	version "github.com/hashicorp/go-version"
	"maps"
	"slices"
	"strings"
)

// VersionSelection retains the chosen spelling and advertised object for movement detection.
// Object is the tag object for annotated tags, not its peeled commit.
type VersionSelection struct {
	Tag     string `json:"tag"`
	Version string `json:"version"`
	Object  string `json:"object"`
}

// VersionSelectionKind identifies selection failures that an importer can handle separately.
type VersionSelectionKind string

const (
	VersionNotFound         VersionSelectionKind = "version-not-found"
	VersionAmbiguous        VersionSelectionKind = "ambiguous-version"
	InvalidTagAdvertisement VersionSelectionKind = "git-failed"
	TagLimitExceeded        VersionSelectionKind = "limit-exceeded"
)

// VersionSelectionError separates discovery failures from malformed user constraints.
type VersionSelectionError struct {
	Kind    VersionSelectionKind
	Problem string
}

// Error returns a human diagnostic; callers should branch on Kind, not message text.
func (e *VersionSelectionError) Error() string { return e.Problem }

const maxAdvertisedTags = 20_000

// SelectReleaseTag picks the highest matching version using native go-version precedence.
// Ordinary non-version tags are skipped. Equal-precedence aliases must identify the
// same peeled commit; otherwise selection fails. Tag spelling breaks safe ties.
// Input is ls-remote --tags output; no Git invocation or filesystem access occurs.
func SelectReleaseTag(availableGitTags string, constraint VersionConstraint) (VersionSelection, error) {
	if len(constraint.comparisons) == 0 {
		return VersionSelection{}, invalid("constraint", "version constraint must be parsed before matching")
	}
	records, err := tagRecords(availableGitTags)
	if err != nil {
		return VersionSelection{}, err
	}
	candidates := make([]VersionSelection, 0, len(records))
	var highest *version.Version
	for _, tag := range slices.SortedFunc(maps.Keys(records), compareText) {
		normalized, err := TagVersion(tag, "tag")
		if err != nil {
			var validation *ValidationError
			if errors.As(err, &validation) {
				continue
			}
			return VersionSelection{}, err
		}
		parsed, err := version.NewVersion(normalized)
		if err != nil {
			return VersionSelection{}, fmt.Errorf("parse validated release version: %v", err)
		}
		candidates = append(candidates, VersionSelection{Tag: tag, Version: normalized, Object: records[tag]})
		if constraint.comparisons.Check(parsed) && (highest == nil || parsed.GreaterThan(highest)) {
			highest = parsed
		}
	}
	if highest == nil {
		return VersionSelection{}, &VersionSelectionError{Kind: VersionNotFound, Problem: "No Git tag satisfies version constraint " + constraint.String() + "."}
	}
	var selected VersionSelection
	var commit string
	for _, candidate := range candidates {
		parsed, err := version.NewVersion(candidate.Version)
		if err != nil {
			return VersionSelection{}, fmt.Errorf("parse candidate version: %v", err)
		}
		if !parsed.Equal(highest) {
			continue
		}
		target, ok := records[candidate.Tag+"^{}"]
		if !ok {
			target = candidate.Object
		}
		if selected.Tag == "" {
			selected = candidate
			commit = target
		} else if commit != target {
			return VersionSelection{}, &VersionSelectionError{Kind: VersionAmbiguous, Problem: "Tags for the highest matching version " + highest.String() + " point to different objects; pin an exact ref or fix the tags."}
		}
	}
	return selected, nil
}

// tagRecords validates Git's bounded line protocol, retaining peeled annotated-tag entries.
func tagRecords(text string) (map[string]string, error) {
	if len(text) > 8*1024*1024 {
		return nil, &VersionSelectionError{Kind: TagLimitExceeded, Problem: "Git tag advertisement exceeds 8 MiB."}
	}
	records := make(map[string]string)
	for line := range strings.SplitSeq(text, "\n") {
		if line == "" {
			continue
		}
		object, ref, ok := strings.Cut(line, "\t")
		tag := strings.TrimPrefix(ref, "refs/tags/")
		if !ok || len(object) != 40 || !commitRef.MatchString(object) || object != strings.ToLower(object) || tag == ref || tag == "" || strings.ContainsAny(tag, "\t\r") {
			return nil, &VersionSelectionError{Kind: InvalidTagAdvertisement, Problem: "Git returned an unsupported tag advertisement."}
		}
		records[tag] = object
		if len(records) > maxAdvertisedTags {
			return nil, &VersionSelectionError{Kind: TagLimitExceeded, Problem: "The repository exceeds the version-discovery tag limit."}
		}
	}
	return records, nil
}
