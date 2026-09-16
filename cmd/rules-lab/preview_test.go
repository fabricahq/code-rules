// Verify the browser preview renders Markdown without enabling authored HTML or unbounded requests.

package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMarkdownPreview renders headings and rejects active content through the real lab handler.
func TestMarkdownPreview(t *testing.T) {
	server := handler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	input := "# Go\n\n**When to read:** When editing Go.\n\n<script>alert(1)</script>\n\n[unsafe](javascript:alert(1))\n"
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/preview-markdown", strings.NewReader(input)))
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, "<h1>Go</h1>") || !strings.Contains(body, "<strong>When to read:</strong>") {
		t.Fatalf("preview did not render Markdown: %d %s", response.Code, body)
	}
	if strings.Contains(body, "<script>") || strings.Contains(body, `href="javascript:`) || !strings.Contains(response.Header().Get("Content-Security-Policy"), "sandbox") {
		t.Fatal("preview permitted active authored content")
	}
}

// TestMarkdownPreviewBoundaries rejects oversized bodies and cross-site preview requests.
func TestMarkdownPreviewBoundaries(t *testing.T) {
	server := handler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/preview-markdown", strings.NewReader(strings.Repeat("x", (1<<20)+1))))
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body returned %d", response.Code)
	}
	request := httptest.NewRequest(http.MethodPost, "/preview-markdown", strings.NewReader("# Go"))
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-site request returned %d", response.Code)
	}
}
