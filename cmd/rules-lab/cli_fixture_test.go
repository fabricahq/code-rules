// Verify CLI walkthrough arguments cannot select an unvalidated project configuration.

package main

import (
	"encoding/json"
	"testing"
)

// TestCLIRejectsDefaultConfig prevents fixture edits from redirecting the child CLI to external repositories.
func TestCLIRejectsDefaultConfig(t *testing.T) {
	input := json.RawMessage(`{"configuration":{"schemaVersion":1,"sources":{}},"libraries":{},"arguments":["sync"],"changes":{".code-rules/config.json":"{\"schemaVersion\":1,\"sources\":{\"external\":{\"repository\":\"git@fixture.invalid:external\",\"ref\":\"v1.0.0\",\"groups\":\"*\"}}}"}}`)
	result, err := cliResponse(input)
	if err != nil || result.OK || result.Error == nil || result.Error.Location != "arguments" {
		t.Fatalf("expected argument rejection before process setup: %+v, %v", result, err)
	}
}
