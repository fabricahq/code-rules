// Package logging configures process logging. Callers pass stderr and configuration
// from the command boundary; domain packages do not depend on logging.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// New returns a logger without changing the process-wide default. Empty settings
// select INFO and text. Level follows slog's syntax; format is text or json.
func New(output io.Writer, level, format string) (*slog.Logger, error) {
	var threshold slog.Level
	if level != "" {
		if err := threshold.UnmarshalText([]byte(level)); err != nil {
			return nil, fmt.Errorf("CODE_RULES_LOG_LEVEL: expected a slog level such as debug, info, warn, or error")
		}
	}
	options := &slog.HandlerOptions{Level: threshold}
	var handler slog.Handler
	switch strings.ToLower(format) {
	case "", "text":
		handler = slog.NewTextHandler(output, options)
	case "json":
		handler = slog.NewJSONHandler(output, options)
	default:
		return nil, fmt.Errorf("CODE_RULES_LOG_FORMAT: expected text or json")
	}
	return slog.New(handler), nil
}
