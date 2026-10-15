// Read a project's authored skills, and plan the installed skill folders build, sync, and update replace or remove.

package project

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/skills"
)

// authoredSkills returns the authored skills in tree, the project's skills/ folder, by ID, skills/<name>, each with
// its files by path in its folder. It fails for a file directly in skills/, a folder name that breaks the skill name
// grammar, and a folder without a SKILL.md. A nil tree has none.
func authoredSkills(tree *filetxn.Tree) (map[string]build.SkillFiles, error) {
	result := map[string]build.SkillFiles{}
	if tree == nil {
		return result, nil
	}
	folders := []string{}
	for _, directory := range tree.Directories {
		if !strings.Contains(directory, "/") {
			folders = append(folders, directory)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(tree.Files)) {
		folder, _, inFolder := strings.Cut(name, "/")
		if !inFolder {
			return nil, &ValidationError{Location: ".syllabus/skills/" + name, Problem: "skills/ holds one folder per skill, each with a SKILL.md; move this file into a skill's folder"}
		}
		if !slices.Contains(folders, folder) {
			folders = append(folders, folder)
		}
	}
	slices.Sort(folders)
	for _, folder := range folders {
		location := ".syllabus/skills/" + folder + "/"
		if !skills.ValidName(folder) {
			return nil, &ValidationError{Location: location, Problem: "a skill's folder name is its name, which uses 1 to 64 lowercase letters, digits, and hyphens, without a leading, trailing, or doubled hyphen; rename the folder"}
		}
		files := map[string][]byte{}
		executable := []string{}
		for name, data := range tree.Files {
			if rel, ok := strings.CutPrefix(name, folder+"/"); ok {
				files[rel] = data
				if slices.Contains(tree.Executable, name) {
					executable = append(executable, rel)
				}
			}
		}
		if _, ok := files[skills.File]; !ok {
			return nil, &ValidationError{Location: location, Problem: "missing SKILL.md; each folder in skills/ is a skill, so add its SKILL.md or move the folder out of skills/"}
		}
		slices.Sort(executable)
		result[skills.AuthoredID(folder)] = build.SkillFiles{Files: files, Executable: executable}
	}
	return result, nil
}

// skillInstall is what a write does to the installed skill folders: the targets and executable files it passes to
// filetxn, the folders it touches as read before planning, and the file changes, by path from the repository root.
type skillInstall struct {
	targets    map[filetxn.Target]map[string][]byte
	executable map[filetxn.Target][]string
	// read holds each folder the write touches, by path from the repository root, as read before planning; a
	// folder that was absent is nil.
	read    map[string]*filetxn.Tree
	changes FileChanges
}

// planSkillFolders plans the installed skill folders of output, given generated, the generated tree before the
// write, whose installed-skills.json lists the folders Syllabus installed. It replaces each folder output
// installs whose files differ, and removes each folder Syllabus installed that output no longer installs. root is
// the project's Syllabus directory, whose parent is the repository root. It fails with skill-folder-not-managed
// when a folder output installs already exists and Syllabus didn't install it, and with unsafe-path when a skill
// target's folder is a symbolic link.
func planSkillFolders(ctx context.Context, root *os.Root, generated *filetxn.Tree, output build.Output) (skillInstall, error) {
	plan := skillInstall{targets: map[filetxn.Target]map[string][]byte{}, executable: map[filetxn.Target][]string{}, read: map[string]*filetxn.Tree{}}
	owned := build.InstalledFolders(treeFiles(generated)[build.InstalledSkillsFile])
	touched := slices.Sorted(maps.Keys(output.Skills))
	for _, folder := range owned {
		if !slices.Contains(touched, folder) {
			touched = append(touched, folder)
		}
	}
	slices.Sort(touched)
	if len(touched) == 0 {
		plan.changes = compareFiles(nil, nil)
		return plan, nil
	}
	read, err := readSkillFolders(ctx, root, touched)
	if err != nil {
		return skillInstall{}, err
	}
	plan.read = read
	before, after := map[string][]byte{}, map[string][]byte{}
	modeChanged := []string{}
	for _, folder := range touched {
		current := read[folder]
		want, installs := output.Skills[folder]
		if current != nil && !slices.Contains(owned, folder) {
			return skillInstall{}, failure("skill-folder-not-managed", fmt.Sprintf("%s already exists, and Syllabus didn't install it, so it won't replace it. Move or delete the folder, or stop including the skill that installs there, then run the command again.", folder), nil)
		}
		if current != nil {
			for name, data := range current.Files {
				before[folder+"/"+name] = data
			}
		}
		if !installs {
			if current != nil {
				plan.targets[filetxn.SkillFolder(folder)] = nil
			}
			continue
		}
		for name, data := range want.Files {
			after[folder+"/"+name] = data
		}
		var executable []string
		if current != nil {
			executable = current.Executable
			for name := range want.Files {
				if _, ok := current.Files[name]; ok && slices.Contains(executable, name) != slices.Contains(want.Executable, name) {
					modeChanged = append(modeChanged, folder+"/"+name)
				}
			}
		}
		if current != nil && maps.EqualFunc(current.Files, want.Files, bytes.Equal) && slices.Equal(nonNil(executable), want.Executable) {
			continue
		}
		target := filetxn.SkillFolder(folder)
		plan.targets[target] = want.Files
		plan.executable[target] = want.Executable
	}
	plan.changes = compareFiles(before, after)
	for _, name := range modeChanged {
		if !slices.Contains(plan.changes.Changed, name) {
			plan.changes.Changed = append(plan.changes.Changed, name)
		}
	}
	slices.Sort(plan.changes.Changed)
	return plan, nil
}

// requireSkillFoldersUnchanged fails with concurrent-change when a folder plan read changed since, so a write never
// replaces or removes edits made while it ran.
func requireSkillFoldersUnchanged(ctx context.Context, root *os.Root, plan skillInstall) error {
	if len(plan.read) == 0 {
		return nil
	}
	after, err := readSkillFolders(ctx, root, slices.Sorted(maps.Keys(plan.read)))
	if err != nil {
		return err
	}
	for folder, tree := range plan.read {
		if (tree == nil) != (after[folder] == nil) || (tree != nil && tree.Digest() != after[folder].Digest()) {
			return failure("concurrent-change", folder+": the installed skill folder changed during the operation; retry after edits finish", nil)
		}
	}
	return nil
}

// readSkillFolders reads each folder, by path from the repository root, the parent of root, keeping a missing one
// as nil.
func readSkillFolders(ctx context.Context, root *os.Root, folders []string) (map[string]*filetxn.Tree, error) {
	repository, err := filetxn.Open(ctx, filepath.Dir(root.Name()))
	if err != nil {
		return nil, err
	}
	defer repository.Close()
	read := map[string]*filetxn.Tree{}
	for _, folder := range folders {
		tree, err := filetxn.ReadTree(ctx, repository, folder)
		if err != nil {
			return nil, err
		}
		read[folder] = tree
	}
	return read, nil
}

// prefixChanges returns changes with prefix before each path.
func prefixChanges(changes FileChanges, prefix string) FileChanges {
	for _, list := range []*[]string{&changes.Added, &changes.Changed, &changes.Removed} {
		prefixed := []string{}
		for _, name := range *list {
			prefixed = append(prefixed, prefix+name)
		}
		*list = prefixed
	}
	return changes
}

// mergeChanges adds the paths of extra to changes, keeping each list sorted.
func mergeChanges(changes *FileChanges, extra FileChanges) {
	for _, list := range []struct {
		to   *[]string
		from []string
	}{{&changes.Added, extra.Added}, {&changes.Changed, extra.Changed}, {&changes.Removed, extra.Removed}} {
		*list.to = append(nonNil(*list.to), list.from...)
		slices.Sort(*list.to)
	}
}

// noSkillTargetsWarning returns the warning for a project that installs skills but has no skill targets, so no copy
// was installed, or empty when it has targets or no skills.
func noSkillTargetsWarning(output build.Output, targets []string) string {
	if len(targets) > 0 || len(output.SkillIDs) == 0 {
		return ""
	}
	return fmt.Sprintf("The project has no skill targets, so no copy of its %d skill(s) was installed. Agents can find them through .syllabus/generated/SKILLS.md. To install copies, set skillTargets in .syllabus/config.yaml, such as [claude, agents], then run syllabus project build.", len(output.SkillIDs))
}

// clashFailure converts a *build.ClashError to its CLI failure, returning other errors unchanged.
func clashFailure(err error) error {
	var clash *build.ClashError
	if errors.As(err, &clash) {
		return failure("skill-name-clash", clash.Error(), nil)
	}
	return err
}
