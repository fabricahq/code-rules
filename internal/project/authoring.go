// Package project owns a consuming project's Code Rules state: initialization, local authoring,
// library adoption, offline generation, and read-only freshness checks. Storage protocols stay private.
package project

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
)

// AuthoringResult lists absolute authored paths and the next explicit user action. Errors return no result.
type AuthoringResult struct {
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

// openProject resolves configuration while keeping root creation inside the storage module.
func openProject(ctx context.Context, options Options, create bool) (*os.Root, string, error) {
	config := options.ConfigPath
	if config == "" {
		config = filepath.Join(".code-rules", "config.json")
	}
	absolute, err := filepath.Abs(config)
	if err != nil {
		return nil, "", err
	}
	var root *os.Root
	if create {
		root, err = filetxn.Create(ctx, filepath.Dir(absolute))
	} else {
		root, err = filetxn.Open(ctx, filepath.Dir(absolute))
	}
	return root, filepath.Base(absolute), err
}

// configuration reads valid project configuration and its exact bytes under the current root.
func configuration(ctx context.Context, root *os.Root, name string) ([]byte, rules.Configuration, error) {
	data, err := filetxn.ReadOptional(ctx, root, name)
	if err != nil {
		return nil, rules.Configuration{}, err
	}
	if data == nil {
		return nil, rules.Configuration{}, failure("needs-init", name+": missing configuration; run init first", nil)
	}
	config, err := rules.ParseConfiguration(data)
	return data, config, err
}

// Initialize creates missing scaffolding and refreshes an unmodified managed guide; valid configuration and local files are preserved.
func Initialize(ctx context.Context, options Options) (AuthoringResult, error) {
	root, name, err := openProject(ctx, options, true)
	if err != nil {
		return AuthoringResult{}, err
	}
	defer root.Close()
	changes, err := filetxn.Edit(ctx, root, func() ([]filetxn.File, error) {
		var files []filetxn.File
		old, err := filetxn.ReadOptional(ctx, root, name)
		if err != nil {
			return nil, err
		}
		if old != nil {
			if _, err = rules.ParseConfiguration(old); err != nil {
				return nil, err
			}
		}
		if _, err = filetxn.ReadTree(ctx, root, "local"); err != nil {
			return nil, err
		}
		readme, err := filetxn.ReadOptional(ctx, root, "local/README.md")
		if err != nil {
			return nil, err
		}
		guide, err := prepareProjectGuide(ctx, root, name)
		if err != nil {
			return nil, err
		}
		if old == nil {
			data, _ := jsonText(map[string]any{"schemaVersion": 1, "sources": map[string]any{}})
			files = append(files, filetxn.File{Path: name, Content: data})
		}
		if readme == nil {
			files = append(files, filetxn.File{Path: "local/README.md", Content: renderLocalReadme(filepath.Join(root.Name(), name))})
		}
		if guide != nil {
			files = append(files, *guide)
		}
		return files, nil
	})
	return authoringResult(changes, "Add a local group and rule, then run build. Or add a source and run sync.", err)
}

// editProject revalidates configuration and local paths under exclusive ownership before publishing authored files.
func editProject(ctx context.Context, options Options, next string, prepare func(*os.Root, string, []byte, rules.Configuration) ([]filetxn.File, error)) (AuthoringResult, error) {
	root, name, err := openProject(ctx, options, false)
	if err != nil {
		return AuthoringResult{}, err
	}
	defer root.Close()
	changes, err := filetxn.Edit(ctx, root, func() ([]filetxn.File, error) {
		var files []filetxn.File
		original, config, err := configuration(ctx, root, name)
		if err != nil {
			return nil, err
		}
		if _, err = filetxn.ReadTree(ctx, root, "local"); err != nil {
			return nil, err
		}
		files, err = prepare(root, name, original, config)
		if err != nil {
			return nil, err
		}
		return files, nil
	})
	return authoringResult(changes, next, err)
}

// AddLocalGroup creates one complete local group definition without overwriting existing metadata.
func AddLocalGroup(ctx context.Context, id string, metadata rules.GroupMetadata, options Options) (AuthoringResult, error) {
	if err := rules.ValidateGroupID(id, "group"); err != nil {
		return AuthoringResult{}, err
	}
	data, err := rules.RenderGroup(metadata)
	if err != nil {
		return AuthoringResult{}, err
	}
	return editProject(ctx, options, "Add a rule to this group, then run build.", func(root *os.Root, name string, _ []byte, _ rules.Configuration) ([]filetxn.File, error) {
		guideName, _ := projectGuide(filepath.Join(root.Name(), name))
		return groupFiles(path.Join("local", id), id, data, guideName), nil
	})
}

// groupAvailable accepts local metadata or a selected group from a verified persisted source snapshot.
func groupAvailable(ctx context.Context, root *os.Root, config rules.Configuration, id string) (bool, error) {
	local, err := filetxn.ReadTree(ctx, root, "local")
	if err != nil {
		return false, err
	}
	if local != nil {
		if data, ok := local.Files[id+"/_group.json"]; ok {
			_, err = rules.ParseGroupMetadata(data, id)
			return err == nil, err
		}
	}
	vendor, err := filetxn.ReadTree(ctx, root, "vendor")
	if err != nil || vendor == nil {
		return false, err
	}
	snapshots, err := decodeSnapshots(config, vendor.Files)
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
func AddLocalRule(ctx context.Context, id string, metadata rules.RuleMetadata, options RuleOptions) (AuthoringResult, error) {
	if strings.HasSuffix(id, ".md") {
		return AuthoringResult{}, failure("invalid-operation", "use a rule ID without the .md extension", nil)
	}
	group, err := rules.GroupFromPath(id+".md", "rule")
	if err != nil {
		return AuthoringResult{}, err
	}
	data, err := rules.RenderRule(id, metadata, options.Body)
	if err != nil {
		return AuthoringResult{}, err
	}

	next := "Review the rule, then run build."
	if options.Body == nil {
		next = "Complete the draft and remove unused template prompts before running build."
	}
	return editProject(ctx, options.Options, next, func(root *os.Root, _ string, _ []byte, config rules.Configuration) ([]filetxn.File, error) {
		available, err := groupAvailable(ctx, root, config, group)
		if err != nil {
			return nil, err
		}
		if !available {
			return nil, failure("missing-group", "group "+group+" does not exist; create it first with code-rules local add group "+group+", then retry adding the rule", nil)
		}
		return []filetxn.File{{Path: path.Join("local", id+".md"), Content: data}}, nil
	})
}

// AddSource records a validated source declaration without Git access, preserving other selections and exceptions.
func AddSource(ctx context.Context, alias string, source json.RawMessage, options Options) (AuthoringResult, error) {
	return editProject(ctx, options, "Run sync to import this source and regenerate resolved rules.", func(_ *os.Root, name string, original []byte, _ rules.Configuration) ([]filetxn.File, error) {
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
		return []filetxn.File{{Path: name, Content: data, Previous: original}}, nil
	})
}

// authoringResult adds the domain's next action to the completed publication report.
func authoringResult(changes filetxn.Changes, next string, err error) (AuthoringResult, error) {
	if err != nil {
		return AuthoringResult{}, err
	}
	return AuthoringResult{Files: changes.Files, Next: next, Warnings: changes.Warnings}, nil
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

func groupFiles(directory, id string, metadata []byte, projectGuideName string) []filetxn.File {
	instructions := fmt.Sprintf("## Add or edit rules\n\nFollow [the project guide](../../../%s) for complete commands and the correct configuration path.\nUse `code-rules local add rule %s/<rule-name>` to add a rule. Edit existing rule files directly.\nRun build and check with that configuration after local changes, then inspect [the resolved rules](../../../generated/RULES.md).\nUse the resolved rules when working on the project: they include imported guidance and apply exclusions and replacements.\n", projectGuideName, id)
	return []filetxn.File{{Path: path.Join(directory, "_group.json"), Content: metadata}, {Path: path.Join(directory, "README.md"), Content: rules.GroupGuide(id, instructions)}}
}
