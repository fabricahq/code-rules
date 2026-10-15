// Run the native CLI with process streams and signal-driven cancellation.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fabricahq/code-rules/internal/cli"
)

// version is set by native release builds; development binaries declare their status explicitly.
var version = "0.0.0-development"

// previewCommit is set only by candidate packaging, independently of the chosen version.
var previewCommit string

// main owns process exit after all command resources and signal handlers have been released.
func main() { os.Exit(run()) }

// run translates interruption into cancellation and waits for command cleanup before returning an exit status.
func run() int {
	if previewCommit != "" {
		fmt.Fprintf(os.Stderr, "WARNING: Unreleased preview from commit %s. For testing only; not for production use.\n", previewCommit)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return cli.Run(ctx, os.Args[1:], cli.Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}, cli.Options{Version: version})
}
