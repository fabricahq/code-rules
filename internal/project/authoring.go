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
	"slices"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
)

// AuthoringResult lists absolute authored paths and cleanup warnings. Errors return no result.
type AuthoringResult struct {
	Files []string `json:"files"`
	// Warnings describe cleanup failures after all requested files were committed.
	Warnings []string `json:"warnings,omitempty"`
}

// RuleOptions distinguishes a supplied body from an unfinished draft in an existing group.
type RuleOptions struct {
	Options
	Body *string
}

// Initialize creates missing scaffolding and refreshes an unmodified managed guide; valid configuration and local files are preserved.
func Initialize(ctx context.Context, options Options) (AuthoringResult, error) {
	root, err := openProject(ctx, options, true)
	if err != nil {
		return AuthoringResult{}, err
	}
	defer root.Close()
	changes, err := filetxn.Edit(ctx, root, func() ([]filetxn.File, error) {
		var files []filetxn.File
		old, err := filetxn.ReadOptional(ctx, root, configurationFile)
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
		guide, err := prepareProjectGuide(ctx, root)
		if err != nil {
			return nil, err
		}
		if old == nil {
			data, _ := jsonText(map[string]any{"schemaVersion": 1, "sources": map[string]any{}})
			files = append(files, filetxn.File{Path: configurationFile, Content: data})
		}
		if readme == nil {
			files = append(files, filetxn.File{Path: "local/README.md", Content: renderLocalReadme()})
		}
		if guide != nil {
			files = append(files, *guide)
		}
		return files, nil
	})
	return authoringResult(changes, err)
}

// editProject revalidates configuration and local paths under exclusive ownership before publishing authored files.
func editProject(ctx context.Context, options Options, prepare func(*os.Root, []byte, rules.Configuration) ([]filetxn.File, error)) (AuthoringResult, error) {
	root, err := openProject(ctx, options, false)
	if err != nil {
		return AuthoringResult{}, err
	}
	defer root.Close()
	changes, err := filetxn.Edit(ctx, root, func() ([]filetxn.File, error) {
		var files []filetxn.File
		original, config, err := configuration(ctx, root)
		if err != nil {
			return nil, err
		}
		if _, err = filetxn.ReadTree(ctx, root, "local"); err != nil {
			return nil, err
		}
		files, err = prepare(root, original, config)
		if err != nil {
			return nil, err
		}
		return files, nil
	})
	return authoringResult(changes, err)
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
	return editProject(ctx, options, func(root *os.Root, _ []byte, _ rules.Configuration) ([]filetxn.File, error) {
		if err := checkLocalGroup(ctx, root, id); err != nil {
			return nil, err
		}
		guideName, _ := projectGuide()
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

// AddLocalRule creates a validated rule or canonical unfinished draft, in an existing group.
func AddLocalRule(ctx context.Context, id string, metadata rules.RuleMetadata, options RuleOptions) (AuthoringResult, error) {
	data, err := rules.RenderRule(id, metadata, options.Body)
	if err != nil {
		return AuthoringResult{}, err
	}

	return editProject(ctx, options.Options, func(root *os.Root, _ []byte, config rules.Configuration) ([]filetxn.File, error) {
		if err := checkLocalRule(ctx, root, config, id); err != nil {
			return nil, err
		}
		return []filetxn.File{{Path: path.Join("local", id+".md"), Content: data}}, nil
	})
}

// AddSource records a validated source declaration without Git access, preserving other selections and exceptions.
func AddSource(ctx context.Context, alias string, source json.RawMessage, options Options) (AuthoringResult, error) {
	return editProject(ctx, options, func(_ *os.Root, original []byte, config rules.Configuration) ([]filetxn.File, error) {
		var fields map[string]json.RawMessage
		json.Unmarshal(original, &fields)
		var sources map[string]json.RawMessage
		json.Unmarshal(fields["sources"], &sources)
		if err := checkSourceAlias(config, alias); err != nil {
			return nil, err
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
		return []filetxn.File{{Path: configurationFile, Content: data, Previous: original}}, nil
	})
}

// authoringResult returns published paths and cleanup warnings only after a successful write.
func authoringResult(changes filetxn.Changes, err error) (AuthoringResult, error) {
	if err != nil {
		return AuthoringResult{}, err
	}
	return AuthoringResult{Files: changes.Files, Warnings: changes.Warnings}, nil
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
	instructions := fmt.Sprintf("## Add or edit rules\n\nFollow [the project guide](../../../%s) for commands to run from the project root.\nUse `code-rules project add rule %s/<rule-name>` to add a rule. Edit existing rule files directly.\nRun code-rules project build and code-rules project check from the project root after local changes, then inspect [the resolved rules](../../../generated/RULES.md).\nUse the resolved rules when working on the project: they include imported guidance and apply exclusions and replacements.\n", projectGuideName, id)
	return []filetxn.File{{Path: path.Join(directory, "_group.json"), Content: metadata}, {Path: path.Join(directory, "README.md"), Content: rules.GroupGuide(id, instructions)}}
}
