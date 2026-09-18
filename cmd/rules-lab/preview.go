// Render generated Markdown for the lab's sandboxed preview, without enabling raw HTML.

package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer/html"
)

// markdownPreview returns a bounded Markdown document as HTML for a sandboxed iframe.
func markdownPreview(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				http.Error(w, "preview exceeds 1 MiB limit; use Markdown source", http.StatusRequestEntityTooLarge)
			} else {
				http.Error(w, "could not read preview text", http.StatusBadRequest)
			}
			return
		}
		var body bytes.Buffer
		if err := html.New().Render(&body, data, parser.New().Parse(data)); err != nil {
			logger.ErrorContext(r.Context(), "render Markdown preview failed", "error", err)
			http.Error(w, "could not render preview", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
		page := `<!doctype html><html><head><meta charset="utf-8"><meta name="color-scheme" content="light dark"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'"><style>
body{font:14px/1.6 system-ui,sans-serif;margin:16px;overflow-wrap:anywhere;color:light-dark(#242424,#e8e8e8);background:light-dark(#fff,#171717)}
h1{font-size:24px;line-height:1.25;margin:0 0 16px}h2{font-size:20px}h3{font-size:17px;margin:24px 0 8px}p{margin:10px 0}a{color:light-dark(#285ec4,#91b6ff);pointer-events:none}code{font-size:12px}pre{white-space:pre-wrap}blockquote{margin-left:0;padding-left:12px;border-left:3px solid #888}
</style></head><body>` + body.String() + `</body></html>`
		if _, err := io.WriteString(w, page); err != nil {
			logger.WarnContext(r.Context(), "write Markdown preview failed", "error", err)
		}
	}
}
