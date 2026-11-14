package server_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"klean/backend/internal/config"
)

// firstChat makes a provider and an untitled chat, and returns their ids.
func firstChat(e *env) (providerID, chatID string) {
	e.t.Helper()
	var p provider
	e.do("POST", "/providers", map[string]any{"name": "Fake", "baseUrl": e.base}, &p)
	var c chat
	e.do("POST", "/chats", map[string]any{}, &c)
	return p.ID, c.ID
}

func (e *env) title(chatID string) string {
	e.t.Helper()
	var c chat
	e.do("GET", "/chats/"+chatID, nil, &c)
	return c.Title
}

func TestSummaryTitle(t *testing.T) {
	e := setup(t)
	pid, id := firstChat(e)
	body := map[string]any{"providerId": pid, "model": "fake-small", "content": "hello there"}

	ev := e.sendAll(id, body)
	n := len(ev)
	if ev[n-2].name != "done" || ev[n-2].data["title"] != "hello there" {
		t.Fatalf("done carries the provisional title: %+v", ev[n-2])
	}
	// The fake answers with quotes and a full stop, which are removed.
	if ev[n-1].name != "title" || len(ev[n-1].data) != 1 || ev[n-1].data["title"] != "A short summary" {
		t.Fatalf("title event: %+v", ev[n-1])
	}
	if got := e.title(id); got != "A short summary" {
		t.Fatalf("saved title: %q", got)
	}
	reqs := e.fake.TitleRequests()
	if len(reqs) != 1 || reqs[0]["model"] != "fake-small" || reqs[0]["tools"] != nil {
		t.Fatalf("title requests: %v", reqs)
	}
	input := reqs[0]["input"].([]any)[0].(map[string]any)["content"].(string)
	if !strings.Contains(input, "hello there") || !strings.Contains(input, "received 1 messages") {
		t.Fatalf("title request input: %q", input)
	}

	// Only the message that named the chat asks for a title, and the request
	// is not kept as a message.
	if ev = e.sendAll(id, body); ev[len(ev)-1].name != "done" {
		t.Fatalf("second message: %+v", ev[len(ev)-1])
	}
	var c chat
	e.do("GET", "/chats/"+id, nil, &c)
	if len(e.fake.TitleRequests()) != 1 || rolesOf(c.Messages) != "user,assistant,user,assistant" {
		t.Fatalf("after second: %d title requests, %s", len(e.fake.TitleRequests()), rolesOf(c.Messages))
	}
}

func TestSummaryTitleFailureKeepsProvisional(t *testing.T) {
	for _, content := range []string{"hi NOTITLE", "hi BLANKTITLE"} {
		e := setup(t)
		pid, id := firstChat(e)
		ev := e.sendAll(id, map[string]any{"providerId": pid, "model": "fake-small", "content": content})
		if last := ev[len(ev)-1]; last.name != "done" || last.data["title"] != content {
			t.Fatalf("%s: last event %+v", content, last)
		}
		if len(e.fake.TitleRequests()) != 1 || e.title(id) != content {
			t.Fatalf("%s: title %q", content, e.title(id))
		}
	}
}

// A chat renamed while its first reply streams keeps the name the user chose.
func TestSummaryTitleKeepsRename(t *testing.T) {
	e := setup(t)
	pid, id := firstChat(e)
	b, _ := json.Marshal(map[string]any{"providerId": pid, "model": "fake-small", "content": "hello there"})
	res, err := http.Post(e.api.URL+"/api/chats/"+id+"/messages", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	r := bufio.NewReader(res.Body)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "event: token\n" {
			break
		}
	}
	e.do("PATCH", "/chats/"+id, map[string]any{"title": "My name"}, nil)
	ev := parseSSE(t, r)
	if last := ev[len(ev)-1]; last.name != "done" {
		t.Fatalf("last event: %+v", last)
	}
	if len(e.fake.TitleRequests()) != 1 || e.title(id) != "My name" {
		t.Fatalf("title: %q", e.title(id))
	}
}

