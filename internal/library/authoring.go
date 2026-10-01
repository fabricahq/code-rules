// Initialize library repositories and publish authored definitions without replacing existing content.

package library

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"go.yaml.in/yaml/v4"
	"io/fs"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/gitexec"
	"github.com/fabricahq/code-rules/internal/rules"
)

// AuthoringResult lists the absolute paths of the library files an operation created and changed, and any
// post-commit cleanup warnings.
type AuthoringResult struct {
	Added   []string `json:"added"`
	Changed []string `json:"changed"`
	// LicenseDeclared describes the manifest after initialization, and HasGroups whether the library already had a
	// technology or practice group.
	LicenseDeclared bool     `json:"-"`
	HasGroups       bool     `json:"-"`
	Warnings        []string `json:"warnings"`
}

// Written returns every path the operation wrote: its created files, then its changed ones.
func (r AuthoringResult) Written() []string { return append(slices.Clone(r.Added), r.Changed...) }

// Options locates the library itself, independently of any consumer project.
type Options struct {
	Directory string
	// Git runs Git in the library's own repository, to read its library releases; zero values use Git from PATH.
	Git gitexec.Options
}

// Terms contains publisher-supplied text; nil Notice omits a notice while an empty value preserves an empty file.
type Terms struct {
	SPDXExpression string
	License        string
	Notice         *string
}

// RuleOptions distinguishes an explicit body from a marked canonical draft in an existing group.
type RuleOptions struct {
	Options
	Body *string
}

// libraryReadme introduces publisher workflows; CLI tests execute its command examples.
//
//go:embed library-guide.md
var libraryReadme string

// checkWorkflow runs code-rules library check on pull requests; checkWorkflowFor fills in checkWorkflowVersion.
//
//go:embed check-workflow.yml
var checkWorkflow string

// checkWorkflowVersion marks where checkWorkflow's install step sets the Code Rules version it installs.
const checkWorkflowVersion = "CODE_RULES_VERSION"

// latestVersion is the shell command that the install step of a development build's workflow runs to find the
// version of the latest Code Rules release, without its v prefix.
const latestVersion = `$(gh release view --repo fabricahq/code-rules --json tagName --jq '.tagName | ltrimstr("v")')`

// developmentVersion prefixes the versions of builds that no release published: development builds and
// unpublished candidates, such as 0.0.0-development and 0.0.0-dev.g0123456789ab.
const developmentVersion = "0.0.0-"

// checkWorkflowPath is the GitHub Actions workflow library init writes.
const checkWorkflowPath = ".github/workflows/code-rules.yml"

// openLibrary enforces the same final-root no-link policy as project authoring.
func openLibrary(ctx context.Context, options Options, create bool) (*os.Root, error) {
	directory := options.Directory
	if directory == "" {
		directory = "."
	}
	if create {
		return filetxn.Create(ctx, directory)
	}
	root, err := filetxn.Open(ctx, directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil, failure("needs-init", directory+": missing library directory; run code-rules library init first", err)
	}
	return root, err
}

// declaredTerms validates the declaration before reading contained term paths, using planned bytes during initialization.
func declaredTerms(ctx context.Context, root *os.Root, manifest []byte, planned map[string][]byte) (map[string][]byte, *rules.LicenseDeclaration, error) {
	license, err := rules.ParseLibraryLicense(manifest, "library")
	if err != nil {
		return nil, nil, err
	}
	inventory := map[string][]byte{"rule-library.yaml": manifest}
	for _, name := range rules.LicensePaths(license) {
		data, ok := planned[name]
		if !ok {
			if err = checkParents(root, name); err != nil {
				return nil, nil, err
			}
			data, err = filetxn.ReadOptional(ctx, root, name)
			if err != nil {
				return nil, nil, err
			}
			if data == nil {
				return nil, nil, failure("missing-license", name+": missing declared license or notice", nil)
			}
		}
		inventory[name] = data
	}
	return inventory, license, nil
}

// checkParents refuses observed symlinks in every term-file parent without traversing unrelated repository files.
func checkParents(root *os.Root, name string) error {
	parts := strings.Split(name, "/")
	for i := 1; i < len(parts); i++ {
		prefix := strings.Join(parts[:i], "/")
		info, err := root.Lstat(prefix)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return failure("unsafe-path", prefix+": expected directories without symlinks", nil)
		}
	}
	return nil
}

