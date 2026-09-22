// Package distribution builds reviewable native archives without publishing or changing installed tools.
package distribution

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fabricahq/code-rules/internal/rules"
)

// Options selects trusted build inputs and an exclusively created output directory.
type Options struct {
	Source, Output, Version string
	Targets                 []string
	Candidate               bool
}

// Artifact identifies an archive and its exact integrity and size measurements.
type Artifact struct {
	Target      string `json:"target"`
	File        string `json:"file"`
	SHA256      string `json:"sha256"`
	Bytes       int64  `json:"bytes"`
	BinaryBytes int64  `json:"binaryBytes"`
}

// Manifest records source identity and review status; it does not imply publication approval.
type Manifest struct {
	Version        string     `json:"version"`
	Candidate      bool       `json:"candidate"`
	License        string     `json:"license"`
	SourceRevision string     `json:"sourceRevision"`
	SourceDirty    bool       `json:"sourceDirty"`
	GoVersion      string     `json:"goVersion"`
	Artifacts      []Artifact `json:"artifacts"`
}

var supported = []string{"darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64"}

// Build compiles an isolated copy of committed HEAD (working edits are excluded), packages deterministic archive entries, and writes a manifest only after every target succeeds.
// Output must not exist. A failed build retains an INCOMPLETE marker for inspection; retry into a new directory.
func Build(ctx context.Context, options Options) (Manifest, error) {
	source, err := filepath.Abs(options.Source)
	if err != nil {
		return Manifest{}, err
	}
	options.Output, err = filepath.Abs(options.Output)
	if err != nil {
		return Manifest{}, err
	}
	revision, err := output(ctx, source, "git", "rev-parse", "HEAD")
	if err != nil {
		return Manifest{}, err
	}
	dirty, err := output(ctx, source, "git", "status", "--porcelain", "--untracked-files=normal")
	if err != nil {
		return Manifest{}, err
	}
	if !options.Candidate && dirty != "" {
		return Manifest{}, fmt.Errorf("release source must be clean")
	}
	captured, err := committedSource(ctx, source, revision)
	if err != nil {
		return Manifest{}, err
	}
	defer os.RemoveAll(captured)
	source = filepath.Join(captured, "tree")
	if options.Version == "" && options.Candidate {
		options.Version = "0.0.0-dev.g" + revision[:12]
	}
	version, err := rules.TagVersion(options.Version, "version")
	if err != nil || version != options.Version {
		return Manifest{}, fmt.Errorf("expected an unprefixed complete version")
	}
	license, licenseErr := os.ReadFile(filepath.Join(source, "LICENSE.md"))
	if !options.Candidate && (licenseErr != nil || len(strings.TrimSpace(string(license))) == 0) {
		return Manifest{}, fmt.Errorf("release artifacts require a nonempty LICENSE.md; use --candidate for unpublished review builds")
	}
	if licenseErr != nil && !os.IsNotExist(licenseErr) {
		return Manifest{}, licenseErr
	}
	targets := slices.Clone(options.Targets)
	if len(targets) == 0 {
		targets = slices.Clone(supported)
	}
	slices.Sort(targets)
	for i, target := range targets {
		if !slices.Contains(supported, target) || i > 0 && target == targets[i-1] {
			return Manifest{}, fmt.Errorf("unsupported or duplicate target %q", target)
		}
	}
	goVersion, err := output(ctx, source, "go", "version")
	if err != nil {
		return Manifest{}, err
	}
	terms := "UNLICENSED"
	if len(strings.TrimSpace(string(license))) > 0 {
		terms = "SEE LICENSE.md"
	}
	manifest := Manifest{Version: version, Candidate: options.Candidate, License: terms, SourceRevision: revision, SourceDirty: dirty != "", GoVersion: goVersion, Artifacts: []Artifact{}}
	if err = os.Mkdir(options.Output, 0755); err != nil {
		return Manifest{}, fmt.Errorf("create new output directory: %w", err)
	}
	marker := filepath.Join(options.Output, "INCOMPLETE")
	if err = os.WriteFile(marker, []byte("Build incomplete. No manifest is valid until this marker is removed.\n"), 0600); err != nil {
		return Manifest{}, err
	}
	for _, target := range targets {
		artifact, err := buildTarget(ctx, source, options.Output, version, revision, target, license, options.Candidate)
		if err != nil {
			return Manifest{}, err
		}
		manifest.Artifacts = append(manifest.Artifacts, artifact)
	}
	if err := ctx.Err(); err != nil {
		return Manifest{}, err
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Manifest{}, err
	}
	if err = os.WriteFile(filepath.Join(options.Output, "manifest.json"), append(encoded, '\n'), 0644); err != nil {
		return Manifest{}, err
	}
	var checksums strings.Builder
	for _, artifact := range manifest.Artifacts {
		fmt.Fprintf(&checksums, "%s  %s\n", artifact.SHA256, artifact.File)
	}
	if err = os.WriteFile(filepath.Join(options.Output, "SHA256SUMS"), []byte(checksums.String()), 0644); err != nil {
		return Manifest{}, err
	}
	if err := ctx.Err(); err != nil {
		return Manifest{}, err
	}
	if err = os.Remove(marker); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// output runs a trusted development tool with captured diagnostics and no shell interpolation.
func output(ctx context.Context, directory, name string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = directory
	result, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(result)))
	}
	return strings.TrimSpace(string(result)), nil
}

