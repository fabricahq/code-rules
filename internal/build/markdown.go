// Rewrite only parsed Markdown destinations and top-level headings, preserving untouched source text.

package build

import (
	"bytes"
	"regexp"
	"slices"
	"strings"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/text"
	"github.com/yuin/goldmark/v2/util"
)

// edit replaces a byte span in the original body.
type edit struct {
	start, end int
	value      string
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

var externalURI = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

var htmlReference = regexp.MustCompile(`(?i)\b(?:href|src)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)

// renderBody rewrites destinations, removes a duplicate title, and nests top-level guidance headings.
func renderBody(body string, active ActiveRule, paths []string, outputPath string) (string, error) {
	source := []byte(body)
	ends := map[ast.Node]int{}
	root := markdownParser(ends).Parse(source)
	edits := []edit{}
	seen := map[text.Index]bool{}
	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var destination text.SingleLineValue
		var reference *ast.ReferenceLink
		image, hasURL := false, true
		switch n := node.(type) {
		case *ast.Link:
			destination, reference = n.Destination, n.Reference
		case *ast.Image:
			destination, reference, image = n.Destination, n.Reference, true
		case *ast.LinkReferenceDefinition:
			destination = n.Destination
		default:
			hasURL = false
		}
		if hasURL {
			if reference != nil {
				image = false
			}
			index := destination.Index()
			if destination.IsOwned() {
				if reference != nil {
					return ast.WalkContinue, nil
				}
				end := ends[node]
				if end < 1 || end > len(source) || source[end-1] != ')' {
					return ast.WalkStop, invalid(active.Rule.ID, "could not locate Markdown destination")
				}
				index = text.NewIndex(end-1, end-1)
			}
			if !seen[index] {
				seen[index] = true
				value, err := relocatedURL(destination.Value(source), active, paths, outputPath, image)
				if err != nil {
					return ast.WalkStop, err
				}
				// The existing delimiters remain intact; escape characters that can terminate a destination.
				value = strings.NewReplacer(" ", "%20", "(", "%28", ")", "%29", "<", "%3C", ">", "%3E", "\\", "%5C").Replace(value)
				edits = append(edits, edit{index.Start, index.Stop, value})
			}
		}
		var html string
		switch n := node.(type) {
		case *ast.RawHTML:
			html = n.Value.Value(source)
		case *ast.HTMLBlock:
			html = string(n.Value.Bytes(source))
		}
		for _, match := range htmlReference.FindAllStringSubmatch(html, -1) {
			value := match[1] + match[2] + match[3]
			if !strings.HasPrefix(value, "//") && !externalURI.MatchString(value) {
				return ast.WalkStop, invalid(active.Rule.ID, "use Markdown links for relative HTML references")
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return "", err
	}
	edits = headingEdits(root, source, active.Rule.Title, edits)
	return strings.TrimSpace(applyEdits(body, edits)), nil
}

// headingEdits folds link edits into top-level headings, preventing overlapping replacements.
func headingEdits(root ast.Node, source []byte, title string, edits []edit) []edit {
	depth := 3
	for node := root.FirstChild(); node != nil; node = node.NextSibling() {
		if h, ok := node.(*ast.Heading); ok {
			if node == root.FirstChild() && plainHeading(h, source) == title {
				continue
			}
			depth = min(depth, h.Level)
		}
	}
	offset := 3 - depth
	for node := root.FirstChild(); node != nil; node = node.NextSibling() {
		h, ok := node.(*ast.Heading)
		if !ok {
			continue
		}
		segments := h.Source()
		if len(segments) == 0 {
			continue
		}
		start := h.Pos()
		last := segments[len(segments)-1].Stop
		end := lineEnd(source, max(start, last-1))
		if h.HeadingKind == ast.HeadingKindSetext {
			end = lineEnd(source, end)
		}
		// A trailing line break belongs to the surrounding document, not this replacement.
		for end > start && (source[end-1] == '\n' || source[end-1] == '\r') {
			end--
		}
		textStart, textEnd := segments[0].Start, segments[len(segments)-1].Stop
		inner := []edit{}
		outer := []edit{}
		for _, change := range edits {
			if change.start >= textStart && change.end <= textEnd {
				inner = append(inner, edit{change.start - textStart, change.end - textStart, change.value})
			} else {
				outer = append(outer, change)
			}
		}
		content := strings.TrimSpace(applyEdits(string(source[textStart:textEnd]), inner))
		content = strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", " "), "\n", " ")
		plain := true
		visible := ""
		for child := h.FirstChild(); child != nil; child = child.NextSibling() {
			if t, ok := child.(*ast.Text); ok {
				visible += t.Value.Value(source)
			} else {
				plain = false
			}
		}
		value := strings.Repeat("#", min(6, h.Level+offset)) + " " + content
		if node == root.FirstChild() && plain && visible == title {
			value = ""
		}
		edits = append(outer, edit{start, end, value})
	}
	return edits
}

// lineEnd returns the first byte after the next newline, or the source length.
func lineEnd(source []byte, start int) int {
	if at := bytes.IndexByte(source[start:], '\n'); at >= 0 {
		return start + at + 1
	}
	return len(source)
}

// applyEdits applies nonoverlapping original-source spans from right to left.
func applyEdits(source string, edits []edit) string {
	slices.SortFunc(edits, func(a, b edit) int { return b.start - a.start })
	for _, change := range edits {
		source = source[:change.start] + change.value + source[change.end:]
	}
	return source
}

// plainHeading returns decoded text only when the heading has no formatted inline children.
func plainHeading(heading *ast.Heading, source []byte) string {
	var result strings.Builder
	for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
		value, ok := child.(*ast.Text)
		if !ok {
			return ""
		}
		result.WriteString(value.Value.Value(source))
	}
	return result.String()
}
