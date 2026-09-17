// Exercise the development adapter through its JSON and HTTP boundaries.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/imports"
	"github.com/fabricahq/code-rules/internal/rules"
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
				if !got.OK || got.Value["id"] != "team:techs/go/example" || got.Value["document"] != test.text {
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
		{"link address error", `{"operation":"repositoryFile","location":"custom.location","input":{"repository":null,"commit":"abc123","path":"a.md","image":false}}`, `{"ok":false,"error":{"name":"ValidationError","message":"repository: expected nonempty text","location":"repository"}}`},
		{"missing repository", `{"operation":"repositoryFile","location":"repository","input":{"commit":"abc123","path":"a.md","image":false}}`, `{"ok":false,"error":{"name":"AdapterError","message":"repositoryFile input must contain repository, commit, path, and image"}}`},
		{"invalid wrapper", `{"operation":"repositoryFile","location":"repository","input":{"repository":"git@github.com:team/rules"}}`, `{"ok":false,"error":{"name":"AdapterError","message":"repositoryFile input must contain repository, commit, path, and image"}}`},
	}
	for _, test := range cases {
		// Exercise each request through the handler and inspect both output and logs.
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
			recorder := httptest.NewRecorder()
			handler(logger).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(test.body)))
			var got, expected any
			if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.expected), &expected); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || !reflect.DeepEqual(got, expected) {
				t.Fatalf("got %d %s; want %s", recorder.Code, recorder.Body, test.expected)
			}
			if strings.Contains(logs.String(), "secret") || strings.Contains(logs.String(), "example.org") {
				t.Fatalf("request content leaked into logs: %s", &logs)
			}
		})
	}
}

// TestHTTPRefs checks native ref results, version validation, and adapter type errors.
func TestHTTPRefs(t *testing.T) {
	for _, test := range []struct{ name, operation, input, expected string }{
		{"tag", "gitRef", `"main"`, `{"ok":true,"value":{"kind":"tag","name":"refs/tags/main"}}`},
		{"invalid ref", "gitRef", `"deadbeef"`, `{"ok":false,"error":{"name":"ValidationError","message":"custom.ref: abbreviated commits are unsupported; use a full SHA or refs/tags/<name>","location":"custom.ref"}}`},
		{"version", "tagVersion", `"v1.2.3-beta.1+build"`, `{"ok":true,"value":"1.2.3-beta.1+build"}`},
		{"non-version", "tagVersion", `"1.2"`, `{"ok":false,"error":{"name":"ValidationError","message":"custom.ref: expected a complete semantic version tag, such as v1.2.3 or 1.2.3-beta.1+build.5","location":"custom.ref"}}`},
	} {
		// Exercise serialization through HTTP, including the caller-owned location for invalid versions.
		t.Run(test.name, func(t *testing.T) {
			payload := `{"operation":"` + test.operation + `","input":` + test.input + `,"location":"custom.ref"}`
			recorder := httptest.NewRecorder()
			handler(slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(payload)))
			var got, expected any
			if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.expected), &expected); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || !reflect.DeepEqual(got, expected) {
				t.Fatalf("HTTP %d: %s; want %s", recorder.Code, recorder.Body, test.expected)
			}
		})
	}
	for _, operation := range []string{"gitRef", "tagVersion"} {
		for _, input := range []string{`null`, `42`, `{}`} {
			got, err := invoke([]byte(`{"operation":"` + operation + `","input":` + input + `,"location":"ref"}`))
			if err != nil || got.OK || got.Value != nil || got.Error == nil || got.Error.Name != "AdapterError" {
				t.Fatalf("%s input %s: got %+v, %v", operation, input, got, err)
			}
		}
	}
}

// TestHTTPVersionConstraints checks matching, nonmatching, and invalid inputs through HTTP.
func TestHTTPVersionConstraints(t *testing.T) {
	for _, test := range []struct{ name, operation, input, expected string }{
		{"parse", "versionConstraint", `"~> 1.2.3"`, `{"ok":true,"value":"~> 1.2.3"}`},
		{"match", "versionMatch", `{"constraint":"~> 1.2.3","version":"1.2.9"}`, `{"ok":true,"value":true}`},
		{"nonmatch", "versionMatch", `{"constraint":"~> 1.2.3","version":"1.3.0"}`, `{"ok":true,"value":false}`},
		{"invalid version", "versionMatch", `{"constraint":"~> 1.2.3","version":"1.2"}`, `{"ok":false,"error":{"name":"ValidationError","message":"release.version: expected a complete semantic version tag, such as v1.2.3 or 1.2.3-beta.1+build.5","location":"release.version"}}`},
		{"invalid constraint first", "versionMatch", `{"constraint":"","version":"1.2"}`, `{"ok":false,"error":{"name":"ValidationError","message":"release.constraint: expected nonempty text","location":"release.constraint"}}`},
	} {
		// Verify the serialized response separates domain false from a returned error.
		t.Run(test.name, func(t *testing.T) {
			payload := `{"operation":"` + test.operation + `","input":` + test.input + `,"location":"release"}`
			recorder := httptest.NewRecorder()
			handler(slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(payload)))
			var got, expected any
			if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.expected), &expected); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || !reflect.DeepEqual(got, expected) {
				t.Fatalf("HTTP %d: %s; want %s", recorder.Code, recorder.Body, test.expected)
			}
		})
	}
	for _, input := range []string{`null`, `42`, `{}`, `{"constraint":null,"version":"1.2.3"}`, `{"constraint":"*","version":42}`, `{"constraint":"1","version":"1.0.0","extra":true}`} {
		got, err := invoke([]byte(`{"operation":"versionMatch","input":` + input + `,"location":"release"}`))
		if err != nil || got.OK || got.Value != nil || got.Error == nil || got.Error.Name != "AdapterError" {
			t.Fatalf("input %s: %+v, %v", input, got, err)
		}
	}
}

