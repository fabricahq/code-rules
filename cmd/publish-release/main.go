// Publish a verified release bundle using a token supplied only to the final CI step.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fabricahq/code-rules/internal/release"
	"github.com/google/go-github/v92/github"
)

func main() { os.Exit(run()) }

func run() int {
	repository := flag.String("repository", "", "GitHub owner/name")
	directory := flag.String("directory", "", "Directory containing the approved release bundle")
	commit := flag.String("commit", "", "Exact approved source commit")
	flag.Parse()
	if *repository == "" || *directory == "" || *commit == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "provide --repository, --directory, and --commit")
		return 2
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "GITHUB_TOKEN is required")
		return 2
	}
	client, err := github.NewClient(github.WithAuthToken(token), github.WithTimeout(2*time.Minute))
	if err != nil {
		fmt.Fprintln(os.Stderr, "configure GitHub client:", err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	url, err := release.Publish(ctx, client, release.PublishOptions{Repository: *repository, Directory: *directory, Commit: *commit})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("Release ready:", url)
	return 0
}
