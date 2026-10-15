// Fork a skill a source installs into skills/, and replace the source's skill with it.

package project

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/fabricahq/code-rules/internal/skills"
)

// SkillForkSource names the configured source and the skill in it that a fork copies.
type SkillForkSource struct {
	Source string
	// Skill is the skill's ID in the source, its folder's path there, such as skills/fix-bug.
	Skill string
}

// String returns SOURCE:SKILL.
func (s SkillForkSource) String() string { return s.Source + ":" + s.Skill }

// ParseSkillForkSource reads SOURCE:SKILL, splitting at the first colon, because source names have none.
func ParseSkillForkSource(value string) (SkillForkSource, error) {
	source, skill, found := strings.Cut(value, ":")
	if !found || source == "" || skill == "" {
		return SkillForkSource{}, &ValidationError{Location: "--from", Problem: fmt.Sprintf("%q: expected SOURCE:SKILL, a configured source and the skill's ID in it, such as fabrica:skills/fix-bug", value)}
	}
	if err := skills.ValidateID(skill, "--from"); err != nil {
		return SkillForkSource{}, err
	}
	return SkillForkSource{Source: source, Skill: skill}, nil
}

// SkillForkPlan is a planned fork of a skill a source installs. Planning reads the project without writing; Commit
// writes the fork.
type SkillForkPlan struct {
	from    SkillForkSource
	options Options
	// configBytes is the configuration planning read; Commit refuses to write when it changed.
	configBytes []byte
	// name is the skill's install name, which names the fork's folder, and commit the commit of the source's
	// repository that the source record locks, which the exclusion records as basedOn.
	name, commit string
	// files holds the fork's files, by path relative to the Syllabus directory, with their executable bits.
	files []filetxn.File
}

