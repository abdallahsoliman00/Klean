// Package fakeprovider is a tiny OpenAI-compatible server (Models and the
// streaming Responses API) for tests and offline UI work. Base URL: {server}/v1.
//
// The reply says how many input messages it received, so tests can check
// context trimming. A message containing FAIL gets a 500, MIDFAIL fails
// partway through the stream, and SLOW streams slowly.
//
// When the request offers tools, a trigger word in the last user message
// makes it call one: /artifact, /code, /svg, /update, /ask, /search, /fetch
// (see triggers below). Once the call's output comes back it answers in text.
// /loop calls a tool on every request and never finishes, NOTOOLS rejects any
// request that carries tools with a 400.
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
	mu       sync.Mutex
	last     map[string]any
	lastHdr  http.Header
	requests []map[string]any
	calls    int // numbers the call ids
}

func New() *Fake { return &Fake{} }

// Last returns the most recent /responses request body and headers.
func (f *Fake) Last() (map[string]any, http.Header) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.last, f.lastHdr
}

// Requests returns every /responses request body received so far.
func (f *Fake) Requests() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]map[string]any(nil), f.requests...)
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

// call is a function call the fake decided to make.
type call struct {
	name string
	args map[string]any
}

// input is what the fake needs to know about a request's input items.
type input struct {
	count      int
	lastUser   string
	toolOutput string // output of the last call made since the last user message
	answered   bool   // such an output exists
	artifactID string // id in the most recent artifact tool output
}

func readInput(items []any) input {
	in := input{count: len(items)}
	for _, it := range items {
		m, _ := it.(map[string]any)
		switch {
		case m["role"] == "user":
			in.lastUser, _ = m["content"].(string)
			in.answered, in.toolOutput = false, ""
		case m["type"] == "function_call_output":
			in.toolOutput, _ = m["output"].(string)
			in.answered = true
			// Artifact tools answer with JSON carrying the artifact's id.
			var out struct{ ID string }
			if json.Unmarshal([]byte(in.toolOutput), &out) == nil && out.ID != "" {
				in.artifactID = out.ID
			}
		}
	}
	return in
}

func offeredTools(body map[string]any) map[string]bool {
	offered := map[string]bool{}
	list, _ := body["tools"].([]any)
	for _, t := range list {
		if m, _ := t.(map[string]any); m != nil {
			if name, _ := m["name"].(string); name != "" {
				offered[name] = true
			}
		}
	}
	return offered
}

// after returns the text following a trigger word, or fallback when there is none.
func after(text, trigger, fallback string) string {
	rest := strings.TrimSpace(text[strings.Index(text, trigger)+len(trigger):])
	if rest == "" {
		return fallback
	}
	return rest
}

const demoPage = `<!doctype html>
<html><head><meta charset="utf-8"><title>Demo</title>
<style>body{font-family:sans-serif;display:grid;place-items:center;height:100vh;margin:0}</style></head>
<body><button onclick="this.textContent='Clicked '+(++n)+' times'">Click me</button><script>let n=0</script></body></html>`

const demoSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120">
  <circle cx="60" cy="60" r="50" fill="#4f46e5"/>
  <path d="M40 62l14 14 28-30" fill="none" stroke="#fff" stroke-width="8" stroke-linecap="round"/>
