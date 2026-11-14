package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"klean/backend/internal/fakeprovider"
	"klean/backend/internal/prompt"
)

// sseServer answers /responses with the given events.
func sseServer(t *testing.T, events ...string) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, e := range events {
			var v struct{ Type string }
			_ = json.Unmarshal([]byte(e), &v)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", v.Type, e)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func stream(t *testing.T, url string) (calls []ToolCall, started []string, err error) {
	req := Request{BaseURL: url, Model: "m", Turns: []prompt.Turn{{Role: "user", Content: "hi"}},
		Tools: []Tool{{Name: "web_search", Description: "d", Parameters: map[string]any{"type": "object"}}}}
	calls, err = StreamReply(context.Background(), req, Handler{
		OnCallStart: func(id, name string) { started = append(started, id+":"+name) },
	})
	return
}

const (
	added     = `{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"web_search","arguments":""}}`
	argsDone  = `{"type":"response.function_call_arguments.done","item_id":"fc_1","output_index":0,"arguments":"{\"query\":\"go\"}"}`
	itemDone  = `{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"web_search","arguments":"{\"query\":\"go\"}"}}`
	completed = `{"type":"response.completed","response":{"id":"r","output":[{"type":"function_call","id":"fc_1","call_id":"call_1","name":"web_search","arguments":"{\"query\":\"go\"}"}]}}`
	bare      = `{"type":"response.completed","response":{"id":"r","output":[]}}`
)

// Whichever of the events a server sends, each call is reported once.
func TestToolCallsFromAnyEventMix(t *testing.T) {
	want := ToolCall{ID: "call_1", Name: "web_search", Arguments: `{"query":"go"}`}
	for name, events := range map[string][]string{
		"all events":                {added, argsDone, itemDone, completed},
		"only the final response":   {completed},
		"item done, bare completed": {added, itemDone, bare},
		"arguments done only":       {added, argsDone, bare},
		"no final event":            {added, argsDone, itemDone},
		// no item ids at all, and arguments as an object rather than a string
		"loose server": {
			`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"web_search"}}`,
			`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"call_1","name":"web_search","arguments":{"query":"go"}}}`,
			bare,
		},
	} {
		calls, started, err := stream(t, sseServer(t, events...))
		if err != nil || len(calls) != 1 || calls[0] != want {
			t.Errorf("%s: calls %+v, err %v", name, calls, err)
		}
		if len(started) != 1 || started[0] != "call_1:web_search" {
			t.Errorf("%s: started %v", name, started)
		}
	}
}

func TestTwoCallsKeepTheirOrder(t *testing.T) {
	second := strings.NewReplacer("fc_1", "fc_2", "call_1", "call_2", `"output_index":0`, `"output_index":1`, "go", "rust")
	calls, _, err := stream(t, sseServer(t, added, second.Replace(added), itemDone, second.Replace(itemDone), bare))
	if err != nil || len(calls) != 2 || calls[0].ID != "call_1" || calls[1].ID != "call_2" || calls[1].Arguments != `{"query":"rust"}` {
		t.Fatalf("calls %+v, err %v", calls, err)
	}
}

func TestRequestShape(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: response.completed\ndata: %s\n\n", bare)
	}))
	defer srv.Close()
	req := Request{BaseURL: srv.URL, Model: "m", Turns: []prompt.Turn{
		{Role: "user", Content: "find it"},
		{Role: "assistant", Call: &prompt.Call{ID: "call_9", Name: "web_search", Arguments: `{"query":"x"}`, Output: "results"}},
	}}
	if _, err := StreamReply(context.Background(), req, Handler{}); err != nil {
		t.Fatal(err)
	}
	if _, has := body["tools"]; has {
		t.Fatal("tools sent though none were given")
	}
	items := body["input"].([]any)
	call, out := items[1].(map[string]any), items[2].(map[string]any)
	if len(items) != 3 || call["type"] != "function_call" || call["call_id"] != "call_9" || call["arguments"] != `{"query":"x"}` ||
		out["type"] != "function_call_output" || out["call_id"] != "call_9" || out["output"] != "results" {
		t.Fatalf("input: %v", items)
	}
	if body["store"] != false || body["previous_response_id"] != nil {
		t.Fatalf("store=%v previous_response_id=%v", body["store"], body["previous_response_id"])
	}
}

