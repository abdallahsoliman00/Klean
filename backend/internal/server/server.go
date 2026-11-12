// Package server is the HTTP API. Everything lives under /api; the frontend
// folder is optionally served at /.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"klean/backend/internal/store"
	"klean/backend/internal/tools"
)

type Server struct {
	db  *store.DB
	web *tools.Web // web_search and fetch_page
}

// Option adjusts a Server; tests use it to keep the web tools off the internet.
type Option func(*Server)

// WithWeb replaces the web tools, for example with ones pointed at a stub.
func WithWeb(w *tools.Web) Option { return func(s *Server) { s.web = w } }

// New returns the HTTP handler. static holds the frontend (index.html at its
// root), or is nil to serve only the API. corsOrigins is a comma-separated
// list, or "*".
func New(db *store.DB, static fs.FS, corsOrigins string, opts ...Option) http.Handler {
	s := &Server{db: db, web: tools.NewWeb()}
	for _, o := range opts {
		o(s)
	}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		// "app" lets a second copy recognise one that is already running.
		writeJSON(w, 200, map[string]any{"ok": true, "app": "klean"})
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

	mux.HandleFunc("DELETE /api/artifacts/{id}", s.deleteArtifact)
	mux.HandleFunc("GET /api/settings", s.getSettings)
	mux.HandleFunc("PATCH /api/settings", s.updateSettings)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		fail(w, http.StatusNotFound, "Not found.")
	})
	if static != nil {
		mux.Handle("/", http.FileServer(http.FS(static)))
	}
	return cors(mux, corsOrigins)
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
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 10<<20))
	if err := dec.Decode(dst); err != nil {
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
	if t < 0 || t > 2 {
		return errorf(http.StatusUnprocessableEntity, "temperature must be between 0 and 2.")
	}
	return nil
}

func checkContextWindow(n int) error {
	if n < 512 || n > 2_000_000 {
		return errorf(http.StatusUnprocessableEntity, "contextWindow must be between 512 and 2000000.")
	}
	return nil
}
