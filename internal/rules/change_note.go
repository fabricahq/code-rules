// Parse a library change note from changes/ without accessing files.

package rules

import (
	"encoding/json"
	"maps"
	"slices"

	"github.com/fabricahq/code-rules/coderules"
	"github.com/fabricahq/code-rules/internal/decode"
	"github.com/fabricahq/code-rules/internal/librarypath"
)

// ChangeNote records one change to one or more rules for the next library release.
type ChangeNote struct {
	// Summary has no surrounding whitespace.
	Summary string `json:"summary"`
	// Rules maps library rule IDs, such as practices/testing/verify-retry-limits, to their change. It is never empty.
	Rules map[string]NoteChange `json:"rules"`
}

// NoteChange is one rule's change in a note. ReplacedBy is set only for a retired rule that has a replacement.
type NoteChange struct {
	Change     coderules.Change `json:"change"`
	ReplacedBy string           `json:"replacedBy,omitempty"`
}

// ParseChangeNote validates one change note's YAML. location names the note, such as changes/2026-09-29-verify-retry-limits-7f3a9c.yaml.
// Rule IDs are checked for syntax only; whether each rule exists, and whether its change matches the library, is a library check.
func ParseChangeNote(input []byte, location string) (ChangeNote, error) {
	_, data, err := decode.YAML(input, location)
	if err != nil {
		return ChangeNote{}, err
	}
	fields, err := decode.Object(data, location)
	if err != nil {
		return ChangeNote{}, err
	}
	if err := decode.KnownFields(fields, []string{"summary", "rules"}, location); err != nil {
		return ChangeNote{}, err
	}
	summary, err := decode.Line(fields["summary"], location+".summary")
	if err != nil {
		return ChangeNote{}, err
	}
	entries, err := decode.Object(fields["rules"], location+".rules")
	if err != nil {
		return ChangeNote{}, err
	}
	if len(entries) == 0 {
		return ChangeNote{}, invalid(location+".rules", "expected at least one rule")
	}
	note := ChangeNote{Summary: summary, Rules: map[string]NoteChange{}}
	for _, id := range slices.Sorted(maps.Keys(entries)) {
		entryLocation := location + ".rules." + id
		if err := librarypath.ValidateRuleID(id, entryLocation); err != nil {
			return ChangeNote{}, err
		}
		change, err := noteChange(entries[id], id, entryLocation)
		if err != nil {
			return ChangeNote{}, err
		}
		note.Rules[id] = change
	}
	return note, nil
}

// noteChange accepts a change name, or an object that retires the rule with an optional replacement.
func noteChange(input json.RawMessage, id, location string) (NoteChange, error) {
	var name string
	if json.Unmarshal(input, &name) == nil {
		switch change := coderules.Change(name); change {
		case coderules.ChangeMajor, coderules.ChangeMinor, coderules.ChangePatch, coderules.ChangeNew, coderules.ChangeRetired:
			return NoteChange{Change: change}, nil
		}
		return NoteChange{}, invalid(location, "unknown change "+decode.Quote(name)+"; expected major, minor, patch, new, or retired")
	}
	fields, err := decode.Object(input, location)
	if err != nil {
		return NoteChange{}, invalid(location, "expected major, minor, patch, new, retired, or an object with change: retired")
	}
	if err := decode.KnownFields(fields, []string{"change", "replacedBy"}, location); err != nil {
		return NoteChange{}, err
	}
	if json.Unmarshal(fields["change"], &name) != nil || coderules.Change(name) != coderules.ChangeRetired {
		return NoteChange{}, invalid(location+".change", "expected retired; only a retirement can name a replacement")
	}
	result := NoteChange{Change: coderules.ChangeRetired}
	if raw, ok := fields["replacedBy"]; ok {
		if json.Unmarshal(raw, &result.ReplacedBy) != nil {
			return NoteChange{}, invalid(location+".replacedBy", "expected a rule ID")
		}
		if err := librarypath.ValidateRuleID(result.ReplacedBy, location+".replacedBy"); err != nil {
			return NoteChange{}, err
		}
		if result.ReplacedBy == id {
			return NoteChange{}, invalid(location+".replacedBy", "a rule can't replace itself")
		}
	}
	return result, nil
}

// changeRank orders version changes so several notes on one rule resolve to the largest; other changes rank zero.
func changeRank(change coderules.Change) int {
	switch change {
	case coderules.ChangePatch:
		return 1
	case coderules.ChangeMinor:
		return 2
	case coderules.ChangeMajor:
		return 3
	}
	return 0
}

// LargerChange returns whichever of two version changes moves a rule further; ties return a.
func LargerChange(a, b coderules.Change) coderules.Change {
	if changeRank(b) > changeRank(a) {
		return b
	}
	return a
}