// Only a refusal that mentions tools is blamed on them, and only when tools were sent.
func TestToolRejection(t *testing.T) {
	reject := func(status int, msg string) string {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			fmt.Fprintf(w, `{"error":{"message":%q}}`, msg)
		}))
		t.Cleanup(srv.Close)
		return srv.URL
	}
	_, _, err := stream(t, reject(400, "registry.ollama.ai/library/gemma does not support tools"))
	if err == nil || !strings.Contains(err.Error(), "doesn't appear to support tools") || !strings.Contains(err.Error(), "gemma") {
		t.Fatalf("tools rejected: %v", err)
	}
	_, _, err = stream(t, reject(422, "Unsupported parameter: function calling"))
	if err == nil || !strings.Contains(err.Error(), "doesn't appear to support tools") {
		t.Fatalf("functions rejected: %v", err)
	}
	for _, url := range []string{reject(400, "context length exceeded"), reject(500, "tool runner crashed")} {
		if _, _, err = stream(t, url); err == nil || strings.Contains(err.Error(), "support tools") {
			t.Fatalf("unrelated error blamed on tools: %v", err)
		}
	}
}

func TestNoSuchEndpoint(t *testing.T) {
	for _, c := range []struct {
		status  int
		detail  string
		missing bool
	}{
		{404, "404 page not found", true},
		{404, "Not Found", true},
		{404, "", true},
		{405, "Method Not Allowed", true},
		{404, "model 'llama9' not found", false},
		{404, "The model `gpt-9` does not exist or you do not have access to it.", false},
		{400, "no such route", false},
		{500, "Not Found", false},
	} {
		if got := noSuchEndpoint(c.status, c.detail); got != c.missing {
			t.Errorf("%d %q: missing=%v", c.status, c.detail, got)
		}
	}
}

var onePixel = prompt.Image{MediaType: "image/png", Data: "iVBORw0KGgo="}

// A provider without /responses is answered through /chat/completions, with
// the same callbacks and result, and is not asked for /responses again.
func TestFallsBackToCompletions(t *testing.T) {
	fake := fakeprovider.NewCompletionsOnly()
	srv := httptest.NewServer(fake.Handler())
	defer srv.Close()
	req := Request{
		BaseURL: srv.URL + "/v1", Model: "m", Instructions: "Be brief.", Temperature: 0.5,
		Turns: []prompt.Turn{{Role: "user", Content: "/search gophers", Images: []prompt.Image{onePixel}}},
		Tools: []Tool{{Name: "web_search", Description: "d", Parameters: map[string]any{"type": "object"}}},
	}
	var text strings.Builder
	var started []string
	h := Handler{
		OnDelta:     func(s string) { text.WriteString(s) },
		OnCallStart: func(id, name string) { started = append(started, id+":"+name) },
	}
	calls, err := StreamReply(context.Background(), req, h)
	if err != nil || len(calls) != 1 || calls[0].Name != "web_search" || calls[0].Arguments != `{"query":"gophers"}` {
		t.Fatalf("calls %+v, err %v", calls, err)
	}
	if text.String() != "On it. " || len(started) != 1 || started[0] != calls[0].ID+":web_search" {
		t.Fatalf("text %q, started %v, id %q", text.String(), started, calls[0].ID)
	}

	// The request, as Chat Completions takes it.
	body, _ := fake.Last()
	msgs := body["messages"].([]any)
	system, user := msgs[0].(map[string]any), msgs[1].(map[string]any)
	parts := user["content"].([]any)
	image := parts[1].(map[string]any)
	if len(msgs) != 2 || system["role"] != "system" || system["content"] != "Be brief." || body["temperature"] != 0.5 ||
		parts[0].(map[string]any)["text"] != "/search gophers" || image["type"] != "image_url" ||
		image["image_url"].(map[string]any)["url"] != "data:image/png;base64,iVBORw0KGgo=" {
		t.Fatalf("messages: %v", msgs)
	}
	fn := body["tools"].([]any)[0].(map[string]any)
	if fn["type"] != "function" || fn["function"].(map[string]any)["name"] != "web_search" {
		t.Fatalf("tools: %v", body["tools"])
	}

	// The call and its output are replayed, and /responses is not tried again.
	text.Reset()
	req.Turns = append(req.Turns, prompt.Turn{Role: "assistant", Content: "On it. "},
		prompt.Turn{Role: "assistant", Call: &prompt.Call{ID: calls[0].ID, Name: "web_search", Arguments: calls[0].Arguments, Output: "results"}})
	calls, err = StreamReply(context.Background(), req, h)
	if err != nil || len(calls) != 0 || !strings.Contains(text.String(), "The tool returned: results") {
		t.Fatalf("second: calls %+v, err %v, text %q", calls, err, text.String())
	}
	body, _ = fake.Last()
	msgs = body["messages"].([]any)
	asked, output := msgs[3].(map[string]any), msgs[4].(map[string]any)
	call := asked["tool_calls"].([]any)[0].(map[string]any)
	if len(msgs) != 5 || asked["role"] != "assistant" || call["id"] != output["tool_call_id"] ||
		call["function"].(map[string]any)["arguments"] != `{"query":"gophers"}` || output["role"] != "tool" || output["content"] != "results" {
		t.Fatalf("replayed call: %v", msgs[3:])
	}
	if n := fake.ResponsesMissed(); n != 1 {
		t.Fatalf("/responses was tried %d times", n)
	}
}

