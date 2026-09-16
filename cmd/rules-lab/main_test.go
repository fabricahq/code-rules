// Exercise the development adapter through its JSON and HTTP boundaries.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHTTPValidationDoesNotLogInput checks that expected validation failures never expose user input in logs.
func TestHTTPValidationDoesNotLogInput(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	req := httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(`{"operation":"groupID","input":"private-user-content","location":"private-field"}`))
	recorder := httptest.NewRecorder()
	handler(logger).ServeHTTP(recorder, req)
	if !strings.Contains(recorder.Body.String(), "ValidationError") || !strings.Contains(logs.String(), `"ok":false`) {
		t.Fatalf("missing validation response or debug outcome: response=%s, logs=%s", recorder.Body, &logs)
	}
	if strings.Contains(logs.String(), "private-") || strings.Contains(logs.String(), `"level":"ERROR"`) {
		t.Fatalf("expected input failures must not expose content or produce error logs: %s", &logs)
	}
}

type brokenBody struct{}

// Read simulates an input failure containing details that must not reach the client.
func (brokenBody) Read([]byte) (int, error) { return 0, errors.New("private read details") }

// Close satisfies io.ReadCloser without any resources to release.
func (brokenBody) Close() error { return nil }

// TestHTTPBodyReadFailure checks that unreadable requests return a safe response without internal details.
func TestHTTPBodyReadFailure(t *testing.T) {
	var logs bytes.Buffer
	req := httptest.NewRequest(http.MethodPost, "/invoke", brokenBody{})
	recorder := httptest.NewRecorder()
	handler(slog.New(slog.NewJSONHandler(&logs, nil))).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest || recorder.Body.String() != "could not read request body\n" || logs.Len() != 0 {
		t.Fatalf("got status=%d, response=%s, logs=%s", recorder.Code, recorder.Body, &logs)
	}
}

type brokenResponse struct{ *httptest.ResponseRecorder }

// Write simulates a client disconnect while the handler sends a response.
func (brokenResponse) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// TestHTTPWriteFailureLoggedOnce checks that failed page and result writes each produce one warning.
func TestHTTPWriteFailureLoggedOnce(t *testing.T) {
	for _, path := range []string{"/", "/invoke"} {
		t.Run(path, func(t *testing.T) {
			var logs bytes.Buffer
			method := http.MethodGet
			if path == "/invoke" {
				method = http.MethodPost
			}
			req := httptest.NewRequest(method, path, strings.NewReader(`{"operation":"selection","input":[],"location":"groups"}`))
			handler(slog.New(slog.NewJSONHandler(&logs, nil))).ServeHTTP(brokenResponse{httptest.NewRecorder()}, req)
			if strings.Count(logs.String(), "\n") != 1 || !strings.Contains(logs.String(), `"level":"WARN"`) || !strings.Contains(logs.String(), "closed pipe") {
				t.Fatalf("want one warning for undeliverable response, got %s", &logs)
			}
		})
	}
}

// TestInvokeBoundary checks request decoding and the distinction between adapter and domain failures.
func TestInvokeBoundary(t *testing.T) {
	cases := []struct{ name, input, errorName string }{
		{"rule group", `{"operation":"ruleGroup","input":"techs/go/a.md","location":"rule"}`, ""},
		{"domain error", `{"operation":"groupID","input":"techs/Go","location":"group"}`, "ValidationError"},
		{"missing selection", `{"operation":"selection","location":"groups"}`, "ValidationError"},
		{"null text", `{"operation":"groupID","input":null,"location":"group"}`, "AdapterError"},
		{"wrong text type", `{"operation":"groupID","input":42,"location":"group"}`, "AdapterError"},
		{"unknown operation", `{"operation":"writeFile","input":"x","location":"group"}`, "AdapterError"},
		{"unknown field", `{"operation":"groupID","input":"techs/go","location":"group","shell":"x"}`, "AdapterError"},
		{"malformed JSON", `{"operation":"selection","input":[}`, "AdapterError"},
		{"document input must be text", `{"operation":"document","input":{},"location":"rule"}`, "AdapterError"},
		{"missing document envelope", `{"operation":"document","input":"Body","location":"rule"}`, "ValidationError"},
		{"malformed metadata document", `{"operation":"groupMetadata","input":"{","location":"group"}`, "ValidationError"},
		{"metadata must be document text", `{"operation":"groupMetadata","input":{},"location":"group"}`, "AdapterError"},
		{"trailing value", `{"operation":"selection","input":[],"location":"groups"} true`, "AdapterError"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := invoke([]byte(test.input))
			if err != nil {
				t.Fatal(err)
			}
			if test.errorName == "" {
				if !got.OK || got.Value != "techs/go" {
					t.Fatalf("unexpected success: %+v", got)
				}
			} else if got.OK || got.Error == nil || got.Error.Name != test.errorName {
				t.Fatalf("unexpected error: %+v", got)
			}
		})
	}
}

