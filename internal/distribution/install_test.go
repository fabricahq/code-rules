// Validate archive boundaries independently of the packaging implementation.

package distribution

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

// TestArchiveRejectsUnsafeMembers ensures extraction cannot accept links, duplicates, or paths outside the install directory.
func TestArchiveRejectsUnsafeMembers(t *testing.T) {
	for _, header := range []tar.Header{
		{Name: "../escape", Typeflag: tar.TypeReg},
		{Name: "code-rules", Typeflag: tar.TypeSymlink, Linkname: "/tmp/other"},
		{Name: "code-rules", Typeflag: tar.TypeReg},
	} {
		t.Run(header.Name+string(header.Typeflag), func(t *testing.T) {
			var data bytes.Buffer
			compressed := gzip.NewWriter(&data)
			archive := tar.NewWriter(compressed)
			for _, name := range []string{"code-rules", "README.txt"} {
				if err := archive.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg}); err != nil {
					t.Fatal(err)
				}
			}
			if err := archive.WriteHeader(&header); err != nil {
				t.Fatal(err)
			}
			if err := archive.Close(); err != nil {
				t.Fatal(err)
			}
			if err := compressed.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := readArchive(data.Bytes(), 0); err == nil {
				t.Fatal("accepted unsafe archive")
			}
		})
	}
}
