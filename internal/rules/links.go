// Interpret CommonMark links and supporting-file ownership without filesystem access.

package rules

import (
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/text"
)

var externalScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// markdownLinks finds Markdown destinations and HTML href/src attributes, excluding code and complete frontmatter envelopes.
func markdownLinks(document string) ([]string, error) {
	if !utf8.ValidString(document) {
		return nil, invalid("document", "expected UTF-8 text")
	}
	offset := 0
	withoutBOM := strings.TrimPrefix(document, "\ufeff")
	if strings.HasPrefix(withoutBOM, "---\n") || strings.HasPrefix(withoutBOM, "---\r\n") {
		if split, err := SplitDocument(withoutBOM, "document"); err == nil {
			offset = len(document) - len(split.Body)
		}
	}
	source := []byte(document[offset:])
	root := parser.New().Parse(source)
	links := []string{}
	var rawHTML strings.Builder
	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.RawHTML:
			rawHTML.WriteString(n.Value.Value(source))
			rawHTML.WriteByte('\n')
		case *ast.HTMLBlock:
			rawHTML.Write(n.Value.Bytes(source))
			rawHTML.WriteByte('\n')
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
		links = append(links, destination.Value(source))
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}
	links = append(links, htmlLinks(rawHTML.String())...)
	slices.Sort(links)
	return slices.Compact(links), nil
}

// htmlLinks reads real href/src attributes; one tokenizer preserves script and textarea context across inline nodes.
func htmlLinks(fragment string) []string {
	links := []string{}
	tokenizer := html.NewTokenizer(strings.NewReader(fragment))
	for {
		kind := tokenizer.Next()
		if kind == html.ErrorToken {
			return links
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		for _, attribute := range tokenizer.Token().Attr {
			if attribute.Key == "href" || attribute.Key == "src" {
				links = append(links, attribute.Val)
			}
		}
	}
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

// RequireAllowedTarget permits self-links, shared/owned assets, and declared terms.
// Links to other rule documents fail regardless of selection or target existence.
func RequireAllowedTarget(file, target string, terms []string) error {
	if file == target || slices.Contains(terms, target) {
		return nil
	}
	if _, err := GroupFromPath(target, file); err == nil {
		return invalid(file, "links to other rule documents are not allowed: "+target+"; move shared supporting material to the library-root assets/ directory")
	}
	own := AssetDirectory(file)
	if own == "" {
		own = RuleAssetDirectory(file)
	}
	destination := AssetDirectory(target)
	if destination == "assets/" || (destination != "" && destination == own) {
		return nil
	}
	return invalid(file, "unsupported supporting-file link: "+target+"; use this rule's assets directory or the library-root assets directory")
}

// MarkdownTargets returns distinct source-relative dependencies in deterministic path order, including HTML href/src attributes.
func MarkdownTargets(document, file string) ([]string, error) {
	links, err := markdownLinks(document)
	if err != nil {
		return nil, err
	}
	targets := []string{}
	for _, link := range links {
		target, _, local, err := RelativeTarget(link, file)
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
