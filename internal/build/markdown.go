// Rewrite only parsed Markdown destinations and headings, preserving untouched source text.

package build

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"

	"github.com/fabricahq/code-rules/internal/rules"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/text"
)

// edit replaces a byte span in the original body.
type edit struct {
	start, end int
	value      string
}

var externalURI = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// renderBody rewrites destinations, removes a duplicate title, and nests guidance headings.
func renderBody(body string, active resolvedRule, paths []string, outputPath string) (string, error) {
	source := []byte(body)
	root := parser.New().Parse(source)
	edits := []edit{}
	bodyHeadingDepth := 3
	if outputPath != rulePath(active.Rule) {
		edits = referenceEdits(root, active.Rule.ID)
		bodyHeadingDepth = 5
	}
	destinations, err := rules.MarkdownDestinations(body)
	if err != nil {
		return "", invalid(active.Rule.ID, err.Error())
	}
	for _, destination := range destinations {
		value, err := relocatedURL(destination.Value, active, paths, outputPath)
		if err != nil {
			return "", err
		}
		// The existing delimiters remain intact.
		edits = append(edits, edit{destination.Start, destination.End, rules.EscapeDestination(value)})
	}
	var rawHTML strings.Builder
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.RawHTML:
			rawHTML.WriteString(n.Value.Value(source))
		case *ast.HTMLBlock:
			rawHTML.Write(n.Value.Bytes(source))
		}
		rawHTML.WriteByte('\n')
		return ast.WalkContinue, nil
	})
	// One tokenizer preserves script and textarea context across separate inline HTML nodes.
	if hasRelativeHTMLReference(rawHTML.String()) {
		return "", invalid(active.Rule.ID, "use Markdown links for relative HTML references")
	}
	edits = headingEdits(root, source, active.Rule.Title, bodyHeadingDepth, edits)
	return strings.TrimSpace(applyEdits(body, edits)), nil
}

// referenceEdits gives each rule its own reference labels so combined documents cannot capture another rule's links.
func referenceEdits(root ast.Node, ruleID string) []edit {
	labels := map[text.Index]string{}
	edits := []edit{}
	// The digest keeps generated labels short even for long source-qualified IDs.
	prefix := fmt.Sprintf("code-rules-%x-", sha256.Sum256([]byte(ruleID)))
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if definition, ok := node.(*ast.LinkReferenceDefinition); entering && ok {
			label := fmt.Sprintf("%s%d", prefix, len(labels))
			labels[definition.Destination.Index()] = label
			indices := definition.Label.Indices()
			edits = append(edits, edit{indices[0].Start, indices[len(indices)-1].Stop, label})
		}
		return ast.WalkContinue, nil
	})
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var reference *ast.ReferenceLink
		var destination text.SingleLineValue
		switch n := node.(type) {
		case *ast.Link:
			reference, destination = n.Reference, n.Destination
		case *ast.Image:
			reference, destination = n.Reference, n.Destination
		}
		if reference != nil {
			label := labels[destination.Index()]
			indices := reference.Value.Indices()
			start, end := indices[0].Start, indices[len(indices)-1].Stop
			if reference.ReferenceLinkKind == ast.ReferenceLinkKindShortcut {
				start, end, label = end+1, end+1, "["+label+"]"
			}
			edits = append(edits, edit{start, end, label})
		}
		return ast.WalkContinue, nil
	})
	return edits
}

// headingEdits folds link edits into headings, preventing overlapping replacements.
func headingEdits(root ast.Node, source []byte, title string, bodyHeadingDepth int, edits []edit) []edit {
	headings := []*ast.Heading{}
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if heading, ok := node.(*ast.Heading); ok {
				headings = append(headings, heading)
			}
		}
		return ast.WalkContinue, nil
	})
	depth := bodyHeadingDepth
	for _, h := range headings {
		if h == root.FirstChild() && plainHeading(h, source) == title {
			continue
		}
		depth = min(depth, h.Level)
	}
	offset := bodyHeadingDepth - depth
	for _, h := range headings {
		segments := h.Source()
		if len(segments) == 0 {
			end := lineEnd(source, h.Pos())
			for end > h.Pos() && (source[end-1] == '\n' || source[end-1] == '\r') {
				end--
			}
			edits = append(edits, edit{h.Pos(), end, strings.Repeat("#", min(6, h.Level+offset))})
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
		content := headingContent(source, segments, inner)
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
		if h == root.FirstChild() && plain && visible == title {
			value = ""
		}
		edits = append(outer, edit{start, end, value})
	}
	return edits
}

// headingContent applies edits across source lines before flattening a heading, excluding container prefixes.
func headingContent(source []byte, segments []text.Segment, edits []edit) string {
	start := segments[0].Start
	end := segments[len(segments)-1].Stop
	for i := 1; i < len(segments); i++ {
		gapStart, gapEnd := segments[i-1].Stop-start, segments[i].Start-start
		if gapStart == gapEnd {
			continue
		}
		covered := false
		for _, change := range edits {
			if change.start < gapEnd && change.end > gapStart {
				covered = true
				break
			}
		}
		if !covered {
			edits = append(edits, edit{gapStart, gapEnd, ""})
		}
	}
	lines := strings.Split(applyEdits(string(source[start:end]), edits), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.TrimSpace(strings.Join(lines, " "))
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

// hasRelativeHTMLReference inspects actual HTML attributes, excluding comments, data attributes, and script text.
func hasRelativeHTMLReference(fragment string) bool {
	tokenizer := html.NewTokenizer(strings.NewReader(fragment))
	for {
		kind := tokenizer.Next()
		if kind == html.ErrorToken {
			return false
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		for _, attribute := range tokenizer.Token().Attr {
			if attribute.Key != "href" && attribute.Key != "src" {
				continue
			}
			if !strings.HasPrefix(attribute.Val, "//") && !externalURI.MatchString(attribute.Val) {
				return true
			}
		}
	}
}
