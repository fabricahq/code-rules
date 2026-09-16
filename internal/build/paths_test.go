// Check the generated path boundary before an output map is handed to a writer.

package build

import "testing"

// TestOutputPathCollisions rejects unsafe and nonportable output trees.
func TestOutputPathCollisions(t *testing.T) {
	for _, files := range []map[string][]byte{
		{"../outside": nil}, {"a.md": nil, "a.md/b.md": nil}, {"A/r.md": nil, "a/x.md": nil}, {"é/r.md": nil, "e\u0301/x.md": nil}, {"bad\nname": nil},
	} {
		if err := validateOutputPaths(files); err == nil {
			t.Fatalf("accepted %v", files)
		}
	}
}
