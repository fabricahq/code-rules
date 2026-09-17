// Test that edited authoring demo requests cannot choose an external file or fetching command.

package main

import "testing"

// TestAuthoringFixtureConfinement rejects escape paths before any child command can be started.
func TestAuthoringFixtureConfinement(t *testing.T) {
	for _, args := range [][]string{{"sync", "--config", "config.json"}, {"init"}, {"init", "--config", "/tmp/config.json"}, {"local", "add", "rule", "techs/go/errors", "--config", "config.json", "--body-file", "../secret"}, {"local", "add", "rule", "techs/go/errors", "--config=config.json", "--body-file=/tmp/secret"}} {
		if err := validateAuthoringArguments(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

// TestLibraryFixtureConfinement refuses external library roots, input files, and unrelated commands.
func TestLibraryFixtureConfinement(t *testing.T) {
	for _, args := range [][]string{{"library", "init"}, {"library", "init", "--directory", "/tmp/library"}, {"library", "init", "--directory", "library", "--license-file", "../terms"}, {"library", "check", "--directory=../../"}, {"sync", "--directory", "library"}} {
		if err := validateLibraryArguments(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
