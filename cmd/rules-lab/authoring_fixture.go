// Execute an explicit authoring command sequence in a disposable project with per-command output and file changes.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/project"
	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/fabricahq/code-rules/internal/terminalfixture"
)

// authoringFixture supplies initial relative text files and a bounded sequence of native authoring commands.
type authoringFixture struct {
	Files    map[string]string        `json:"files"`
	Commands [][]string               `json:"commands"`
	Terminal bool                     `json:"terminal,omitempty"`
	Answers  [][]terminalfixture.Step `json:"answers,omitempty"`
}

// authoringObservation retains command streams and the complete before/after project for review.
type authoringObservation struct {
	Commands []*cliObservation `json:"commands"`
	Before   map[string]string `json:"before"`
	After    map[string]string `json:"after"`
}

// authoringResponse captures real executable statuses; fixture validation is reported separately.
func authoringResponse(input json.RawMessage) (response, error) {
	return authoringSequenceResponse(input, validateAuthoringArguments)
}

// authoringSequenceResponse owns the disposable project and captures a bounded sequence of reviewed commands.
func authoringSequenceResponse(input json.RawMessage, validate func([]string) error) (response, error) {
	var fixture authoringFixture
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return adapterError("expected an authoring fixture"), nil
	}
	if len(fixture.Commands) < 1 || len(fixture.Commands) > 12 {
		return adapterError("provide 1 to 12 authoring commands"), nil
	}
	for _, args := range fixture.Commands {
		if err := validate(args); err != nil {
			return response{Error: describeProjectError(err)}, nil
		}
	}
	directory, err := os.MkdirTemp("", "rules-lab-authoring-")
	if err != nil {
		return response{}, err
	}
	defer os.RemoveAll(directory)
	root, err := os.OpenRoot(directory)
	if err != nil {
		return response{}, err
	}
	defer root.Close()
	for name, text := range fixture.Files {
		if err := projectFixturePath(name); err != nil {
			return response{Error: describeProjectError(err)}, nil
		}
		if err := writeOfflineFixtureFile(root, name, []byte(text)); err != nil {
			return response{}, err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	before, err := project.ReadTree(ctx, root, ".")
	if err != nil {
		return response{}, err
	}
	observed := authoringObservation{Commands: []*cliObservation{}, Before: displayProjectTree(before)}
	for index, args := range fixture.Commands {
		var value any
		var err error
		if fixture.Terminal {
			var steps []terminalfixture.Step
			if index < len(fixture.Answers) {
				steps = fixture.Answers[index]
			}
			value, err = executeTerminalCLI(ctx, root, args, steps, before)
		} else {
			value, err = executeFixtureCLI(ctx, root, imports.Options{Environment: append(os.Environ(), "PATH="+directory+"/no-runtime")}, args, before)
		}
		if err != nil {
			return response{}, err
		}
		command := value.(*cliObservation)
		observed.Commands = append(observed.Commands, command)
		observed.After = command.After
		if command.ExitCode != 0 {
			break
		}
		before, err = project.ReadTree(ctx, root, ".")
		if err != nil {
			return response{}, err
		}
	}
	return response{OK: true, Value: observed}, nil
}

// validateAuthoringArguments confines file options and allows no Git-fetching or arbitrary execution commands.
func validateAuthoringArguments(args []string) error {
	invalid := func(problem string) error { return &rules.ValidationError{Location: "commands", Problem: problem} }
	if len(args) == 0 {
		return invalid("expected a command")
	}
	allowed := args[0] == "init" || args[0] == "build" || args[0] == "check" || (len(args) > 2 && args[0] == "local" && args[1] == "add" && (args[2] == "group" || args[2] == "rule")) || (len(args) > 1 && args[0] == "add" && args[1] == "source")
	if !allowed {
		return invalid("this walkthrough supports init, add source, local add group/rule, build, and check")
	}
	for i := 0; i < len(args); i++ {
		flag, value, hasValue := strings.Cut(args[i], "=")
		if flag != "--config" && flag != "--body-file" {
			continue
		}
		if !hasValue {
			if i+1 >= len(args) {
				return invalid(flag + " requires a value")
			}
			i++
			value = args[i]
		}
		if flag == "--config" {
			if value != ".code-rules/config.json" {
				return invalid("config must be .code-rules/config.json")
			}
		} else if err := projectFixturePath(value); err != nil {
			return invalid("body-file must be a contained fixture path")
		}
	}
	return nil
}
