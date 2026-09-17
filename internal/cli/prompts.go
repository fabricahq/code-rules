// Collect terminal-only input before taking any authoring writer lock.

package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fabricahq/code-rules/internal/authoring"
	"github.com/fabricahq/code-rules/internal/rules"
	"golang.org/x/sys/unix"
)

// interactive requires terminal input and diagnostics, and honors explicit unattended mode.
func (f *authoringFlags) interactive() bool {
	disabled, _ := f.command.Flags().GetBool("non-interactive")
	in, inputFile := f.command.InOrStdin().(*os.File)
	out, outputFile := f.command.ErrOrStderr().(*os.File)
	return !disabled && inputFile && outputFile && terminalFile(in) && terminalFile(out)
}

// ask reads one bounded canonical terminal line, polling so cancellation needs no abandoned reader goroutine.
func (f *authoringFlags) ask(label string) (string, error) {
	if !f.interactive() {
		return "", fmt.Errorf("%s Missing input; supply explicit flags in non-interactive mode", label)
	}
	if _, err := fmt.Fprint(f.command.ErrOrStderr(), label+" "); err != nil {
		return "", err
	}
	input := f.command.InOrStdin().(*os.File)
	fd := int(input.Fd())
	var line strings.Builder
	var one [1]byte
	for {
		if err := f.command.Context().Err(); err != nil {
			return "", err
		}
		ready := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		n, err := unix.Poll(ready, 100)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("wait for terminal input: %w", err)
		}
		if n == 0 {
			continue
		}
		if ready[0].Revents&(unix.POLLERR|unix.POLLNVAL) != 0 {
			return "", fmt.Errorf("terminal input is unavailable")
		}
		n, err = unix.Read(fd, one[:])
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("read terminal input: %w", err)
		}
		if n == 0 {
			return "", fmt.Errorf("input ended; no files were written: %w", io.EOF)
		}
		if one[0] == '\n' {
			return strings.TrimSpace(line.String()), nil
		}
		if line.Len() >= 65536 {
			return "", fmt.Errorf("terminal answer exceeds 64 KiB")
		}
		line.WriteByte(one[0])
	}
}

// collectSource prompts only for missing source inputs and preserves explicitly supplied flags.
func (f *authoringFlags) collectSource(groups *[]string) error {
	if f.value("ref") != "" && f.value("version") != "" {
		return fmt.Errorf("specify exactly one of --ref or --version")
	}
	if err := f.require("repository"); err != nil {
		return err
	}
	if f.value("ref") == "" && f.value("version") == "" {
		kind, err := f.ask("Revision kind (ref or version):")
		if err != nil {
			return err
		}
		if kind != "ref" && kind != "version" {
			return fmt.Errorf("choose ref or version")
		}
		if err := f.require(kind); err != nil {
			return err
		}
	}
	if len(*groups) == 0 {
		text, err := f.ask("Groups (comma-separated IDs, *, practices/*, or techs/*):")
		if err != nil {
			return err
		}
		if text == "" {
			return fmt.Errorf("provide --groups")
		}
		for _, group := range strings.Split(text, ",") {
			*groups = append(*groups, strings.TrimSpace(group))
		}
	}
	return nil
}

// offerGroup asks before creating missing metadata; publication validates it again under its writer lock.
func (f *authoringFlags) offerGroup(id string, create *bool, library bool) error {
	if !*create {
		for _, name := range []string{"group-name", "group-description", "group-when-to-read"} {
			if f.value(name) != "" {
				return fmt.Errorf("group metadata requires --create-group")
			}
		}
	}
	group, err := rules.GroupFromPath(id+".md", "rule")
	if err != nil {
		return err
	}
	if *create || !f.interactive() {
		return nil
	}
	var exists bool
	if library {
		exists, err = authoring.HasLibraryGroup(f.command.Context(), group, f.libraryOptions())
	} else {
		exists, err = authoring.HasLocalRuleGroup(f.command.Context(), group, f.options())
	}
	if err != nil || exists {
		return err
	}
	answer, err := f.ask("Create missing group " + group + "? [y/N]")
	if err != nil {
		return err
	}
	if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
		return fmt.Errorf("cancelled; no rule was created")
	}
	*create = true
	return nil
}
