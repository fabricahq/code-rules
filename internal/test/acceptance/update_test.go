// Publish a library release with code-rules library release, then preview and apply it in a project with
// code-rules project update.

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

// updateScenario has the library author publish release/2, which makes techs/go/errors a major change, retires
// techs/go/naming in favor of the new techs/go/wrapping, and then has the project preview it, keep the
// retirement, exclude the new rule, and apply the rest.
func updateScenario(ctx context.Context, report *Report, invoke invocation, fixture *gitfixture.Fixture, directory, consumer string, online, offline []string) error {
	author, err := fixture.Clone(ctx)
	if err != nil {
		return err
	}
	errorsRule := filepath.Join(author, "techs/go/errors.md")
	document, err := os.ReadFile(errorsRule)
	if err != nil {
		return err
	}
	document = bytes.ReplaceAll(document, []byte("Return every failure."), []byte("Wrap every failure with the operation that failed."))
	if err := os.WriteFile(errorsRule, document, 0600); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(author, "techs/go/naming.md")); err != nil {
		return err
	}
	body := filepath.Join(directory, "wrapping.md")
	if err := os.WriteFile(body, []byte("Wrap errors with the operation that failed.\n"), 0600); err != nil {
		return err
	}
	report.Steps = append(report.Steps, Step{Label: "Author edit: make techs/go/errors stricter and delete techs/go/naming"})
	for _, args := range [][]string{
		{"library", "add", "rule", "techs/go/wrapping", "--title", "Wrap errors", "--impact", "HIGH", "--impact-description", "Trace failures.", "--when-to-read", "When returning errors.", "--body-file", body},
		{"library", "change", "techs/go/errors", "--bump", "major", "--summary", "Require wrapping every failure."},
		{"library", "change", "techs/go/naming", "--retire", "--replaced-by", "techs/go/wrapping", "--summary", "Covered by the wrapping rule."},
		{"library", "change", "techs/go/wrapping", "--summary", "Add the rule."},
		{"library", "check"},
	} {
		if err := invoke("Record the library's changes", author, online, 0, args...); err != nil {
			return err
		}
	}
	if _, err := fixture.Commit(ctx, author, "Require wrapping and retire naming", nil); err != nil {
		return err
	}
	if _, err := fixture.CommandIn(ctx, author, "push", "--quiet", "origin", "main"); err != nil {
		return err
	}
	report.Steps = append(report.Steps, Step{Label: "Author edit: commit and push the changes"})
	if err := invoke("Publish library release release/2", author, online, 0, "library", "release"); err != nil {
		return err
	}
	report.Verified = append(report.Verified, "code-rules library release publishes release/2 from change notes")

	before, err := readTree(ctx, consumer)
	if err != nil {
		return err
	}
	if err := invoke("Preview the update without a terminal", consumer, online, 0, "project", "update"); err != nil {
		return err
	}
	preview := report.Steps[len(report.Steps)-1].Stdout
	for _, row := range []string{"  major     techs/go/errors    1.0.0 -> 2.0.0\n            Require wrapping every failure.\n", "  new       techs/go/wrapping  1.0.0\n            Add the rule.\n", "  retired   techs/go/naming    1.0.0\n            Replaced by techs/go/wrapping.\n            Covered by the wrapping rule.\n", "This is a preview; no files were written."} {
		if !strings.Contains(preview, row) {
			return fmt.Errorf("preview lacks %q:\n%s", row, preview)
		}
	}
	if err := invoke("Preview the update as JSON", consumer, online, 0, "project", "update", "--json"); err != nil {
		return err
	}
	if !strings.Contains(report.Steps[len(report.Steps)-1].Stdout, `"applied": false`) {
		return fmt.Errorf("JSON preview isn't marked unapplied")
	}
	after, err := readTree(ctx, consumer)
	if err != nil {
		return err
	}
	if !equalTrees(before, after) {
		return fmt.Errorf("a preview changed the project")
	}
	report.Verified = append(report.Verified, "A preview without --yes lists major, new, and retired rules with their summaries, exits 0, and writes nothing")

	if err := invoke("Apply the update, keeping the retired rule and excluding the new one", consumer, online, 0, "project", "update", "--yes", "--keep", "team:techs/go/naming", "--exclude", "team:techs/go/wrapping", "--reason", "Adopting wrapping next quarter."); err != nil {
		return err
	}
	files, err := readTree(ctx, consumer)
	if err != nil {
		return err
	}
	config := string(files.Files[".code-rules/config.yaml"])
	for _, text := range []string{"    pins:\n      techs/go/naming:\n        version: \"1.0.0\"\n        reason: Adopting wrapping next quarter.\n", "    exclude:\n      techs/go/wrapping:\n        reason: Adopting wrapping next quarter.\n"} {
		if !strings.Contains(config, text) {
			return fmt.Errorf("configuration lacks %q:\n%s", text, config)
		}
	}
	if !bytes.Equal(files.Files[".code-rules/vendor/team/techs/go/errors.md"], document) || !strings.Contains(string(files.Files[".code-rules/generated/rules/team/techs/go/errors.md"]), "Version: 2.0.0") {
		return fmt.Errorf("the update didn't install techs/go/errors 2.0.0")
	}
	if !strings.Contains(string(files.Files[".code-rules/generated/rules/team/techs/go/naming.md"]), "Version: 1.0.0") {
		return fmt.Errorf("the kept rule isn't in generated guidance at 1.0.0")
	}
	if files.Files[".code-rules/vendor/team/techs/go/wrapping.md"] == nil || files.Files[".code-rules/generated/rules/team/techs/go/wrapping.md"] != nil {
		return fmt.Errorf("the excluded rule should be imported but not generated")
	}
	if err := invoke("Check the update offline", consumer, offline, 0, "project", "check"); err != nil {
		return err
	}
	report.Verified = append(report.Verified, "--keep and --exclude write a quoted pin and an exclusion to config.yaml with the updated output, which checks offline")
	return nil
}
