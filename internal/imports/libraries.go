// Coordinate complete source imports and preserve original bytes with verified Git provenance.

package imports

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
)

// Library keeps the parsed catalog and its byte-preserving, verified source snapshot together.
type Library struct {
	Catalog  library.Catalog  `json:"catalog"`
	Snapshot project.Snapshot `json:"snapshot"`
}

// ImportLibraries imports every configured source or returns no partial result.
// It never installs files in a consuming project. Source temporary state is closed on every path.
func ImportLibraries(ctx context.Context, configuration rules.Configuration, options Options) (map[string]Library, error) {
	if err := ctx.Err(); err != nil {
		return nil, contextFailure(err)
	}
	result := make(map[string]Library, len(configuration.Sources))
	for _, source := range configuration.Sources {
		if _, exists := result[source.Name]; exists {
			return nil, fail("invalid-configuration", "Duplicate source alias.", nil)
		}
		imported, err := importLibrary(ctx, source, options)
		if err != nil {
			return nil, fmt.Errorf("import source %q: %w. No libraries were returned because all configured sources must succeed", source.Name, err)
		}
		result[source.Name] = imported
	}
	return result, nil
}

// importLibrary applies one deadline across fetch, blob reads, validation, and snapshot construction.
func importLibrary(ctx context.Context, source rules.Source, options Options) (_ Library, err error) {
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	if timeout < 0 {
		return Library{}, fail("invalid-options", "Import timeout must be positive or zero for the default.", nil)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	revision, err := FetchRevision(ctx, source, options)
	if err != nil {
		return Library{}, err
	}
	defer func() { err = errors.Join(err, revision.Close()) }()
	input, err := revision.openTree(ctx)
	if err != nil {
		return Library{}, err
	}
	catalog, err := library.LoadSource(ctx, input, source.Name, source.Groups)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Library{}, contextFailure(err)
		}
		return Library{}, err
	}
	files := make(map[string][]byte, len(catalog.SupportingFiles))
	for name, data := range catalog.SupportingFiles {
		files[name] = bytes.Clone(data)
	}
	groups := make([]string, 0, len(catalog.Groups))
	for _, group := range catalog.Groups {
		groups = append(groups, group.ID)
		for _, rule := range group.Rules {
			files[rule.Path] = []byte(rule.Document)
		}
	}
	selection := catalog.Selection
	selection.Groups = slices.Clone(selection.Groups)
	snapshot := project.Snapshot{Repository: source.Repository, Ref: source.Ref, Version: source.Version, Tag: revision.Tag, ResolvedVersion: revision.Version, Commit: revision.Commit, Selection: selection, Groups: groups, Files: files}
	if err := ctx.Err(); err != nil {
		return Library{}, contextFailure(err)
	}
	return Library{Catalog: catalog, Snapshot: snapshot}, nil
}
