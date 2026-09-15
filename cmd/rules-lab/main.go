// rules-lab is a development adapter, not the Code Rules CLI. By default it
// reads one JSON request per line from stdin; -serve opens a loopback browser lab.

package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fabricahq/code-rules/internal/logging"
	"github.com/fabricahq/code-rules/internal/rules"
)

//go:embed index.html
var page []byte

const maxRequestBytes = 1 << 20

type request struct {
	Operation string          `json:"operation"`
	Input     json.RawMessage `json:"input"`
	// Location is an input field path for diagnostics, not a file to read.
	Location string `json:"location"`
}

type failure struct {
	Name     string `json:"name"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}

type response struct {
	OK bool `json:"ok"`
	// Value is nil only on failure. A successful empty selection holds a non-nil
	// []string in this interface, so JSON includes "value": [] rather than null.
	Value any      `json:"value,omitempty"`
	Error *failure `json:"error,omitempty"`
}

// invoke returns expected input failures as responses and unexpected failures as errors.
func invoke(data []byte) (response, error) {
	var req request
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return adapterError("invalid request JSON: " + err.Error()), nil
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return adapterError("expected one request object"), nil
	}
	if req.Location == "" {
		return adapterError("location must be nonempty"), nil
	}
	var value any
	var err error
	switch req.Operation {
	case "groupID", "ruleGroup", "groupMetadata":
		var text string
		if len(req.Input) == 0 || bytes.Equal(bytes.TrimSpace(req.Input), []byte("null")) || json.Unmarshal(req.Input, &text) != nil {
			return adapterError("input must be a string for " + req.Operation), nil
		}
		switch req.Operation {
		case "groupID":
			err = rules.ValidateGroupID(text, req.Location)
			value = text
		case "ruleGroup":
			value, err = rules.GroupFromPath(text, req.Location)
		case "groupMetadata":
			value, err = rules.ParseGroupMetadata(json.RawMessage(text), req.Location)
		}
	case "selection":
		var selection rules.GroupSelection
		selection, err = rules.ParseGroupSelection(req.Input, req.Location)
		if selection.Pattern != "" {
			value = selection.Pattern
		} else {
			value = selection.Groups
		}
	default:
		return adapterError("unknown operation " + req.Operation), nil
	}
	if err != nil {
		var validation *rules.ValidationError
		if errors.As(err, &validation) {
			return response{Error: &failure{Name: "ValidationError", Message: err.Error(), Location: validation.Location}}, nil
		}
		return response{}, fmt.Errorf("invoke %s: %v", req.Operation, err)
	}
	return response{OK: true, Value: value}, nil
}

func adapterError(message string) response {
	return response{Error: &failure{Name: "AdapterError", Message: message}}
}

func handler(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if _, err := w.Write(page); err != nil {
			logger.WarnContext(r.Context(), "write lab page failed", "error", err)
		}
	})
	mux.HandleFunc("POST /invoke", func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBytes))
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				http.Error(w, "request body exceeds 1 MiB limit", http.StatusRequestEntityTooLarge)
			} else {
				http.Error(w, "could not read request body", http.StatusBadRequest)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		result, err := invoke(data)
		if err != nil {
			logger.ErrorContext(r.Context(), "invoke failed", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			// The failure is already reported here; a disconnected client cannot receive it.
			_ = json.NewEncoder(w).Encode(response{Error: &failure{Name: "InternalError", Message: "invocation failed; see server logs"}})
			return
		}
		if err := json.NewEncoder(w).Encode(result); err != nil {
			logger.WarnContext(r.Context(), "write invocation response failed", "error", err)
			return
		}
		// Expected validation failures belong in the response. Never log the input
		// or diagnostic message: either may contain user-authored content.
		logger.DebugContext(r.Context(), "invocation completed", "ok", result.OK)
	})
	return http.NewCrossOriginProtection().Handler(mux)
}

func run(logger *slog.Logger) error {
	serve := flag.Bool("serve", false, "serve the interactive lab on loopback")
	port := flag.Int("port", 0, "loopback port (0 chooses an available port)")
	flag.Parse()
	if *serve {
		listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", *port))
		if err != nil {
			return fmt.Errorf("start rules lab: %v", err)
		}
		logger.Info("rules lab listening", "url", "http://"+listener.Addr().String())
		server := &http.Server{
			Handler:           handler(logger),
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       30 * time.Second,
		}
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve rules lab at %s: %v", listener.Addr(), err)
		}
		return nil
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), maxRequestBytes)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		result, err := invoke(scanner.Bytes())
		if err != nil {
			return err
		}
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("write response: %v", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read request: %v", err)
	}
	return nil
}

// loggerFromEnvironment parses command settings without exposing their values in errors.
func loggerFromEnvironment(output io.Writer) (*slog.Logger, error) {
	var level slog.Level
	if text := os.Getenv("CODE_RULES_LOG_LEVEL"); text != "" {
		if err := level.UnmarshalText([]byte(text)); err != nil {
			return nil, errors.New("CODE_RULES_LOG_LEVEL: expected a slog level such as debug, info, warn, or error")
		}
	}
	format := logging.Format(strings.ToLower(os.Getenv("CODE_RULES_LOG_FORMAT")))
	if format == "" {
		format = logging.FormatText
	}
	logger, err := logging.New(output, level, format)
	if err != nil {
		return nil, fmt.Errorf("CODE_RULES_LOG_FORMAT: %v", err)
	}
	return logger, nil
}

func main() {
	logger, err := loggerFromEnvironment(os.Stderr)
	if err != nil {
		// Default settings are always valid, even when the environment is not.
		logger, _ = logging.New(os.Stderr, slog.LevelInfo, logging.FormatText)
		logger.Error("invalid logging configuration", "error", err)
		os.Exit(1)
	}
	if err := run(logger); err != nil {
		logger.Error("rules lab failed", "error", err)
		os.Exit(1)
	}
}
