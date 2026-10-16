// Identify the explicit unfinished-rule marker shared by project and library authoring.

package rules

import "bytes"

// DraftMarker marks an authored rule that must be completed before consumption.
const DraftMarker = "<!-- syllabus:draft -->"

// legacyDraftMarker is DraftMarker as authoring wrote it before Code Rules was renamed Syllabus.
const legacyDraftMarker = "<!-- code-rules:draft -->"

// HasDraftMarker reports whether a rule document still carries its unfinished marker, in either spelling.
func HasDraftMarker(document []byte) bool {
	return bytes.Contains(document, []byte(DraftMarker)) || bytes.Contains(document, []byte(legacyDraftMarker))
}
