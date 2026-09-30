// Locate Markdown link destinations by byte span, and write relative links, so callers can rewrite links in place.

package rules

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/text"
	"github.com/yuin/goldmark/v2/util"
)

// MarkdownDestination is one link, image, or link reference definition destination in a Markdown text.
type MarkdownDestination struct {
	// Start and End delimit the destination's bytes, excluding any <> or parentheses around it. They are equal for
	// an empty destination, such as [text](), at the position a new destination belongs.
	Start, End int
	// Value is the destination as CommonMark reads it, with escapes and entities decoded.
	Value string
}

// MarkdownDestinations returns each distinct destination in the CommonMark text, in document order. A reference
// link shares its definition's destination, which is listed once. It parses the whole text; callers skip
// frontmatter with MarkdownBodyStart. It fails when an empty destination can't be located.
func MarkdownDestinations(markdown string) ([]MarkdownDestination, error) {
	source := []byte(markdown)
	ends := map[ast.Node]int{}
	root := markdownParser(ends).Parse(source)
	destinations := []MarkdownDestination{}
	seen := map[text.Index]bool{}
	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var destination text.SingleLineValue
		var reference *ast.ReferenceLink
		switch n := node.(type) {
		case *ast.Link:
			destination, reference = n.Destination, n.Reference
		case *ast.Image:
			destination, reference = n.Destination, n.Reference
		case *ast.LinkReferenceDefinition:
			destination = n.Destination
		default:
			return ast.WalkContinue, nil
		}
		index := destination.Index()
		if destination.IsOwned() {
			if reference != nil {
				return ast.WalkContinue, nil
			}
			end := ends[node]
			if end < 1 || end > len(source) || source[end-1] != ')' {
				return ast.WalkStop, fmt.Errorf("could not locate Markdown destination")
			}
			index = text.NewIndex(end-1, end-1)
		}
		if !seen[index] {
			seen[index] = true
			destinations = append(destinations, MarkdownDestination{Start: index.Start, End: index.Stop, Value: destination.Value(source)})
		}
		return ast.WalkContinue, nil
	})
	return destinations, err
}

// EscapeDestination escapes the characters that could end a destination written between existing delimiters, so
// the value stays one destination.
func EscapeDestination(value string) string {
	return strings.NewReplacer("&", "&amp;", " ", "%20", "(", "%28", ")", "%29", "<", "%3C", ">", "%3E", "\\", "%5C").Replace(value)
}

// MarkdownBodyStart returns the byte offset where Markdown starts in document: after a complete frontmatter
// envelope at its start, allowing a byte order mark before it, or 0 when it has none.
func MarkdownBodyStart(document string) int {
	withoutBOM := strings.TrimPrefix(document, "\ufeff")
	if strings.HasPrefix(withoutBOM, "---\n") || strings.HasPrefix(withoutBOM, "---\r\n") {
		if split, err := SplitDocument(withoutBOM, "document"); err == nil {
			return len(document) - len(split.Body)
		}
	}
	return 0
}

// RelativeLink returns a link from the file at from to the file at to, both relative to one root, with each path
// segment URL-escaped.
func RelativeLink(from, to string) string {
	relative, _ := filepath.Rel(filepath.FromSlash(path.Dir(from)), filepath.FromSlash(to))
	return EncodeLinkPath(filepath.ToSlash(relative))
}

// EncodeLinkPath URL-escapes each segment of a slash-separated path, keeping the slashes.
func EncodeLinkPath(file string) string {
	parts := strings.Split(file, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

// linkParser records consumed endpoints for empty destinations that Goldmark stores without a source index.
type linkParser struct {
	parser.InlineParser
	ends map[ast.Node]int
}

// Parse delegates CommonMark syntax and records each successfully parsed link's endpoint.
func (p *linkParser) Parse(parent ast.Node, reader text.Reader, context parser.Context) ast.Node {
	node := p.InlineParser.Parse(parent, reader, context)
	if node != nil {
		_, position := reader.Position()
		p.ends[node] = position.Start
	}
	return node
}

// CloseBlock preserves the delegated parser's reference and delimiter cleanup.
func (p *linkParser) CloseBlock(parent ast.Node, reader text.Reader, context parser.Context) {
	p.InlineParser.(parser.CloseBlocker).CloseBlock(parent, reader, context)
}

// markdownParser installs the standard CommonMark parsers with one endpoint-recording link parser.
func markdownParser(ends map[ast.Node]int) parser.Parser {
	return parser.New(parser.WithDefaultParsers(false), parser.WithBlockParsers(
		util.Prioritized(parser.NewSetextHeadingParser(), 100), util.Prioritized(parser.NewThematicBreakParser(), 200),
		util.Prioritized(parser.NewListParser(), 300), util.Prioritized(parser.NewListItemParser(), 400),
		util.Prioritized(parser.NewCodeBlockParser(), 500), util.Prioritized(parser.NewATXHeadingParser(), 600),
		util.Prioritized(parser.NewFencedCodeBlockParser(), 700), util.Prioritized(parser.NewBlockquoteParser(), 800),
		util.Prioritized(parser.NewHTMLBlockParser(), 900), util.Prioritized(parser.NewParagraphParser(), 1000)),
		parser.WithInlineParsers(util.Prioritized(parser.NewCodeSpanParser(), 100), util.Prioritized[parser.InlineParser](&linkParser{parser.NewLinkParser(), ends}, 200), util.Prioritized(parser.NewAutoLinkParser(), 300), util.Prioritized(parser.NewRawHTMLParser(), 400), util.Prioritized(parser.NewEmphasisParser(), 500)),
		parser.WithParagraphTransformers(util.Prioritized(parser.LinkReferenceParagraphTransformer, 100)))
}
