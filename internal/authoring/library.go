// Author library manifests, group metadata, and rule documents using protected publication.

package authoring

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
)

// LibraryOptions locates the library itself, independently of any consumer project.
type LibraryOptions struct{ Directory string }

// LibraryTerms contains publisher-supplied text; nil Notice omits a notice while an empty value preserves an empty file.
type LibraryTerms struct {
	SPDXExpression string
	License        string
	Notice         *string
}

// LibraryRuleOptions distinguishes an explicit body from a marked canonical draft and optional new metadata.
type LibraryRuleOptions struct {
	LibraryOptions
	Body  *string
	Group *rules.GroupMetadata
}

const libraryReadme = `# Rule library

Describe shared engineering guidance under techs/<group>/ or practices/<group>/.
Start with code-rules library add group, then code-rules library add rule.
Complete drafts and run code-rules library check before committing and tagging a version.

Follow the [canonical authoring rubric](https://github.com/fabricahq/code-rules/blob/main/docs/src/content/docs/reference/rule-authoring.md).
Declare one license for the whole library in rule-library.json and retain its actual license text and notices.
Consumers select this Git repository and a version with code-rules add source, then run code-rules sync.
`

// openLibrary enforces the same final-root no-link policy as project authoring.
func openLibrary(ctx context.Context, options LibraryOptions, create bool) (*os.Root, error) {
	directory := options.Directory
	if directory == "" {
		directory = "."
	}
	root, _, err := openProject(ctx, Options{ConfigPath: filepath.Join(directory, "rule-library.json")}, create)
	return root, err
}

