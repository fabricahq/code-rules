// Validate archive boundaries independently of the packaging implementation.

package distribution

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
				if err := archive.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Size: 1}); err != nil {
					t.Fatal(err)
				}
				if _, err := archive.Write([]byte("x")); err != nil {
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
			if _, err := readArchive(data.Bytes(), 1); err == nil || !strings.Contains(err.Error(), "unexpected archive member") {
				t.Fatal("unsafe member was not rejected at member validation", err)
			}
		})
	}
}

// TestInstallWriteFailureRemovesPartialDestination verifies retry works after a later member fails.
func TestInstallWriteFailureRemovesPartialDestination(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "installed")
	entries := []archiveEntry{{"code-rules", []byte("binary"), 0755}, {"README.txt", []byte("readme"), 0644}}
	failure := errors.New("disk full")
	err := installEntries(destination, entries, func(dir string, entry archiveEntry) error {
		if entry.name == "README.txt" {
			return failure
		}
		return writeEntry(dir, entry)
	})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("partial install left behind", err)
	}
	if err := installEntries(destination, entries, writeEntry); err != nil {
		t.Fatal("retry failed", err)
	}
}

// cancelWriter simulates cancellation once compressed output starts.
type cancelWriter struct{ cancel context.CancelFunc }

// Write accepts one output chunk and cancels subsequent writes.
func (w cancelWriter) Write(data []byte) (int, error) { w.cancel(); return len(data), nil }

// TestArchiveCancellation prevents a final target's compression from masking interruption.
func TestArchiveCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := writeArchive(ctx, cancelWriter{cancel}, []archiveEntry{{"code-rules", bytes.Repeat([]byte("data"), 10000), 0755}})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

// TestRejectEmptyExecutable refuses a correctly encoded archive whose executable is empty.
func TestRejectEmptyExecutable(t *testing.T) {
	var data bytes.Buffer
	if err := writeArchive(context.Background(), &data, []archiveEntry{{"code-rules", nil, 0755}, {"README.txt", []byte("readme"), 0644}}); err != nil {
		t.Fatal(err)
	}
	if _, err := readArchive(data.Bytes(), 0); err == nil {
		t.Fatal("accepted empty executable")
	}
}
