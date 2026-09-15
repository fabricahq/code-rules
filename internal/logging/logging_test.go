// Verify log configuration, level filtering, and structured output through writers.

package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/fabricahq/code-rules/internal/logging"
)

func TestDefaultTextLogger(t *testing.T) {
	var output bytes.Buffer
	logger, err := logging.New(&output, slog.LevelInfo, logging.FormatText)
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
	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		t.Run(level.String(), func(t *testing.T) {
			var output bytes.Buffer
			logger, err := logging.New(&output, level, logging.FormatJSON)
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
			want := map[slog.Level]string{slog.LevelDebug: "debug,info,warn,error", slog.LevelInfo: "info,warn,error", slog.LevelWarn: "warn,error", slog.LevelError: "error"}[level]
			if strings.Join(messages, ",") != want {
				t.Fatalf("got %v, want %s", messages, want)
			}
		})
	}
}

func TestInvalidFormat(t *testing.T) {
	for _, format := range []logging.Format{"", "private-value", "JSON"} {
		var output bytes.Buffer
		logger, err := logging.New(&output, slog.LevelInfo, format)
		if logger != nil || err == nil || err.Error() != "invalid log format: expected text or json" || output.Len() != 0 {
			t.Fatalf("expected a format error without logging or echoing the value: logger=%v, error=%v, output=%s", logger, err, &output)
		}
	}
}