// TestReleaseTagTransportLimits lets domain-sized inputs reach Go through both transports.
func TestReleaseTagTransportLimits(t *testing.T) {
	const tagLine = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\trefs/tags/v1.2.3\n"
	for _, extra := range []int{0, 1} {
		// Six-byte JSON escapes must not make an otherwise valid tag listing too large.
		listing := tagLine + strings.Repeat("\n", 8*1024*1024-len(tagLine)+extra)
		encoded, err := json.Marshal(listing)
		if err != nil {
			t.Fatal(err)
		}
		payload := `{"operation":"selectReleaseTag","location":"selection","input":{"constraint":">= 1.0.0","availableGitTags":` + strings.ReplaceAll(string(encoded), `\n`, `\u000a`) + `}}`
		for _, transport := range []string{"HTTP", "stdin"} {
			// Invoke the same payload through the actual HTTP or newline-delimited boundary.
			t.Run(fmt.Sprintf("%s/extra=%d", transport, extra), func(t *testing.T) {
				var output bytes.Buffer
				if transport == "HTTP" {
					recorder := httptest.NewRecorder()
					handler(slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(payload)))
					if recorder.Code != http.StatusOK {
						t.Fatalf("HTTP %d: %s", recorder.Code, recorder.Body)
					}
					output.Write(recorder.Body.Bytes())
				} else if err := runRequests(strings.NewReader(payload+"\n"), &output); err != nil {
					t.Fatal(err)
				}
				var got struct {
					OK    bool
					Value rules.VersionSelection
					Error *failure
				}
				if err := json.Unmarshal(output.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if extra == 0 {
					want := rules.VersionSelection{Tag: "v1.2.3", Version: "1.2.3", Object: strings.Repeat("a", 40)}
					if !got.OK || got.Error != nil || got.Value != want {
						t.Fatalf("unexpected selection: %s", &output)
					}
				} else if got.OK || got.Error == nil || got.Error.Code != string(rules.TagLimitExceeded) {
					t.Fatalf("expected domain size error: %s", &output)
				}
			})
		}
	}
}

// TestLibraryFixtureBoundary exercises actual disposable filesystem loading and rejects escaped writes.
func TestLibraryFixtureBoundary(t *testing.T) {
	for _, test := range []struct {
		name, input string
		ok          bool
	}{
		{"empty library", `{"files":{"rule-library.json":"{\"formatVersion\":1}"},"groups":"*","source":"team"}`, true},
		{"escape", `{"files":{"../escape":"x"},"groups":"*","source":"team"}`, false},
		{"missing manifest", `{"files":{},"groups":"*","source":"team"}`, false},
	} {
		// Execute the serialized lab boundary so fixture safety is checked before writes.
		t.Run(test.name, func(t *testing.T) {
			got, err := invoke([]byte(`{"operation":"loadLibrary","input":` + test.input + `,"location":"fixture"}`))
			if err != nil || got.OK != test.ok || (!test.ok && (got.Error == nil || got.Error.Name != "ValidationError")) {
				t.Fatalf("%+v, %v", got, err)
			}
		})
	}
}

// TestFixtureRejectsNullText prevents JSON null from silently becoming an empty fixture file.
func TestFixtureRejectsNullText(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(`{"operation":"loadLibrary","location":"fixture","input":{"files":{"rule-library.json":"{\"formatVersion\":1}","extra.txt":null},"groups":[],"source":"team"}}`))
	recorder := httptest.NewRecorder()
	handler(slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(recorder, req)
	if !strings.Contains(recorder.Body.String(), `"ok":false`) || !strings.Contains(recorder.Body.String(), `AdapterError`) {
		t.Fatalf("null text accepted: %s", recorder.Body)
	}
}

// TestMarkdownTargetsRequiresExplicitText rejects malformed requests rather than silently substituting empty input.
func TestMarkdownTargetsRequiresExplicitText(t *testing.T) {
	for _, input := range []string{`{"file":"techs/go/r.md"}`, `{"text":null,"file":"techs/go/r.md"}`, `{"text":"","file":"techs/go/r.md","extra":true}`} {
		req := httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(`{"operation":"markdownTargets","location":"links","input":`+input+`}`))
		recorder := httptest.NewRecorder()
		handler(slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(recorder, req)
		if !strings.Contains(recorder.Body.String(), `"ok":false`) {
			t.Fatal(recorder.Body.String())
		}
	}
}

