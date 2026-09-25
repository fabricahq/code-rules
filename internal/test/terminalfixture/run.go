// Package terminalfixture runs reviewed native CLI demonstrations in an isolated pseudo-terminal.
package terminalfixture

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

// Step waits for a visible prompt before supplying one answer or an explicit interruption.
type Step struct {
	Prompt    string `json:"prompt"`
	Answer    string `json:"answer,omitempty"`
	Interrupt bool   `json:"interrupt,omitempty"`
	EOF       bool   `json:"eof,omitempty"`
}

// Result separates machine output from the real terminal conversation and process status.
type Result struct {
	Stdout      string `json:"stdout"`
	Transcript  string `json:"transcript"`
	ExitCode    int    `json:"exitCode"`
	AnswersSent int    `json:"answersSent"`
}

// Run owns a child terminal, bounded output, and cancellation, without executing a shell or changing parent streams.
func Run(ctx context.Context, binary, directory string, args []string, steps []Step) (Result, error) {
	if len(steps) > 20 {
		return Result{}, fmt.Errorf("at most 20 prompt answers are allowed")
	}
	for _, step := range steps {
		if step.Prompt == "" || len(step.Answer) > 4096 || strings.ContainsAny(step.Answer, "\r\n\x00") {
			return Result{}, fmt.Errorf("each step needs a prompt and one bounded answer line")
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	master, slave, err := pty.Open()
	if err != nil {
		return Result{}, err
	}
	defer master.Close()
	defer slave.Close()
	if err := unix.SetNonblock(int(master.Fd()), true); err != nil {
		return Result{}, err
	}
	command := exec.CommandContext(ctx, binary, args...)
	command.Dir = directory
	command.Env = append(os.Environ(), "PATH="+directory+"/no-runtime")
	command.Stdin, command.Stderr = slave, slave
	var stdout bytes.Buffer
	command.Stdout = &stdout
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	command.Cancel = func() error { return command.Process.Signal(os.Interrupt) }
	command.WaitDelay = time.Second
	if err := command.Start(); err != nil {
		return Result{}, err
	}
	// Keep the parent's slave open until capture has drained the master: macOS can discard
	// unread output when the last slave descriptor closes, which loses a quick child's output.
	exited := make(chan struct{})
	var waitErr error
	go func() {
		waitErr = command.Wait()
		close(exited)
	}()
	var transcript strings.Builder
	sent, offset := 0, 0
	readErr := capture(ctx, master, command, exited, steps, &transcript, &sent, &offset)
	if readErr != nil {
		cancel()
	}
	<-exited
	result := Result{Stdout: stdout.String(), Transcript: transcript.String(), AnswersSent: sent}
	if readErr != nil {
		return result, readErr
	}
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if waitErr != nil {
		var exit *exec.ExitError
		if !errors.As(waitErr, &exit) {
			return result, waitErr
		}
		result.ExitCode = exit.ExitCode()
	}
	if sent != len(steps) {
		return result, fmt.Errorf("process ended before prompt %q", steps[sent].Prompt)
	}
	return result, nil
}

// capture drains terminal bytes and sends a step only after its expected prompt appears.
// It ends once a poll that began after the process exited finds no more output.
func capture(ctx context.Context, master *os.File, command *exec.Cmd, exited <-chan struct{}, steps []Step, transcript *strings.Builder, sent, offset *int) error {
	fd := int(master.Fd())
	buffer := make([]byte, 4096)
	var pending []byte
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		finished := false
		select {
		case <-exited:
			finished, pending = true, nil
		default:
		}
		ready := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		if len(pending) > 0 {
			ready[0].Events |= unix.POLLOUT
		}
		n, err := unix.Poll(ready, 100)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return err
		}
		if n == 0 {
			if finished {
				return nil
			}
			continue
		}
		if ready[0].Revents&unix.POLLOUT != 0 && len(pending) > 0 {
			written, writeErr := unix.Write(fd, pending[:min(len(pending), 256)])
			if written > 0 {
				pending = pending[written:]
			}
			if writeErr != nil && writeErr != unix.EAGAIN && writeErr != unix.EINTR {
				return writeErr
			}
		}
		if ready[0].Revents&(unix.POLLIN|unix.POLLHUP|unix.POLLERR) == 0 {
			continue
		}
		n, err = unix.Read(fd, buffer)
		if err == unix.EINTR || err == unix.EAGAIN {
			continue
		}
		if err == unix.EIO || n == 0 && err == nil {
			return nil
		}
		if err != nil {
			return err
		}
		if transcript.Len()+n > 2*1024*1024 {
			return fmt.Errorf("terminal output exceeds 2 MiB")
		}
		transcript.Write(buffer[:n])
		if *sent >= len(steps) {
			continue
		}
		step := steps[*sent]
		index := strings.Index(transcript.String()[*offset:], step.Prompt)
		if index < 0 {
			continue
		}
		*offset += index + len(step.Prompt)
		*sent++
		switch {
		case step.Interrupt:
			err = command.Process.Signal(os.Interrupt)
		case step.EOF:
			pending = []byte{4}
		default:
			pending = []byte(step.Answer + "\r")
		}
		if err != nil {
			return err
		}
	}
}
