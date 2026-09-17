// Keep successful long-running native invocations observable over the browser's HTTP boundary.

package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestLongInvocationResponse exercises a real HTTP response beyond the former ten-second write deadline.
func TestLongInvocationResponse(t *testing.T) {
	server := httptest.NewUnstartedServer(nil)
	server.Config = labServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := server.Config.Handler
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow Git import before entering the unchanged native request handler.
		time.Sleep(10500 * time.Millisecond)
		handler.ServeHTTP(w, r)
	})
	server.Start()
	defer server.Close()
	client := &http.Client{Timeout: 20 * time.Second}
	result, err := client.Post(server.URL+"/invoke", "application/json", strings.NewReader(`{"operation":"gitImport","location":"libraries","input":{"configuration":{"schemaVersion":1,"sources":{}},"libraries":{},"scenario":"import"}}`))
	if err != nil {
		t.Fatalf("long invocation lost its HTTP response: %v", err)
	}
	defer result.Body.Close()
	var output response
	if err := json.NewDecoder(result.Body).Decode(&output); err != nil || !output.OK {
		t.Fatalf("long invocation: %+v %v", output, err)
	}
}
