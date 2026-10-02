// Package coderules parses the files and Git tags of a Code Rules library exactly as Code Rules reads them:
// library release tag names and messages, the release records in those messages, rule files, group metadata
// (_group.yaml), and the canonical group list. The format is specified at
// https://code-rules.fabricahq.com/reference/library-format/ and
// https://code-rules.fabricahq.com/reference/rule-versions/.
//
// The package only parses. It has no Git, filesystem, or network access: every function takes bytes or text the
// caller has already read, such as a tag message read with go-git, and returns typed values. Code Rules itself
// reads libraries through this package, so the two can't disagree.
//
// A catalog typically reads a library in three steps. ParseReleaseTag recognizes the release/<number> tag names
// among a repository's tags. ParseReleaseMessage splits each such annotated tag's message into its release notes
// and its release record, which lists every rule's version and what changed. ParseRule and ParseGroupMetadata then
// read the rule files and _group.yaml files at the tagged commit.
//
// Errors start with the input's location, such as a tag name, a field in a record, or a file path, and describe
// the problem for people. Callers can inspect three errors: ErrNotReleaseTag, with errors.Is, for a tag name that
// isn't a library release tag; *UnsupportedReleaseRecordError, with errors.As, for a release record in a newer
// format than this version reads; and *ValidationError, with errors.As, for the location and problem of any other
// invalid input. Every error these functions return means the input is invalid.
//
// This is a public API that follows Code Rules' own version: each Code Rules release includes it, and it may
// change before Code Rules 1.0.0.
package coderules
