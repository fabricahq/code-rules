// identity-lab is a development adapter, not the Code Rules CLI. By default it
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
	"net"
	"net/http"
	"os"
	"time"

	"github.com/fabricahq/code-rules/internal/rules"
)

//go:embed index.html
var page []byte

const maxRequestBytes = 1 << 20

type request struct {
	Operation string          `json:"operation"`
	Input     json.RawMessage `json:"input"`
	Location  string          `json:"location"`
}

type failure struct {
	Name     string `json:"name"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}

type response struct {
	OK    bool     `json:"ok"`
	Value any      `json:"value,omitempty"`
	Error *failure `json:"error,omitempty"`
}

func invoke(data []byte) response {
	var req request
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return adapterError("invalid request JSON: " + err.Error())
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return adapterError("expected one request object")
	}
	if req.Location == "" {
		return adapterError("location must be nonempty")
	}
	var value any
	var err error
	switch req.Operation {
	case "groupID", "ruleGroup":
		var text string
		if len(req.Input) == 0 || bytes.Equal(bytes.TrimSpace(req.Input), []byte("null")) || json.Unmarshal(req.Input, &text) != nil {
			return adapterError("input must be a string for " + req.Operation)
		}
		if req.Operation == "groupID" {
			err = rules.ValidateGroupID(text, req.Location)
			value = text
		} else {
			value, err = rules.GroupFromPath(text, req.Location)
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
		return adapterError("unknown operation " + req.Operation)
	}
	if err != nil {
		var validation *rules.ValidationError
		if errors.As(err, &validation) {
			return response{Error: &failure{Name: "ValidationError", Message: err.Error(), Location: validation.Location}}
		}
		return adapterError(err.Error())
	}
	return response{OK: true, Value: value}
}

func adapterError(message string) response {
	return response{Error: &failure{Name: "AdapterError", Message: message}}
}

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(page)
	})
	mux.HandleFunc("POST /invoke", func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBytes))
		if err != nil {
			http.Error(w, "request body exceeds limit or could not be read", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(invoke(data))
	})
	return http.NewCrossOriginProtection().Handler(mux)
}

func run() error {
	serve := flag.Bool("serve", false, "serve the interactive lab on loopback")
	port := flag.Int("port", 0, "loopback port (0 chooses an available port)")
	flag.Parse()
	if *serve {
		listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", *port))
		if err != nil {
			return fmt.Errorf("start identity lab: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Identity lab: http://%s\n", listener.Addr())
		server := &http.Server{Handler: handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
		return server.Serve(listener)
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), maxRequestBytes)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		if err := encoder.Encode(invoke(scanner.Bytes())); err != nil {
			return fmt.Errorf("write response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read request: %w", err)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
