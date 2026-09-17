// Run complete native CLI pilots and expose their actual command evidence in the final migration walkthrough.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"time"
	"unicode/utf8"

	"github.com/fabricahq/code-rules/internal/acceptance"
)

// acceptanceResponse runs only predefined disposable workflows and returns assertions alongside original project files.
func acceptanceResponse(parent context.Context, input json.RawMessage) (response, error) {
	var fixture struct {
		Scenario string `json:"scenario"`
	}
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return adapterError("expected an acceptance scenario"), nil
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return adapterError("expected one acceptance scenario"), nil
	}
	if err := parent.Err(); err != nil {
		return response{Error: &failure{Name: "AcceptanceError", Message: err.Error()}}, nil
	}
	binary, err := fixtureCLIPath()
	if err != nil {
		return response{}, err
	}
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	report, err := acceptance.Run(ctx, binary, fixture.Scenario)
	observed := map[string]any{"steps": report.Steps, "verified": report.Verified, "files": acceptanceFiles(report.Files)}
	if err != nil {
		return response{Observation: observed, Error: &failure{Name: "AcceptanceError", Message: err.Error()}}, nil
	}
	return response{OK: true, Value: observed}, nil
}

// acceptanceFiles preserves text readability and encodes non-UTF-8 evidence without losing bytes.
func acceptanceFiles(files map[string][]byte) map[string]any {
	display := map[string]any{}
	for name, data := range files {
		if utf8.Valid(data) {
			display[name] = string(data)
		} else {
			display[name] = map[string]any{"encoding": "base64", "bytes": data}
		}
	}
	return display
}