// TestLibraryLicenseJSON exposes no declaration as null and one declaration with multiple notices as an object.
func TestLibraryLicenseJSON(t *testing.T) {
	for _, test := range []struct{ name, manifest, want string }{
		{"undeclared", `{"formatVersion":1}`, `null`},
		{"one with notices", `{"formatVersion":1,"license":{"file":"LICENSE","notices":["NOTICE","AUTHORS"],"spdxExpression":"MIT"}}`, `{"spdxExpression":"MIT","files":["LICENSE"],"attributionFiles":["NOTICE","AUTHORS"]}`},
	} {
		// Use real temporary library loading behind the same HTTP boundary as the browser.
		t.Run(test.name, func(t *testing.T) {
			payload, err := json.Marshal(map[string]any{"operation": "loadLibrary", "location": "fixture", "input": map[string]any{"files": map[string]string{"rule-library.json": test.manifest, "LICENSE": "Terms\r\n", "NOTICE": "Notice", "AUTHORS": "Authors"}, "groups": []string{}, "source": "team"}})
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			handler(slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/invoke", bytes.NewReader(payload)))
			var result struct {
				OK    bool                       `json:"ok"`
				Value map[string]json.RawMessage `json:"value"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil || !result.OK {
				t.Fatalf("load failed: %s, %v", recorder.Body, err)
			}
			if string(result.Value["license"]) != test.want {
				t.Fatalf("license = %s; want %s", result.Value["license"], test.want)
			}
			if _, exists := result.Value["licenses"]; exists {
				t.Fatal("obsolete licenses array is still exposed")
			}
		})
	}
}

// TestResolveFixtureRejectsExcludedRuleLinks exercises real library loading before either linked rule can be excluded.
func TestResolveFixtureRejectsExcludedRuleLinks(t *testing.T) {
	document := "---\ntitle: Return errors\nimpact: HIGH\nimpactDescription: Preserve failures.\nwhenToRead: When calling functions.\n---\nReturn errors.\n"
	for _, excluded := range []string{"techs/go/one", "techs/go/two"} {
		t.Run(excluded, func(t *testing.T) {
			input := map[string]any{
				"configuration": map[string]any{"schemaVersion": 1, "sources": map[string]any{"team": map[string]any{
					"repository": "https://github.com/acme/rules", "ref": "v1.0.0", "groups": []string{"techs/go"},
					"exclude": map[string]string{excluded: "Project policy"}, "replace": map[string]any{},
				}}},
				"libraries": map[string]any{"team": map[string]any{
					"commit": strings.Repeat("a", 40), "files": map[string]string{
						"rule-library.json":    `{"formatVersion":1}`,
						"techs/go/_group.json": `{"name":"Go","description":"Go guidance.","whenToRead":"When editing Go."}`,
						"techs/go/one.md":      document + "\n[other](two.md)\n", "techs/go/two.md": document,
					},
				}},
				"localFiles": map[string]string{},
			}
			payload, err := json.Marshal(map[string]any{"operation": "resolveRules", "input": input, "location": "fixture"})
			if err != nil {
				t.Fatal(err)
			}
			got, err := invoke(payload)
			if err != nil || got.OK || got.Value != nil || got.Error == nil || !strings.Contains(got.Error.Message, "links to other rule documents are not allowed") {
				t.Fatalf("expected rule-link failure before exclusion: %+v, %v", got, err)
			}
		})
	}
}

// TestProjectFixtureSetupFailureIsHTTP500 exercises unavailable temporary storage through the HTTP endpoint.
func TestProjectFixtureSetupFailureIsHTTP500(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir()+"/missing")
	var logs bytes.Buffer
	rec := httptest.NewRecorder()
	handler(slog.New(slog.NewJSONHandler(&logs, nil))).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(`{"operation":"projectWrite","location":"project","input":{"scenario":"apply","output":{"generated":{}}}}`)))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if logs.Len() == 0 {
		t.Fatal("operational failure was not logged")
	}
	if strings.Contains(rec.Body.String(), "missing") {
		t.Fatal("filesystem path leaked to response")
	}
}

// TestGitFailureKeepsCleanupDiagnostic exposes both the primary import failure and a joined cleanup failure.
func TestGitFailureKeepsCleanupDiagnostic(t *testing.T) {
	primary := &imports.Error{Code: "git-failed", Problem: "Fetch failed."}
	cleanup := &imports.Error{Code: "cleanup-failed", Problem: "Temporary repository cleanup failed."}
	got := gitFailure(errors.Join(primary, cleanup))
	if got == nil || got.Code != "git-failed" || !strings.Contains(got.Message, primary.Problem) || !strings.Contains(got.Message, cleanup.Problem) {
		t.Fatalf("lost primary or cleanup diagnostic: %+v", got)
	}
}
