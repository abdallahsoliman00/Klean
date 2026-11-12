package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
