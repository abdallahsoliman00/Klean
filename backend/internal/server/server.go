// Package server is the HTTP API. Everything lives under /api; the frontend
// folder is optionally served at /.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"unicode/utf8"

	"klean/backend/internal/config"
	"klean/backend/internal/pdf"
	"klean/backend/internal/store"
	"klean/backend/internal/tools"
)

type Server struct {
	db  *store.DB
	web *tools.Web // web_search and fetch_page
	// browser prints reports to PDF; nil when this machine has none.
	browser *pdf.Browser
	handler http.Handler

	// The replies being written, by chat. See runs.go.
	ctx    context.Context // cancelled by Close
	cancel context.CancelFunc
	mu     sync.Mutex
	runs   map[string]*run
	closed bool
	active sync.WaitGroup // one for every run that hasn't ended

	upd updateCache // see update.go
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.handler.ServeHTTP(w, r) }

// Option adjusts a Server; tests use it to keep the web tools off the internet.
type Option func(*Server)

// WithWeb replaces the web tools, for example with ones pointed at a stub.
func WithWeb(w *tools.Web) Option { return func(s *Server) { s.web = w } }

// WithBrowser sets the browser reports are printed to PDF with.
func WithBrowser(b *pdf.Browser) Option { return func(s *Server) { s.browser = b } }

// New returns the server, which is the HTTP handler. Close it when done. static holds the frontend (index.html at its
// root), or is nil to serve only the API. corsOrigins is a comma-separated
// list, or "*".
func New(db *store.DB, static fs.FS, corsOrigins string, opts ...Option) *Server {
	s := &Server{db: db, web: tools.NewWeb(), runs: map[string]*run{}}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	for _, o := range opts {
		o(s)
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		// "app" lets a second copy recognise one that is already running.
		writeJSON(w, 200, map[string]any{"ok": true, "app": config.AppID, "version": config.Version})
	})

	mux.HandleFunc("GET /api/providers", s.listProviders)
	mux.HandleFunc("POST /api/providers", s.createProvider)
	mux.HandleFunc("PATCH /api/providers/{id}", s.updateProvider)
	mux.HandleFunc("DELETE /api/providers/{id}", s.deleteProvider)
	mux.HandleFunc("GET /api/providers/{id}/models", s.providerModels)

	mux.HandleFunc("GET /api/chats", s.listChats)
	mux.HandleFunc("POST /api/chats", s.createChat)
	mux.HandleFunc("GET /api/chats/{id}", s.getChat)
	mux.HandleFunc("PATCH /api/chats/{id}", s.updateChat)
	mux.HandleFunc("DELETE /api/chats/{id}", s.deleteChat)
	mux.HandleFunc("POST /api/chats/{id}/restore", s.restoreChat)
	mux.HandleFunc("POST /api/chats/{id}/messages", s.sendMessage)
	mux.HandleFunc("GET /api/chats/{id}/stream", s.streamReply)
	mux.HandleFunc("POST /api/chats/{id}/stop", s.stopReply)

	mux.HandleFunc("GET /api/images/{id}", s.getImage)
	mux.HandleFunc("GET /api/artifacts", s.listArtifacts)
	mux.HandleFunc("GET /api/artifacts/{id}/versions", s.artifactVersions)
	mux.HandleFunc("DELETE /api/artifacts/{id}", s.deleteArtifact)
	mux.HandleFunc("POST /api/artifacts/{id}/pdf", s.artifactPDF)

	mux.HandleFunc("GET /api/presets", s.listPresets)
	mux.HandleFunc("POST /api/presets", s.createPreset)
	mux.HandleFunc("PATCH /api/presets/{id}", s.updatePreset)
	mux.HandleFunc("DELETE /api/presets/{id}", s.deletePreset)

	mux.HandleFunc("GET /api/skills", s.listSkills)
	mux.HandleFunc("POST /api/skills", s.createSkill)
	mux.HandleFunc("PATCH /api/skills/{id}", s.updateSkill)
	mux.HandleFunc("DELETE /api/skills/{id}", s.deleteSkill)
	mux.HandleFunc("GET /api/settings", s.getSettings)
	mux.HandleFunc("PATCH /api/settings", s.updateSettings)
	mux.HandleFunc("GET /api/update", s.checkUpdate)
	mux.HandleFunc("POST /api/update", s.applyUpdate)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		fail(w, http.StatusNotFound, "Not found.")
	})
	if static != nil {
		mux.Handle("/", http.FileServer(http.FS(static)))
	}
	s.handler = cors(mux, corsOrigins)
	return s
}

// ---------------- helpers ----------------

type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

func errorf(status int, format string, a ...any) error {
	return &httpError{status, fmt.Sprintf(format, a...)}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// fail sends {"detail": msg}, the shape the frontend reads.
func fail(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"detail": msg})
}

// reply sends v, or the error: an *httpError as is, anything else as a 500.
func reply(w http.ResponseWriter, status int, v any, err error) {
	var he *httpError
	switch {
	case errors.As(err, &he):
		fail(w, he.status, he.msg)
	case err != nil:
		log.Printf("internal error: %v", err)
		fail(w, http.StatusInternalServerError, "Something went wrong on the server.")
	case status == http.StatusNoContent:
		w.WriteHeader(status)
	default:
		writeJSON(w, status, v)
	}
}

func decode(r *http.Request, dst any) error {
	return decodeLimit(r, dst, config.MaxRequestBody)
}

// decodeLimit is decode for an endpoint with its own limit on the body's size.
func decodeLimit(r *http.Request, dst any, maxBytes int64) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxBytes))
	if err := dec.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return errorf(http.StatusUnprocessableEntity, "The request is too large: it may be at most %d MB.", maxBytes>>20)
		}
		return errorf(http.StatusUnprocessableEntity, "The request body isn't valid JSON for this endpoint: %v", err)
	}
	return nil
}

func cors(next http.Handler, origins string) http.Handler {
	allowed := map[string]bool{}
	for _, o := range strings.Split(origins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			allowed[o] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (allowed["*"] || allowed[origin]) {
			if allowed["*"] {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func cleanURL(u string) (string, error) {
	u = strings.TrimRight(strings.TrimSpace(u), "/")
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return "", errorf(http.StatusUnprocessableEntity, "The base URL needs to start with http:// or https://")
	}
	return u, nil
}

func checkLen(field, v string, min, max int) error {
	if n := utf8.RuneCountInString(v); n < min || n > max {
		return errorf(http.StatusUnprocessableEntity, "%s must be between %d and %d characters.", field, min, max)
	}
	return nil
}

func checkTemperature(t float64) error {
	if t < config.MinTemperature || t > config.MaxTemperature {
		return errorf(http.StatusUnprocessableEntity, "temperature must be between %g and %g.",
			config.MinTemperature, config.MaxTemperature)
	}
	return nil
}

func checkContextWindow(n int) error {
	if n < config.MinContextWindow || n > config.MaxContextWindow {
		return errorf(http.StatusUnprocessableEntity, "contextWindow must be between %d and %d.",
			config.MinContextWindow, config.MaxContextWindow)
	}
	return nil
}
