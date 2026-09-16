// rules-lab is a development adapter, not the Code Rules CLI. By default it
// reads one JSON request per line from stdin; -serve opens a loopback browser lab.

package main

import (
	"bufio"
	"bytes"
	"embed"
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

	"github.com/fabricahq/code-rules/internal/build"
	"github.com/fabricahq/code-rules/internal/logging"
	"github.com/fabricahq/code-rules/internal/rules"
)

//go:embed index.html
var page []byte

//go:embed walkthroughs/*.html
var walkthroughs embed.FS

//go:embed lab-assets/*
var labAssets embed.FS

// Allow an 8 MiB tag listing, up to sixfold JSON escaping, and the request envelope.
const maxRequestBytes = 64 << 20

type request struct {
	Operation string          `json:"operation"`
	Input     json.RawMessage `json:"input"`
	// Location is an input field path for diagnostics, not a file to read.
	Location string `json:"location"`
}

type failure struct {
	Code     string `json:"code,omitempty"`
	Name     string `json:"name"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}

type response struct {
	OK bool `json:"ok"`
	// Value holds successful results. A typed nil *string represents an unknown
	// repository web link as JSON null; a non-nil []string preserves empty selections.
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
	if req.Location == "" && req.Operation != "rule" {
		return adapterError("location must be nonempty"), nil
	}
	var value any
	var err error
	switch req.Operation {
	case "indexPages":
		var input struct {
			File     string   `json:"file"`
			Header   string   `json:"header"`
			Entries  []string `json:"entries"`
			Footer   string   `json:"footer"`
			MaxBytes int      `json:"maxBytes"`
		}
		fields := json.NewDecoder(bytes.NewReader(req.Input))
		fields.DisallowUnknownFields()
		if fields.Decode(&input) != nil {
			return adapterError("expected index page fields"), nil
		}
		value, err = build.IndexPages(input.File, input.Header, input.Entries, input.Footer, input.MaxBytes)
	case "renderIndexes":
		var input struct {
			Fixture  json.RawMessage `json:"fixture"`
			MaxBytes int             `json:"maxBytes"`
		}
		fields := json.NewDecoder(bytes.NewReader(req.Input))
		fields.DisallowUnknownFields()
		if fields.Decode(&input) != nil {
			return adapterError("expected fixture and maxBytes"), nil
		}
		var resolved build.Resolved
		resolved, err = resolveBuildFixture(input.Fixture)
		if err == nil {
			value, err = build.RenderIndexes(resolved, input.MaxBytes)
		}
	case "renderRules":
		var resolved build.Resolved
		resolved, err = resolveBuildFixture(req.Input)
		if err == nil {
			value, err = build.RenderRules(resolved)
		}
	case "resolveRules":
		value, err = resolveBuildFixture(req.Input)
	case "markdownTargets":
		var input struct {
			Text *string `json:"text"`
			File *string `json:"file"`
		}
		fields := json.NewDecoder(bytes.NewReader(req.Input))
		fields.DisallowUnknownFields()
		if fields.Decode(&input) != nil || input.Text == nil || input.File == nil || *input.File == "" {
			return adapterError("expected text and file"), nil
		}
		value, err = rules.MarkdownTargets(*input.Text, *input.File)
	case "loadLibrary":
		if err := validateFixtureText(req.Input); err != nil {
			return adapterError(err.Error()), nil
		}
		var input libraryFixture
		fields := json.NewDecoder(bytes.NewReader(req.Input))
		fields.DisallowUnknownFields()
		if fields.Decode(&input) != nil || input.Files == nil || input.Source == "" {
			return adapterError("loadLibrary input must contain files, groups, and source"), nil
		}
		value, err = loadLibraryFixture(input)
	case "selectReleaseTag":
		var input struct {
			AvailableGitTags *string `json:"availableGitTags"`
			Constraint       *string `json:"constraint"`
		}
		fields := json.NewDecoder(bytes.NewReader(req.Input))
		fields.DisallowUnknownFields()
		if fields.Decode(&input) != nil || input.AvailableGitTags == nil || input.Constraint == nil {
			return adapterError("selectReleaseTag input must contain availableGitTags and constraint strings"), nil
		}
		constraint, parseErr := rules.ParseVersionConstraint(*input.Constraint, req.Location+".constraint")
		if parseErr != nil {
			err = parseErr
		} else {
			value, err = rules.SelectReleaseTag(*input.AvailableGitTags, constraint)
		}
	case "licenses":
		var input struct {
			Manifest *string   `json:"manifest"`
			Paths    *[]string `json:"paths"`
		}
		fields := json.NewDecoder(bytes.NewReader(req.Input))
		fields.DisallowUnknownFields()
		if fields.Decode(&input) != nil || input.Manifest == nil || input.Paths == nil {
			return adapterError("licenses input must contain manifest text and a paths array"), nil
		}
		files := make(map[string][]byte)
		for _, path := range *input.Paths {
			files[path] = nil
		}
		files["rule-library.json"] = []byte(*input.Manifest)
		value, err = rules.ReadLibraryLicenses(files, req.Location)
	case "versionMatch":
		var input struct {
			Constraint *string `json:"constraint"`
			Version    *string `json:"version"`
		}
		fields := json.NewDecoder(bytes.NewReader(req.Input))
		fields.DisallowUnknownFields()
		if fields.Decode(&input) != nil || input.Constraint == nil || input.Version == nil {
			return adapterError("versionMatch input must contain constraint and version strings"), nil
		}
		constraint, parseErr := rules.ParseVersionConstraint(*input.Constraint, req.Location+".constraint")
		if parseErr != nil {
			err = parseErr
		} else {
			value, err = constraint.Matches(*input.Version, req.Location+".version")
		}
	case "repository":
		value, err = rules.ParseRepository(req.Input, req.Location)
	case "repositoryFile":
		var input struct {
			Repository json.RawMessage `json:"repository"`
			Commit     *string         `json:"commit"`
			Path       *string         `json:"path"`
			Image      *bool           `json:"image"`
		}
		fields := json.NewDecoder(bytes.NewReader(req.Input))
		fields.DisallowUnknownFields()
		if fields.Decode(&input) != nil || input.Repository == nil || input.Commit == nil || input.Path == nil || input.Image == nil {
			return adapterError("repositoryFile input must contain repository, commit, path, and image"), nil
		}
		var repository rules.Repository
		// Mirror repositoryFileUrl: its diagnostic location is always "repository".
		// The standalone repository operation accepts caller-defined locations.
		repository, err = rules.ParseRepository(input.Repository, "repository")
		var link *string
		if err == nil {
			if text, known := repository.FileURL(*input.Commit, *input.Path, *input.Image); known {
				link = &text
			}
		}
		value = link
	case "rule":
		var input struct {
			Text   *string `json:"text"`
			Path   *string `json:"path"`
			Source *string `json:"source"`
		}
		fields := json.NewDecoder(bytes.NewReader(req.Input))
		fields.DisallowUnknownFields()
		if fields.Decode(&input) != nil || input.Text == nil || input.Path == nil || input.Source == nil {
			return adapterError("rule input must contain text, path, and source strings"), nil
		}
		value, err = rules.Parse(*input.Text, *input.Path, *input.Source)
	case "groupID", "ruleGroup", "groupMetadata", "document", "gitRef", "tagVersion", "versionConstraint", "configuration":
		var text string
		if len(req.Input) == 0 || bytes.Equal(bytes.TrimSpace(req.Input), []byte("null")) || json.Unmarshal(req.Input, &text) != nil {
			return adapterError("input must be a string for " + req.Operation), nil
		}
		switch req.Operation {
		case "configuration":
			value, err = rules.ParseConfiguration(json.RawMessage(text))
		case "versionConstraint":
			var constraint rules.VersionConstraint
			constraint, err = rules.ParseVersionConstraint(text, req.Location)
			value = constraint.String()
		case "gitRef":
			value, err = rules.ParseGitRef(text, req.Location)
		case "tagVersion":
			value, err = rules.TagVersion(text, req.Location)
		case "groupID":
			err = rules.ValidateGroupID(text, req.Location)
			value = text
		case "ruleGroup":
			value, err = rules.GroupFromPath(text, req.Location)
		case "groupMetadata":
			value, err = rules.ParseGroupMetadata(json.RawMessage(text), req.Location)
		case "document":
			value, err = rules.SplitDocument(text, req.Location)
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
		var selection *rules.VersionSelectionError
		if errors.As(err, &selection) {
			return response{Error: &failure{Name: "VersionSelectionError", Code: string(selection.Kind), Message: err.Error()}}, nil
		}
		var validation *rules.ValidationError
		if errors.As(err, &validation) {
			return response{Error: &failure{Name: "ValidationError", Message: err.Error(), Location: validation.Location}}, nil
		}
		return response{}, fmt.Errorf("invoke %s: %v", req.Operation, err)
	}
	return response{OK: true, Value: value}, nil
}

// adapterError builds a response for a request rejected before calling a domain function.
func adapterError(message string) response {
	return response{Error: &failure{Name: "AdapterError", Message: message}}
}

// handler serves the walkthrough and bounded, same-origin invocations without logging user content.
func handler(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	// Serve the embedded walkthrough and report an undeliverable page once.
	mux.Handle("GET /lab-assets/", http.FileServerFS(labAssets))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if _, err := w.Write(page); err != nil {
			logger.WarnContext(r.Context(), "write lab page failed", "error", err)
		}
	})
	// Serve immutable review pages so later slices do not replace earlier walkthroughs.
	mux.HandleFunc("GET /walkthrough/{name}", func(w http.ResponseWriter, r *http.Request) {
		data, err := walkthroughs.ReadFile("walkthroughs/" + r.PathValue("name") + ".html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if _, err := w.Write(data); err != nil {
			logger.WarnContext(r.Context(), "write walkthrough failed", "error", err)
		}
	})
	// Decode a bounded invocation and return its result without logging request contents.
	mux.HandleFunc("POST /invoke", func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBytes))
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				http.Error(w, "request body exceeds 64 MiB limit", http.StatusRequestEntityTooLarge)
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

// run runs the loopback HTTP server or processes JSON requests from standard input.
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
	return runRequests(os.Stdin, os.Stdout)
}

// runRequests processes bounded newline-delimited requests and writes native results.
func runRequests(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), maxRequestBytes)
	encoder := json.NewEncoder(output)
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

// main configures logging and reports startup or execution failures once before exiting.
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