// libraryManifest requires an initialized manifest and valid declared terms.
func libraryManifest(ctx context.Context, root *os.Root) (map[string][]byte, *rules.LicenseDeclaration, error) {
	data, err := filetxn.ReadOptional(ctx, root, "rule-library.yaml")
	if err != nil {
		return nil, nil, err
	}
	if data == nil {
		return nil, nil, failure("needs-init", "missing rule-library.yaml; run library init first", nil)
	}
	return declaredTerms(ctx, root, data, nil)
}

// Initialize creates missing scaffolding and explicit terms without overwriting an existing manifest, terms,
// README, or check workflow. The workflow installs codeRulesVersion, an unprefixed version such as 0.2.0,
// from the Code Rules release tagged with it, or the latest release for a 0.0.0- development build.
func Initialize(ctx context.Context, options Options, terms *Terms, codeRulesVersion string) (AuthoringResult, error) {
	workflow, err := checkWorkflowFor(codeRulesVersion)
	if err != nil {
		return AuthoringResult{}, err
	}
	root, err := openLibrary(ctx, options, true)
	if err != nil {
		return AuthoringResult{}, err
	}
	defer root.Close()
	var license *rules.LicenseDeclaration
	changes, err := filetxn.Edit(ctx, root, func() ([]filetxn.File, error) {
		var files []filetxn.File
		existing, err := filetxn.ReadOptional(ctx, root, "rule-library.yaml")
		if err != nil {
			return nil, err
		}
		if existing != nil && terms != nil {
			return nil, failure("already-initialized", "library manifest exists; edit its license declaration explicitly", nil)
		}
		manifest := existing
		planned := map[string][]byte{}
		if existing == nil {
			value := map[string]any{"formatVersion": 1}
			if terms != nil {
				notices := []string{}
				if terms.Notice != nil {
					notices = append(notices, "NOTICE.md")
				}
				value["license"] = map[string]any{"spdxExpression": terms.SPDXExpression, "file": "LICENSE.md", "notices": notices}
			}
			manifest, err = yamlText(value)
			if err != nil {
				return nil, err
			}
			files = append(files, filetxn.File{Path: "rule-library.yaml", Content: manifest})
		}
		if terms != nil {
			planned["LICENSE.md"] = []byte(terms.License)
			files = append(files, filetxn.File{Path: "LICENSE.md", Content: planned["LICENSE.md"]})
			if terms.Notice != nil {
				planned["NOTICE.md"] = []byte(*terms.Notice)
				files = append(files, filetxn.File{Path: "NOTICE.md", Content: planned["NOTICE.md"]})
			}
		}
		_, license, err = declaredTerms(ctx, root, manifest, planned)
		if err != nil {
			return nil, err
		}
		readme, err := filetxn.ReadOptional(ctx, root, "README.md")
		if err != nil {
			return nil, err
		}
		if readme == nil {
			files = append(files, filetxn.File{Path: "README.md", Content: []byte(libraryReadme)})
		}
		existingWorkflow, err := readOptionalBelow(ctx, root, checkWorkflowPath)
		if err != nil {
			return nil, err
		}
		if existingWorkflow == nil {
			files = append(files, filetxn.File{Path: checkWorkflowPath, Content: workflow})
		}
		return files, nil
	})
	if err != nil {
		return AuthoringResult{}, err
	}
	result, err := authoringResult(changes, err)
	result.LicenseDeclared = license != nil
	for _, directory := range []string{"practices", "techs"} {
		if entries, readErr := fs.ReadDir(root.FS(), directory); readErr == nil && len(entries) > 0 {
			result.HasGroups = true
		}
	}
	return result, err
}

// checkWorkflowFor returns the check workflow pinned to version, which must be a complete semantic version
// without a v prefix, so it can't change the workflow's structure. Its install step downloads and verifies only
// that version's archive. A development build has no release to pin, so its workflow finds the latest release's
// version first and then installs exactly that one.
func checkWorkflowFor(version string) ([]byte, error) {
	if canonical, err := rules.TagVersion(version, "version"); err != nil || canonical != version {
		return nil, failure("invalid-version", "Code Rules version "+strconv.Quote(version)+" is not a complete semantic version", err)
	}
	installed := version
	if strings.HasPrefix(version, developmentVersion) {
		installed = latestVersion
	}
	return []byte(strings.ReplaceAll(checkWorkflow, checkWorkflowVersion, installed)), nil
}

