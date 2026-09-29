// Record rule changes as new change notes in changes/, and read the notes library check compares.

package library

import (
	"context"
	"io/fs"
	"strings"
)

// changesDirectory holds change notes at the library root; projects never import it.
const changesDirectory = "changes"

// readChangeNotes captures every note under changes/, which may be absent. It holds only .yaml files;
// subdirectories, links, and other files fail.
func readChangeNotes(ctx context.Context, input FileSource) (map[string][]byte, error) {
	r := reader{ctx: ctx, input: portableInventorySource{input}, files: map[string][]byte{}}
	entries, err := r.entries(changesDirectory, true)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		name := changesDirectory + "/" + entry.Name()
		if entry.Type()&fs.ModeType != 0 || !strings.HasSuffix(name, ".yaml") {
			return nil, bad(name, "expected only change notes, as .yaml files, in changes/")
		}
		if _, err := r.read(name); err != nil {
			return nil, err
		}
	}
	return r.files, nil
}
