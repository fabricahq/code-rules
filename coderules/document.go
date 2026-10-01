// Separate a rule's frontmatter envelope from its body without interpreting either.

package coderules

import (
	"regexp"

	"github.com/fabricahq/code-rules/internal/authored"
)

var documentPattern = regexp.MustCompile(`(?s)^---\r?\n(.*?)\r?\n---(?:\r?\n|$)(.*)$`)

// DocumentText contains exact input text excluding the delimiter lines and their
// adjacent newlines. Either field may be empty; neither is parsed or trimmed.
type DocumentText struct {
	Frontmatter string `json:"frontmatter"`
	Body        string `json:"body"`
}

// SplitDocument separates a rule document's required YAML frontmatter, between --- lines, from its Markdown body,
// such as a parsed Rule's Document. It validates only the delimiters, not YAML syntax, metadata fields, or Markdown
// content; location names the document in errors. On error, the returned document is the zero value.
func SplitDocument(text, location string) (DocumentText, error) {
	parts := documentPattern.FindStringSubmatch(text)
	if parts == nil {
		return DocumentText{}, authored.Invalid(location, "expected YAML frontmatter followed by Markdown")
	}
	return DocumentText{Frontmatter: parts[1], Body: parts[2]}, nil
}
