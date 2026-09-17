// Create project scaffolding, source declarations, and local definitions without importing or rendering.

package authoring

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
)

// Options identifies a project's config; its sibling directories contain local, vendor, and generated files.
type Options struct{ ConfigPath string }

// Result lists absolute authored paths and the next explicit user action. Errors return no result.
type Result struct {
	Files []string `json:"files"`
	Next  string   `json:"next"`
	// Warnings describe cleanup failures after all requested files were committed.
	Warnings []string `json:"warnings,omitempty"`
}

// RuleOptions distinguishes a supplied body from an unfinished draft in an existing group.
type RuleOptions struct {
	Options
	Body *string
}

// openProject opens the final project directory without accepting a symlink and optionally creates its parents.
func openProject(ctx context.Context, options Options, create bool) (*os.Root, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	config := options.ConfigPath
	if config == "" {
		config = filepath.Join(".code-rules", "config.json")
	}
	absolute, err := filepath.Abs(config)
	if err != nil {
		return nil, "", err
	}
	directory := filepath.Dir(absolute)
	if create {
		if err := createDirectory(directory); err != nil {
			return nil, "", err
		}
	}
	info, err := os.Lstat(directory)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, "", failure("needs-init", "project is not initialized; run init first", err)
	}
	if err != nil {
		return nil, "", err
	}
	if !info.IsDir() {
		return nil, "", failure("unsafe-path", directory+": expected a directory without symlinks", nil)
	}
	root, err := os.OpenRoot(directory)
	return root, filepath.Base(absolute), err
}

// createDirectory resolves existing parent aliases before enforcing the final directory's own no-link boundary.
func createDirectory(name string) error {
	parent := filepath.Dir(name)
	if parent == name {
		return nil
	}
	if _, err := os.Stat(parent); errors.Is(err, fs.ErrNotExist) {
		if err = createDirectory(parent); err != nil {
			return err
		}
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer root.Close()
	created := []string{}
	return ensureParents(root, filepath.Base(name), &created)
}

// configuration reads valid project configuration and its exact bytes under the current root.
func configuration(ctx context.Context, root *os.Root, name string) ([]byte, rules.Configuration, error) {
	data, err := optionalFile(ctx, root, name)
	if err != nil {
		return nil, rules.Configuration{}, err
	}
	if data == nil {
		return nil, rules.Configuration{}, failure("needs-init", name+": missing configuration; run init first", nil)
	}
	config, err := rules.ParseConfiguration(data)
	return data, config, err
}

// InitializeProject creates missing scaffolding and refreshes an unmodified managed guide; valid configuration and local files are preserved.
func InitializeProject(ctx context.Context, options Options) (Result, error) {
	root, name, err := openProject(ctx, options, true)
	if err != nil {
		return Result{}, err
	}
	defer root.Close()
	var files []authoredFile
	committed := false
	err = project.WithWriter(ctx, root, func(_ *project.Writer) error {
		old, err := optionalFile(ctx, root, name)
		if err != nil {
			return err
		}
		if old != nil {
			if _, err = rules.ParseConfiguration(old); err != nil {
				return err
			}
		}
		if _, err = project.ReadTree(ctx, root, "local"); err != nil {
			return err
		}
		readme, err := optionalFile(ctx, root, "local/README.md")
		if err != nil {
			return err
		}
		guide, err := prepareProjectGuide(ctx, root, name)
		if err != nil {
			return err
		}
		if old == nil {
			data, _ := jsonText(map[string]any{"schemaVersion": 1, "sources": map[string]any{}})
			files = append(files, authoredFile{name: name, data: data})
		}
		if readme == nil {
			files = append(files, authoredFile{name: "local/README.md", data: renderLocalReadme(filepath.Join(root.Name(), name))})
		}
		if guide != nil {
			files = append(files, *guide)
		}
		err = publishAuthored(ctx, root, files)
		committed = publicationComplete(err)
		return err
	})
	return finishAuthoring(root, files, "Add a local group and rule, then run build. Or add a source and run sync.", committed, err)
}

// authoredResult reports only paths published by this operation in deterministic publication order.
func authoredResult(root *os.Root, files []authoredFile, next string) Result {
	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, filepath.Join(root.Name(), filepath.FromSlash(file.name)))
	}
	return Result{Files: names, Next: next}
}

// editProject revalidates configuration and local paths under exclusive ownership before publishing authored files.
func editProject(ctx context.Context, options Options, next string, prepare func(*os.Root, string, []byte, rules.Configuration) ([]authoredFile, error)) (Result, error) {
	root, name, err := openProject(ctx, options, false)
	if err != nil {
		return Result{}, err
	}
	defer root.Close()
	var files []authoredFile
	committed := false
	err = project.WithWriter(ctx, root, func(_ *project.Writer) error {
		original, config, err := configuration(ctx, root, name)
		if err != nil {
			return err
		}
		if _, err = project.ReadTree(ctx, root, "local"); err != nil {
			return err
		}
		files, err = prepare(root, name, original, config)
		if err != nil {
			return err
		}
		err = publishAuthored(ctx, root, files)
		committed = publicationComplete(err)
		return err
	})
	return finishAuthoring(root, files, next, committed, err)
}