// declaredTerms validates the declaration before reading contained term paths, using planned bytes during initialization.
func declaredTerms(ctx context.Context, root *os.Root, manifest []byte, planned map[string][]byte) (map[string][]byte, *rules.LicenseDeclaration, error) {
	inventory := map[string][]byte{"rule-library.json": manifest}
	var raw struct {
		License struct {
			File    string
			Notices []string
		}
	}
	// Discovery supplies only placeholder presence; the domain parser validates every path and field before reads.
	if json.Unmarshal(manifest, &raw) == nil {
		if raw.License.File != "rule-library.json" {
			inventory[raw.License.File] = nil
		}
		for _, name := range raw.License.Notices {
			if name != "rule-library.json" {
				inventory[name] = nil
			}
		}
	}
	license, err := rules.ReadLibraryLicense(inventory, "library")
	if err != nil {
		return nil, nil, err
	}
	inventory = map[string][]byte{"rule-library.json": manifest}
	for _, name := range rules.LicensePaths(license) {
		data, ok := planned[name]
		if !ok {
			if err = checkParents(root, name); err != nil {
				return nil, nil, err
			}
			data, err = optionalFile(ctx, root, name)
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
	data, err := optionalFile(ctx, root, "rule-library.json")
	if err != nil {
		return nil, nil, err
	}
	if data == nil {
		return nil, nil, failure("needs-init", "missing rule-library.json; run library init first", nil)
	}
	return declaredTerms(ctx, root, data, nil)
}

// InitializeLibrary creates missing scaffolding and explicit terms without overwriting an existing manifest or terms.
func InitializeLibrary(ctx context.Context, options LibraryOptions, terms *LibraryTerms) (Result, error) {
	root, err := openLibrary(ctx, options, true)
	if err != nil {
		return Result{}, err
	}
	defer root.Close()
	var files []authoredFile
	committed := false
	var license *rules.LicenseDeclaration
	err = project.WithWriter(ctx, root, func(_ *project.Writer) error {
		existing, err := optionalFile(ctx, root, "rule-library.json")
		if err != nil {
			return err
		}
		if existing != nil && terms != nil {
			return failure("already-initialized", "library manifest exists; edit its license declaration explicitly", nil)
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
			manifest, err = jsonText(value)
			if err != nil {
				return err
			}
			files = append(files, authoredFile{name: "rule-library.json", data: manifest})
		}
		if terms != nil {
			planned["LICENSE.md"] = []byte(terms.License)
			files = append(files, authoredFile{name: "LICENSE.md", data: planned["LICENSE.md"]})
			if terms.Notice != nil {
				planned["NOTICE.md"] = []byte(*terms.Notice)
				files = append(files, authoredFile{name: "NOTICE.md", data: planned["NOTICE.md"]})
			}
		}
		_, license, err = declaredTerms(ctx, root, manifest, planned)
		if err != nil {
			return err
		}
		readme, err := optionalFile(ctx, root, "README.md")
		if err != nil {
			return err
		}
		if readme == nil {
			files = append(files, authoredFile{name: "README.md", data: []byte(libraryReadme)})
		}
		err = publishAuthored(ctx, root, files)
		committed = publicationComplete(err)
		return err
	})
	if err != nil && !committed {
		return Result{}, err
	}
	next := "Add a group and rule, then run library check."
	if license == nil {
		next = "License is undeclared. Decide terms before sharing. " + next
	}
	return finishAuthoring(root, files, next, committed, err)
}

// editLibrary revalidates manifest and the target category under exclusive ownership before publication.
func editLibrary(ctx context.Context, options LibraryOptions, group, next string, prepare func(*os.Root) ([]authoredFile, error)) (Result, error) {
	root, err := openLibrary(ctx, options, false)
	if err != nil {
		return Result{}, err
	}
	defer root.Close()
	var files []authoredFile
	committed := false
	err = project.WithWriter(ctx, root, func(_ *project.Writer) error {
		if _, _, err := libraryManifest(ctx, root); err != nil {
			return err
		}
		if _, err := project.ReadTree(ctx, root, strings.Split(group, "/")[0]); err != nil {
			return err
		}
		var err error
		files, err = prepare(root)
		if err != nil {
			return err
		}
		err = publishAuthored(ctx, root, files)
		committed = publicationComplete(err)
		return err
	})
	return finishAuthoring(root, files, next, committed, err)
}

// AddLibraryGroup creates complete metadata without changing existing library content.
func AddLibraryGroup(ctx context.Context, id string, metadata rules.GroupMetadata, options LibraryOptions) (Result, error) {
	if err := rules.ValidateGroupID(id, "group"); err != nil {
		return Result{}, err
	}
	data, err := renderGroup(metadata)
	if err != nil {
		return Result{}, err
	}
	return editLibrary(ctx, options, id, "Add a library rule, then run library check.", func(_ *os.Root) ([]authoredFile, error) {
		return []authoredFile{{name: path.Join(id, "_group.json"), data: data}}, nil
	})
}

// hasGroupMetadata validates an existing group definition without creating it.
func hasGroupMetadata(ctx context.Context, root *os.Root, id string) (bool, error) {
	if err := checkParents(root, id+"/_group.json"); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	data, err := optionalFile(ctx, root, id+"/_group.json")
	if err != nil || data == nil {
		return false, err
	}
	_, err = rules.ParseGroupMetadata(data, id)
	return err == nil, err
}

// HasLibraryGroup checks current metadata before an interactive prompt; writes recheck under the lock.
func HasLibraryGroup(ctx context.Context, id string, options LibraryOptions) (bool, error) {
	if err := rules.ValidateGroupID(id, "group"); err != nil {
		return false, err
	}
	root, err := openLibrary(ctx, options, false)
	if err != nil {
		return false, err
	}
	defer root.Close()
	if err = project.RequireIdle(root); err != nil {
		return false, err
	}
	if _, _, err = libraryManifest(ctx, root); err != nil {
		return false, err
	}
	return hasGroupMetadata(ctx, root, id)
}

// AddLibraryRule creates supplied guidance or a marked unfinished canonical draft, optionally creating its group.
func AddLibraryRule(ctx context.Context, id string, metadata RuleMetadata, options LibraryRuleOptions) (Result, error) {
	if strings.HasSuffix(id, ".md") {
		return Result{}, failure("invalid-operation", "use a rule ID without the .md extension", nil)
	}
	group, err := rules.GroupFromPath(id+".md", "rule")
	if err != nil {
		return Result{}, err
	}
	data, err := renderRule(id, metadata, options.Body)
	if err != nil {
		return Result{}, err
	}
	next := "Review the rule and run library check."
	if options.Body == nil {
		data = append(data, []byte("\n<!-- code-rules:draft -->\n")...)
		next = "Complete the draft and remove its code-rules:draft marker, then run library check."
	}
	var groupData []byte
	if options.Group != nil {
		groupData, err = renderGroup(*options.Group)
		if err != nil {
			return Result{}, err
		}
	}
	return editLibrary(ctx, options.LibraryOptions, group, next, func(root *os.Root) ([]authoredFile, error) {
		files := []authoredFile{}
		if groupData != nil {
			files = append(files, authoredFile{name: group + "/_group.json", data: groupData})
		} else {
			exists, err := hasGroupMetadata(ctx, root, group)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, failure("missing-group", "run library add group "+group+" first or supply --create-group and metadata", nil)
			}
		}
		return append(files, authoredFile{name: id + ".md", data: data}), nil
	})
}
