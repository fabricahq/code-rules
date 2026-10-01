// Fork one published version of a library rule into local/, and replace the imported rule with it.

package project

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/fabricahq/code-rules/internal/filetxn"
	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/rules"
)

// ForkSource names the library and the published rule version a fork copies.
type ForkSource struct {
	// Library is a configured source name, or a repository address, which always contains a colon.
	Library string
	Version rules.RuleVersion
}

// ParseForkSource reads LIBRARY@VERSION, splitting at the last @, because repository addresses can contain @.
func ParseForkSource(value string) (ForkSource, error) {
	at := strings.LastIndex(value, "@")
	if at <= 0 {
		return ForkSource{}, &rules.ValidationError{Location: "--from", Problem: fmt.Sprintf("%q: expected LIBRARY@VERSION, such as team@1.3.0", value)}
	}
	version, err := rules.ParseRuleVersion(value[at+1:], "--from")
	if err != nil {
		return ForkSource{}, err
	}
	return ForkSource{Library: value[:at], Version: version}, nil
}

// ForkPlan is a planned fork of one published rule version. Planning reads the project and the library without
// writing; Commit writes the fork.
type ForkPlan struct {
	id, group string
	options   Options
	// configBytes is the configuration planning read; Commit refuses to write when it changed.
	configBytes []byte
	// replaces is the configured source whose imported rule the fork replaces, or empty when no source imports it.
	// pin is that source's pin of the rule, which Commit removes, or nil when it has none.
	replaces string
	pin      *rules.Pin
	// version is the forked version, which the exclusion records as basedOn.
	version rules.RuleVersion
	release int
	// files holds the fork's files by path relative to local/, and groupMetadata the library's metadata for the
	// rule's group, which Commit copies when the project has no local metadata for it.
	files         map[string][]byte
	groupMetadata []byte
}

