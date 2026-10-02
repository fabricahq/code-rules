// Adapt managed directories, the two project-guide filenames, the configuration, and the local files sync and update
// write to the same recoverable transaction.

package filetxn

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/fabricahq/code-rules/internal/librarytree"
	"github.com/fabricahq/code-rules/internal/rules"
)

// GuideReadme and GuideStandalone are managed files, not permission to replace a general project README.
// Callers must establish guide ownership and recheck its original bytes before publication.
const (
	GuideReadme     Target = "README.md"
	GuideStandalone Target = "CODE_RULES.md"
)

// Config is the project configuration. Only project update replaces it, together with vendor and generated output,
// so its pins and exclusions and the output they select are installed or recovered together.
const Config Target = "config.yaml"

// LocalGroupMetadata is the metadata file of the local group with ID group, local/<group>/_group.yaml; group must be
// a valid group ID. Sync and update create it together with vendor and generated output when the project's local
// rules in the group lose the only imported copy of its metadata, so recovery keeps the two consistent. Its
// directory must already exist.
func LocalGroupMetadata(group string) Target { return Target("local/" + group + "/_group.yaml") }

// maxLocalGroupMetadata bounds the local group metadata files one transaction can create, so its journal stays
// small enough to read during recovery.
const maxLocalGroupMetadata = 1000

// LocalRule is the Markdown file of the local rule at file, a rule path relative to local/, such as
// techs/go/errors.md, and LocalRuleAssets is that rule's asset directory, local/techs/go/assets/errors. Only
// project update replaces them, when it replaces a fork with a newer one, so recovery restores or finishes the rule,
// its assets, the configuration that records the fork's version, and the output generated from them together. A
// nil file map for LocalRuleAssets removes the directory. Apply creates their missing parent directories.
func LocalRule(file string) Target { return Target("local/" + file) }

// LocalRuleAssets is the asset directory of the local rule at file; see LocalRule.
func LocalRuleAssets(file string) Target {
	return Target("local/" + strings.TrimSuffix(rules.RuleAssetDirectory(file), "/"))
}

// maxLocalRules bounds the local rules one transaction can replace, each with its asset directory, so its journal
// stays small enough to read during recovery.
const maxLocalRules = 1000

func guideTarget(target Target) bool { return target == GuideReadme || target == GuideStandalone }

// localGroupMetadataTarget reports whether target is a LocalGroupMetadata target of a valid group ID.
func localGroupMetadataTarget(target Target) bool {
	group, local := strings.CutPrefix(string(target), "local/")
	group, metadata := strings.CutSuffix(group, "/_group.yaml")
	return local && metadata && librarytree.ValidateGroupID(group, "group") == nil
}

// localRuleTarget reports whether target is a LocalRule target: the Markdown file of a rule path under local/.
func localRuleTarget(target Target) bool {
	file, local := strings.CutPrefix(string(target), "local/")
	id, versioned := librarytree.VersionedRule(file)
	return local && versioned && file == id+".md"
}

// localRuleAssetsTarget reports whether target is a LocalRuleAssets target: the asset directory of a rule path
// under local/.
func localRuleAssetsTarget(target Target) bool {
	directory, local := strings.CutPrefix(string(target), "local/")
	parent := path.Dir(directory)
	if !local || path.Base(parent) != "assets" {
		return false
	}
	file := path.Dir(parent) + "/" + path.Base(directory) + ".md"
	return localRuleTarget(LocalRule(file)) && LocalRuleAssets(file) == target
}

// localTarget reports whether target is an authored file or directory under local/, which follows the fixed targets
// in a transaction.
func localTarget(target Target) bool {
	return localGroupMetadataTarget(target) || localRuleTarget(target) || localRuleAssetsTarget(target)
}

// fileTarget reports whether target is a single file rather than a directory tree.
func fileTarget(target Target) bool {
	return guideTarget(target) || target == Config || localGroupMetadataTarget(target) || localRuleTarget(target)
}

func validTarget(target Target) bool {
	return target == Vendor || target == Generated || fileTarget(target) || localRuleAssetsTarget(target)
}

// requireRealParents fails with unsafe-path when a parent directory of target, such as local/techs for local group
// metadata, is a symbolic link or not a directory, so publication and recovery never reach another file through
// it. A missing parent passes: the target is absent.
func requireRealParents(root *os.Root, target Target) error {
	parts := strings.Split(string(target), "/")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[:i], "/")
		info, err := root.Lstat(parent)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return failure("unsafe-path", parent+": expected a directory without symlinks", nil)
		}
	}
	return nil
}

// entryName names target's staged output, backup, and displaced copy inside the transaction directory, which holds
// no subdirectories for them: the slashes of a path under local/ become +, which no target contains.
func entryName(target Target) string { return strings.ReplaceAll(string(target), "/", "+") }

// renameCheck is where Apply probes that this filesystem can publish a file target without replacing another file.
var renameCheck = path.Join(transactionName, "rename-check")

// readTarget represents a managed file as a one-file inventory with a stable key across staging and recovery.
func readTarget(ctx context.Context, root *os.Root, target Target, location string) (*Tree, error) {
	if !fileTarget(target) {
		return ReadTree(ctx, root, location)
	}
	data, err := ReadOptional(ctx, root, location)
	if err != nil || data == nil {
		return nil, err
	}
	return &Tree{Files: map[string][]byte{string(target): data}}, nil
}

// writeTarget creates staged output exclusively; file payloads must contain exactly their declared filename.
func writeTarget(ctx context.Context, root *os.Root, target Target, location string, files map[string][]byte) error {
	if !fileTarget(target) {
		return writeTree(ctx, root, location, files)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	data, ok := files[string(target)]
	if !ok || len(files) != 1 {
		return failure("invalid-target", fmt.Sprintf("%s: expected exactly one matching file", target), nil)
	}
	file, err := root.OpenFile(location, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	return errors.Join(writeErr, closeErr)
}

// renameManaged publishes file targets, and probes publishing them, without replacing a file created after the last
// validation. Parent descriptors come from os.Root so the platform rename retains root confinement.
func renameManaged(root *os.Root, from, to string) error {
	return renameManagedWith(root, from, to, renameExclusive)
}

// renameManagedWith supplies the platform boundary for filesystem capability tests.
func renameManagedWith(root *os.Root, from, to string, exclusive func(int, string, int, string) error) error {
	if !fileTarget(Target(to)) && path.Dir(to) != renameCheck {
		return root.Rename(from, to)
	}
	source, err := root.Open(path.Dir(from))
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := root.Open(path.Dir(to))
	if err != nil {
		return err
	}
	defer destination.Close()
	return exclusive(int(source.Fd()), path.Base(from), int(destination.Fd()), path.Base(to))
}

// probeFileRename checks actual staged bytes on this filesystem before any live output is displaced.
// Hard-link fallbacks are unsuitable: interrupted link/unlink pairs violate recovery's no-hard-link policy.
func (w *Writer) probeFileRename(target Target, staged string) error {
	if err := w.root.Mkdir(renameCheck, 0700); err != nil {
		return err
	}
	destination := path.Join(renameCheck, entryName(target))
	if err := w.rename(staged, destination); err != nil {
		return fmt.Errorf("cannot safely publish %s on this filesystem; existing files were not changed: %w", target, err)
	}
	if err := w.root.Rename(destination, staged); err != nil {
		return err
	}
	return w.root.Remove(renameCheck)
}
