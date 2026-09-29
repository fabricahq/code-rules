// Record rule changes as new change notes in changes/, and read the notes library check compares.

package library

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.yaml.in/yaml/v4"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/rules"
)

// changesDirectory holds change notes at the library root; projects never import it.
const changesDirectory = "changes"

// maxNoteNameAttempts bounds the numbered suffixes tried for a unique note name on one day.
const maxNoteNameAttempts = 1000

// ChangeRequest describes a new change note for one or more rules.
type ChangeRequest struct {
	// IDs are library rule IDs, such as practices/testing/verify-retry-limits, in the order given.
	IDs []string
	// Bump is major, minor, or patch for rules that have a version, and empty for new or retired rules.
	Bump rules.Change
	// Summary is one line for project maintainers; for a retirement, it explains why.
	Summary string
	// Retire records that the rules were retired; their Markdown files must already be gone.
	Retire bool
	// ReplacedBy names the replacement of a single retired rule, and is empty when nothing replaces it.
	ReplacedBy string
}

// ChangePlan is a change note request checked against the library and its latest library release.
// It owns no filesystem resources; Commit checks everything again before writing.
type ChangePlan struct {
	request ChangeRequest
	options Options
	// Versioned reports that the rules have versions, so the note needs a major, minor, or patch change.
	Versioned bool
}

// PlanChange checks a request before prompts supply a missing Bump or Summary. It fails before the first
// library release, when rules need no notes; for an ID that isn't a rule, unless it's a published rule being
// retired; and when Bump is given for new or retired rules.
func PlanChange(ctx context.Context, request ChangeRequest, options Options) (*ChangePlan, error) {
	directory, err := filepath.Abs(options.Directory)
	if err != nil {
		return nil, err
	}
	options.Directory = directory
	root, err := openLibrary(ctx, options, false)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	versioned, _, err := checkChange(ctx, root, options, request)
	if err != nil {
		return nil, err
	}
	return &ChangePlan{request: request, options: options, Versioned: versioned}, nil
}

// Commit writes a new, uniquely named note, such as changes/2026-09-29-verify-retry-limits.yaml, named by
// date's calendar day and the first rule. It never edits or deletes an existing note, and revalidates the
// complete request under writer ownership.
func (p *ChangePlan) Commit(ctx context.Context, bump rules.Change, summary string, date time.Time) (AuthoringResult, error) {
	if p == nil || len(p.request.IDs) == 0 {
		return AuthoringResult{}, failure("invalid-operation", "expected a planned change note", nil)
	}
	request := p.request
	request.Bump, request.Summary = bump, summary
	if err := requireComplete(request, p.Versioned); err != nil {
		return AuthoringResult{}, err
	}
	root, err := openLibrary(ctx, p.options, false)
	if err != nil {
		return AuthoringResult{}, err
	}
	defer root.Close()
	changes, err := filetxn.Edit(ctx, root, func() ([]filetxn.File, error) {
		versioned, published, err := checkChange(ctx, root, p.options, request)
		if err != nil {
			return nil, err
		}
		if err := requireComplete(request, versioned); err != nil {
			return nil, err
		}
		name, err := noteName(root, request.IDs[0], date, published)
		if err != nil {
			return nil, err
		}
		note, err := renderNote(request, versioned, name)
		if err != nil {
			return nil, err
		}
		return []filetxn.File{{Path: name, Content: note}}, nil
	})
	return authoringResult(changes, err)
}

// checkChange validates a request against the working tree and the latest library release reachable from HEAD.
// It reports whether the rules have versions, and the files that library release published.
func checkChange(ctx context.Context, root *os.Root, options Options, request ChangeRequest) (bool, map[string]string, error) {
	if err := validateRequest(request); err != nil {
		return false, nil, err
	}
	if _, _, err := libraryManifest(ctx, root); err != nil {
		return false, nil, err
	}
	git, err := openLibraryGit(ctx, root.Name(), options.Git)
	if err != nil {
		return false, nil, err
	}
	history, err := git.history(ctx)
	if err != nil {
		return false, nil, err
	}
	if history.latest == nil {
		return false, nil, failure("no-library-release", "this library has no library release yet, so rules need no change notes. Its first library release gives every rule version 1.0.0", nil)
	}
	var versioned, unversioned []string
	for _, id := range request.IDs {
		exists, err := ruleExists(root, id)
		if err != nil {
			return false, nil, err
		}
		_, published := history.latest.record.Rules[id]
		release, retired := history.retired[id]
		switch {
		case request.Retire && exists:
			return false, nil, failure("invalid-change", id+" still exists. Delete its Markdown file and asset directory before recording its retirement.", nil)
		case request.Retire && retired:
			return false, nil, failure("invalid-change", id+" was already retired by release/"+strconv.Itoa(release)+".", nil)
		case request.Retire && !published:
			return false, nil, failure("invalid-change", id+" was never published, so it can't be retired. Delete its Markdown file, and remove it from any pending note.", nil)
		case request.Retire:
		case !exists:
			return false, nil, failure("unknown-rule", unknownRule(id, published), nil)
		case retired:
			return false, nil, failure("invalid-change", id+" reuses the ID of a rule that release/"+strconv.Itoa(release)+" retired. Retired IDs can't be reused; give the rule a new ID.", nil)
		case published:
			versioned = append(versioned, id)
		default:
			unversioned = append(unversioned, id)
		}
	}
	switch {
	case len(versioned) > 0 && len(unversioned) > 0:
		return false, nil, failure("invalid-change", "--bump is required for rules that have a version ("+strings.Join(versioned, ", ")+") and not accepted for new rules ("+strings.Join(unversioned, ", ")+"). Record them in separate notes.", nil)
	case request.Bump != "" && request.Retire:
		return false, nil, failure("invalid-change", "--bump isn't accepted for retired rules.", nil)
	case request.Bump != "" && len(unversioned) > 0:
		return false, nil, failure("invalid-change", "--bump isn't accepted for new rules, which start at version 1.0.0: "+strings.Join(unversioned, ", ")+".", nil)
	}
	return len(versioned) > 0, history.latest.files, nil
}