// PlanFork checks the project, reads the published rule version from the library, and prepares the fork's files,
// without writing. The fork replaces the rule when the configured source for the library imports it, as the
// source's synced record shows. It fails before reading the library when local/ already has the rule, when that
// source already excludes it, since existing local rules and exclusions are never replaced, or when the source
// selects the rule but the project hasn't synced the source's current configuration.
func PlanFork(ctx context.Context, id string, from ForkSource, options Options, git imports.Options) (*ForkPlan, error) {
	plan := &ForkPlan{id: id, version: from.Version}
	var library rules.Source
	options, err := planProjectAuthoring(ctx, options, func(root *os.Root, _ rules.Configuration) error {
		original, config, err := configuration(ctx, root)
		if err != nil {
			return err
		}
		plan.configBytes = original
		if plan.group, err = checkForkTarget(ctx, root, id); err != nil {
			return err
		}
		var replaces *rules.Source
		if library, replaces, err = forkLibrary(config, from.Library, id, plan.group); err != nil {
			return err
		}
		if replaces != nil {
			imported, err := importsRule(ctx, root, *replaces, id)
			if err != nil || !imported {
				return err
			}
			if _, excluded := replaces.Exclude[id]; excluded {
				return &rules.ValidationError{Location: "sources." + replaces.Name + ".exclude." + id, Problem: "the source already excludes this rule, and a fork never replaces an existing exclusion; delete the entry to fork the rule"}
			}
			plan.replaces = replaces.Name
			if pin, pinned := replaces.Pins[id]; pinned {
				plan.pin = &pin
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	plan.options = options
	published, err := imports.ReadPublishedRule(ctx, library, id, from.Version, git)
	if err != nil {
		return nil, fmt.Errorf("fork %s@%s: %w", id, from.Version, err)
	}
	attribution, err := forkAttribution(library.Repository, id, from.Version, published)
	if err != nil {
		return nil, err
	}
	if plan.files, err = forkFiles(id, published, attribution); err != nil {
		return nil, fmt.Errorf("fork %s@%s: %w", id, from.Version, err)
	}
	if published.GroupMetadata != nil {
		if _, err := rules.ParseGroupMetadataYAML(published.GroupMetadata, fmt.Sprintf("release/%d: %s/_group.yaml", published.Release, plan.group)); err != nil {
			return nil, err
		}
	}
	plan.release, plan.groupMetadata = published.Release, published.GroupMetadata
	return plan, nil
}

// Replaces returns the configured source whose imported rule the fork replaces, or empty when no source imports
// the rule. Commit needs a reason exactly when it isn't empty.
func (p *ForkPlan) Replaces() string { return p.replaces }

// Release returns the number of the library release that published the forked version.
func (p *ForkPlan) Release() int { return p.release }

// Commit writes the fork under writer ownership, creating the local group from the library's group metadata when
// the project has no local metadata for it. When the fork replaces an imported rule, the same edit adds the
// source's exclusion with reason, the fork as replacedBy, and the forked version as basedOn, so updates list the
// library's later changes, and removes the source's pin of the rule, with a
// warning, since the fork, not a pinned import, now decides what agents read. It writes nothing when reason is blank for a
// replacement or given without one, when the configuration changed after planning, or when any file exists.
func (p *ForkPlan) Commit(ctx context.Context, reason string) (AuthoringResult, error) {
	if p == nil || p.id == "" {
		return AuthoringResult{}, failure("invalid-operation", "expected a planned fork", nil)
	}
	replacedBy := path.Join("local", p.id+".md")
	switch {
	case p.replaces != "" && strings.TrimSpace(reason) == "":
		return AuthoringResult{}, &rules.ValidationError{Location: "--reason", Problem: "give the reason the project uses the fork instead of the rule it imports from " + p.replaces}
	case p.replaces == "" && reason != "":
		return AuthoringResult{}, &rules.ValidationError{Location: "--reason", Problem: "the project doesn't import this rule, so the fork replaces nothing and has no exclusion to record a reason in"}
	}
	result, err := editProject(ctx, p.options, func(root *os.Root, original []byte, config rules.Configuration) ([]filetxn.File, error) {
		if !bytes.Equal(original, p.configBytes) {
			return nil, failure("concurrent-change", "the project's configuration changed after the fork was planned; run the command again", nil)
		}
		files, err := p.groupFiles(ctx, root, config)
		if err != nil {
			return nil, err
		}
		for name, data := range p.files {
			files = append(files, filetxn.File{Path: path.Join("local", name), Content: data})
		}
		slices.SortFunc(files, func(a, b filetxn.File) int { return strings.Compare(a.Path, b.Path) })
		for _, file := range files {
			if err := filetxn.RequireAbsent(ctx, root, file.Path); err != nil {
				return nil, err
			}
		}
		if p.replaces != "" {
			edit := rules.SourceEdit{Exclude: map[string]rules.Exclusion{p.id: {Reason: reason, ReplacedBy: replacedBy, BasedOn: &p.version}}}
			if p.pin != nil {
				edit.Unpin = []string{p.id}
			}
			edited, err := rules.EditConfigurationSource(original, p.replaces, edit)
			if err != nil {
				return nil, err
			}
			files = append(files, filetxn.File{Path: configurationFile, Content: edited, Previous: original})
		}
		return files, nil
	})
	if err == nil && p.pin != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Removed sources.%s.pins.%s, which kept the rule at %s, because the fork replaces the imported rule. Updates now list the library's changes after %s, the version the fork is based on, for you to compare with the fork.", p.replaces, p.id, p.pin.Version, p.version))
	}
	return result, err
}

// groupFiles returns the files that create the rule's local group from the library's metadata, and the group's
// authoring README when it's missing too. It returns none when the project has local metadata for the group, or
// when a source the project imports, as its synced record shows, supplies the group, because local metadata would
// replace that library's description of the group. It fails when the library has no metadata for the group either.
func (p *ForkPlan) groupFiles(ctx context.Context, root *os.Root, config rules.Configuration) ([]filetxn.File, error) {
	group := p.group
	directory := path.Join("local", group)
	metadata, err := filetxn.ReadOptional(ctx, root, path.Join(directory, "_group.yaml"))
	if err != nil || metadata != nil {
		return nil, err
	}
	if imported, err := importedGroup(ctx, root, config, group); err != nil || imported {
		return nil, err
	}
	if p.groupMetadata == nil {
		return nil, failure("missing-group", fmt.Sprintf("release/%d has no metadata for group %s; create the group with code-rules project add group %s, then retry the fork", p.release, group, group), nil)
	}
	guideName, _ := projectGuide()
	files := []filetxn.File{}
	for _, file := range groupFiles(directory, group, p.groupMetadata, guideName) {
		existing, err := filetxn.ReadOptional(ctx, root, file.Path)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			files = append(files, file)
		}
	}
	return files, nil
}

// importedGroup reports whether the synced record of a configured source imports group, as a selected group or
// the group of an imported rule; either supplies the group's metadata.
func importedGroup(ctx context.Context, root *os.Root, config rules.Configuration, group string) (bool, error) {
	vendor, err := filetxn.ReadTree(ctx, root, "vendor")
	if err != nil {
		return false, err
	}
	recorded, err := recordedSnapshots(config, treeFiles(vendor))
	if err != nil {
		return false, err
	}
	for _, snapshot := range recorded {
		if slices.Contains(snapshot.Groups, group) {
			return true, nil
		}
		for id := range snapshot.Rules {
			if strings.HasPrefix(id, group+"/") {
				return true, nil
			}
		}
	}
	return false, nil
}

// checkForkTarget checks that id names a rule path and that local/ doesn't have the rule yet, and returns the
// rule's group.
func checkForkTarget(ctx context.Context, root *os.Root, id string) (string, error) {
	if strings.HasSuffix(id, ".md") {
		return "", failure("invalid-rule-path", "use a rule path without the .md extension", nil)
	}
	group, err := rules.GroupFromPath(id+".md", "rule")
	if err != nil {
		return "", err
	}
	return group, filetxn.RequireAbsent(ctx, root, path.Join("local", id+".md"))
}

// forkLibrary returns the library that name identifies, labeled for diagnostics, and the configured source that
// selects rule id of group from it, through its groups or its rules list, or nil when none does. A name without a
// colon is a configured source name; any other is a repository address, matched to a configured source by
// repository.
func forkLibrary(config rules.Configuration, name, id, group string) (rules.Source, *rules.Source, error) {
	var configured *rules.Source
	library := rules.Source{Name: name, Repository: name}
	if !strings.Contains(name, ":") {
		index := slices.IndexFunc(config.Sources, func(source rules.Source) bool { return source.Name == name })
		if index < 0 {
			return rules.Source{}, nil, &rules.ValidationError{Location: "--from", Problem: "no source named " + name + " in .code-rules/config.yaml; name a configured source or the library's repository address"}
		}
		configured = &config.Sources[index]
		library.Repository = configured.Repository
	} else {
		address, err := repositoryIdentity(name, "--from")
		if err != nil {
			return rules.Source{}, nil, err
		}
		for i, source := range config.Sources {
			if identity, err := repositoryIdentity(source.Repository, "sources."+source.Name+".repository"); err == nil && identity == address {
				configured = &config.Sources[i]
			}
		}
	}
	if configured != nil && (configured.Groups.Includes(group) || slices.Contains(configured.Rules, id)) {
		return library, configured, nil
	}
	return library, nil, nil
}

// importsRule reports whether source, which selects rule id, imports it, as its record from the last sync shows:
// a ref, for example, can name a revision without the rule. It fails, asking to sync, when the source has no
// record or its record doesn't match its configuration.
func importsRule(ctx context.Context, root *os.Root, source rules.Source, id string) (bool, error) {
	location := source.Name + "/_source.json"
	data, err := filetxn.ReadOptional(ctx, root, path.Join("vendor", location))
	if err != nil {
		return false, err
	}
	if data == nil {
		return false, invalidSnapshot(location, "missing source record, so whether the source imports "+id+" isn't known; run code-rules project sync, then fork the rule")
	}
	record, err := parseSourceRecord(data, source.Name)
	if err != nil {
		return false, err
	}
	if err := matchSnapshotSource(source, record); err != nil {
		return false, err
	}
	_, imported := record.Rules[id]
	return imported, nil
}

// repositoryIdentity returns the identity that recognizes two spellings of one repository address.
func repositoryIdentity(address, location string) (string, error) {
	raw, _ := json.Marshal(address)
	repository, err := rules.ParseRepository(raw, location)
	return repository.Identity, err
}

// forkAttribution returns the attribution entry that links a fork to the published rule: the rule's file at the
// library release's commit for GitHub.com and GitLab.com, or the repository address for other hosts' HTTPS
// addresses. It returns nil for other addresses, which have no HTTP(S) URL to cite.
func forkAttribution(repository, id string, version rules.RuleVersion, published imports.PublishedRule) (*rules.Attribution, error) {
	raw, _ := json.Marshal(repository)
	parsed, err := rules.ParseRepository(raw, "--from")
	if err != nil {
		return nil, err
	}
	url, ok := parsed.FileURL(published.Commit, id+".md", false)
	if !ok {
		if !strings.HasPrefix(repository, "https://") {
			return nil, nil
		}
		url = repository
	}
	return &rules.Attribution{URL: url, Description: fmt.Sprintf("Forked from version %s of %s, published in library release %d at commit %s.", version, id, published.Release, published.Commit)}, nil
}

// forkFiles returns the fork's files by path relative to local/. The rule and its asset directory keep their
// library paths; each shared asset moves from assets/ into the rule's asset directory, keeping its path below
// assets/. Markdown links to moved files are rewritten, and attribution, when not nil, is added to the rule. It
// fails when two files would share a path, or when a Markdown file links to anything outside the fork, such as
// a declared license file, because local rules can't depend on library files.
func forkFiles(id string, published imports.PublishedRule, attribution *rules.Attribution) (map[string][]byte, error) {
	rulePath := id + ".md"
	moved := map[string]string{}
	sources := map[string]string{}
	for _, file := range slices.Sorted(maps.Keys(published.Files)) {
		target := file
		if shared, ok := strings.CutPrefix(file, "assets/"); ok {
			target = rules.RuleAssetDirectory(rulePath) + shared
		}
		if other, taken := sources[target]; taken {
			return nil, &rules.ValidationError{Location: rulePath, Problem: fmt.Sprintf("the library's %s and %s would both become %s in the fork; rename one in the library", other, file, target)}
		}
		moved[file], sources[target] = target, file
	}
	result := map[string][]byte{}
	for file, target := range moved {
		data := published.Files[file]
		if strings.HasSuffix(file, ".md") {
			text, err := relocateLinks(string(data), file, target, moved)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", file, err)
			}
			data = []byte(text)
		}
		result[target] = data
	}
	document := string(result[rulePath])
	var err error
	if attribution != nil {
		document, err = rules.AddRuleAttribution(document, rulePath, *attribution)
	} else {
		_, err = rules.Parse(document, rulePath, "local")
	}
	if err != nil {
		return nil, err
	}
	result[rulePath] = []byte(document)
	return result, requireSelfContained(result)
}

// relocateLinks returns the Markdown text of the file moving from from to to, with each local link rewritten as a
// relative link when its destination moves or the file itself moves, including links that name the file itself.
// Pathless links, such as #top, external links, and other links keep their original text.
func relocateLinks(text, from, to string, moved map[string]string) (string, error) {
	start := rules.MarkdownBodyStart(text)
	destinations, err := rules.MarkdownDestinations(text[start:])
	if err != nil {
		return "", err
	}
	result := text
	for _, destination := range slices.Backward(destinations) {
		target, suffix, local, err := rules.RelativeTarget(destination.Value, from)
		if err != nil {
			return "", err
		}
		// A pathless link, such as #top, reaches its own document wherever the document moves.
		if !local || strings.IndexAny(destination.Value, "?#") == 0 || destination.Value == "" {
			continue
		}
		relocated, ok := moved[target]
		if !ok {
			relocated = target
		}
		if relocated == target && to == from {
			continue
		}
		link := rules.EscapeDestination(rules.RelativeLink(to, relocated) + suffix)
		result = result[:start+destination.Start] + link + result[start+destination.End:]
	}
	return result, nil
}

// requireSelfContained fails when a Markdown file among files links to a file that files doesn't hold, or has a
// relative link in raw HTML, which a fork can't relocate and generation rejects in local rules.
func requireSelfContained(files map[string][]byte) error {
	for _, file := range slices.Sorted(maps.Keys(files)) {
		if !strings.HasSuffix(file, ".md") {
			continue
		}
		links, err := rules.RawHTMLLinks(string(files[file]))
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		for _, link := range links {
			if _, _, local, err := rules.RelativeTarget(link, file); err != nil || local {
				return &rules.ValidationError{Location: file, Problem: fmt.Sprintf("links to %s in raw HTML, which a fork can't use: local rules can't have relative links in raw HTML; use a Markdown link in the library", link)}
			}
		}
		targets, err := rules.MarkdownTargets(string(files[file]), file)
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		for _, target := range targets {
			if _, ok := files[target]; !ok {
				return &rules.ValidationError{Location: file, Problem: fmt.Sprintf("links to %s, which a fork can't copy: a fork holds only the rule, its asset directory, and the shared assets its Markdown links reach, because local rules can't depend on library files", target)}
			}
		}
	}
	return nil
}
