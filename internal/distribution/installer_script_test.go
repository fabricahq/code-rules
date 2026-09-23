// Exercise standalone installation, upgrades, and refusal paths through the shipped shell script.

package distribution

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestStandaloneInstaller verifies real shell, tar, checksums, and file updates with isolated downloads.
func TestStandaloneInstaller(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("standalone installer supports macOS and Linux")
	}
	t.Run("PATH setup", testInstallerPath)
	t.Run("piped latest installs and sets up PATH", func(t *testing.T) {
		f := newInstallerFixture(t)
		profile := filepath.Join(f.home, ".profile")
		original := []byte("# keep my configuration\n")
		f.write(profile, original, 0644)
		output := f.success()
		binary := filepath.Join(f.destination, "code-rules")
		f.content(binary, f.binary)
		info, err := os.Stat(binary)
		if err != nil || info.Mode()&0111 == 0 {
			t.Fatalf("installed binary is not executable: %v", err)
		}
		f.absent(filepath.Join(f.destination, "code-rules.LICENSE"))
		if !strings.HasPrefix(string(f.read(profile)), string(original)) || !strings.Contains(string(f.read(profile)), "export PATH=") {
			t.Fatal("default installation must preserve the profile and add PATH setup")
		}
		if !strings.Contains(output, "export PATH=") {
			t.Fatal("missing PATH instructions", output)
		}
		matches, err := filepath.Glob(filepath.Join(f.destination, ".code-rules-install.*"))
		if err != nil || len(matches) != 0 {
			t.Fatalf("staging files left behind: %v %v", matches, err)
		}
	})
	t.Run("native executable installs and runs", func(t *testing.T) {
		f := newInstallerFixture(t)
		if err := os.Remove(filepath.Join(f.commands, "uname")); err != nil {
			t.Fatal(err)
		}
		binary := filepath.Join(f.root, "native-code-rules")
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
		defer cancel()
		build := exec.CommandContext(ctx, "go", "build", "-ldflags", "-X main.version=1.2.3", "-o", binary, "./cmd/code-rules")
		build.Dir = filepath.Join("..", "..")
		if output, err := build.CombinedOutput(); err != nil {
			t.Fatalf("build native CLI: %v\n%s", err, output)
		}
		f.binary = f.read(binary)
		f.release("1.2.3", runtime.GOOS+"_"+runtime.GOARCH, nil)
		f.success()
		help := exec.CommandContext(ctx, filepath.Join(f.destination, "code-rules"), "--help")
		help.Dir = f.root
		help.Env = f.environment()
		output, err := help.CombinedOutput()
		if err != nil || !strings.Contains(string(output), "sync") {
			t.Fatalf("installed CLI help: %v\n%s", err, output)
		}
		license := exec.CommandContext(ctx, filepath.Join(f.destination, "code-rules"), "--license")
		license.Dir = f.root
		license.Env = f.environment()
		output, err = license.CombinedOutput()
		if err != nil || string(output) != string(f.read(filepath.Join("..", "..", "LICENSE.md"))) {
			t.Fatalf("installed CLI license: %v\n%s", err, output)
		}
		entries, err := os.ReadDir(f.destination)
		if err != nil || len(entries) != 1 || entries[0].Name() != "code-rules" {
			t.Fatal("installer must place only the executable", entries, err)
		}
	})
	t.Run("all four platforms", func(t *testing.T) {
		for _, platform := range []struct{ os, arch, target string }{{"Darwin", "arm64", "darwin_arm64"}, {"Darwin", "x86_64", "darwin_amd64"}, {"Linux", "aarch64", "linux_arm64"}, {"Linux", "amd64", "linux_amd64"}} {
			t.Run(platform.target, func(t *testing.T) {
				f := newInstallerFixture(t)
				f.platform = platform.os
				f.arch = platform.arch
				f.release("1.2.3", platform.target, nil)
				f.success("--version", "v1.2.3")
			})
		}
	})
	t.Run("upgrade to explicit prerelease in custom directory", func(t *testing.T) {
		f := newInstallerFixture(t)
		destination := filepath.Join(f.root, "custom bin")
		f.success("--install-dir", destination)
		f.binary = []byte("#!/bin/sh\necho 'code-rules 1.3.0-rc.1'\n")
		f.release("1.3.0-rc.1", "linux_amd64", nil)
		f.success("--version", "1.3.0-rc.1", "--install-dir", destination)
		f.content(filepath.Join(destination, "code-rules"), f.binary)
	})
	t.Run("download and checksum failures preserve installed binary", func(t *testing.T) {
		for _, kind := range []string{"missing", "corrupt", "duplicate checksum"} {
			t.Run(kind, func(t *testing.T) {
				f := newInstallerFixture(t)
				f.success()
				archive := f.release("1.2.3", "linux_amd64", nil)
				message := ""
				switch kind {
				case "missing":
					if err := os.Remove(archive); err != nil {
						t.Fatal(err)
					}
					message = "could not download"
				case "corrupt":
					f.write(archive, []byte("corrupted download"), 0644)
					message = "checksum mismatch"
				case "duplicate checksum":
					sums := filepath.Join(filepath.Dir(archive), "SHA256SUMS")
					data := f.read(sums)
					f.write(sums, append(data, data...), 0644)
					message = "exactly one valid entry"
				}
				f.failure(message)
				f.content(filepath.Join(f.destination, "code-rules"), f.binary)
			})
		}
	})
	t.Run("reject unsafe incomplete and duplicate archive members", func(t *testing.T) {
		for _, kind := range []string{"traversal", "symlink", "missing license", "duplicate", "empty executable"} {
			t.Run(kind, func(t *testing.T) {
				f := newInstallerFixture(t)
				binary := installerMember{"code-rules", f.binary, false}
				license := installerMember{"LICENSE.md", []byte("MIT"), false}
				var entries []installerMember
				message := "unexpected or missing release archive files"
				switch kind {
				case "traversal":
					entries = []installerMember{{"../escape", []byte("bad"), false}, binary, license}
				case "symlink":
					entries = []installerMember{{"code-rules", nil, true}, license}
					message = "link or non-regular file"
				case "missing license":
					entries = []installerMember{binary}
				case "duplicate":
					entries = []installerMember{binary, binary, license}
				case "empty executable":
					entries = []installerMember{{"code-rules", nil, false}, license}
					message = "executable or license is empty"
				}
				f.release("1.2.3", "linux_amd64", entries)
				f.failure(message)
				f.absent(filepath.Join(f.destination, "code-rules"))
				f.absent(filepath.Join(f.root, "escape"))
			})
		}
	})
	t.Run("unrunnable download preserves installed binary", func(t *testing.T) {
		f := newInstallerFixture(t)
		f.success()
		f.release("1.2.3", "linux_amd64", []installerMember{{"code-rules", []byte("#!/bin/sh\nexit 1\n"), false}, {"LICENSE.md", []byte("MIT"), false}})
		f.failure("downloaded executable cannot run")
		f.content(filepath.Join(f.destination, "code-rules"), f.binary)
	})
	t.Run("refuse to replace symlink or directory", func(t *testing.T) {
		f := newInstallerFixture(t)
		if err := os.MkdirAll(f.destination, 0755); err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(f.destination, "code-rules")
		target := filepath.Join(f.root, "other-manager-binary")
		f.write(target, []byte("preserve"), 0644)
		if err := os.Symlink(target, destination); err != nil {
			t.Fatal(err)
		}
		f.failure("destination is a symlink")
		f.content(target, []byte("preserve"))
		if err := os.Remove(destination); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(destination, 0755); err != nil {
			t.Fatal(err)
		}
		f.failure("destination is a directory")
		if info, err := os.Stat(destination); err != nil || !info.IsDir() {
			t.Fatalf("directory was replaced: %v", err)
		}
	})
	t.Run("reject unsupported platforms and invalid options", func(t *testing.T) {
		for _, test := range []struct {
			args    []string
			message string
		}{
			{[]string{"--version", "../../bad"}, "version must look like"},
			{[]string{"--version"}, "requires a value"},
			{[]string{"--unknown"}, "unknown option"},
			{[]string{"--install-dir", "relative"}, "must be an absolute path"},
			{[]string{"--install-dir", "/tmp/a:b"}, "cannot contain a colon"},
		} {
			t.Run(strings.Join(test.args, " "), func(t *testing.T) {
				f := newInstallerFixture(t)
				f.failure(test.message, test.args...)
				f.absent(f.destination)
			})
		}
		for _, platform := range []struct{ os, arch, message string }{{"FreeBSD", "x86_64", "supported operating systems"}, {"Linux", "riscv64", "supported processor architectures"}} {
			t.Run(platform.os+platform.arch, func(t *testing.T) {
				f := newInstallerFixture(t)
				f.platform = platform.os
				f.arch = platform.arch
				f.failure(platform.message)
				f.absent(f.destination)
			})
		}
	})
	t.Run("printed PATH command safely quotes custom directory", func(t *testing.T) {
		f := newInstallerFixture(t)
		directory := filepath.Join(f.root, "it's a $(touch unwanted) bin")
		output := f.success("--install-dir", directory)
		line := ""
		for _, candidate := range strings.Split(output, "\n") {
			if strings.HasPrefix(candidate, "export PATH=") {
				line = candidate
				break
			}
		}
		if line == "" {
			t.Fatal("missing PATH command", output)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()
		check := exec.CommandContext(ctx, "sh", "-c", line+"\ncommand -v code-rules")
		check.Dir = f.root
		check.Env = f.environment()
		checked, err := check.CombinedOutput()
		if err != nil || strings.TrimSpace(string(checked)) != filepath.Join(directory, "code-rules") {
			t.Fatalf("PATH command failed: %v\n%s", err, checked)
		}
		f.absent(filepath.Join(f.root, "unwanted"))
	})
	t.Run("help needs no download", func(t *testing.T) {
		f := newInstallerFixture(t)
		if err := os.RemoveAll(f.releases); err != nil {
			t.Fatal(err)
		}
		f.success("--help")
		f.absent(f.destination)
	})
}
