// Identify the explicit unfinished-rule marker shared by project and library authoring.

package rules

import "bytes"

// DraftMarker marks an authored rule that must be completed before consumption.
const DraftMarker = "<!-- code-rules:draft -->"

// HasDraftMarker reports whether a rule document still carries its unfinished marker.
func HasDraftMarker(document []byte) bool {
	return bytes.Contains(document, []byte(DraftMarker))
}