</svg>`

// triggers are checked in order; the first whose word is in the last user
// message and whose tool was offered decides the call.
var triggers = []struct {
	word, tool string
	args       func(in input) map[string]any
}{
	{"/artifact", "create_artifact", func(input) map[string]any {
		return map[string]any{"title": "Demo page", "kind": "html", "content": demoPage}
	}},
	{"/code", "create_artifact", func(input) map[string]any {
		return map[string]any{"title": "hello.py", "kind": "code", "language": "python",
			"content": "def greet(name):\n    return f\"Hello, {name}!\"\n\nprint(greet(\"world\"))\n"}
	}},
	{"/svg", "create_artifact", func(input) map[string]any {
		return map[string]any{"title": "Check mark", "kind": "svg", "content": demoSVG}
	}},
	{"/update", "update_artifact", func(in input) map[string]any {
		id := in.artifactID
		if id == "" {
			id = "missing" // nothing to update: exercises the tool's error path
		}
		return map[string]any{"id": id, "title": "Demo page (updated)",
			"content": strings.Replace(demoPage, "Click me", "Click me again", 1)}
	}},
	{"/ask", "ask_user", func(input) map[string]any {
		return map[string]any{"question": "Which colour should the button be?", "multiple": false, "options": []map[string]any{
			{"label": "Blue", "description": "Calm and conventional"},
			{"label": "Green", "description": "Reads as a confirmation"},
			{"label": "Red"},
		}}
	}},
	{"/fetch", "fetch_page", func(in input) map[string]any {
		return map[string]any{"url": strings.Fields(after(in.lastUser, "/fetch", "https://example.com/"))[0]}
	}},
	{"/search", "web_search", func(in input) map[string]any {
		return map[string]any{"query": after(in.lastUser, "/search", "klean chat ui")}
	}},
	{"/loop", "create_artifact", func(in input) map[string]any {
		return map[string]any{"title": "Loop", "kind": "markdown", "content": fmt.Sprintf("Request with %d input items.", in.count)}
	}},
}

// pickCall decides whether this request should be answered with a tool call.
func pickCall(in input, offered map[string]bool) *call {
	for _, t := range triggers {
		if !strings.Contains(in.lastUser, t.word) || !offered[t.tool] {
			continue
		}
		// The call was already made and its output is here: answer in text.
		if in.answered && t.word != "/loop" {
			return nil
		}
		return &call{t.tool, t.args(in)}
	}
	return nil
}

func clip(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

func (f *Fake) responses(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"error": map[string]any{"message": err.Error()}})
		return
	}
	f.mu.Lock()
	f.last, f.lastHdr = body, r.Header.Clone()
	f.requests = append(f.requests, body)
	f.calls++
	callNo := f.calls
	f.mu.Unlock()

	items, _ := body["input"].([]any)
	in := readInput(items)
	lastUser := in.lastUser
	offered := offeredTools(body)
	if strings.Contains(lastUser, "FAIL") && !strings.Contains(lastUser, "MIDFAIL") {
		writeJSON(w, 500, map[string]any{"error": map[string]any{"message": "model exploded"}})
		return
	}
	if strings.Contains(lastUser, "NOTOOLS") && len(offered) > 0 {
		writeJSON(w, 400, map[string]any{"error": map[string]any{"message": "this model does not support tools"}})
		return
	}
	pause := 10 * time.Millisecond
	if strings.Contains(lastUser, "SLOW") {
		pause = 300 * time.Millisecond
	}
	toolCall := pickCall(in, offered)
	text := fmt.Sprintf("You said: **%s**. I received %d messages.\n\n```python\nprint('hello')\n```\n\nDone.", lastUser, len(items))
	switch {
	case toolCall != nil:
		text = "On it. "
	case in.answered:
		text = fmt.Sprintf("The tool returned: %s\n\nDone.", clip(in.toolOutput, 160))
	}
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
	wait := func() bool {
		select {
		case <-r.Context().Done():
			return false
		case <-time.After(pause):
			return true
		}
	}
	resp := map[string]any{"id": "resp_fake", "object": "response", "status": "in_progress", "model": body["model"], "output": []any{}}
	emit(map[string]any{"type": "response.created", "response": resp})
	for i, word := range words {
		if word == "" {
			continue
		}
		if i == 3 && strings.Contains(lastUser, "MIDFAIL") {
			emit(map[string]any{"type": "error", "code": "server_error", "message": "lost the thread", "param": nil})
			return
		}
		if !emit(map[string]any{"type": "response.output_text.delta", "item_id": "msg_1", "output_index": 0, "content_index": 0, "delta": word}) {
			return
		}
		if !wait() {
			return
		}
	}
	if toolCall != nil {
		item, ok := streamCall(emit, wait, toolCall, callNo)
		if !ok {
			return
		}
		resp["output"] = []any{item}
	}
	resp["status"] = "completed"
	emit(map[string]any{"type": "response.completed", "response": resp})
}

// streamCall sends a function call the way the Responses API does: the item
// is added with empty arguments, the arguments stream in pieces, then the
// finished item follows. It returns the finished item.
func streamCall(emit func(map[string]any) bool, wait func() bool, c *call, n int) (map[string]any, bool) {
	argsJSON, _ := json.Marshal(c.args)
	args := string(argsJSON)
	item := map[string]any{
		"type": "function_call", "id": fmt.Sprintf("fc_%d", n), "call_id": fmt.Sprintf("call_fake_%d", n),
		"name": c.name, "arguments": "", "status": "in_progress",
	}
	if !emit(map[string]any{"type": "response.output_item.added", "output_index": 1, "item": item}) || !wait() {
		return nil, false
	}
	const chunk = 120
	for i := 0; i < len(args); i += chunk {
		end := min(i+chunk, len(args))
		if !emit(map[string]any{"type": "response.function_call_arguments.delta", "item_id": item["id"],
			"output_index": 1, "delta": args[i:end]}) {
			return nil, false
		}
	}
	if !wait() {
		return nil, false
	}
	done := map[string]any{}
	for k, v := range item {
		done[k] = v
	}
	done["arguments"], done["status"] = args, "completed"
	ok := emit(map[string]any{"type": "response.function_call_arguments.done", "item_id": item["id"],
		"output_index": 1, "name": c.name, "arguments": args}) &&
		emit(map[string]any{"type": "response.output_item.done", "output_index": 1, "item": done})
	return done, ok
}
