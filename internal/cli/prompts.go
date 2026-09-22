// Collect terminal-only input before taking any authoring writer lock.

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fabricahq/code-rules/internal/library"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// interactive requires terminal input and diagnostics, and honors explicit unattended mode.
func (f *authoringFlags) interactive() bool {
	disabled, _ := f.command.Flags().GetBool("non-interactive")
	structured, _ := f.command.Flags().GetBool("json")
	in, inputFile := f.command.InOrStdin().(*os.File)
	out, outputFile := f.command.ErrOrStderr().(*os.File)
	return !disabled && !structured && inputFile && outputFile && term.IsTerminal(int(in.Fd())) && term.IsTerminal(int(out.Fd()))
}

// ask uses Go's terminal editor for pasted text and restores terminal settings on every return path.
func (f *authoringFlags) ask(label string) (answer string, err error) {
	if !f.interactive() {
		return "", fmt.Errorf("%s Missing input; supply explicit flags in non-interactive mode", label)
	}
	if f.introduction != "" {
		if _, err := io.WriteString(f.command.ErrOrStderr(), f.introduction); err != nil {
			return "", err
		}
		f.introduction = ""
	}
	input := f.command.InOrStdin().(*os.File)
	fd := int(input.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, term.Restore(fd, state)) }()
	terminal := term.NewTerminal(&promptStream{ctx: f.command.Context(), fd: fd, output: f.command.ErrOrStderr()}, label+" ")
	if width, height, sizeErr := term.GetSize(fd); sizeErr == nil && width > 0 && height > 0 {
		if err := terminal.SetSize(width, height); err != nil {
			return "", err
		}
	}
	answer, err = terminal.ReadLine()
	if err != nil {
		return "", fmt.Errorf("terminal input ended; no files were written: %w", err)
	}
	f.prompted = true
	return strings.TrimSpace(answer), nil
}

// promptStream preserves stderr output while polling input for cancellation without a background reader.
type promptStream struct {
	ctx       context.Context
	fd        int
	output    io.Writer
	bytesRead int
}

// Write forwards editor output to the caller's diagnostic terminal.
func (p *promptStream) Write(data []byte) (int, error) { return p.output.Write(data) }

// Read supplies one byte at a time so terminal buffering cannot consume the next prompt's answer.
func (p *promptStream) Read(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	for {
		if err := p.ctx.Err(); err != nil {
			return 0, err
		}
		ready := []unix.PollFd{{Fd: int32(p.fd), Events: unix.POLLIN}}
		n, err := unix.Poll(ready, 100)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return 0, err
		}
		if n == 0 {
			continue
		}
		if ready[0].Revents&(unix.POLLERR|unix.POLLNVAL) != 0 {
			return 0, fmt.Errorf("terminal input is unavailable")
		}
		n, err = unix.Read(p.fd, data[:1])
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, io.EOF
		}
		if data[0] == 3 {
			return 0, context.Canceled
		}
		if data[0] != '\n' && data[0] != '\r' {
			p.bytesRead++
			if p.bytesRead > 4096 {
				return 0, fmt.Errorf("terminal answer exceeds 4096 input bytes")
			}
		}
		return n, nil
	}
}

// collectSource prompts only for missing source inputs and preserves explicitly supplied flags.
func (f *authoringFlags) collectSource(groups *[]string) error {
	if f.value("ref") != "" {
		if _, _, err := libraryRef(f.value("ref")); err != nil {
			return err
		}
	}
	if err := f.require("repository", "ref"); err != nil {
		return err
	}
	if len(*groups) == 0 {
		text, err := f.askValidated("Groups (comma-separated paths, *, practices/*, or techs/*):", func(value string) error {
			if err := validateAnswer("groups", value); err != nil {
				return err
			}
			data, err := json.Marshal(sourceGroupSelection(splitPromptGroups(value)))
			if err != nil {
				return err
			}
			_, err = rules.ParseGroupSelection(data, "--groups")
			return err
		})
		if err != nil {
			return err
		}
		*groups = splitPromptGroups(text)
	}
	return nil
}

// requireRuleGroup refuses absent metadata before any prompts; publication rechecks under its writer lock.
func (f *authoringFlags) requireRuleGroup(id string, isLibrary bool) error {
	group, err := rules.GroupFromPath(id+".md", "rule")
	if err != nil {
		return err
	}
	var exists bool
	command := "code-rules project add group " + group
	if isLibrary {
		exists, err = library.HasGroup(f.command.Context(), group, f.libraryOptions())
		command = "code-rules library add group " + group
		if directory := f.value("directory"); directory != "" {
			command += " --directory='" + strings.ReplaceAll(directory, "'", "'\"'\"'") + "'"
		}
	} else {
		exists, err = project.HasLocalRuleGroup(f.command.Context(), group, f.options())
	}
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("group %s does not exist; create it first with %s, then retry adding the rule", group, command)
	}
	return nil
}
