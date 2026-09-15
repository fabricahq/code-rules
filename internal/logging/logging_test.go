// Verify log configuration, level filtering, and structured output through writers.

package logging_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/logging"
)

func TestDefaultTextLogger(t *testing.T) {
	var output bytes.Buffer
	logger, err := logging.New(&output, "", "")
	if err != nil {
		t.Fatal(err)
	}
	logger.Debug("hidden")
	logger.Info("listening", "url", "http://127.0.0.1:4391")
	if strings.Contains(output.String(), "hidden") || !strings.Contains(output.String(), "level=INFO") || !strings.Contains(output.String(), "url=http://127.0.0.1:4391") {
		t.Fatalf("unexpected default output: %s", &output)
	}
}

func TestJSONLoggerLevels(t *testing.T) {
	for _, level := range []string{"debug", "INFO", "warn", "error"} {
		t.Run(level, func(t *testing.T) {
			var output bytes.Buffer
			logger, err := logging.New(&output, level, "json")
			if err != nil {
				t.Fatal(err)
			}
			logger.Debug("debug")
			logger.Info("info")
			logger.Warn("warn")
			logger.Error("error", "attempt", 3)
			var messages []string
			decoder := json.NewDecoder(&output)
			for decoder.More() {
				var record struct {
					Message string `json:"msg"`
					Attempt int    `json:"attempt"`
				}
				if err := decoder.Decode(&record); err != nil {
					t.Fatal(err)
				}
				messages = append(messages, record.Message)
				if record.Message == "error" && record.Attempt != 3 {
					t.Fatal("structured attribute lost its numeric value")
				}
			}
			want := map[string]string{"debug": "debug,info,warn,error", "INFO": "info,warn,error", "warn": "warn,error", "error": "error"}[level]
			if strings.Join(messages, ",") != want {
				t.Fatalf("got %v, want %s", messages, want)
			}
		})
	}
}

func TestInvalidLoggingConfiguration(t *testing.T) {
	for _, test := range []struct{ level, format, field string }{
		{"private-value", "", "CODE_RULES_LOG_LEVEL"},
		{"", "private-value", "CODE_RULES_LOG_FORMAT"},
	} {
		var output bytes.Buffer
		logger, err := logging.New(&output, test.level, test.format)
		if logger != nil || err == nil || !strings.Contains(err.Error(), test.field) || strings.Contains(err.Error(), "private-value") || output.Len() != 0 {
			t.Fatalf("expected a contextual error without logging or echoing the value: logger=%v, error=%v, output=%s", logger, err, &output)
		}
	}
}
