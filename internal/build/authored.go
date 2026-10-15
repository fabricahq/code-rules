// Place a project's authored files in the Syllabus directory, where their links resolve.

package build

import (
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
)

// authoredLocation returns where the authored file keyed file sits in the Syllabus directory. Authored files are
// keyed as a library keys its files: a shared asset, keyed assets/<path>, sits in assets/, and every other file,
// keyed by its path in rules/, such as techs/go/errors.md, sits there.
func authoredLocation(file string) string {
	if rules.AssetDirectory(file) == "assets/" {
		return file
	}
	return "rules/" + file
}

// authoredTarget resolves destination, a link in the authored file keyed file, from where that file sits in the
// Syllabus directory, and returns the key of the authored file it names, as rules.RelativeTarget does for a
// library file. A link outside rules/ and assets/ fails, because authored files can depend only on each other.
func authoredTarget(destination, file string) (target, suffix string, local bool, err error) {
	location := authoredLocation(file)
	target, suffix, local, err = rules.RelativeTarget(destination, location)
	if err != nil || !local {
		return "", "", local, err
	}
	key, err := authoredKey(target, location)
	return key, suffix, true, err
}

// authoredKey returns the key of the authored file at target, a path in the Syllabus directory that a link in the
// authored file at location names. It fails for a path outside rules/ and assets/.
func authoredKey(target, location string) (string, error) {
	if rules.AssetDirectory(target) == "assets/" {
		return target, nil
	}
	if key, ok := strings.CutPrefix(target, "rules/"); ok && rules.AssetDirectory(key) != "assets/" {
		return key, nil
	}
	return "", invalid(location, "links to "+target+", which is outside .syllabus/rules/ and .syllabus/assets/; link to this rule's own assets directory, or to a shared asset in .syllabus/assets/")
}
