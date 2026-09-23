// Verify default PATH setup and explicit opt-out through the installer and real shell profile evaluation.

package distribution

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testInstallerPath(t *testing.T) {
	for _, shell := range []string{"sh", "bash", "zsh", "fish"} {
		t.Run(shell+" preserves profiles and configures a usable path", func(t *testing.T) {
			f := newInstallerFixture(t)
			f.shell = "/bin/" + shell
			profile := filepath.Join(f.home, ".profile")
			switch shell {
			case "bash":
				profile = filepath.Join(f.home, ".bashrc")
			case "zsh":
				f.zdotdir = filepath.Join(f.home, "zsh config")
				profile = filepath.Join(f.zdotdir, ".zshrc")
			case "fish":
				f.configHome = filepath.Join(f.home, "fish config")
				profile = filepath.Join(f.configHome, "fish/config.fish")
			}
			if err := os.MkdirAll(filepath.Dir(profile), 0755); err != nil {
				t.Fatal(err)
			}
			original := []byte("# keep this comment without a final newline")
			f.write(profile, original, 0644)
			// Quotes, glob characters, backslashes, and substitutions must stay literal.
			destination := filepath.Join(f.root, "it's a \\ * $(touch unwanted) `touch unwanted` bin")
			output := f.success("--install-dir", destination)
			first := f.read(profile)
			if !strings.HasPrefix(string(first), string(original)+"\n") || !strings.Contains(output, profile) || !strings.Contains(output, "Open a new terminal") {
				t.Fatal("profile content or result message", string(first), output)
			}
			f.success("--install-dir", destination)
			f.content(profile, first)
			if shell == "zsh" {
				f.absent(filepath.Join(f.home, ".zshrc"))
			}
			if shell == "fish" {
				f.absent(filepath.Join(f.home, ".config/fish/config.fish"))
			}
			// Run the emitted setup twice with the real shell to check escaping and PATH deduplication.
			binary, err := exec.LookPath(shell)
			if err != nil {
				if os.Getenv("INSTALLER_REQUIRE_SHELLS") == "1" {
					t.Fatalf("required shell unavailable: %s", shell)
				}
				t.Skipf("profile written and checked; %s unavailable for evaluation", shell)
			}
			command := `. "$1"; . "$1"; command -v code-rules; printf '%s\n' "$PATH"`
			args := []string{"-c", command, "profile-check", profile}
			if shell == "fish" {
				args = []string{"--no-config", "-c", `source $argv[1]; source $argv[1]; command -s code-rules; string join : $PATH`, profile}
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, args...)
			cmd.Dir = f.root
			cmd.Env = f.environment()
			checked, err := cmd.CombinedOutput()
			lines := strings.Split(strings.TrimSpace(string(checked)), "\n")
			if err != nil || len(lines) != 2 || lines[0] != filepath.Join(destination, "code-rules") || strings.Count(lines[1], destination) != 1 {
				t.Fatalf("profile failed in %s: %v\n%s\n%s", shell, err, first, checked)
			}
			f.absent(filepath.Join(f.root, "unwanted"))
			if shell == "bash" {
				// Both login and non-login startup paths receive identical, guarded setup.
				if !strings.Contains(string(f.read(filepath.Join(f.home, ".profile"))), "case ") {
					t.Fatal("missing login setup")
				}
			}
		})
	}
	t.Run("opt-out preserves existing profiles and creates none", func(t *testing.T) {
		for _, shell := range []string{"sh", "bash", "zsh", "fish"} {
			t.Run(shell, func(t *testing.T) {
				f := newInstallerFixture(t)
				f.shell = "/bin/" + shell
				profiles := []string{".profile", ".bashrc", ".bash_profile", ".bash_login", ".zshrc", ".config/fish/config.fish"}
				output := f.success("--no-update-path")
				if !strings.Contains(output, "Shell configuration was left unchanged") {
					t.Fatal(output)
				}
				for _, name := range profiles {
					profile := filepath.Join(f.home, name)
					f.absent(profile)
					if err := os.MkdirAll(filepath.Dir(profile), 0755); err != nil {
						t.Fatal(err)
					}
					f.write(profile, []byte("# preserve without final newline"), 0644)
				}
				f.success("--no-update-path")
				for _, name := range profiles {
					f.content(filepath.Join(f.home, name), []byte("# preserve without final newline"))
				}
			})
		}
	})
	t.Run("bash honors first existing login file", func(t *testing.T) {
		for _, login := range []string{".bash_profile", ".bash_login", ".profile"} {
			t.Run(login, func(t *testing.T) {
				f := newInstallerFixture(t)
				f.shell = "/bin/bash"
				profile := filepath.Join(f.home, login)
				f.write(profile, []byte("# login\n"), 0644)
				output := f.success()
				if !strings.Contains(output, profile) || !strings.Contains(string(f.read(profile)), "export PATH=") {
					t.Fatal(output)
				}
				if login != ".bash_profile" {
					f.absent(filepath.Join(f.home, ".bash_profile"))
				}
			})
		}
	})
	t.Run("failed install leaves profile unchanged", func(t *testing.T) {
		f := newInstallerFixture(t)
		profile := filepath.Join(f.home, ".profile")
		original := []byte("# preserve\n")
		f.write(profile, original, 0644)
		f.binary = []byte("#!/bin/sh\nexit 1\n")
		f.release("1.2.3", "linux_amd64", nil)
		f.failure("downloaded executable cannot run")
		f.content(profile, original)
	})
	t.Run("unknown shell leaves profiles alone", func(t *testing.T) {
		f := newInstallerFixture(t)
		f.shell = "/bin/unknown"
		output := f.success()
		if !strings.Contains(output, "automatic PATH setup could not be completed") || !strings.Contains(output, "export PATH=") {
			t.Fatal(output)
		}
		f.absent(filepath.Join(f.home, ".profile"))
		f.content(filepath.Join(f.destination, "code-rules"), f.binary)
	})
	t.Run("unsafe profile remains untouched with manual fallback", func(t *testing.T) {
		for _, kind := range []string{"symlink", "directory", "read-only"} {
			t.Run(kind, func(t *testing.T) {
				f := newInstallerFixture(t)
				profile := filepath.Join(f.home, ".profile")
				target := filepath.Join(f.root, "managed-profile")
				original := []byte("# owned elsewhere\n")
				f.write(target, original, 0644)
				switch kind {
				case "symlink":
					if err := os.Symlink(target, profile); err != nil {
						t.Fatal(err)
					}
				case "directory":
					if err := os.Mkdir(profile, 0755); err != nil {
						t.Fatal(err)
					}
				case "read-only":
					if os.Geteuid() == 0 {
						t.Skip("root bypasses permission checks")
					}
					f.write(profile, original, 0444)
				}
				output := f.success()
				if !strings.Contains(output, "automatic PATH setup could not be completed") || !strings.Contains(output, "export PATH=") {
					t.Fatal(output)
				}
				f.content(target, original)
				if kind == "read-only" {
					f.content(profile, original)
				}
				f.content(filepath.Join(f.destination, "code-rules"), f.binary)
			})
		}
	})
}
