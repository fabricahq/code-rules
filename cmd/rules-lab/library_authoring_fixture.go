// Confine editable library-authoring CLI demonstrations to one disposable library directory.

package main

import (
	"strings"

	"github.com/fabricahq/code-rules/internal/rules"
)

// validateLibraryArguments accepts only library authoring/check commands and contained input files.
func validateLibraryArguments(args []string) error {
	invalid := func(problem string) error { return &rules.ValidationError{Location: "commands", Problem: problem} }
	if len(args) < 2 || args[0] != "library" {
		return invalid("expected a library command")
	}
	allowed := args[1] == "init" || args[1] == "check" || (len(args) > 3 && args[1] == "add" && (args[2] == "group" || args[2] == "rule"))
	if !allowed {
		return invalid("use library init, check, add group, or add rule")
	}
	directory := false
	for i := 0; i < len(args); i++ {
		flag, value, equals := strings.Cut(args[i], "=")
		switch flag {
		case "--directory", "--body-file", "--license-file", "--notice-file":
			if !equals {
				if i+1 >= len(args) {
					return invalid(flag + " requires a value")
				}
				i++
				value = args[i]
			}
			if flag == "--directory" {
				if value != "library" {
					return invalid("the library directory must be library")
				}
				directory = true
			} else if err := projectFixturePath(value); err != nil {
				return invalid("input files must be contained fixture paths")
			}
		}
	}
	if !directory {
		return invalid("commands require --directory library")
	}
	return nil
}
