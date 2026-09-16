// Interpret CommonMark links and supporting-file ownership without filesystem access.

package rules

import (
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/text"
)

// MarkdownLink retains a decoded destination and its byte range in the original document.
// Reference uses point to the definition's destination; ranges are unique and source ordered.
type MarkdownLink struct {
	URL   string `json:"url"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

var externalScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// MarkdownLinks finds links, images, and reference definitions, excluding code and frontmatter.
// HTML is left opaque, matching the existing Markdown contract.
func MarkdownLinks(document string) ([]MarkdownLink, error) {
	if !utf8.ValidString(document) {
		return nil, invalid("document", "expected UTF-8 text")
	}
	offset := 0
	if strings.HasPrefix(document, "---\n") || strings.HasPrefix(document, "---\r\n") {
		split, err := SplitDocument(document, "document")
		if err != nil {
			return nil, err
		}
		offset = len(document) - len(split.Body)
	}
	source := []byte(document[offset:])
	root := parser.New().Parse(source)
	links := []MarkdownLink{}
	seen := map[text.Index]bool{}
	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var destination text.SingleLineValue
		switch n := node.(type) {
		case *ast.Link:
			destination = n.Destination
		case *ast.Image:
			destination = n.Destination
		case *ast.LinkReferenceDefinition:
			destination = n.Destination
		default:
			return ast.WalkContinue, nil
		}
		index := destination.Index()
		if !destination.IsOwned() && !seen[index] {
			seen[index] = true
			links = append(links, MarkdownLink{URL: destination.Value(source), Start: index.Start + offset, End: index.Stop + offset})
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(links, func(a, b MarkdownLink) int { return a.Start - b.Start })
	return links, nil
}

// RelativeTarget resolves a decoded Markdown destination within its source root.
// External URLs return local=false. Query and fragment spelling remains unchanged.
func RelativeTarget(destination, file string) (target, suffix string, local bool, err error) {
	if externalScheme.MatchString(destination) || strings.HasPrefix(destination, "//") {
		return "", "", false, nil
	}
	raw := destination
	if at := strings.IndexAny(raw, "?#"); at >= 0 {
		raw, suffix = raw[:at], raw[at:]
	}
	decoded, decodeErr := url.PathUnescape(raw)
	if decodeErr != nil || !utf8.ValidString(decoded) {
		return "", "", false, invalid(file, "invalid encoded link "+destination)
	}
	if strings.ContainsRune(decoded, '\\') || strings.ContainsFunc(decoded, func(r rune) bool { return r < 32 || r == 127 }) {
		return "", "", false, invalid(file, "unsafe relative link "+destination)
	}
	target = file
	if decoded != "" {
		if strings.HasPrefix(decoded, "/") {
			target = path.Clean(strings.TrimLeft(decoded, "/"))
		} else {
			target = path.Clean(path.Join(path.Dir(file), decoded))
		}
	}
	if target == ".." || strings.HasPrefix(target, "../") {
		return "", "", false, invalid(file, "link escapes source root: "+destination)
	}
	return target, suffix, true, nil
}

// AssetDirectory returns a shared or rule-owned asset directory, or empty for other paths.
func AssetDirectory(file string) string {
	parts := strings.Split(file, "/")
	if len(parts) > 1 && parts[0] == "assets" {
		return "assets/"
	}
	for i := 2; i < len(parts)-2; i++ {
		if parts[i] == "assets" {
			return strings.Join(parts[:i+2], "/") + "/"
		}
	}
	return ""
}

// RuleAssetDirectory returns the directory owned by one rule, including nested rules.
func RuleAssetDirectory(file string) string {
	return path.Dir(file) + "/assets/" + strings.TrimSuffix(path.Base(file), ".md") + "/"
}

// RequireAllowedTarget enforces shared assets, own assets, rule links, and declared terms.
func RequireAllowedTarget(file, target string, terms []string) error {
	if file == target || slices.Contains(terms, target) {
		return nil
	}
	own := AssetDirectory(file)
	if own == "" {
		own = RuleAssetDirectory(file)
	}
	destination := AssetDirectory(target)
	if destination == "assets/" || (destination != "" && destination == own) {
		return nil
	}
	if _, err := GroupFromPath(target, file); err == nil {
		return nil
	}
	return invalid(file, "unsupported supporting-file link: "+target+"; use this rule's assets directory or the library-root assets directory")
}

// MarkdownTargets returns distinct source-relative dependencies in deterministic path order.
func MarkdownTargets(document, file string) ([]string, error) {
	links, err := MarkdownLinks(document)
	if err != nil {
		return nil, err
	}
	targets := []string{}
	for _, link := range links {
		target, _, local, err := RelativeTarget(link.URL, file)
		if err != nil {
			return nil, err
		}
		if local {
			targets = append(targets, target)
		}
	}
	slices.Sort(targets)
	return slices.Compact(targets), nil
}
