// Classify exact Git refs and validate strict semantic-version tags without Git access.

package rules

import (
	"regexp"
	"strconv"
	"strings"
)

// GitRefKind distinguishes a complete commit SHA from an exact tag name.
type GitRefKind string

const (
	GitRefCommit GitRefKind = "commit"
	GitRefTag    GitRefKind = "tag"
)

// GitRef describes an exact revision, without establishing that it exists.
type GitRef struct {
	Kind GitRefKind `json:"kind"`
	// SHA is a lowercase 40-digit commit ID; populated only for GitRefCommit.
	SHA string `json:"sha,omitempty"`
	// Name includes refs/tags/; populated only for GitRefTag. Branches are never inferred.
	Name string `json:"name,omitempty"`
}

var (
	commitRef      = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)
	shortCommitRef = regexp.MustCompile(`^[a-fA-F0-9]{4,39}$`)
	unsafeRef      = regexp.MustCompile(`[\x00-\x20\x7f~^:?*\[\\]|\.\.|@\{`)
)

// ParseGitRef accepts a full commit SHA or an exact tag, optionally prefixed with
// refs/tags/. A bare name such as main means a tag, never a branch. Abbreviated
// commit-shaped tags require the explicit tag prefix. Errors return a zero GitRef.
// Location is the caller's complete diagnostic field path, such as sources.team.ref.
func ParseGitRef(ref, location string) (GitRef, error) {
	if strings.TrimFunc(ref, jsWhitespace) == "" {
		return GitRef{}, invalid(location, "expected nonempty text")
	}
	if commitRef.MatchString(ref) {
		return GitRef{Kind: GitRefCommit, SHA: strings.ToLower(ref)}, nil
	}
	tag := strings.TrimPrefix(ref, "refs/tags/")
	if unsafeRef.MatchString(tag) || tag == "@" || strings.HasPrefix(tag, "-") ||
		(strings.HasPrefix(ref, "refs/") && !strings.HasPrefix(ref, "refs/tags/")) {
		return GitRef{}, invalid(location, "expected a full commit SHA or exact tag name")
	}
	for _, part := range strings.Split(tag, "/") {
		if part == "" || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") || strings.HasSuffix(part, ".") {
			return GitRef{}, invalid(location, "expected a full commit SHA or exact tag name")
		}
	}
	if shortCommitRef.MatchString(ref) {
		return GitRef{}, invalid(location, "abbreviated commits are unsupported; use a full SHA or refs/tags/<name>")
	}
	return GitRef{Kind: GitRefTag, Name: "refs/tags/" + tag}, nil
}

const (
	versionNumber     = `(?:0|[1-9][0-9]*)`
	versionPrerelease = `(?:0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)`
	// The pinned npm parser bounds core components to JavaScript's safe integer limit.
	maxVersionComponent = 9007199254740991
)

var versionTag = regexp.MustCompile(`^(` + versionNumber + `)\.(` + versionNumber + `)\.(` + versionNumber + `)(?:-` + versionPrerelease + `(?:\.` + versionPrerelease + `)*)?(?:\+[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*)?$`)

// TagVersion validates a complete SemVer tag with an optional lowercase v.
// It retains prerelease and build metadata and never coerces partial versions.
// Invalid input returns an empty string and a ValidationError at the caller's location.
func TagVersion(tag, location string) (string, error) {
	version := strings.TrimPrefix(tag, "v")
	// All accepted characters are ASCII, so this byte limit equals npm's string limit.
	if len(version) > 256 {
		return "", invalid(location, "version must be at most 256 characters, excluding the optional v prefix")
	}
	parts := versionTag.FindStringSubmatch(version)
	if parts == nil {
		return "", invalid(location, "expected a complete semantic version tag, such as v1.2.3 or 1.2.3-beta.1+build.5")
	}
	for _, part := range parts[1:4] {
		number, err := strconv.ParseUint(part, 10, 64)
		if err != nil || number > maxVersionComponent {
			return "", invalid(location, "major, minor, and patch must each be at most 9007199254740991")
		}
	}
	return version, nil
}
