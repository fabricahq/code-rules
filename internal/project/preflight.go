// Reject duplicate project authoring targets before interactive input; publication retains its own checks.

package project

import (
	"context"
	"path"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
)

// CheckNewLocalGroup requires both group files to be absent, allowing imported groups to have local metadata.
func CheckNewLocalGroup(ctx context.Context, id string, options Options) error {
	if err := rules.ValidateGroupID(id, "group"); err != nil {
		return err
	}
	return checkNewLocalFiles(ctx, options, path.Join("local", id, "_group.json"), path.Join("local", id, "README.md"))
}

// CheckNewLocalRule rejects an existing local rule without conflating it with an imported rule of the same name.
func CheckNewLocalRule(ctx context.Context, id string, options Options) error {
	if _, err := rules.GroupFromPath(id+".md", "rule"); err != nil {
		return err
	}
	return checkNewLocalFiles(ctx, options, path.Join("local", id+".md"))
}

func checkNewLocalFiles(ctx context.Context, options Options, names ...string) error {
	root, configName, err := openProject(ctx, options, false)
	if err != nil {
		return err
	}
	defer root.Close()
	if _, _, err := configuration(ctx, root, configName); err != nil {
		return err
	}
	return filetxn.RequireAbsent(ctx, root, names...)
}

// CheckNewSource rejects an alias already declared in configuration without fetching the library.
func CheckNewSource(ctx context.Context, alias string, options Options) error {
	root, name, err := openProject(ctx, options, false)
	if err != nil {
		return err
	}
	defer root.Close()
	_, config, err := configuration(ctx, root, name)
	if err != nil {
		return err
	}
	for _, source := range config.Sources {
		if source.Name == alias {
			return failure("source-exists", "library alias "+alias+" already exists; edit .code-rules/config.json or choose a different alias", nil)
		}
	}
	return nil
}
