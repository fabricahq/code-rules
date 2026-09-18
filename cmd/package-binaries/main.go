// Build native candidate or release archives without publishing them.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fabricahq/code-rules/internal/distribution"
)

// main owns signals and the packaging command exit status.
func main() { os.Exit(run()) }

// run validates explicit output ownership and reports the completed artifact manifest.
func run() int {
	source := flag.String("source", ".", "Source checkout")
	output := flag.String("output", "", "New output directory (must not exist)")
	version := flag.String("version", "", "Semantic version without the leading 'v' (for example, 0.1.0; required for releases; candidates default to 0.0.0-dev.g<commit>)")
	targets := flag.String("targets", "", "Comma-separated OS/architecture targets (defaults to four supported targets)")
	candidate := flag.Bool("candidate", false, "Build unpublished review artifacts without claiming release readiness")
	flag.Parse()
	if *output == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "provide --output with a new directory")
		return 2
	}
	var selected []string
	if *targets != "" {
		selected = strings.Split(*targets, ",")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	manifest, err := distribution.Build(ctx, distribution.Options{Source: *source, Output: *output, Version: *version, Targets: selected, Candidate: *candidate})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
