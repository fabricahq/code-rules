// Retain original library bytes with the revision and selection that produced them.

package library

import "github.com/fabricahq/code-rules/internal/rules"

// Snapshot owns original library bytes and the identity recorded when they were imported.
// A verified digest establishes integrity against the record, not authenticity of its origin.
// Files excludes _source.json. Binary content remains bytes; required text is checked by the loader.
type Snapshot struct {
	Repository string `json:"repository"`
	Ref        string `json:"ref,omitempty"`
	Version    string `json:"version,omitempty"`
	// Tag and ResolvedVersion are present only for a version constraint, not an exact ref.
	Tag             string `json:"resolvedTag,omitempty"`
	ResolvedVersion string `json:"resolvedVersion,omitempty"`
	Commit          string `json:"resolvedCommit"`
	// Groups lists the resolved IDs; Selection retains the requested IDs or wildcard.
	Groups    []string             `json:"groups"`
	Selection rules.GroupSelection `json:"groupSelection"`
	Files     map[string][]byte    `json:"files"`
}
