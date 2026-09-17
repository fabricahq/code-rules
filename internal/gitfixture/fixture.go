// Package gitfixture supplies disposable real Git repositories for tests and the development lab.
package gitfixture

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Fixture owns a local repository and a private SSH transport that runs Git upload-pack over stdio.
// No listener, external network, user Git configuration, or process-wide environment change is needed.
type Fixture struct {
	Directory    string
	Repository   string
	FirstCommit  string
	LatestCommit string
	GitPath      string
	Environment  []string
}

// New creates two real commits, a lightweight v1.0.0 tag, and an annotated v1.2.0 tag.
func New(ctx context.Context, files map[string][]byte) (_ *Fixture, err error) {
	executable, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("fixture requires Git: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "code-rules-git-fixture-*")
	if err != nil {
		return nil, err
	}
	f := &Fixture{Directory: dir, Repository: "git@fixture.invalid:rules", GitPath: executable}
	defer func() {
		if err != nil {
			err = errors.Join(err, f.Close())
		}
	}()
	home := filepath.Join(dir, "home")
	repo := filepath.Join(dir, "repository")
	for _, name := range []string{home, repo} {
		if err := os.Mkdir(name, 0700); err != nil {
			return nil, err
		}
	}
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if !strings.HasPrefix(key, "GIT_") && key != "HOME" && key != "XDG_CONFIG_HOME" {
			f.Environment = append(f.Environment, item)
		}
	}
	f.Environment = append(f.Environment, "HOME="+home, "XDG_CONFIG_HOME="+home, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid", "GIT_COMMITTER_NAME=Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid", "GIT_TERMINAL_PROMPT=0")
	if _, err = f.Command(ctx, "init", "--quiet", "--template=", "--initial-branch=main"); err != nil {
		return nil, err
	}
	for name, data := range files {
		if !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, "\\\x00") {
			return nil, fmt.Errorf("invalid fixture file path")
		}
		for _, part := range strings.Split(name, "/") {
			if strings.EqualFold(part, ".git") {
				return nil, fmt.Errorf("reserved fixture file path")
			}
		}
		file := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(file, data, 0600); err != nil {
			return nil, err
		}
	}
	if _, err = f.Command(ctx, "add", "--all"); err != nil {
		return nil, err
	}
	if _, err = f.Command(ctx, "-c", "commit.gpgsign=false", "commit", "--quiet", "--allow-empty", "-m", "Fixture first revision"); err != nil {
		return nil, err
	}
	f.FirstCommit, err = f.Command(ctx, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if _, err = f.Command(ctx, "tag", "v1.0.0"); err != nil {
		return nil, err
	}
	if _, err = f.Command(ctx, "-c", "commit.gpgsign=false", "commit", "--quiet", "--allow-empty", "-m", "Fixture second revision"); err != nil {
		return nil, err
	}
	f.LatestCommit, err = f.Command(ctx, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if _, err = f.Command(ctx, "-c", "tag.gpgsign=false", "tag", "-a", "v1.2.0", "-m", "Fixture annotated release"); err != nil {
		return nil, err
	}
	transport := filepath.Join(dir, "ssh")
	// This trusted helper ignores remote arguments and serves only this owned fixture repository.
	script := "#!/bin/sh\nexec " + Quote(executable) + " upload-pack " + Quote(repo) + "\n"
	if err := os.WriteFile(transport, []byte(script), 0700); err != nil {
		return nil, err
	}
	f.Environment = append(f.Environment, "GIT_SSH_COMMAND="+Quote(transport), "GIT_SSH_VARIANT=simple")
	return f, nil
}

// Command mutates or inspects only the owned fixture repository using isolated Git configuration.
func (f *Fixture) Command(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, f.GitPath, append([]string{"-c", "core.hooksPath=/dev/null"}, args...)...)
	cmd.Dir = filepath.Join(f.Directory, "repository")
	cmd.Env = f.Environment
	data, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("fixture Git setup failed: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// Close releases all repository and helper files owned by the fixture.
func (f *Fixture) Close() error {
	if f == nil || f.Directory == "" {
		return nil
	}
	return os.RemoveAll(f.Directory)
}

// Quote escapes a trusted local path for a generated shell helper's literal argument.
func Quote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
