package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
		{"trailing value", `{"operation":"selection","input":[],"location":"groups"} true`, "AdapterError"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := invoke([]byte(test.input))
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

func TestHTTPInvokesNativeFunction(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/invoke", strings.NewReader(`{"operation":"selection","input":["techs/go","practices/testing"],"location":"groups"}`))
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	recorder := httptest.NewRecorder()
	handler().ServeHTTP(recorder, req)
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

func TestHTTPRejectsCrossOriginAndOversizedRequests(t *testing.T) {
	for _, test := range []struct {
		name, origin, body string
		status             int
	}{
		{"cross origin", "https://unrelated.example", `{}`, http.StatusForbidden},
		{"body limit", "http://127.0.0.1:8080", strings.Repeat(" ", maxRequestBytes+1), http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/invoke", strings.NewReader(test.body))
			req.Header.Set("Origin", test.origin)
			recorder := httptest.NewRecorder()
			handler().ServeHTTP(recorder, req)
			if recorder.Code != test.status {
				t.Fatalf("got HTTP %d, want %d", recorder.Code, test.status)
			}
		})
	}
}
