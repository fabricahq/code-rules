// Package rules parses and validates the formats only Code Rules itself reads, such as project configuration,
// change notes, library license declarations, Git references, and Markdown links, and renders the rules and groups
// that authors create, without accessing the filesystem. Rule files, group metadata, and release records belong to
// the public coderules package, which other tools also use.
package rules
