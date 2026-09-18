// Verify that JSON transport retains exact binary evidence alongside readable text.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestAcceptanceFilesPreserveBytes round-trips the pilot's binary fixture through its actual display JSON.
func TestAcceptanceFilesPreserveBytes(t *testing.T) {
	original := []byte{0, 255, 13, 10, 42}
	data, err := json.Marshal(acceptanceFiles(map[string][]byte{"binary": original, "text": []byte("Original\r\n")}))
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Binary struct {
			Encoding string
			Bytes    []byte
		}
		Text string
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.Binary.Encoding != "base64" || !bytes.Equal(original, result.Binary.Bytes) || result.Text != "Original\r\n" {
		t.Fatalf("evidence changed: %s", data)
	}
}

// TestAcceptanceBoundary rejects trailing values and propagates canceled callers before launching a pilot.
func TestAcceptanceBoundary(t *testing.T) {
	result, err := acceptanceResponse(context.Background(), json.RawMessage(`{"scenario":"lifecycle"} {}`))
	if err != nil || result.OK || result.Error == nil || !strings.Contains(result.Error.Message, "one acceptance scenario") {
		t.Fatal(result, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = invokeContext(ctx, []byte(`{"operation":"nativeAcceptance","input":{"scenario":"lifecycle"},"location":"pilot"}`))
	if err != nil || result.OK || result.Error == nil || !strings.Contains(result.Error.Message, "canceled") {
		t.Fatal(result, err)
	}
}