type preset struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	ProviderID    *string `json:"providerId"`
	Model         *string `json:"model"`
	SystemPrompt  string  `json:"systemPrompt"`
	Temperature   float64 `json:"temperature"`
	ContextWindow int     `json:"contextWindow"`
}

func TestPresetCRUD(t *testing.T) {
	e := setup(t)
	var p provider
	e.do("POST", "/providers", map[string]any{"name": "Fake", "baseUrl": e.base}, &p)

	var list []preset
	if code := e.do("GET", "/presets", nil, &list); code != 200 || list == nil || len(list) != 0 {
		t.Fatalf("empty list: %d %v", code, list)
	}
	var raw map[string]any
	if code := e.do("POST", "/presets", map[string]any{"name": "  writer "}, &raw); code != 201 {
		t.Fatalf("create: %d %v", code, raw)
	}
	if got := keys(raw); got != "contextWindow,id,model,name,providerId,systemPrompt,temperature" {
		t.Fatalf("preset keys: %s", got)
	}
	if raw["name"] != "writer" || raw["providerId"] != nil || raw["model"] != nil || raw["systemPrompt"] != "" ||
		raw["temperature"] != config.DefaultTemperature || raw["contextWindow"] != float64(config.DefaultContextWindow) {
		t.Fatalf("defaults: %v", raw)
	}
	var coder preset
	e.do("POST", "/presets", map[string]any{"name": "Coder", "providerId": p.ID, "model": "fake-large",
		"systemPrompt": "Write code.", "temperature": 0.2, "contextWindow": 4096}, &coder)
	if *coder.ProviderID != p.ID || *coder.Model != "fake-large" || coder.SystemPrompt != "Write code." ||
		coder.Temperature != 0.2 || coder.ContextWindow != 4096 {
		t.Fatalf("created: %+v", coder)
	}
	e.do("GET", "/presets", nil, &list)
	if len(list) != 2 || list[0].Name != "Coder" || list[1].Name != "writer" {
		t.Fatalf("list is ordered by name: %+v", list)
	}

	// A patch changes only what it carries; null clears the model.
	var patched preset
	if code := e.do("PATCH", "/presets/"+coder.ID, map[string]any{"name": "Zed", "temperature": 1.5, "model": nil}, &patched); code != 200 {
		t.Fatalf("patch: %d", code)
	}
	if patched.Name != "Zed" || patched.Temperature != 1.5 || patched.Model != nil || *patched.ProviderID != p.ID ||
		patched.SystemPrompt != "Write code." || patched.ContextWindow != 4096 {
		t.Fatalf("patched: %+v", patched)
	}

	// Deleting the provider clears it on the preset.
	e.do("DELETE", "/providers/"+p.ID, nil, nil)
	e.do("GET", "/presets", nil, &list)
	if len(list) != 2 || list[1].ID != coder.ID || list[1].ProviderID != nil {
		t.Fatalf("after provider delete: %+v", list)
	}

	if code := e.do("DELETE", "/presets/"+coder.ID, nil, nil); code != 204 {
		t.Fatalf("delete: %d", code)
	}
	e.do("GET", "/presets", nil, &list)
	if len(list) != 1 {
		t.Fatalf("after delete: %+v", list)
	}
}

