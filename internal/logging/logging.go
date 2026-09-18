// Package logging configures process logging. Callers pass stderr and configuration
// from the command boundary; domain packages do not depend on logging.
package logging

import (
	"errors"
	"io"
	"log/slog"
)

// Format selects the encoding of each log record.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// New returns a logger without changing the process-wide default. Only FormatText
// and FormatJSON are valid; callers parse external settings before calling New.
func New(output io.Writer, level slog.Level, format Format) (*slog.Logger, error) {
	options := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch format {
	case FormatText:
		handler = slog.NewTextHandler(output, options)
	case FormatJSON:
		handler = slog.NewJSONHandler(output, options)
	default:
		return nil, errors.New("invalid log format: expected text or json")
	}
	return slog.New(handler), nil
}
