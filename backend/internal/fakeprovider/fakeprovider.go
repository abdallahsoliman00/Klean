// Package fakeprovider is a tiny OpenAI-compatible server (Models, and
// streaming replies over both the Responses API and Chat Completions) for
// tests and offline UI work. Base URL: {server}/v1. NewCompletionsOnly makes
// one without the Responses API, as many local servers are.
//
// The reply says how many input messages it received, so tests can check
// context trimming, and how many images came with them. A message containing
// FAIL gets a 500, MIDFAIL fails partway through the stream, SLOW streams
// slowly, and NOIMAGES rejects any request that carries images with a 400.
//
// When the request offers tools, a trigger word in the last user message
// makes it call one: /artifact, /code, /svg, /report, /skill <name>, /update,
// /ask, /search, /fetch
// (see triggers below). Once the call's output comes back it answers in text.
// /loop and /webloop call a tool on every request and never finish, NOTOOLS
// rejects any request that carries tools with a 400.
//
// A request to name a chat (its instructions are config.TitleInstructions)
// is answered with a fixed title and kept apart from the other requests.
// NOTITLE in the chat's first message fails it, BLANKTITLE answers with nothing.
package fakeprovider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"klean/backend/internal/config"
)

type Fake struct {
	mu       sync.Mutex
	last     map[string]any
	lastHdr  http.Header
	requests []map[string]any
	titles   []map[string]any // requests to name a chat
	calls    int              // numbers the call ids

	completionsOnly bool // /responses answers 404
	missed          int  // requests /responses turned away
}

func New() *Fake { return &Fake{} }

// NewCompletionsOnly returns a fake that has no Responses API: /responses
// answers 404 and replies come from /chat/completions.
func NewCompletionsOnly() *Fake { return &Fake{completionsOnly: true} }

// ResponsesMissed is how many requests found /responses missing.
func (f *Fake) ResponsesMissed() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.missed
}

// Last returns the most recent reply request body and headers.
func (f *Fake) Last() (map[string]any, http.Header) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.last, f.lastHdr
}

// Requests returns every reply request body received so far, from either API.
func (f *Fake) Requests() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]map[string]any(nil), f.requests...)
}

// TitleRequests returns every request to name a chat received so far.
func (f *Fake) TitleRequests() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]map[string]any(nil), f.titles...)
}

// titleReply is what a title request is answered with; the quotes and the
// full stop are there for the server to remove.
const titleReply = `"A short summary."`

func (f *Fake) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", f.models)
	mux.HandleFunc("POST /v1/chat/completions", f.reply(writeCompletions))
	if f.completionsOnly {
		mux.HandleFunc("POST /v1/responses", f.noResponses)
	} else {
		mux.HandleFunc("POST /v1/responses", f.reply(writeResponses))
	}
	return mux
}

