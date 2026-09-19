// Validate a committed release request and print the exact build and publication inputs.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fabricahq/code-rules/internal/release"
	"os"
)

func main() { os.Exit(run()) }

func run() int {
	base := flag.String("base", "", "Commit before the release request")
	head := flag.String("head", "HEAD", "Commit containing the approved notes")
	flag.Parse()
	if *base == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "provide --base and optionally --head")
		return 2
	}
	plan, err := release.Read(context.Background(), ".", *base, *head)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := json.NewEncoder(os.Stdout).Encode(plan); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
