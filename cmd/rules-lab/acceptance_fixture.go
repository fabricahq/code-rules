// Run complete native CLI pilots and expose their actual command evidence in the final migration walkthrough.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/fabricahq/code-rules/internal/acceptance"
	"github.com/fabricahq/code-rules/internal/project"
)

// acceptanceResponse runs only predefined disposable workflows and returns assertions alongside original project files.
func acceptanceResponse(input json.RawMessage) (response, error) {
	var fixture struct {
		Scenario string `json:"scenario"`
	}
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return adapterError("expected an acceptance scenario"), nil
	}
	binary, err := fixtureCLIPath()
	if err != nil {
		return response{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	report, err := acceptance.Run(ctx, binary, fixture.Scenario)
	observed := map[string]any{"steps": report.Steps, "verified": report.Verified, "files": displayProjectTree(&project.Tree{Files: report.Files})}
	if err != nil {
		return response{Observation: observed, Error: &failure{Name: "AcceptanceError", Message: err.Error()}}, nil
	}
	return response{OK: true, Value: observed}, nil
}