// buildTarget creates one statically linked native executable and archives it with original tool terms.
func buildTarget(ctx context.Context, source, directory, version, revision, target string, license []byte, candidate bool) (Artifact, error) {
	temporary, err := os.MkdirTemp(directory, ".build-")
	if err != nil {
		return Artifact{}, err
	}
	defer os.RemoveAll(temporary)
	binary := filepath.Join(temporary, "code-rules")
	platform := strings.Split(target, "/")
	flags := "-s -w -X main.version=" + version
	if candidate {
		flags += " -X main.previewCommit=" + revision
	}
	command := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-ldflags", flags, "-o", binary, "./cmd/code-rules")
	command.Dir = source
	command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+platform[0], "GOARCH="+platform[1], "GOWORK=off", "GOFLAGS=", "GOENV=off", "GOAMD64=v1", "GOARM64=v8.0", "GOEXPERIMENT=")
	if text, err := command.CombinedOutput(); err != nil {
		return Artifact{}, fmt.Errorf("build %s: %w: %s", target, err, text)
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		return Artifact{}, err
	}
	name := "code-rules_" + version + "_" + strings.ReplaceAll(target, "/", "_") + ".tar.gz"
	note := "Code Rules " + version + "\nExtract code-rules to a private directory and run ./code-rules --help.\nOffline commands need no Node or Bun. Sync also requires Git on PATH.\n"
	if candidate {
		note += "UNPUBLISHED REVIEW CANDIDATE. This archive does not grant a tool license or imply release approval.\n"
	}
	entries := []archiveEntry{{"code-rules", data, 0755}, {"README.txt", []byte(note), 0644}}
	if len(license) > 0 {
		entries = append(entries, archiveEntry{"LICENSE.md", license, 0644})
	}
	file, err := os.OpenFile(filepath.Join(temporary, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return Artifact{}, err
	}
	digest := sha256.New()
	counter := &byteCounter{}
	err = writeArchive(ctx, io.MultiWriter(file, digest, counter), entries)
	closeErr := file.Close()
	if err != nil {
		return Artifact{}, err
	}
	if closeErr != nil {
		return Artifact{}, closeErr
	}
	if err = os.Rename(filepath.Join(temporary, name), filepath.Join(directory, name)); err != nil {
		return Artifact{}, err
	}
	return Artifact{Target: target, File: name, SHA256: hex.EncodeToString(digest.Sum(nil)), Bytes: counter.n, BinaryBytes: int64(len(data))}, nil
}

// archiveEntry fixes each relative archive member's exact bytes and permissions.
type archiveEntry struct {
	name string
	data []byte
	mode int64
}

// writeArchive uses stable ordering and timestamps so repeated builds can produce identical archive bytes.
func writeArchive(ctx context.Context, output io.Writer, entries []archiveEntry) error {
	gzipWriter := gzip.NewWriter(contextWriter{ctx, output})
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		if err := tarWriter.WriteHeader(&tar.Header{Name: entry.name, Mode: entry.mode, Size: int64(len(entry.data)), ModTime: time.Unix(0, 0), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		if _, err := tarWriter.Write(entry.data); err != nil {
			return err
		}
	}
	if err := tarWriter.Close(); err != nil {
		return err
	}
	return gzipWriter.Close()
}

// byteCounter measures compressed bytes without buffering an archive in memory.
type byteCounter struct{ n int64 }

// Write counts bytes forwarded to the archive file and digest.
func (c *byteCounter) Write(data []byte) (int, error) { c.n += int64(len(data)); return len(data), nil }

// committedSource extracts a fixed Git commit into an isolated build tree so edits cannot change artifact inputs.
func committedSource(ctx context.Context, source, revision string) (_ string, err error) {
	directory, err := os.MkdirTemp("", "code-rules-build-source-")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(directory)
		}
	}()
	archivePath := filepath.Join(directory, "source.tar")
	if _, err = output(ctx, source, "git", "archive", "--format=tar", "--output", archivePath, revision); err != nil {
		return "", err
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	root := filepath.Join(directory, "tree")
	if err = os.Mkdir(root, 0700); err != nil {
		return "", err
	}
	reader := tar.NewReader(file)
	var total int64
	for {
		header, nextErr := reader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return "", nextErr
		}
		if header.Typeflag == tar.TypeXGlobalHeader {
			continue // Git archive stores the commit ID in global PAX metadata.
		}
		name := strings.TrimSuffix(header.Name, "/")
		if !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, "\\\x00") {
			return "", fmt.Errorf("unsafe committed source path")
		}
		destination := filepath.Join(root, filepath.FromSlash(name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(destination, 0700); err != nil {
				return "", err
			}
		case tar.TypeReg:
			if header.Size < 0 || header.Size > 256*1024*1024-total {
				return "", fmt.Errorf("committed source exceeds 256 MiB")
			}
			total += header.Size
			if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
				return "", err
			}
			data, readErr := io.ReadAll(reader)
			if readErr != nil {
				return "", readErr
			}
			if err = os.WriteFile(destination, data, 0600); err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("unsupported committed source entry %s", name)
		}
	}
	if err = file.Close(); err != nil {
		return "", err
	}
	if err = os.Remove(archivePath); err != nil {
		return "", err
	}
	// The caller removes the container; return its tree through a dedicated cleanup boundary.
	return directory, nil
}

// contextWriter stops archive compression as soon as cancellation reaches an output write.
type contextWriter struct {
	ctx    context.Context
	writer io.Writer
}

// Write preserves cancellation identity instead of completing an interrupted artifact.
func (w contextWriter) Write(data []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.writer.Write(data)
}