// statusServer answers every path with the status and error message, and
// counts the requests per path.
func statusServer(t *testing.T, status int, msg string) (string, map[string]int) {
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprintf(w, `{"error":{"message":%q}}`, msg)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, hits
}

// A 404 for the model is the provider's answer, not a missing endpoint.
func TestModelNotFoundIsNotAFallback(t *testing.T) {
	url, hits := statusServer(t, 404, "model 'm' not found, try pulling it first")
	_, _, err := stream(t, url)
	if err == nil || !strings.Contains(err.Error(), "try pulling it first") || hits["/chat/completions"] != 0 {
		t.Fatalf("err %v, hits %v", err, hits)
	}
	// With neither endpoint there, the advice is about the base URL, and the
	// provider is not remembered as one that needs completions.
	url, hits = statusServer(t, 404, "Not Found")
	for range 2 {
		if _, _, err = stream(t, url); err == nil || !strings.Contains(err.Error(), "neither /responses nor /chat/completions") {
			t.Fatalf("err %v", err)
		}
	}
	if hits["/responses"] != 2 || hits["/chat/completions"] != 2 {
		t.Fatalf("hits %v", hits)
	}
}

func TestImagesInResponsesInputAndRejection(t *testing.T) {
	fake := fakeprovider.New()
	srv := httptest.NewServer(fake.Handler())
	defer srv.Close()
	completions := httptest.NewServer(fakeprovider.NewCompletionsOnly().Handler())
	defer completions.Close()
	req := Request{BaseURL: srv.URL + "/v1", Model: "m", Turns: []prompt.Turn{{Role: "user", Images: []prompt.Image{onePixel}}}}
	if _, err := StreamReply(context.Background(), req, Handler{}); err != nil {
		t.Fatal(err)
	}
	body, _ := fake.Last()
	parts := body["input"].([]any)[0].(map[string]any)["content"].([]any)
	if p := parts[0].(map[string]any); len(parts) != 1 || p["type"] != "input_image" || p["image_url"] != "data:image/png;base64,iVBORw0KGgo=" {
		t.Fatalf("content: %v", parts)
	}

	// A refusal of the images is explained, on either API; without images
	// in the request the same words are passed on as they are.
	req.Turns[0].Content = "NOIMAGES"
	for _, base := range []string{srv.URL, completions.URL} {
		req.BaseURL = base + "/v1"
		_, err := StreamReply(context.Background(), req, Handler{})
		if err == nil || !strings.Contains(err.Error(), "can't read images") || !strings.Contains(err.Error(), "does not support image input") {
			t.Fatalf("%s: %v", base, err)
		}
	}
	url, _ := statusServer(t, 400, "image input is not supported")
	if _, _, err := stream(t, url); err == nil || strings.Contains(err.Error(), "can't read images") {
		t.Fatalf("blamed on images that weren't sent: %v", err)
	}
}
