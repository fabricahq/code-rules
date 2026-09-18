// Demonstrate real terminal conversations inside the same confined authoring fixture.

package main

import (
	"context"
	"os"

	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/terminalfixture"
)

// validateInteractiveArguments uses the existing path confinement for either authoring command family.
func validateInteractiveArguments(args []string) error {
	if len(args) > 0 && args[0] == "library" {
		return validateLibraryArguments(args)
	}
	return validateAuthoringArguments(args)
}

// executeTerminalCLI captures a real PTY while keeping JSON stdout separate from prompts and input echo.
func executeTerminalCLI(ctx context.Context, root *os.Root, args []string, steps []terminalfixture.Step, before *project.Tree) (any, error) {
	binary, err := fixtureCLIPath()
	if err != nil {
		return nil, err
	}
	result, err := terminalfixture.Run(ctx, binary, root.Name(), args, steps)
	if err != nil {
		return nil, err
	}
	after, err := project.ReadTree(ctx, root, ".")
	if err != nil {
		return nil, err
	}
	return &cliObservation{Arguments: args, Stdout: result.Stdout, Stderr: result.Transcript, ExitCode: result.ExitCode, Before: displayProjectTree(before), After: displayProjectTree(after)}, nil
}
