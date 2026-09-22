// Adapt managed directories and the two project-guide filenames to the same recoverable transaction.

package filetxn

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
)

// GuideReadme and GuideStandalone are managed files, not permission to replace a general project README.
// Callers must establish guide ownership and recheck its original bytes before publication.
const (
	GuideReadme     Target = "README.md"
	GuideStandalone Target = "CODE_RULES.md"
)

func guideTarget(target Target) bool { return target == GuideReadme || target == GuideStandalone }
func validTarget(target Target) bool {
	return target == Vendor || target == Generated || guideTarget(target)
}

// readTarget represents a managed file as a one-file inventory with a stable key across staging and recovery.
func readTarget(ctx context.Context, root *os.Root, target Target, location string) (*Tree, error) {
	if !guideTarget(target) {
		return ReadTree(ctx, root, location)
	}
	data, err := ReadOptional(ctx, root, location)
	if err != nil || data == nil {
		return nil, err
	}
	return &Tree{Files: map[string][]byte{string(target): data}}, nil
}

// writeTarget creates staged output exclusively; guide payloads must contain exactly their declared filename.
func writeTarget(ctx context.Context, root *os.Root, target Target, location string, files map[string][]byte) error {
	if !guideTarget(target) {
		return writeTree(ctx, root, location, files)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	data, ok := files[string(target)]
	if !ok || len(files) != 1 {
		return failure("invalid-target", fmt.Sprintf("%s: expected exactly one matching guide file", target), nil)
	}
	file, err := root.OpenFile(location, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	return errors.Join(writeErr, closeErr)
}

// renameManaged publishes guide files without replacing a file created after the last validation.
// Parent descriptors come from os.Root so the platform rename retains root confinement.
func renameManaged(root *os.Root, from, to string) error {
	if !guideTarget(Target(to)) {
		return root.Rename(from, to)
	}
	source, err := root.Open(path.Dir(from))
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := root.Open(".")
	if err != nil {
		return err
	}
	defer destination.Close()
	return renameExclusive(int(source.Fd()), path.Base(from), int(destination.Fd()), to)
}
