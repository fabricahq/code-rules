// Test how the update preview compares a source's skills and which files it flags as scripts.

package imports

import (
	"reflect"
	"testing"

	"github.com/fabricahq/trailguide/internal/library"
)

func TestSkillChanges_FlagsOnlyExecutableFilesTheUpdateAddsOrChanges(t *testing.T) {
	recorded := &library.Snapshot{
		Skills: map[string]library.ImportedSkill{"skills/build": {}, "skills/same": {}, "skills/gone": {}},
		Files: map[string][]byte{
			"skills/build/SKILL.md":         []byte("old"),
			"skills/build/scripts/run.sh":   []byte("echo old"),
			"skills/build/scripts/lint.sh":  []byte("echo lint"),
			"skills/build/scripts/mode.sh":  []byte("echo mode"),
			"skills/build/scripts/plain.sh": []byte("echo plain"),
			"skills/build/scripts/old.sh":   []byte("echo removed"),
			"skills/same/SKILL.md":          []byte("same"),
			"skills/same/scripts/same.sh":   []byte("echo same"),
			"skills/gone/SKILL.md":          []byte("gone"),
		},
		Executable: []string{"skills/build/scripts/lint.sh", "skills/build/scripts/old.sh", "skills/build/scripts/plain.sh", "skills/build/scripts/run.sh", "skills/same/scripts/same.sh"},
	}
	installed := importedSkills{
		skills: map[string]library.ImportedSkill{"skills/build": {}, "skills/same": {}, "skills/new": {}},
		files: map[string][]byte{
			"skills/build/SKILL.md":          []byte("new"),
			"skills/build/scripts/run.sh":    []byte("echo new"),
			"skills/build/scripts/lint.sh":   []byte("echo lint"),
			"skills/build/scripts/mode.sh":   []byte("echo mode"),
			"skills/build/scripts/plain.sh":  []byte("echo plain"),
			"skills/build/scripts/setup.sh":  []byte("echo setup"),
			"skills/build/references/a.md":   []byte("notes"),
			"skills/same/SKILL.md":           []byte("same"),
			"skills/same/scripts/same.sh":    []byte("echo same"),
			"skills/new/SKILL.md":            []byte("new"),
			"skills/new/scripts/new.sh":      []byte("echo new"),
			"skills/new/references/notes.md": []byte("notes"),
		},
		executable: []string{"skills/build/scripts/lint.sh", "skills/build/scripts/mode.sh", "skills/build/scripts/run.sh", "skills/build/scripts/setup.sh", "skills/new/scripts/new.sh", "skills/same/scripts/same.sh"},
	}
	want := []SkillUpdate{
		{
			ID: "skills/build", Change: SkillChanged,
			// lint.sh is unchanged and plain.sh is no longer executable, so neither is flagged.
			Files: []string{"SKILL.md", "references/a.md", "scripts/mode.sh", "scripts/old.sh", "scripts/plain.sh", "scripts/run.sh", "scripts/setup.sh"},
			Scripts: []ScriptUpdate{
				{Path: "scripts/mode.sh", Change: SkillChanged},
				{Path: "scripts/run.sh", Change: SkillChanged},
				{Path: "scripts/setup.sh", Change: SkillAdded},
			},
		},
		{ID: "skills/new", Change: SkillAdded, Files: []string{}, Scripts: []ScriptUpdate{{Path: "scripts/new.sh", Change: SkillAdded}}},
		{ID: "skills/gone", Change: SkillRemoved, Files: []string{}, Scripts: []ScriptUpdate{}},
	}
	if got := skillChanges(recorded, installed); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestSkillChanges_AddsEverySkillAndScriptWithoutARecord(t *testing.T) {
	installed := importedSkills{
		skills:     map[string]library.ImportedSkill{"tools/review": {}},
		files:      map[string][]byte{"tools/review/SKILL.md": []byte("review"), "tools/review/check.sh": []byte("echo check")},
		executable: []string{"tools/review/check.sh"},
	}
	want := []SkillUpdate{{ID: "tools/review", Change: SkillAdded, Files: []string{}, Scripts: []ScriptUpdate{{Path: "check.sh", Change: SkillAdded}}}}
	if got := skillChanges(nil, installed); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestSkillChanges_ListsNothingWhenNoSkillChanges(t *testing.T) {
	files := map[string][]byte{"skills/a/SKILL.md": []byte("a"), "skills/a/run.sh": []byte("echo a"), "skills/ab/SKILL.md": []byte("ab")}
	recorded := &library.Snapshot{Skills: map[string]library.ImportedSkill{"skills/a": {}, "skills/ab": {}}, Files: files, Executable: []string{"skills/a/run.sh"}}
	installed := importedSkills{skills: map[string]library.ImportedSkill{"skills/a": {}, "skills/ab": {}}, files: files, executable: []string{"skills/a/run.sh"}}
	if got := skillChanges(recorded, installed); got == nil || len(got) != 0 {
		t.Fatalf("got %#v; want an empty list", got)
	}
	if got := skillChanges(nil, importedSkills{skills: map[string]library.ImportedSkill{}}); got == nil || len(got) != 0 {
		t.Fatalf("got %#v; want an empty list for no skills", got)
	}
}
