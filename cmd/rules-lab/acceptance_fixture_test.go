// Verify that JSON transport retains exact binary evidence alongside readable text.

package main

import (
	"bytes"
	"encoding/json"
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