// readOptionalBelow reads a file whose parent directories may be missing; nil means the file is absent.
// Observed symlinks in its parents fail.
func readOptionalBelow(ctx context.Context, root *os.Root, name string) ([]byte, error) {
	if err := checkParents(root, name); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return filetxn.ReadOptional(ctx, root, name)
}

// editLibrary revalidates manifest and the target category under exclusive ownership before publication.
func editLibrary(ctx context.Context, options Options, group string, prepare func(*os.Root) ([]filetxn.File, error)) (AuthoringResult, error) {
	root, err := openLibrary(ctx, options, false)
	if err != nil {
		return AuthoringResult{}, err
	}
	defer root.Close()
	changes, err := filetxn.Edit(ctx, root, func() ([]filetxn.File, error) {
		var files []filetxn.File
		if _, _, err := libraryManifest(ctx, root); err != nil {
			return nil, err
		}
		if _, err := filetxn.ReadTree(ctx, root, strings.Split(group, "/")[0]); err != nil {
			return nil, err
		}
		var err error
		files, err = prepare(root)
		if err != nil {
			return nil, err
		}
		return files, nil
	})
	return authoringResult(changes, err)
}

// AddGroup creates complete metadata without changing existing library content.
func AddGroup(ctx context.Context, id string, metadata rules.GroupMetadata, options Options) (AuthoringResult, error) {
	if err := rules.ValidateGroupID(id, "group"); err != nil {
		return AuthoringResult{}, err
	}
	data, err := rules.RenderGroup(metadata)
	if err != nil {
		return AuthoringResult{}, err
	}
	return editLibrary(ctx, options, id, func(root *os.Root) ([]filetxn.File, error) {
		if err := checkNewGroup(ctx, root, id); err != nil {
			return nil, err
		}
		return groupFiles(id, id, data), nil
	})
}

// hasGroupMetadata validates an existing group definition without creating it.
func hasGroupMetadata(ctx context.Context, root *os.Root, id string) (bool, error) {
	if err := checkParents(root, id+"/_group.yaml"); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	data, err := filetxn.ReadOptional(ctx, root, id+"/_group.yaml")
	if err != nil || data == nil {
		return false, err
	}
	_, err = rules.ParseGroupMetadataYAML(data, id)
	return err == nil, err
}

// AddRule creates supplied guidance or a marked unfinished canonical draft, in an existing group.
func AddRule(ctx context.Context, id string, metadata rules.RuleMetadata, options RuleOptions) (AuthoringResult, error) {
	group, err := rules.GroupFromPath(id+".md", "rule")
	if err != nil {
		return AuthoringResult{}, err
	}
	data, err := rules.RenderRule(id, metadata, options.Body)
	if err != nil {
		return AuthoringResult{}, err
	}
	return editLibrary(ctx, options.Options, group, func(root *os.Root) ([]filetxn.File, error) {
		if err := checkNewRule(ctx, root, id, options.Options); err != nil {
			return nil, err
		}
		return []filetxn.File{{Path: id + ".md", Content: data}}, nil
	})
}

// authoringResult returns published paths and cleanup warnings only after a successful write.
func authoringResult(changes filetxn.Changes, err error) (AuthoringResult, error) {
	if err != nil {
		return AuthoringResult{}, err
	}
	return AuthoringResult{Added: changes.Added, Changed: changes.Changed, Warnings: changes.Warnings}, nil
}

func yamlText(value any) ([]byte, error) {
	var out bytes.Buffer
	e := yaml.NewEncoder(&out)
	e.SetIndent(2)
	if err := e.Encode(value); err != nil {
		return nil, err
	}
	return out.Bytes(), e.Close()
}

func failure(code, problem string, cause error) error {
	return &filetxn.Error{Code: code, Problem: problem, Cause: cause}
}

func groupFiles(directory, id string, metadata []byte) []filetxn.File {
	instructions := fmt.Sprintf("## Add or edit rules\n\nRun commands from the library root, two directories above this folder.\nUse `code-rules library add rule %s/<rule-name>` to add a rule; run `code-rules library add rule --help` for metadata and body options.\nEdit existing rule files directly, then run `code-rules library check`. Resolve errors before committing or publishing the library.\n\nA consuming project selects this group in its configuration and runs sync. Its generated/RULES.md identifies the adopted rules after exclusions and replacements.\n", id)
	return []filetxn.File{{Path: path.Join(directory, "_group.yaml"), Content: metadata}, {Path: path.Join(directory, "README.md"), Content: rules.GroupGuide(id, instructions)}}
}
