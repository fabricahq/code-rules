// Package syllabus provides the tool license embedded from the repository's canonical notice.
package syllabus

import _ "embed"

//go:embed LICENSE.md
var licenseText string

// License returns the full MIT license and copyright notice included in this build.
func License() string { return licenseText }
