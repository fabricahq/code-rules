// Verify and extract a locally built artifact into a new installation directory.

package distribution

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
)

const maxArchiveBytes = 256 * 1024 * 1024

// Install verifies the selected archive against its manifest before creating a new installation.
// Checksums detect corruption, not malicious replacement of both archive and manifest; use trusted artifacts.
func Install(directory, target, destination string) (Manifest, error) {
	if _, err := os.Lstat(filepath.Join(directory, "INCOMPLETE")); !os.IsNotExist(err) {
		return Manifest{}, fmt.Errorf("artifact build is incomplete or unreadable")
	}
	metadata, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err = json.Unmarshal(metadata, &manifest); err != nil {
		return Manifest{}, err
	}
	var artifact *Artifact
	for i := range manifest.Artifacts {
		if manifest.Artifacts[i].Target == target {
			if artifact != nil {
				return Manifest{}, fmt.Errorf("duplicate artifact target")
			}
			artifact = &manifest.Artifacts[i]
		}
	}
	if artifact == nil || !slices.Contains(supported, target) {
		return Manifest{}, fmt.Errorf("no artifact for %s", target)
	}
	if filepath.Base(artifact.File) != artifact.File || artifact.File == "." || artifact.Bytes <= 0 || artifact.Bytes > maxArchiveBytes {
		return Manifest{}, fmt.Errorf("invalid artifact metadata")
	}
	file, err := os.Open(filepath.Join(directory, artifact.File))
	if err != nil {
		return Manifest{}, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxArchiveBytes+1))
	closeErr := file.Close()
	if readErr != nil {
		return Manifest{}, readErr
	}
	if closeErr != nil {
		return Manifest{}, closeErr
	}
	sum := sha256.Sum256(data)
	if int64(len(data)) != artifact.Bytes || hex.EncodeToString(sum[:]) != artifact.SHA256 {
		return Manifest{}, fmt.Errorf("artifact checksum or size mismatch; installation was not changed")
	}
	entries, err := readArchive(data, artifact.BinaryBytes)
	if err != nil {
		return Manifest{}, err
	}
	if err = os.Mkdir(destination, 0755); err != nil {
		return Manifest{}, fmt.Errorf("installation directory must be new: %w", err)
	}
	for _, entry := range entries {
		file, err := os.OpenFile(filepath.Join(destination, entry.name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(entry.mode))
		if err != nil {
			return Manifest{}, err
		}
		_, writeErr := file.Write(entry.data)
		closeErr := file.Close()
		if writeErr != nil {
			return Manifest{}, writeErr
		}
		if closeErr != nil {
			return Manifest{}, closeErr
		}
	}
	return manifest, nil
}

// readArchive validates the entire bounded archive before any installation file is written.
func readArchive(data []byte, binaryBytes int64) ([]archiveEntry, error) {
	compressed, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer compressed.Close()
	archive := tar.NewReader(compressed)
	seen := map[string]bool{}
	entries := []archiveEntry{}
	var total int64
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag != tar.TypeReg || seen[header.Name] || !slices.Contains([]string{"code-rules", "README.txt", "LICENSE.md"}, header.Name) {
			return nil, fmt.Errorf("unexpected archive member %q", header.Name)
		}
		if header.Size < 0 || header.Size > maxArchiveBytes-total {
			return nil, fmt.Errorf("archive exceeds extraction limit")
		}
		total += header.Size
		content, err := io.ReadAll(archive)
		if err != nil {
			return nil, err
		}
		mode := int64(0644)
		if header.Name == "code-rules" {
			mode = 0755
			if int64(len(content)) != binaryBytes {
				return nil, fmt.Errorf("binary size mismatch")
			}
		}
		entries = append(entries, archiveEntry{header.Name, content, mode})
		seen[header.Name] = true
	}
	// Drain gzip's footer so a valid tar prefix cannot conceal a corrupt compressed stream.
	if n, err := io.Copy(io.Discard, io.LimitReader(compressed, maxArchiveBytes+1)); err != nil {
		return nil, err
	} else if n != 0 {
		return nil, fmt.Errorf("unexpected data after tar archive")
	}
	if !seen["code-rules"] || !seen["README.txt"] {
		return nil, fmt.Errorf("archive is missing the executable or readme")
	}
	return entries, nil
}
