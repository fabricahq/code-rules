// Verify that editable sync scenarios cannot escape the lab's disposable Git fixtures.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestSyncFixtureRejectsChangedRemote keeps post-setup configuration edits inside the local fixture router.
func TestSyncFixtureRejectsChangedRemote(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(http.StatusNotFound) }))
	defer server.Close()
	config := func(repository string) string {
		return `{"schemaVersion":1,"sources":{"team":{"repository":"` + repository + `","ref":"v1.0.0","groups":[],"exclude":{},"replace":{}}}}`
	}
	changed := config(strings.Replace(server.URL, "http://", "https://", 1) + "/rules.git")
	raw, err := json.Marshal(map[string]any{
		"configuration": json.RawMessage(config("git@fixture.invalid:team")),
		"libraries":     map[string]any{"team": map[string]any{"files": map[string]string{"rule-library.json": `{"formatVersion":1}`}}},
		"scenario":      "sync", "changes": map[string]string{"config.json": changed},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := syncResponse(raw)
	if err != nil || result.OK || result.Error == nil || !strings.Contains(result.Error.Message, "supplied local Git fixture") {
		t.Fatalf("changed remote was not rejected: %+v %v", result.Error, err)
	}
	if requests.Load() != 0 {
		t.Fatal("walkthrough contacted an edited remote")
	}
}
