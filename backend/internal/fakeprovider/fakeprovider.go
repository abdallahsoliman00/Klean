// Package fakeprovider is a tiny OpenAI-compatible server (Models and the
// streaming Responses API) for tests and offline UI work. Base URL: {server}/v1.
//
// The reply says how many input messages it received, so tests can check
// context trimming. A message containing FAIL gets a 500, MIDFAIL fails
// partway through the stream, and SLOW streams slowly.
package fakeprovider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Fake struct {
	mu      sync.Mutex
	last    map[string]any
	lastHdr http.Header
}

func New() *Fake { return &Fake{} }

// Last returns the most recent /responses request body and headers.
func (f *Fake) Last() (map[string]any, http.Header) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.last, f.lastHdr
}

func (f *Fake) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", f.models)
	mux.HandleFunc("POST /v1/responses", f.responses)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (f *Fake) models(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") == "Bearer bad" {
		writeJSON(w, 401, map[string]any{"error": map[string]any{"message": "invalid key"}})
		return
	}
	writeJSON(w, 200, map[string]any{"object": "list", "data": []map[string]any{
		{"id": "fake-small", "object": "model", "owned_by": "test"},
		{"id": "fake-large", "object": "model", "owned_by": "test"},
	}})
}

func (f *Fake) responses(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"error": map[string]any{"message": err.Error()}})
		return
	}
	f.mu.Lock()
	f.last, f.lastHdr = body, r.Header.Clone()
	f.mu.Unlock()

	items, _ := body["input"].([]any)
	lastUser := ""
	for _, it := range items {
		m, _ := it.(map[string]any)
		if m["role"] == "user" {
			lastUser, _ = m["content"].(string)
		}
	}
	if strings.Contains(lastUser, "FAIL") && !strings.Contains(lastUser, "MIDFAIL") {
		writeJSON(w, 500, map[string]any{"error": map[string]any{"message": "model exploded"}})
		return
	}
	pause := 10 * time.Millisecond
	if strings.Contains(lastUser, "SLOW") {
		pause = 300 * time.Millisecond
	}
	text := fmt.Sprintf("You said: **%s**. I received %d messages.\n\n```python\nprint('hello')\n```\n\nDone.", lastUser, len(items))
	words := strings.SplitAfter(text, " ")

	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(200)
	rc := http.NewResponseController(w)
	seq := 0
	emit := func(v map[string]any) bool {
		v["sequence_number"] = seq
		seq++
		b, _ := json.Marshal(v)
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", v["type"], b); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	resp := map[string]any{"id": "resp_fake", "object": "response", "status": "in_progress", "model": body["model"], "output": []any{}}
	emit(map[string]any{"type": "response.created", "response": resp})
	for i, word := range words {
		if i == 3 && strings.Contains(lastUser, "MIDFAIL") {
			emit(map[string]any{"type": "error", "code": "server_error", "message": "lost the thread", "param": nil})
			return
		}
		if !emit(map[string]any{"type": "response.output_text.delta", "item_id": "msg_1", "output_index": 0, "content_index": 0, "delta": word}) {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(pause):
		}
	}
	resp["status"] = "completed"
	emit(map[string]any{"type": "response.completed", "response": resp})
}
