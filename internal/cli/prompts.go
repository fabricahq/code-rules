// Collect terminal-only input before taking any authoring writer lock.

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
		return "", usage(fmt.Errorf("%s Missing input; supply explicit flags in non-interactive mode", label))
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
		ended := fmt.Errorf("terminal input ended; no files were written: %w", err)
		if errors.Is(err, io.EOF) {
			return "", usage(ended)
		}
		return "", ended
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