func TestPresetValidation(t *testing.T) {
	e := setup(t)
	var p preset
	e.do("POST", "/presets", map[string]any{"name": "ok"}, &p)
	long := strings.Repeat("n", config.PresetNameMaxChars+1)
	cases := []struct {
		method, path string
		body         any
		want         int
	}{
		{"POST", "/presets", map[string]any{}, 422},
		{"POST", "/presets", map[string]any{"name": "   "}, 422},
		{"POST", "/presets", map[string]any{"name": long}, 422},
		{"POST", "/presets", map[string]any{"name": "x", "temperature": 3}, 422},
		{"POST", "/presets", map[string]any{"name": "x", "contextWindow": 10}, 422},
		{"POST", "/presets", map[string]any{"name": "x", "providerId": "nope"}, 422},
		{"PATCH", "/presets/" + p.ID, map[string]any{"name": ""}, 422},
		{"PATCH", "/presets/" + p.ID, map[string]any{"temperature": -1}, 422},
		{"PATCH", "/presets/" + p.ID, map[string]any{"contextWindow": config.MaxContextWindow + 1}, 422},
		{"PATCH", "/presets/nope", map[string]any{"name": "x"}, 404},
		{"DELETE", "/presets/nope", nil, 404},
	}
	for _, tc := range cases {
		var detail struct{ Detail string }
		if code := e.do(tc.method, tc.path, tc.body, &detail); code != tc.want || detail.Detail == "" {
			t.Errorf("%s %s %v: got %d %q, want %d", tc.method, tc.path, tc.body, code, detail.Detail, tc.want)
		}
	}
	var list []preset
	e.do("GET", "/presets", nil, &list)
	if len(list) != 1 || list[0].Name != "ok" || list[0].Temperature != config.DefaultTemperature {
		t.Fatalf("rejected requests changed something: %+v", list)
	}
}

func TestChatDefaultsRoundTrip(t *testing.T) {
	e := setup(t)
	var p provider
	e.do("POST", "/providers", map[string]any{"name": "Fake", "baseUrl": e.base}, &p)
	var st struct {
		SearchProvider string
		ChatDefaults   map[string]any
	}
	e.do("GET", "/settings", nil, &st)
	if st.ChatDefaults != nil {
		t.Fatalf("defaults before any were saved: %v", st.ChatDefaults)
	}

	want := map[string]any{"providerId": p.ID, "model": "fake-large", "systemPrompt": "Be brief.",
		"temperature": 1.2, "contextWindow": float64(4096)}
	if code := e.do("PATCH", "/settings", map[string]any{"chatDefaults": want}, &st); code != 200 {
		t.Fatalf("patch: %d", code)
	}
	st.ChatDefaults = nil
	e.do("GET", "/settings", nil, &st)
	if len(st.ChatDefaults) != len(want) {
		t.Fatalf("chatDefaults keys: %v", st.ChatDefaults)
	}
	for k, v := range want {
		if st.ChatDefaults[k] != v {
			t.Fatalf("chatDefaults[%s] = %v, want %v", k, st.ChatDefaults[k], v)
		}
	}

	// Invalid values are refused and change nothing; other settings leave it alone.
	for _, bad := range []map[string]any{{"temperature": 3}, {"contextWindow": 10}, {"providerId": "nope"}} {
		var detail struct{ Detail string }
		if code := e.do("PATCH", "/settings", map[string]any{"chatDefaults": bad}, &detail); code != 422 || detail.Detail == "" {
			t.Errorf("%v: got %d %q", bad, code, detail.Detail)
		}
	}
	e.do("PATCH", "/settings", map[string]any{"searchProvider": "brave"}, &st)
	if st.SearchProvider != "brave" || st.ChatDefaults["temperature"] != 1.2 {
		t.Fatalf("after other changes: %+v", st)
	}

	// A deleted provider is cleared, so a new chat can still be made from it.
	e.do("DELETE", "/providers/"+p.ID, nil, nil)
	e.do("GET", "/settings", nil, &st)
	if st.ChatDefaults["providerId"] != nil || st.ChatDefaults["model"] != "fake-large" {
		t.Fatalf("after provider delete: %v", st.ChatDefaults)
	}

	st.ChatDefaults = map[string]any{}
	e.do("PATCH", "/settings", map[string]any{"chatDefaults": nil}, &st)
	if st.ChatDefaults != nil {
		t.Fatalf("after clearing: %v", st.ChatDefaults)
	}
}
