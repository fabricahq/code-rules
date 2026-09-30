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

// GitRef is a validated, exact revision: a full commit SHA or a tag, without establishing that it exists. It keeps
// the text as authored for display and records, and compares by its canonical form. Outside this package, only
// ParseGitRef and UnmarshalText build a nonzero GitRef, so every nonzero one is valid. The zero value means no ref.
type GitRef struct {
	kind GitRefKind
	// canonical is the lowercase 40-digit SHA of a commit, or a tag's full name, including refs/tags/.
	canonical string
	// text is the ref as authored, such as release/5 or an uppercase SHA.
	text string
}

// IsZero reports whether the ref is the zero value, which means no ref.
func (r GitRef) IsZero() bool { return r.kind == "" }

// Kind returns GitRefCommit or GitRefTag, or "" for the zero value.
func (r GitRef) Kind() GitRefKind { return r.kind }

// Canonical returns the lowercase SHA of a commit, or a tag's full name including refs/tags/, which Git resolves
// exactly; it is empty for the zero value.
func (r GitRef) Canonical() string { return r.canonical }

// String returns the ref as authored, such as release/5 even when it equals refs/tags/release/5; it is empty for
// the zero value.
func (r GitRef) String() string { return r.text }

// Equal reports whether two refs have the same canonical form, so release/5 equals refs/tags/release/5 and a
// commit SHA equals its uppercase spelling. Two zero values are equal.
func (r GitRef) Equal(other GitRef) bool {
	return r.kind == other.kind && r.canonical == other.canonical
}

// MarshalText returns the ref as authored, or empty text for the zero value.
func (r GitRef) MarshalText() ([]byte, error) { return []byte(r.text), nil }

// UnmarshalText parses text with ParseGitRef, reading empty text as the zero value, so MarshalText's output
// round-trips. Invalid text fails with a *ValidationError and leaves r unchanged.
func (r *GitRef) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*r = GitRef{}
		return nil
	}
	parsed, err := ParseGitRef(string(text), "ref")
	if err != nil {
		return err
	}
	*r = parsed
	return nil
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
		if strings.Trim(ref, "0") == "" {
			return GitRef{}, invalid(location, "the all-zero SHA names no commit; use a full commit SHA or a tag")
		}
		return GitRef{kind: GitRefCommit, canonical: strings.ToLower(ref), text: ref}, nil
	}
	if strings.HasPrefix(ref, "refs/heads/") {
		return GitRef{}, invalid(location, ref+" is a branch; ref accepts a tag or a full commit SHA, not a branch")
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
	return GitRef{kind: GitRefTag, canonical: "refs/tags/" + tag, text: ref}, nil
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