func (f *Fake) noResponses(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.missed++
	f.mu.Unlock()
	http.NotFound(w, r)
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

// input is what the fake needs to know about a request: the input items of
// the Responses API, or the messages of Chat Completions.
type input struct {
	instructions string
	count        int    // items, not counting a system message
	images       int    // images attached to the user messages
	lastUser     string // text of the last user message
	toolOutput   string // output of the last call made since the last user message
	answered     bool   // such an output exists
	artifactID   string // id in the most recent artifact tool output
}

// readContent returns the text of a message's content, which is a string or
// a list of text and image parts, and how many images it has.
func readContent(content any) (text string, images int) {
	if s, ok := content.(string); ok {
		return s, 0
	}
	parts, _ := content.([]any)
	for _, p := range parts {
		m, _ := p.(map[string]any)
		switch m["type"] {
		case "input_text", "text":
			t, _ := m["text"].(string)
			text += t
		case "input_image", "image_url":
			images++
		}
	}
	return text, images
}

func readInput(body map[string]any) input {
	in := input{}
	in.instructions, _ = body["instructions"].(string)
	items, _ := body["input"].([]any)
	if items == nil {
		items, _ = body["messages"].([]any)
	}
	for _, it := range items {
		m, _ := it.(map[string]any)
		output, isOutput := m["output"].(string) // Responses: a function_call_output item
		if m["role"] == "tool" {                 // Chat Completions: a tool message
			output, isOutput = m["content"].(string)
		}
		switch {
		case m["role"] == "system":
			in.instructions, _ = m["content"].(string)
			continue
		case m["role"] == "user":
			text, images := readContent(m["content"])
			in.lastUser, in.images = text, in.images+images
			in.answered, in.toolOutput = false, ""
		case isOutput:
			in.toolOutput, in.answered = output, true
			// Artifact tools answer with JSON carrying the artifact's id.
			var out struct{ ID string }
			if json.Unmarshal([]byte(output), &out) == nil && out.ID != "" {
				in.artifactID = out.ID
			}
		}
		in.count++
	}
	return in
}

// offeredTools reads the tool names from either API's tools list.
func offeredTools(body map[string]any) map[string]bool {
	offered := map[string]bool{}
	list, _ := body["tools"].([]any)
	for _, t := range list {
		m, _ := t.(map[string]any)
		if fn, ok := m["function"].(map[string]any); ok { // Chat Completions nests it
			m = fn
		}
		if name, _ := m["name"].(string); name != "" {
			offered[name] = true
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

const demoReport = `<!doctype html>
<html><head><meta charset="utf-8"><title>Quarterly report</title>
<style>body{font-family:Georgia,serif;line-height:1.5}h1,h2{font-family:system-ui,sans-serif}
section{break-inside:avoid}table{border-collapse:collapse;width:100%}td,th{border:1px solid #ccc;padding:4px 8px;text-align:left}
tr{break-inside:avoid}.new-page{break-before:page}</style></head>
<body><h1>Quarterly report</h1><section><h2>Summary</h2><p>Revenue grew and costs held steady.</p></section>
<table><tr><th>Quarter</th><th>Revenue</th></tr><tr><td>Q1</td><td>120</td></tr><tr><td>Q2</td><td>140</td></tr></table>
<section class="new-page"><h2>Outlook</h2><p>More of the same is expected.</p></section></body></html>`

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
	{"/report", "create_report", func(input) map[string]any {
		return map[string]any{"title": "Quarterly report", "content": demoReport}
	}},
	{"/skill", "use_skill", func(in input) map[string]any {
		return map[string]any{"name": strings.Fields(after(in.lastUser, "/skill", "missing"))[0]}
	}},
	{"/update", "update_artifact", func(in input) map[string]any {
		id := in.artifactID
		if id == "" {
			id = "missing" // nothing to update: exercises the tool's error path
		}
		return map[string]any{"id": id, "title": "Demo page (updated)",
			"content": strings.Replace(demoPage, "Click me", "Click me again", 1)}
	}},
	{"/edit", "edit_artifact", func(in input) map[string]any {
		// "/edit <old> => <new>" replaces <old>; the default swaps the button label.
		old, repl, _ := strings.Cut(after(in.lastUser, "/edit", "Click me => Press me"), " => ")
		return map[string]any{"id": in.artifactID, "edits": []map[string]any{{"old_text": old, "new_text": repl}}}
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
	{"/webloop", "web_search", func(input) map[string]any {
		return map[string]any{"query": "loop"}
	}},
	{"/loop", "create_artifact", func(in input) map[string]any {
		return map[string]any{"title": "Loop", "kind": "markdown", "content": fmt.Sprintf("Request with %d input items.", in.count)}
	}},
}

// repeatTriggers keep calling on every request instead of answering once
// their output comes back.
var repeatTriggers = map[string]bool{"/loop": true, "/webloop": true}

// pickCall decides whether this request should be answered with a tool call.
func pickCall(in input, offered map[string]bool) *call {
	for _, t := range triggers {
		if !strings.Contains(in.lastUser, t.word) || !offered[t.tool] {
			continue
		}
		// The call was already made and its output is here: answer in text.
		if in.answered && !repeatTriggers[t.word] {
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

// answer is what the fake decided to reply with.
type answer struct {
	model   any
	words   []string // the text, a word at a time
	midFail bool     // break off with an error after a few words
	call    *call    // made after the text
	callNo  int
}

// stream writes Server-Sent Events at the reply's pace.
type stream struct {
	w     http.ResponseWriter
	r     *http.Request
	pause time.Duration
}

// emit sends one event; event is left out of the frame when "".
func (s stream) emit(event string, data any) bool {
	b, ok := data.(string)
	if !ok {
		raw, _ := json.Marshal(data)
		b = string(raw)
	}
	if event != "" {
		fmt.Fprintf(s.w, "event: %s\n", event)
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", b); err != nil {
		return false
	}
	return http.NewResponseController(s.w).Flush() == nil
}

// wait pauses between events; false means the client went away.
func (s stream) wait() bool {
	select {
	case <-s.r.Context().Done():
		return false
	case <-time.After(s.pause):
		return true
	}
}

// midFailAfter is how many words MIDFAIL lets through.
const midFailAfter = 3

// reply handles a reply request of either API: it records the request,
// decides the answer, and has write stream it in that API's format.
func (f *Fake) reply(write func(stream, answer)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]any{"error": map[string]any{"message": err.Error()}})
			return
		}
		in := readInput(body)
		isTitle := in.instructions == config.TitleInstructions
		f.mu.Lock()
		if isTitle {
			f.titles = append(f.titles, body)
		} else {
			f.last, f.lastHdr = body, r.Header.Clone()
			f.requests = append(f.requests, body)
		}
		f.calls++
		callNo := f.calls
		f.mu.Unlock()

		lastUser := in.lastUser
		offered := offeredTools(body)
		refuse := func(status int, msg string) {
			writeJSON(w, status, map[string]any{"error": map[string]any{"message": msg}})
		}
		switch {
		case strings.Contains(lastUser, "FAIL") && !strings.Contains(lastUser, "MIDFAIL"):
			refuse(500, "model exploded")
			return
		case isTitle && strings.Contains(lastUser, "NOTITLE"):
			refuse(500, "no title today")
			return
		case strings.Contains(lastUser, "NOTOOLS") && len(offered) > 0:
			refuse(400, "this model does not support tools")
			return
		case strings.Contains(lastUser, "NOIMAGES") && in.images > 0:
			refuse(400, "this model does not support image input")
			return
		}
		pause := 10 * time.Millisecond
		if strings.Contains(lastUser, "SLOW") && !isTitle {
			pause = 300 * time.Millisecond
		}
		toolCall := pickCall(in, offered)
		seen := ""
		if in.images > 0 {
			seen = fmt.Sprintf(" and %d images", in.images)
		}
		text := fmt.Sprintf("You said: **%s**. I received %d messages%s.\n\n```python\nprint('hello')\n```\n\nDone.", lastUser, in.count, seen)
		switch {
		case isTitle && strings.Contains(lastUser, "BLANKTITLE"):
			text = " "
		case isTitle:
			text = titleReply
		case toolCall != nil:
			text = "On it. "
		case in.answered:
			text = fmt.Sprintf("The tool returned: %s\n\nDone.", clip(in.toolOutput, 160))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		write(stream{w, r, pause}, answer{
			model: body["model"], words: strings.SplitAfter(text, " "),
			midFail: strings.Contains(lastUser, "MIDFAIL"), call: toolCall, callNo: callNo,
		})
	}
}

// argsChunk is how many bytes of a call's arguments go into one event.
const argsChunk = 120

// chunks cuts s into pieces of at most argsChunk bytes.
func chunks(s string) []string {
	var out []string
	for i := 0; i < len(s); i += argsChunk {
		out = append(out, s[i:min(i+argsChunk, len(s))])
	}
	return out
}

// writeResponses streams the answer as Responses API events.
func writeResponses(s stream, a answer) {
	seq := 0
	emit := func(v map[string]any) bool {
		v["sequence_number"] = seq
		seq++
		return s.emit(v["type"].(string), v)
	}
	resp := map[string]any{"id": "resp_fake", "object": "response", "status": "in_progress", "model": a.model, "output": []any{}}
	emit(map[string]any{"type": "response.created", "response": resp})
	for i, word := range a.words {
		if word == "" {
			continue
		}
		if i == midFailAfter && a.midFail {
			emit(map[string]any{"type": "error", "code": "server_error", "message": "lost the thread", "param": nil})
			return
		}
		if !emit(map[string]any{"type": "response.output_text.delta", "item_id": "msg_1", "output_index": 0, "content_index": 0, "delta": word}) {
			return
		}
		if !s.wait() {
			return
		}
	}
	if a.call != nil {
		item, ok := streamCall(emit, s.wait, a.call, a.callNo)
		if !ok {
			return
		}
		resp["output"] = []any{item}
	}
	resp["status"] = "completed"
	emit(map[string]any{"type": "response.completed", "response": resp})
}

// writeCompletions streams the answer as Chat Completions chunks: the text,
// then the call (its id and name first, its arguments in pieces), then [DONE].
func writeCompletions(s stream, a answer) {
	chunk := func(delta map[string]any, finish any) bool {
		return s.emit("", map[string]any{"id": "chatcmpl_fake", "object": "chat.completion.chunk", "model": a.model,
			"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}})
	}
	for i, word := range a.words {
		if word == "" {
			continue
		}
		if i == midFailAfter && a.midFail {
			s.emit("", map[string]any{"error": map[string]any{"message": "lost the thread"}})
			return
		}
		if !chunk(map[string]any{"content": word}, nil) || !s.wait() {
			return
		}
	}
	finish := "stop"
	if a.call != nil {
		finish = "tool_calls"
		argsJSON, _ := json.Marshal(a.call.args)
		piece := func(call map[string]any) bool {
			call["index"] = 0
			return chunk(map[string]any{"tool_calls": []any{call}}, nil)
		}
		if !piece(map[string]any{"id": fmt.Sprintf("call_fake_%d", a.callNo), "type": "function",
			"function": map[string]any{"name": a.call.name, "arguments": ""}}) || !s.wait() {
			return
		}
		for _, part := range chunks(string(argsJSON)) {
			if !piece(map[string]any{"function": map[string]any{"arguments": part}}) {
				return
			}
		}
	}
	if chunk(map[string]any{}, finish) {
		s.emit("", "[DONE]")
	}
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
	for _, part := range chunks(args) {
		if !emit(map[string]any{"type": "response.function_call_arguments.delta", "item_id": item["id"],
			"output_index": 1, "delta": part}) {
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