// PlanSkillFork checks the project and prepares a fork of the skill from names, copied from vendor/ exactly as the
// source's last sync recorded it, into skills/<name>/, where name is the skill's install name, without writing. It
// fails when the source isn't configured, when its record doesn't match its configuration, when its last sync didn't
// install the skill, when the source's library requires the skill, when the source already excludes the skill, or when the project already has an authored skill
// with that name or another exclusion names it as a replacement, since existing authored skills and exclusions are
// never replaced.
func PlanSkillFork(ctx context.Context, from SkillForkSource, options Options) (*SkillForkPlan, error) {
	plan := &SkillForkPlan{from: from}
	options, err := planProjectAuthoring(ctx, options, func(root *os.Root, _ rules.Configuration) error {
		original, config, err := configuration(ctx, root)
		if err != nil {
			return err
		}
		plan.configBytes = original
		index := slices.IndexFunc(config.Sources, func(source rules.Source) bool { return source.Name == from.Source })
		if index < 0 {
			return &ValidationError{Location: "--from", Problem: "no source named " + from.Source + " in .syllabus/config.yaml; name a configured source"}
		}
		source := config.Sources[index]
		if _, excluded := source.SkillExclude[from.Skill]; excluded {
			return &ValidationError{Location: "sources." + source.Name + ".skills.exclude." + from.Skill, Problem: "the source already excludes this skill, and a fork never replaces an existing exclusion; delete the entry to fork the skill"}
		}
		vendor, err := filetxn.ReadTree(ctx, root, "vendor")
		if err != nil {
			return err
		}
		snapshots, err := decodeSnapshots(config, treeFiles(vendor))
		if err != nil {
			return err
		}
		if err := verifyExecutable(snapshots, vendor); err != nil {
			return err
		}
		snapshot := snapshots[source.Name]
		if snapshot.Skills[from.Skill].Level == skills.LevelRequired {
			return &ValidationError{Location: "--from", Problem: rules.RequiredSkillProblem(source.Name, from.Skill, snapshot.Release)}
		}
		if _, installed := snapshot.Skills[from.Skill]; !installed {
			return &ValidationError{Location: "--from", Problem: fmt.Sprintf("source %s doesn't install skill %s, as its last sync recorded; name a skill its skills.include selects, and if you just added it, run syllabus project sync first", source.Name, from.Skill)}
		}
		files := snapshot.SkillFiles(from.Skill)
		location := "vendor/" + source.Name + "/" + from.Skill + "/" + skills.File
		metadata, err := skills.ReadMetadata(files[skills.File], location)
		if err != nil {
			return err
		}
		if plan.name, err = skills.InstallName(from.Skill, metadata, location); err != nil {
			return err
		}
		plan.commit = snapshot.Commit
		folder := skills.AuthoredID(plan.name)
		if err := requireSkillForkTarget(ctx, root, config, folder); err != nil {
			return err
		}
		executable := snapshot.SkillExecutable(from.Skill)
		for _, name := range slices.Sorted(maps.Keys(files)) {
			plan.files = append(plan.files, filetxn.File{Path: path.Join(folder, name), Content: files[name], Executable: slices.Contains(executable, name)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	plan.options = options
	return plan, nil
}

// requireSkillForkTarget fails when the Syllabus directory already has folder, the fork's authored skill folder,
// skills/<name>, or when an exclusion already names that authored skill as its replacement.
func requireSkillForkTarget(ctx context.Context, root *os.Root, config rules.Configuration, folder string) error {
	for _, source := range config.Sources {
		for _, id := range slices.Sorted(maps.Keys(source.SkillExclude)) {
			if source.SkillExclude[id].ReplacedBy == folder {
				return &ValidationError{Location: "sources." + source.Name + ".skills.exclude." + id + ".replacedBy", Problem: fmt.Sprintf("already names %s, the fork's folder, as a replacement, and each authored skill replaces one skill; remove that replacement first", folder)}
			}
		}
	}
	return filetxn.RequireAbsent(ctx, root, folder)
}

// Name returns the fork's name, which is the skill's install name, so its ID is skills/<name>.
func (p *SkillForkPlan) Name() string { return p.name }

// BasedOn returns the full commit SHA the fork copies the skill from, which the exclusion records as basedOn.
func (p *SkillForkPlan) BasedOn() string { return p.commit }

// Commit writes the fork under writer ownership, and, in the same edit, adds the source's exclusion of the skill
// with reason, the fork as replacedBy, and the commit it copied as basedOn, so the fork installs in place of the
// source's skill and updates list the source's later changes to it. It writes nothing when reason is blank, when the
// configuration changed after planning, or when any of the fork's files exists.
func (p *SkillForkPlan) Commit(ctx context.Context, reason string) (AuthoringResult, error) {
	if p == nil || p.name == "" {
		return AuthoringResult{}, failure("invalid-operation", "expected a planned skill fork", nil)
	}
	if strings.TrimSpace(reason) == "" {
		return AuthoringResult{}, &ValidationError{Location: "--reason", Problem: "give the reason the project uses the fork instead of the skill it installs from " + p.from.Source}
	}
	basedOn, err := rules.ParseGitRef(p.commit, "vendor/"+p.from.Source+"/_source.json")
	if err != nil {
		return AuthoringResult{}, err
	}
	folder := skills.AuthoredID(p.name)
	return editProject(ctx, p.options, func(root *os.Root, original []byte, config rules.Configuration) ([]filetxn.File, error) {
		if !bytes.Equal(original, p.configBytes) {
			return nil, failure("concurrent-change", "the project's configuration changed after the fork was planned; run the command again", nil)
		}
		if err := requireSkillForkTarget(ctx, root, config, folder); err != nil {
			return nil, err
		}
		edit := rules.SourceEdit{SkillExclude: map[string]rules.SkillExclusion{p.from.Skill: {Reason: reason, ReplacedBy: folder, BasedOn: basedOn}}}
		edited, err := rules.EditConfigurationSource(original, p.from.Source, edit)
		if err != nil {
			return nil, err
		}
		return append(slices.Clone(p.files), filetxn.File{Path: configurationFile, Content: edited, Previous: original}), nil
	})
}
