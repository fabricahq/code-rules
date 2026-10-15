// Test how a source's skills are found in a commit's tree and chosen by skills.include, without Git.

package imports

import (
	"errors"
	"io/fs"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/rules"
)

// entries returns a tree of regular files at names, keyed by repository path.
func entries(names ...string) map[string]treeEntry {
	tree := map[string]treeEntry{}
	for _, name := range names {
		tree[name] = treeEntry{name: name, mode: 0o644, object: "0"}
	}
	return tree
}

func TestSkillFoldersFindsEachSkillAndItsFiles(t *testing.T) {
	tests := map[string]struct {
		tree      map[string]treeEntry
		isLibrary bool
		want      map[string][]string
	}{
		"library skills are folders directly in .syllabus/skills": {
			tree: entries(".syllabus/skills/build/SKILL.md", ".syllabus/skills/build/scripts/run.sh", ".syllabus/skills/notes.md", ".syllabus/skills/empty/README.md",
				".syllabus/skills/group/nested/SKILL.md", ".syllabus/rules/techs/go/errors.md", "tools/review/SKILL.md"),
			isLibrary: true,
			want:      map[string][]string{"skills/build": {"skills/build/SKILL.md", "skills/build/scripts/run.sh"}},
		},
		"another repository's skills are folders with a SKILL.md": {
			tree:      entries("SKILL.md", "README.md", "tools/review/SKILL.md", "tools/review/guide.md", "skills/engineering/to-spec/SKILL.md", ".claude/skills/local/SKILL.md"),
			isLibrary: false,
			want: map[string][]string{
				"tools/review":               {"tools/review/SKILL.md", "tools/review/guide.md"},
				"skills/engineering/to-spec": {"skills/engineering/to-spec/SKILL.md"},
				".claude/skills/local":       {".claude/skills/local/SKILL.md"},
			},
		},
		"a skill inside another skill's folder belongs to it": {
			tree:      entries("outer/SKILL.md", "outer/inner/SKILL.md", "outer/inner/notes.md", "outer-two/SKILL.md"),
			isLibrary: false,
			want:      map[string][]string{"outer": {"outer/SKILL.md", "outer/inner/SKILL.md", "outer/inner/notes.md"}, "outer-two": {"outer-two/SKILL.md"}},
		},
		"no skills": {tree: entries("README.md"), want: map[string][]string{}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			folders := skillFolders(test.tree, test.isLibrary)
			got := map[string][]string{}
			for id, files := range folders {
				got[id] = slices.Sorted(maps.Keys(files))
				for stored, entry := range files {
					if entry.name != stored {
						t.Errorf("%s: entry name %s, want its snapshot path %s", id, entry.name, stored)
					}
				}
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("skillFolders = %v, want %v", got, test.want)
			}
		})
	}
}

func TestChooseSkillsTakesEverySkillFromThePlannedRevision(t *testing.T) {
	const locked, newer = "1111111111111111111111111111111111111111", "2222222222222222222222222222222222222222"
	available := []string{".claude/skills/local", "skills/build", "skills/review"}
	release := func(_ *testing.T, include ...string) rules.Source {
		return rules.Source{Name: "team", Skills: include}
	}
	pinned := func(t *testing.T, include ...string) rules.Source {
		return rules.Source{Name: "team", Skills: include, Ref: gitRef(t, "v1.0.0")}
	}
	tests := map[string]struct {
		source  func(*testing.T, ...string) rules.Source
		include []string
		plan    sourcePlan
		want    []string
		code    string
		problem string
	}{
		"wildcard skips dot folders":           {release, []string{"*"}, sourcePlan{commit: locked}, []string{"skills/build", "skills/review"}, "", ""},
		"named skills":                         {release, []string{"skills/review"}, sourcePlan{commit: locked, locked: locked}, []string{"skills/review"}, "", ""},
		"a dot folder can be named":            {release, []string{".claude/skills/local"}, sourcePlan{commit: locked}, []string{".claude/skills/local"}, "", ""},
		"missing from the locked release":      {release, []string{"skills/new"}, sourcePlan{release: 1, commit: locked, locked: locked}, nil, "skill-not-in-revision", "Run syllabus project update team"},
		"missing from the locked ref":          {pinned, []string{"skills/new"}, sourcePlan{commit: locked, locked: locked}, nil, "skill-not-in-revision", "set sources.team.ref to a tag or commit that has it"},
		"missing from a newly chosen revision": {release, []string{"skills/new"}, sourcePlan{release: 2, commit: newer, locked: locked}, nil, "", "the source has no skill skills/new at release/2"},
		"missing without a record":             {release, []string{"skills/new"}, sourcePlan{commit: newer}, nil, "", "the source has no skill skills/new at commit " + newer},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := chooseSkills(test.source(t, test.include...), test.plan, available, nil)
			if test.problem == "" {
				if err != nil || !slices.Equal(got, test.want) {
					t.Fatalf("chooseSkills = %v, %v; want %v", got, err, test.want)
				}
				return
			}
			var failure *Error
			var validation *ValidationError
			switch {
			case test.code != "" && (!errors.As(err, &failure) || failure.Code != test.code):
				t.Fatalf("got %v; want code %s", err, test.code)
			case test.code == "" && (!errors.As(err, &validation) || validation.Location != "sources.team.skills.include"):
				t.Fatalf("got %v; want a validation error at sources.team.skills.include", err)
			case !strings.Contains(err.Error(), test.problem):
				t.Fatalf("got %v; want %q", err, test.problem)
			}
		})
	}
}

func TestSkillFoldersKeepsFileModes(t *testing.T) {
	tree := entries("tools/run/SKILL.md")
	tree["tools/run/run.sh"] = treeEntry{name: "tools/run/run.sh", mode: 0o755, object: "1"}
	tree["tools/run/link"] = treeEntry{name: "tools/run/link", mode: fs.ModeSymlink, object: "2"}
	files := skillFolders(tree, false)["tools/run"]
	if files["tools/run/run.sh"].mode.Perm()&0o100 == 0 || files["tools/run/link"].mode.IsRegular() {
		t.Fatalf("modes = %v", files)
	}
}