// validateRequest checks the request's own consistency, independently of the library.
func validateRequest(request ChangeRequest) error {
	if len(request.IDs) == 0 {
		return failure("invalid-change", "name at least one rule", nil)
	}
	for i, id := range request.IDs {
		if err := rules.ValidateRuleID(id, "ID"); err != nil {
			return err
		}
		if slices.Contains(request.IDs[:i], id) {
			return failure("invalid-change", id+" is named more than once", nil)
		}
	}
	switch request.Bump {
	case "", rules.ChangeMajor, rules.ChangeMinor, rules.ChangePatch:
	default:
		return failure("invalid-change", "--bump must be major, minor, or patch", nil)
	}
	if request.ReplacedBy != "" {
		if !request.Retire || len(request.IDs) != 1 {
			return failure("invalid-change", "--replaced-by requires --retire and a single rule", nil)
		}
		if err := rules.ValidateRuleID(request.ReplacedBy, "--replaced-by"); err != nil {
			return err
		}
		if request.ReplacedBy == request.IDs[0] {
			return failure("invalid-change", "a rule can't replace itself", nil)
		}
	}
	return nil
}

// requireComplete requires a summary, and a bump exactly when the rules have versions.
func requireComplete(request ChangeRequest, versioned bool) error {
	if strings.TrimSpace(request.Summary) == "" {
		return failure("invalid-change", "--summary is required", nil)
	}
	if versioned && request.Bump == "" {
		return failure("invalid-change", "--bump is required for rules that have a version: major, minor, or patch", nil)
	}
	return nil
}

// ruleExists reports whether id's Markdown file is an ordinary file in the library.
func ruleExists(root *os.Root, id string) (bool, error) {
	info, err := root.Lstat(id + ".md")
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Mode().IsRegular(), nil
}

// unknownRule explains a missing rule, suggesting a retirement for a published rule.
func unknownRule(id string, published bool) string {
	message := id + " isn't a rule in the library."
	if published {
		message += " To record its retirement, add --retire."
	} else if strings.HasSuffix(id, ".md") {
		message += " Rule IDs omit the .md extension."
	}
	return message
}

// noteName returns changes/<date>-<rule name>.yaml, adding -2, -3, and so on when that name exists in the
// working tree or was published by the latest library release.
func noteName(root *os.Root, id string, date time.Time, published map[string]string) (string, error) {
	base := changesDirectory + "/" + date.Format(time.DateOnly) + "-" + path.Base(id)
	for attempt := 1; attempt <= maxNoteNameAttempts; attempt++ {
		name := base + ".yaml"
		if attempt > 1 {
			name = base + "-" + strconv.Itoa(attempt) + ".yaml"
		}
		if _, ok := published[name]; ok {
			continue
		}
		if _, err := root.Lstat(name); errors.Is(err, fs.ErrNotExist) {
			return name, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", failure("invalid-change", "too many change notes named "+base+"; name the note by hand", nil)
}

// renderNote writes the note's YAML and parses it back, so the command never writes a note check rejects.
func renderNote(request ChangeRequest, versioned bool, name string) ([]byte, error) {
	text := func(value string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value} }
	entries := &yaml.Node{Kind: yaml.MappingNode}
	for _, id := range slices.Sorted(slices.Values(request.IDs)) {
		var change *yaml.Node
		switch {
		case request.Retire && request.ReplacedBy != "":
			change = &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{text("change"), text(string(rules.ChangeRetired)), text("replacedBy"), text(request.ReplacedBy)}}
		case request.Retire:
			change = text(string(rules.ChangeRetired))
		case versioned:
			change = text(string(request.Bump))
		default:
			change = text(string(rules.ChangeNew))
		}
		entries.Content = append(entries.Content, text(id), change)
	}
	document := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{text("summary"), text(strings.TrimSpace(request.Summary)), text("rules"), entries}}
	// A width of -1 keeps a long summary on one line.
	out, err := yaml.Dump(document, yaml.WithV3Defaults(), yaml.WithIndent(2), yaml.WithLineWidth(-1))
	if err != nil {
		return nil, err
	}
	if _, err := rules.ParseChangeNote(out, name); err != nil {
		return nil, err
	}
	return out, nil
}

// readChangeNotes captures every note under changes/, which may be absent. It holds only .yaml files;
// subdirectories, links, and other files fail.
func readChangeNotes(ctx context.Context, input FileSource) (map[string][]byte, error) {
	r := reader{ctx: ctx, input: portableInventorySource{input}, files: map[string][]byte{}}
	entries, err := r.entries(changesDirectory, true)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		name := changesDirectory + "/" + entry.Name()
		if entry.Type()&fs.ModeType != 0 || !strings.HasSuffix(name, ".yaml") {
			return nil, bad(name, "expected only change notes, as .yaml files, in changes/")
		}
		if _, err := r.read(name); err != nil {
			return nil, err
		}
	}
	return r.files, nil
}