// TestHTTPParsesGroupMetadata checks metadata parsing and whitespace trimming through HTTP.
func TestHTTPParsesGroupMetadata(t *testing.T) {
	body := `{"operation":"groupMetadata","input":"{\"name\":\"Go\",\"description\":\"Go rules\",\"whenToRead\":\" When editing. \"}","location":"techs/go/_group.json"}`
	req := httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler(slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(recorder, req)
	var got struct {
		OK    bool
		Value struct {
			Name, Description string
			WhenToRead        string
		}
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || !got.OK || got.Value.Name != "Go" || got.Value.Description != "Go rules" || got.Value.WhenToRead != "When editing." {
		t.Fatalf("unexpected metadata response: HTTP %d %s", recorder.Code, recorder.Body)
	}
}

// TestHTTPInvokesNativeFunction checks that a same-origin request returns the native selection result.
func TestHTTPInvokesNativeFunction(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/invoke", strings.NewReader(`{"operation":"selection","input":["techs/go","practices/testing"],"location":"groups"}`))
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	recorder := httptest.NewRecorder()
	handler(slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("got HTTP %d: %s", recorder.Code, recorder.Body)
	}
	var got struct {
		OK    bool
		Value []string
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.OK || strings.Join(got.Value, ",") != "practices/testing,techs/go" {
		t.Fatalf("unexpected response: %s", recorder.Body)
	}
}

// TestHTTPRejectsCrossOriginAndOversizedRequests checks the HTTP origin and request-size limits.
func TestHTTPRejectsCrossOriginAndOversizedRequests(t *testing.T) {
	for _, test := range []struct {
		name, origin, body string
		status             int
	}{
		{"cross origin", "https://unrelated.example", `{}`, http.StatusForbidden},
		{"body limit", "http://127.0.0.1:8080", strings.Repeat(" ", maxRequestBytes+1), http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/invoke", strings.NewReader(test.body))
			req.Header.Set("Origin", test.origin)
			recorder := httptest.NewRecorder()
			handler(slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(recorder, req)
			if recorder.Code != test.status {
				t.Fatalf("got HTTP %d, want %d", recorder.Code, test.status)
			}
		})
	}
}

// TestLoggerFromEnvironment checks logging defaults, configured formats and levels, and private configuration errors.
func TestLoggerFromEnvironment(t *testing.T) {
	for _, test := range []struct {
		name, level, format, errorField string
		wantJSON, wantDebug             bool
	}{
		{name: "defaults"},
		{name: "mixed case", level: "dEbUg", format: "JsOn", wantJSON: true, wantDebug: true},
		{name: "numeric offset", level: "INFO+2", format: "json", wantJSON: true},
		{name: "invalid level", level: "private-value", errorField: "CODE_RULES_LOG_LEVEL"},
		{name: "invalid format", format: "private-value", errorField: "CODE_RULES_LOG_FORMAT"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("CODE_RULES_LOG_LEVEL", test.level)
			t.Setenv("CODE_RULES_LOG_FORMAT", test.format)
			var output bytes.Buffer
			logger, err := loggerFromEnvironment(&output)
			if test.errorField != "" {
				if logger != nil || err == nil || !strings.Contains(err.Error(), test.errorField) || strings.Contains(err.Error(), "private-value") || output.Len() != 0 {
					t.Fatalf("unexpected configuration failure: logger=%v error=%v output=%s", logger, err, &output)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			logger.Debug("debug event")
			logger.Warn("warning event")
			if strings.Contains(output.String(), "debug event") != test.wantDebug || !strings.Contains(output.String(), "warning event") {
				t.Fatalf("unexpected level filtering: %s", &output)
			}
			if strings.HasPrefix(output.String(), "{") != test.wantJSON {
				t.Fatalf("unexpected format: %s", &output)
			}
		})
	}
}

// TestHTTPDocumentPreservesTextAndEmptyFields checks exact document captures through HTTP, including empty strings.
func TestHTTPDocumentPreservesTextAndEmptyFields(t *testing.T) {
	for _, test := range []struct {
		name, input, frontmatter, body string
	}{
		{"CRLF and Unicode", "---\r\ntitle: 🐹\r\nnote: exact  \r\n---\r\n\tBody  \r\n", "title: 🐹\r\nnote: exact  ", "\tBody  \r\n"},
		{"empty body", "---\nx\n---", "x", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload, err := json.Marshal(map[string]string{"operation": "document", "input": test.input, "location": "rule"})
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			handler(slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/invoke", bytes.NewReader(payload)))
			var got struct {
				OK    bool
				Value map[string]string
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			body, present := got.Value["body"]
			if recorder.Code != http.StatusOK || !got.OK || !present || len(got.Value) != 2 || got.Value["frontmatter"] != test.frontmatter || body != test.body {
				t.Fatalf("unexpected document response: HTTP %d %s", recorder.Code, recorder.Body)
			}
		})
	}
}

// TestHTTPParsesCompleteRule checks complete-rule results, typed failures, and logging privacy through HTTP.
func TestHTTPParsesCompleteRule(t *testing.T) {
	metadata := "title: Go\r\nimpact: HIGH\r\nimpactDescription: Avoid failures\r\nwhenToRead: Editing Go"
	for _, test := range []struct{ name, text, errorName string }{
		{"valid", "---\r\n" + metadata + "\r\n---\r\n Body  \r\n", ""},
		{"empty body", "---\n" + metadata + "\n---", "ValidationError"},
		{"malformed YAML", "---\ntitle: [\n---\nBody", "ValidationError"},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload, err := json.Marshal(map[string]any{"operation": "rule", "input": map[string]string{"text": test.text, "path": "techs/go/example.md", "source": "team"}})
			if err != nil {
				t.Fatal(err)
			}
			var logs bytes.Buffer
			recorder := httptest.NewRecorder()
			handler(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/invoke", bytes.NewReader(payload)))
			var got struct {
				OK    bool
				Value map[string]any
				Error *struct{ Name, Location string }
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK {
				t.Fatalf("HTTP %d: %s", recorder.Code, recorder.Body)
			}
			if test.errorName == "" {
				if !got.OK || got.Value["id"] != "team:techs/go/example" || got.Value["metadata"] != metadata || got.Value["body"] != " Body  \r\n" {
					t.Fatalf("unexpected rule: %s", recorder.Body)
				}
			} else if got.OK || got.Value != nil || got.Error == nil || got.Error.Name != test.errorName || !strings.HasPrefix(got.Error.Location, "team:techs/go/example.md") {
				t.Fatalf("unexpected failure: %s", recorder.Body)
			}
			if strings.Contains(logs.String(), "Avoid failures") || strings.Contains(logs.String(), "example.md") {
				t.Fatalf("input leaked to logs: %s", &logs)
			}
		})
	}
}

// TestRuleAdapterRejectsInvalidInput checks that malformed rule arguments fail before domain parsing.
func TestRuleAdapterRejectsInvalidInput(t *testing.T) {
	for _, input := range []string{`null`, `"text"`, `{}`, `{"text":null,"path":"x","source":"s"}`, `{"text":1,"path":"x","source":"s"}`, `{"text":"x","path":"x","source":"s","extra":true}`} {
		got, err := invoke([]byte(`{"operation":"rule","input":` + input + `}`))
		if err != nil || got.OK || got.Value != nil || got.Error == nil || got.Error.Name != "AdapterError" {
			t.Fatalf("input %s: got %+v, error %v", input, got, err)
		}
	}
}

// TestHTTPRepositories checks native links, null results, and credential privacy through HTTP.
func TestHTTPRepositories(t *testing.T) {
	cases := []struct{ name, body, expected string }{
		{"file", `{"operation":"repositoryFile","location":"repository","input":{"repository":"git@github.com:Team/Rules.git","commit":"abc123","path":"a b.md","image":false}}`, `{"ok":true,"value":"https://github.com/Team/Rules/blob/abc123/a%20b.md"}`},
		{"unknown web", `{"operation":"repositoryFile","location":"repository","input":{"repository":"git@host.example:Rules.git","commit":"abc123","path":"a.md","image":false}}`, `{"ok":true,"value":null}`},
		{"credentials", `{"operation":"repository","location":"repository","input":"https://user:secret@example.org/rules"}`, `{"ok":false,"error":{"name":"ValidationError","message":"repository: repository requires a host and must not embed credentials; SSH may specify a username","location":"repository"}}`},
		{"link address error", `{"operation":"repositoryFile","location":"repository","input":{"repository":null,"commit":"abc123","path":"a.md","image":false}}`, `{"ok":false,"error":{"name":"ValidationError","message":"repository: expected nonempty text","location":"repository"}}`},
		{"invalid wrapper", `{"operation":"repositoryFile","location":"repository","input":{"repository":"git@github.com:team/rules"}}`, `{"ok":false,"error":{"name":"AdapterError","message":"repositoryFile input must contain repository, commit, path, and image"}}`},
	}
	for _, test := range cases {
		// Exercise each request through the handler and inspect both output and logs.
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
			recorder := httptest.NewRecorder()
			handler(logger).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(test.body)))
			if recorder.Code != http.StatusOK || strings.TrimSpace(recorder.Body.String()) != test.expected {
				t.Fatalf("got %d %s; want %s", recorder.Code, recorder.Body, test.expected)
			}
			if strings.Contains(logs.String(), "secret") || strings.Contains(logs.String(), "example.org") {
				t.Fatalf("request content leaked into logs: %s", &logs)
			}
		})
	}
}
