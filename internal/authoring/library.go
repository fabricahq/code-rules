// Package authoring owns library scaffolding, definitions, and complete library checks.
// Consuming-project operations belong to project; shared publication belongs to filetxn.
package authoring

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
)

// Result lists published library files, next actions, and any post-commit cleanup warnings.
type Result struct {
	Files    []string `json:"files"`
	Next     string   `json:"next"`
	Warnings []string `json:"warnings,omitempty"`
}

// LibraryOptions locates the library itself, independently of any consumer project.
type LibraryOptions struct{ Directory string }

// LibraryTerms contains publisher-supplied text; nil Notice omits a notice while an empty value preserves an empty file.
type LibraryTerms struct {
	SPDXExpression string
	License        string
	Notice         *string
}

// LibraryRuleOptions distinguishes an explicit body from a marked canonical draft in an existing group.
type LibraryRuleOptions struct {
	LibraryOptions
	Body *string
}

// libraryReadme introduces publisher workflows; CLI tests execute its command examples.
//
//go:embed library-guide.md
var libraryReadme string

// openLibrary enforces the same final-root no-link policy as project authoring.
func openLibrary(ctx context.Context, options LibraryOptions, create bool) (*os.Root, error) {
	directory := options.Directory
	if directory == "" {
		directory = "."
	}
	if create {
		return filetxn.Create(ctx, directory)
	}
	return filetxn.Open(ctx, directory)
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
	data, err := filetxn.ReadOptional(ctx, root, "rule-library.json")
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
	var license *rules.LicenseDeclaration
	changes, err := filetxn.Edit(ctx, root, func() ([]filetxn.File, error) {
		var files []filetxn.File
		existing, err := filetxn.ReadOptional(ctx, root, "rule-library.json")
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
			manifest, err = jsonText(value)
			if err != nil {
				return nil, err
			}
			files = append(files, filetxn.File{Path: "rule-library.json", Content: manifest})
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
		return files, nil
	})
	if err != nil {
		return Result{}, err
	}
	next := "Add a group and rule, then run library check."
	if license == nil {
		next = "License is undeclared. Decide terms before sharing. " + next
	}
	return authoringResult(changes, next, err)
}

// editLibrary revalidates manifest and the target category under exclusive ownership before publication.
func editLibrary(ctx context.Context, options LibraryOptions, group, next string, prepare func(*os.Root) ([]filetxn.File, error)) (Result, error) {
	root, err := openLibrary(ctx, options, false)
	if err != nil {
		return Result{}, err
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
	return authoringResult(changes, next, err)
}

// AddLibraryGroup creates complete metadata without changing existing library content.
func AddLibraryGroup(ctx context.Context, id string, metadata rules.GroupMetadata, options LibraryOptions) (Result, error) {
	if err := rules.ValidateGroupID(id, "group"); err != nil {
		return Result{}, err
	}
	data, err := rules.RenderGroup(metadata)
	if err != nil {
		return Result{}, err
	}
	return editLibrary(ctx, options, id, "Add a library rule, then run library check.", func(_ *os.Root) ([]filetxn.File, error) {
		return groupFiles(id, id, data), nil
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
	data, err := filetxn.ReadOptional(ctx, root, id+"/_group.json")
	if err != nil || data == nil {
		return false, err
	}
	_, err = rules.ParseGroupMetadata(data, id)
	return err == nil, err
}

// HasLibraryGroup checks current metadata before an interactive prompt; writes recheck under the lock.
// This advisory read does not require an idle writer; publication owns recovery and revalidation.
func HasLibraryGroup(ctx context.Context, id string, options LibraryOptions) (bool, error) {
	if err := rules.ValidateGroupID(id, "group"); err != nil {
		return false, err
	}
	root, err := openLibrary(ctx, options, false)
	if err != nil {
		return false, err
	}
	defer root.Close()
	if _, _, err = libraryManifest(ctx, root); err != nil {
		return false, err
	}
	return hasGroupMetadata(ctx, root, id)
}

// AddLibraryRule creates supplied guidance or a marked unfinished canonical draft, in an existing group.
func AddLibraryRule(ctx context.Context, id string, metadata rules.RuleMetadata, options LibraryRuleOptions) (Result, error) {
	if strings.HasSuffix(id, ".md") {
		return Result{}, failure("invalid-operation", "use a rule ID without the .md extension", nil)
	}
	group, err := rules.GroupFromPath(id+".md", "rule")
	if err != nil {
		return Result{}, err
	}
	data, err := rules.RenderRule(id, metadata, options.Body)
	if err != nil {
		return Result{}, err
	}
	next := "Review the rule and run library check."
	if options.Body == nil {
		data = append(data, []byte("\n<!-- code-rules:draft -->\n")...)
		next = "Complete the draft and remove its code-rules:draft marker, then run library check."
	}

	return editLibrary(ctx, options.LibraryOptions, group, next, func(root *os.Root) ([]filetxn.File, error) {
		exists, err := hasGroupMetadata(ctx, root, group)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, failure("missing-group", "group "+group+" does not exist; create it first with code-rules library add group "+group+", then retry adding the rule", nil)
		}
		return []filetxn.File{{Path: id + ".md", Content: data}}, nil
	})
}

// authoringResult adds the domain's next action to the completed publication report.
func authoringResult(changes filetxn.Changes, next string, err error) (Result, error) {
	if err != nil {
		return Result{}, err
	}
	return Result{Files: changes.Files, Next: next, Warnings: changes.Warnings}, nil
}

func jsonText(value any) ([]byte, error) {
	var out bytes.Buffer
	e := json.NewEncoder(&out)
	e.SetEscapeHTML(false)
	e.SetIndent("", "  ")
	err := e.Encode(value)
	return out.Bytes(), err
}

func failure(code, problem string, cause error) error {
	return &filetxn.Error{Code: code, Problem: problem, Cause: cause}
}

func groupFiles(directory, id string, metadata []byte) []filetxn.File {
	instructions := fmt.Sprintf("## Add or edit rules\n\nRun commands from the library root, two directories above this folder.\nUse `code-rules library add rule %s/<rule-name>` to add a rule; run `code-rules library add rule --help` for metadata and body options.\nEdit existing rule files directly, then run `code-rules library check`. Resolve errors before committing or publishing the library.\n\nA consuming project selects this group in its configuration and runs sync. Its generated/RULES.md identifies the adopted rules after exclusions and replacements.\n", id)
	return []filetxn.File{{Path: path.Join(directory, "_group.json"), Content: metadata}, {Path: path.Join(directory, "README.md"), Content: rules.GroupGuide(id, instructions)}}
}