// AddLocalGroup creates one complete local group definition without overwriting existing metadata.
func AddLocalGroup(ctx context.Context, id string, metadata rules.GroupMetadata, options Options) (Result, error) {
	if err := rules.ValidateGroupID(id, "group"); err != nil {
		return Result{}, err
	}
	data, err := renderGroup(metadata)
	if err != nil {
		return Result{}, err
	}
	return editProject(ctx, options, "Add a rule to this group, then run build.", func(root *os.Root, name string, _ []byte, _ rules.Configuration) ([]authoredFile, error) {
		guideName, _ := ProjectGuide(filepath.Join(root.Name(), name))
		return groupFiles(path.Join("local", id), id, data, false, guideName), nil
	})
}

// groupAvailable accepts local metadata or a selected group from a verified persisted source snapshot.
func groupAvailable(ctx context.Context, root *os.Root, config rules.Configuration, id string) (bool, error) {
	local, err := project.ReadTree(ctx, root, "local")
	if err != nil {
		return false, err
	}
	if local != nil {
		if data, ok := local.Files[id+"/_group.json"]; ok {
			_, err = rules.ParseGroupMetadata(data, id)
			return err == nil, err
		}
	}
	vendor, err := project.ReadTree(ctx, root, "vendor")
	if err != nil || vendor == nil {
		return false, err
	}
	snapshots, err := project.DecodeSnapshots(config, vendor.Files)
	if err != nil {
		return false, err
	}
	for _, source := range config.Sources {
		snapshot := snapshots[source.Name]
		if !slices.Contains(snapshot.Groups, id) {
			continue
		}
		if data, ok := snapshot.Files[id+"/_group.json"]; ok {
			_, err = rules.ParseGroupMetadata(data, id)
			return err == nil, err
		}
	}
	return false, nil
}

// HasLocalRuleGroup checks current metadata before prompting; writes must still recheck under the writer lock.
// This advisory read does not require an idle writer; publication owns recovery and revalidation.
func HasLocalRuleGroup(ctx context.Context, id string, options Options) (bool, error) {
	if err := rules.ValidateGroupID(id, "group"); err != nil {
		return false, err
	}
	root, name, err := openProject(ctx, options, false)
	if err != nil {
		return false, err
	}
	defer root.Close()
	_, config, err := configuration(ctx, root, name)
	if err != nil {
		return false, err
	}
	return groupAvailable(ctx, root, config, id)
}

// AddLocalRule creates a validated rule or canonical unfinished draft, in an existing group.
func AddLocalRule(ctx context.Context, id string, metadata RuleMetadata, options RuleOptions) (Result, error) {
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

	next := "Review the rule, then run build."
	if options.Body == nil {
		next = "Complete the draft and remove unused template prompts before running build."
	}
	return editProject(ctx, options.Options, next, func(root *os.Root, _ string, _ []byte, config rules.Configuration) ([]authoredFile, error) {
		available, err := groupAvailable(ctx, root, config, group)
		if err != nil {
			return nil, err
		}
		if !available {
			return nil, failure("missing-group", "group "+group+" does not exist; create it first with code-rules local add group "+group+", then retry adding the rule", nil)
		}
		return []authoredFile{{name: path.Join("local", id+".md"), data: data}}, nil
	})
}

// AddSource records a validated source declaration without Git access, preserving other selections and exceptions.
func AddSource(ctx context.Context, alias string, source json.RawMessage, options Options) (Result, error) {
	return editProject(ctx, options, "Run sync to import this source and regenerate resolved rules.", func(_ *os.Root, name string, original []byte, _ rules.Configuration) ([]authoredFile, error) {
		var fields map[string]json.RawMessage
		json.Unmarshal(original, &fields)
		var sources map[string]json.RawMessage
		json.Unmarshal(fields["sources"], &sources)
		if _, ok := sources[alias]; ok {
			return nil, failure("source-exists", "source "+alias+" already exists; edit its configuration explicitly", nil)
		}
		sources[alias] = source
		encoded, err := jsonText(sources)
		if err != nil {
			return nil, err
		}
		fields["sources"] = encoded
		data, err := jsonText(fields)
		if err != nil {
			return nil, err
		}
		if _, err = rules.ParseConfiguration(data); err != nil {
			return nil, err
		}
		return []authoredFile{{name: name, data: data, before: original}}, nil
	})
}

// finishAuthoring preserves committed paths and reports cleanup failures as visible result warnings.
func finishAuthoring(root *os.Root, files []authoredFile, next string, committed bool, err error) (Result, error) {
	if err != nil && !committed {
		return Result{}, err
	}
	result := authoredResult(root, files, next)
	if err != nil {
		result.Warnings = []string{"All requested files were committed. Cleanup needs attention before another authoring operation: " + err.Error()}
	}
	return result, nil
}
