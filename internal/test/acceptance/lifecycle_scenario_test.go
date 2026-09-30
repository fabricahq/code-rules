// Carry a library and a project through the whole versioning lifecycle: change notes, library releases,
// previewed updates, pins, a scoped update, a retirement, and a fork.

package acceptance

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fabricahq/code-rules/internal/test/gitfixture"
)

// invocation runs one CLI command in dir with env and requires exit status want.
type invocation func(label, dir string, env []string, want int, args ...string) error

// lifecycleScenario continues from a project that synced the library's first library release. In author, the
// library's checkout, the author publishes release/2: techs/go/errors becomes a major change, techs/go/naming
// retires in favor of the new techs/go/wrapping, and techs/go/panics is new. The project keeps errors at 1.0.0 with
// --keep, excludes panics with --exclude, and applies the rest. After release/3 changes wrapping, the project deletes
// the pin and adopts errors 2.0.0 alone with a scoped update, then forks errors into its local rules.
// project check passes after every step that changes the project.
func lifecycleScenario(ctx context.Context, report *Report, invoke invocation, fixture *gitfixture.Fixture, author, directory, consumer string, online, offline []string) error {
	last := func() Step { return report.Steps[len(report.Steps)-1] }
	edit := func(label string) { report.Steps = append(report.Steps, Step{Label: label}) }
	check := func(label string) error { return invoke(label, consumer, offline, 0, "project", "check") }
	// publish commits and pushes every change in the author's checkout, then publishes it as library release number.
	publish := func(number int, message string) error {
		if _, err := fixture.Commit(ctx, author, message, nil); err != nil {
			return err
		}
		if _, err := fixture.CommandIn(ctx, author, "push", "--quiet", "origin", "main"); err != nil {
			return err
		}
		edit("Author edit: commit and push the changes")
		if err := invoke(fmt.Sprintf("Publish library release release/%d", number), author, online, 0, "library", "release"); err != nil {
			return err
		}
		return contains("library release output", last().Stdout, fmt.Sprintf("Published library release %d.", number))
	}
	project := func() (map[string][]byte, error) {
		tree, err := readTree(ctx, consumer)
		if err != nil {
			return nil, err
		}
		return tree.Files, nil
	}

	// The author changes the library: check requires a note for the changed rule, then accepts the recorded changes.
	errorsRule := filepath.Join(author, "techs/go/errors.md")
	errorsV1, err := os.ReadFile(errorsRule)
	if err != nil {
		return err
	}
	errorsV2 := bytes.ReplaceAll(errorsV1, []byte("Return every failure."), []byte("Wrap every failure with the operation that failed."))
	if err := os.WriteFile(errorsRule, errorsV2, 0600); err != nil {
		return err
	}
	edit("Author edit: make techs/go/errors stricter")
	if err := invoke("Library check requires a change note", author, online, 1, "library", "check"); err != nil {
		return err
	}
	if err := contains("library check failure", last().Stderr, "techs/go/errors changed since release/1, where its version is 1.0.0, and no pending change note names it. Record it with: code-rules library change techs/go/errors"); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(author, "techs/go/naming.md")); err != nil {
		return err
	}
	edit("Author edit: delete techs/go/naming")
	bodies := map[string]string{"wrapping.md": "Wrap errors with the operation that failed.\n", "panics.md": "Return errors instead of panicking.\n"}
	for name, body := range bodies {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(body), 0600); err != nil {
			return err
		}
	}
	for _, args := range [][]string{
		{"library", "change", "techs/go/errors", "--bump", "major", "--summary", "Require wrapping every failure."},
		{"library", "add", "rule", "techs/go/wrapping", "--title", "Wrap errors", "--impact", "HIGH", "--impact-description", "Trace failures.", "--when-to-read", "When returning errors.", "--body-file", filepath.Join(directory, "wrapping.md")},
		{"library", "add", "rule", "techs/go/panics", "--title", "Avoid panics", "--impact", "MEDIUM", "--impact-description", "Keep failures recoverable.", "--when-to-read", "When handling unexpected states.", "--body-file", filepath.Join(directory, "panics.md")},
		{"library", "change", "techs/go/naming", "--retire", "--replaced-by", "techs/go/wrapping", "--summary", "Covered by the wrapping rule."},
		{"library", "change", "techs/go/wrapping", "--summary", "Add the rule."},
		{"library", "change", "techs/go/panics", "--summary", "Add the rule."},
	} {
		if err := invoke("Record the library's changes", author, online, 0, args...); err != nil {
			return err
		}
	}
	if err := invoke("Library check previews the pending library release", author, online, 0, "library", "check"); err != nil {
		return err
	}
	if err := contains("library check preview", last().Stdout, "Pending library release 2\n", "  techs/go/errors    major    1.0.0 -> 2.0.0\n", "  techs/go/naming    retired  1.0.0, replaced by techs/go/wrapping\n", "  techs/go/panics    new      1.0.0\n", "  techs/go/wrapping  new      1.0.0\n"); err != nil {
		return err
	}
	wrappingV1, err := os.ReadFile(filepath.Join(author, "techs/go/wrapping.md"))
	if err != nil {
		return err
	}
	if err := publish(2, "Require wrapping, retire naming, and add two rules"); err != nil {
		return err
	}
	report.Verified = append(report.Verified, "Library check fails for a changed rule without a note, then previews release/2 from the notes that code-rules library release publishes")

	// The project previews the update without a terminal: nothing is written.
	before, err := readTree(ctx, consumer)
	if err != nil {
		return err
	}
	if err := invoke("Preview the update without a terminal", consumer, online, 0, "project", "update"); err != nil {
		return err
	}
	if err := contains("update preview", last().Stdout,
		"  major     techs/go/errors    1.0.0 -> 2.0.0\n            Require wrapping every failure.\n",
		"  new       techs/go/panics    1.0.0\n            Add the rule.\n",
		"  new       techs/go/wrapping  1.0.0\n            Add the rule.\n",
		"  retired   techs/go/naming    1.0.0\n            Replaced by techs/go/wrapping.\n            Covered by the wrapping rule.\n",
		"This is a preview; no files were written."); err != nil {
		return err
	}
	if err := invoke("Preview the update as JSON", consumer, online, 0, "project", "update", "--json"); err != nil {
		return err
	}
	if err := contains("JSON update preview", last().Stdout, `"applied": false`); err != nil {
		return err
	}
	after, err := readTree(ctx, consumer)
	if err != nil {
		return err
	}
	if !equalTrees(before, after) {
		return fmt.Errorf("a preview changed the project")
	}
	if err := check("Check after the preview"); err != nil {
		return err
	}
	report.Verified = append(report.Verified, "A preview without --yes lists major, new, and retired rules with their summaries, exits 0, and writes nothing")

	// The project keeps the major change, excludes one new rule, and applies the rest, including the retirement.
	reason := "Adopt after the error handling migration."
	if err := invoke("Apply the update, keeping the major change and excluding a new rule", consumer, online, 0, "project", "update", "--yes", "--keep", "team:techs/go/errors", "--exclude", "team:techs/go/panics", "--reason", reason); err != nil {
		return err
	}
	files, err := project()
	if err != nil {
		return err
	}
	pin := "    pins:\n      techs/go/errors:\n        version: \"1.0.0\"\n        reason: " + reason + "\n"
	if err := contains("configuration", string(files[".code-rules/config.yaml"]), pin, "    exclude:\n      techs/go/panics:\n        reason: "+reason+"\n"); err != nil {
		return err
	}
	if !bytes.Equal(files[".code-rules/vendor/team/techs/go/errors.md"], errorsV1) || !strings.Contains(string(files[".code-rules/generated/rules/team/techs/go/errors.md"]), "Version: 1.0.0") {
		return fmt.Errorf("the kept rule didn't stay at 1.0.0")
	}
	if !strings.Contains(string(files[".code-rules/generated/rules/team/techs/go/wrapping.md"]), "Version: 1.0.0") {
		return fmt.Errorf("the new rule wasn't added at 1.0.0")
	}
	if files[".code-rules/vendor/team/techs/go/naming.md"] != nil || files[".code-rules/generated/rules/team/techs/go/naming.md"] != nil {
		return fmt.Errorf("the retired rule wasn't dropped")
	}
	if files[".code-rules/vendor/team/techs/go/panics.md"] == nil || files[".code-rules/generated/rules/team/techs/go/panics.md"] != nil {
		return fmt.Errorf("the excluded rule should be imported but not generated")
	}
	if err := check("Check the applied update"); err != nil {
		return err
	}
	if err := invoke("Preview the pinned rule", consumer, online, 0, "project", "update"); err != nil {
		return err
	}
	if err := contains("pinned preview", last().Stdout, "  pinned    techs/go/errors  1.0.0\n            Newest version: 2.0.0.\n            Reason: "+reason+"\n"); err != nil {
		return err
	}
	report.Verified = append(report.Verified, "--keep pins a major change and --exclude excludes a new rule in config.yaml, the retirement drops its rule, the new rule joins, and later previews list the pin")

	// Library release 3 changes another rule, so the scoped update below has something to leave alone.
	wrappingV2 := append(bytes.Clone(wrappingV1), []byte("\nFor example, wrap with the operation's name.\n")...)
	if err := os.WriteFile(filepath.Join(author, "techs/go/wrapping.md"), wrappingV2, 0600); err != nil {
		return err
	}
	edit("Author edit: add an example to techs/go/wrapping")
	if err := invoke("Record the example", author, online, 0, "library", "change", "techs/go/wrapping", "--bump", "minor", "--summary", "Add an example."); err != nil {
		return err
	}
	if err := publish(3, "Add a wrapping example"); err != nil {
		return err
	}

	// The project deletes the pin and adopts errors 2.0.0 alone.
	config := strings.Replace(string(files[".code-rules/config.yaml"]), pin, "", 1)
	if err := os.WriteFile(filepath.Join(consumer, ".code-rules/config.yaml"), []byte(config), 0600); err != nil {
		return err
	}
	edit("Configuration edit: delete the pin on techs/go/errors")
	if err := invoke("Preview a scoped update", consumer, online, 0, "project", "update", "team:techs/go/errors"); err != nil {
		return err
	}
	if err := contains("scoped preview", last().Stdout, "  major     techs/go/errors  1.0.0 -> 2.0.0\n"); err != nil {
		return err
	}
	if strings.Contains(last().Stdout, "techs/go/wrapping") {
		return fmt.Errorf("the scoped preview lists another rule:\n%s", last().Stdout)
	}
	if err := invoke("Apply the scoped update", consumer, online, 0, "project", "update", "team:techs/go/errors", "--yes"); err != nil {
		return err
	}
	files, err = project()
	if err != nil {
		return err
	}
	if !bytes.Equal(files[".code-rules/vendor/team/techs/go/errors.md"], errorsV2) || !strings.Contains(string(files[".code-rules/generated/rules/team/techs/go/errors.md"]), "Version: 2.0.0") {
		return fmt.Errorf("the scoped update didn't move techs/go/errors to 2.0.0")
	}
	if !bytes.Equal(files[".code-rules/vendor/team/techs/go/wrapping.md"], wrappingV1) || !strings.Contains(string(files[".code-rules/generated/rules/team/techs/go/wrapping.md"]), "Version: 1.0.0") {
		return fmt.Errorf("the scoped update moved another rule")
	}
	if err := check("Check the scoped update"); err != nil {
		return err
	}
	if err := invoke("Preview the rest of the update", consumer, online, 0, "project", "update"); err != nil {
		return err
	}
	if err := contains("remaining preview", last().Stdout, "  minor     techs/go/wrapping  1.0.0 -> 1.1.0\n            Add an example.\n"); err != nil {
		return err
	}
	report.Verified = append(report.Verified, "After the pin is deleted, a scoped update moves only its rule and leaves the rest for the next full update")

	// The project forks errors 2.0.0, with its asset, so it controls the text; the fork replaces the imported rule.
	forkReason := "Adapt wrapping to our error types."
	if err := invoke("Fork a library rule", consumer, online, 0, "project", "add", "rule", "techs/go/errors", "--from", "team@2.0.0", "--reason", forkReason, "--non-interactive"); err != nil {
		return err
	}
	if err := invoke("Build the fork offline", consumer, offline, 0, "project", "build"); err != nil {
		return err
	}
	files, err = project()
	if err != nil {
		return err
	}
	if err := contains("forked rule", string(files[".code-rules/local/techs/go/errors.md"]), "Wrap every failure with the operation that failed.", "![diagram](assets/errors/diagram.bin)"); err != nil {
		return err
	}
	if !bytes.Equal(files[".code-rules/local/techs/go/assets/errors/diagram.bin"], files[".code-rules/vendor/team/techs/go/assets/errors/diagram.bin"]) {
		return fmt.Errorf("the fork didn't copy the rule's asset")
	}
	if err := contains("configuration", string(files[".code-rules/config.yaml"]), "      techs/go/errors:\n        reason: "+forkReason+"\n        replacedBy: local/techs/go/errors.md\n"); err != nil {
		return err
	}
	if files[".code-rules/generated/rules/local/techs/go/errors.md"] == nil || files[".code-rules/generated/rules/team/techs/go/errors.md"] != nil {
		return fmt.Errorf("generated guidance doesn't replace the imported rule with the fork")
	}
	if err := check("Check the fork"); err != nil {
		return err
	}
	report.Verified = append(report.Verified, "A fork copies one published rule version and its asset into local rules and replaces the imported rule, which checks offline")
	return nil
}

// contains reports which of parts text, a named command output or file, lacks.
func contains(what, text string, parts ...string) error {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return fmt.Errorf("%s lacks %q:\n%s", what, part, text)
		}
	}
	return nil
}
