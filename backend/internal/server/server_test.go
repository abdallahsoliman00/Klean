package server_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"klean/backend/internal/config"
	"klean/backend/internal/fakeprovider"
	"klean/backend/internal/server"
	"klean/backend/internal/store"
	"klean/frontend"
)

type env struct {
	t    *testing.T
	api  *httptest.Server
	fake *fakeprovider.Fake
	base string // fake provider base URL
}

func setup(t *testing.T) *env {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	fake := fakeprovider.New()
	up := httptest.NewServer(fake.Handler())
	srv := server.New(db, nil, "*")
	api := httptest.NewServer(srv)
	t.Cleanup(func() { srv.Close(); api.Close(); up.Close() })
	return &env{t, api, fake, up.URL + "/v1"}
}

// do sends a JSON request and decodes the JSON reply into out (if non-nil).
func (e *env) do(method, path string, body any, out any) int {
	e.t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.api.URL+"/api"+path, r)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			e.t.Fatalf("%s %s: decoding: %v", method, path, err)
		}
	}
	return res.StatusCode
}

type event struct {
	name string
	data map[string]any
}

func parseSSE(t *testing.T, r io.Reader) []event {
	t.Helper()
	var out []event
	sc := bufio.NewScanner(r)
	var cur event
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			cur.name = line[7:]
		case strings.HasPrefix(line, "data: "):
			if err := json.Unmarshal([]byte(line[6:]), &cur.data); err != nil {
				t.Fatalf("bad SSE data %q: %v", line, err)
			}
		case line == "" && cur.name != "":
			out = append(out, cur)
			cur = event{}
		}
	}
	return out
}

// send returns the reply's events. The `title` event that may follow `done`
// is left out; sendAll returns it too.
func (e *env) send(chatID string, body map[string]any) []event {
	e.t.Helper()
	return e.withoutTitle(e.sendAll(chatID, body))
}

func (e *env) withoutTitle(ev []event) []event {
	if n := len(ev); n > 0 && ev[n-1].name == "title" {
		return ev[:n-1]
	}
	return ev
}

func (e *env) sendAll(chatID string, body map[string]any) []event {
	e.t.Helper()
	b, _ := json.Marshal(body)
	res, err := http.Post(e.api.URL+"/api/chats/"+chatID+"/messages", "application/json", bytes.NewReader(b))
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		raw, _ := io.ReadAll(res.Body)
		e.t.Fatalf("send: %d %s", res.StatusCode, raw)
	}
	return parseSSE(e.t, res.Body)
}

type provider struct {
	ID      string `json:"id"`
	BaseURL string `json:"baseUrl"`
	HasKey  bool   `json:"hasKey"`
}

type message struct {
	ID, Role, Content string
	Stopped           bool
}

type chat struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	ContextWindow int       `json:"contextWindow"`
	Messages      []message `json:"messages"`
}

func TestProviderCRUDAndModels(t *testing.T) {
	e := setup(t)
	var p provider
	var raw map[string]any
	if code := e.do("POST", "/providers", map[string]any{"name": "Fake", "baseUrl": e.base + "/", "apiKey": "k"}, &raw); code != 201 {
		t.Fatalf("create: %d %v", code, raw)
	}
	if _, leaked := raw["apiKey"]; leaked {
		t.Fatal("API key was sent to the browser")
	}
	b, _ := json.Marshal(raw)
	_ = json.Unmarshal(b, &p)
	if p.BaseURL != e.base || !p.HasKey {
		t.Fatalf("unexpected provider %+v", p)
	}

	var models []struct{ ID string }
	e.do("GET", "/providers/"+p.ID+"/models", nil, &models)
	if len(models) != 2 || models[0].ID != "fake-large" || models[1].ID != "fake-small" {
		t.Fatalf("models: %+v", models)
	}

	e.do("PATCH", "/providers/"+p.ID, map[string]any{"apiKey": "bad"}, nil)
	var detail struct{ Detail string }
	if code := e.do("GET", "/providers/"+p.ID+"/models", nil, &detail); code != 502 || !strings.Contains(detail.Detail, "API key") {
		t.Fatalf("bad key: %d %q", code, detail.Detail)
	}
	// null keeps the key, "" clears it
	e.do("PATCH", "/providers/"+p.ID, map[string]any{"apiKey": nil, "name": "Renamed"}, &p)
	if !p.HasKey {
		t.Fatal("null apiKey cleared the key")
	}
	e.do("PATCH", "/providers/"+p.ID, map[string]any{"apiKey": ""}, &p)
	if p.HasKey {
		t.Fatal(`"" did not clear the key`)
	}
	e.do("DELETE", "/providers/"+p.ID, nil, nil)
	var list []provider
	e.do("GET", "/providers", nil, &list)
	if len(list) != 0 {
		t.Fatalf("providers left: %+v", list)
	}
}

func TestUnreachableAndBadProviders(t *testing.T) {
	e := setup(t)
	var p provider
	e.do("POST", "/providers", map[string]any{"name": "Off", "baseUrl": "http://127.0.0.1:1/v1"}, &p)
	var detail struct{ Detail string }
	if code := e.do("GET", "/providers/"+p.ID+"/models", nil, &detail); code != 502 || !strings.Contains(detail.Detail, "Couldn't reach") {
		t.Fatalf("unreachable: %d %q", code, detail.Detail)
	}
	if code := e.do("POST", "/providers", map[string]any{"name": "x", "baseUrl": "ftp://nope"}, nil); code != 422 {
		t.Fatalf("ftp URL accepted: %d", code)
	}
	if code := e.do("POST", "/providers", map[string]any{"baseUrl": "http://x"}, nil); code != 422 {
		t.Fatalf("missing name accepted: %d", code)
	}

	// A server with a plain-text 404 on every path (wrong base URL).
	e.do("POST", "/providers", map[string]any{"name": "Wrong", "baseUrl": strings.TrimSuffix(e.base, "/v1") + "/nope"}, &p)
	if code := e.do("GET", "/providers/"+p.ID+"/models", nil, &detail); code != 502 || !strings.Contains(detail.Detail, "404") {
		t.Fatalf("plain 404 on models: %d %q", code, detail.Detail)
	}
	var c chat
	e.do("POST", "/chats", map[string]any{}, &c)
	ev := e.send(c.ID, map[string]any{"providerId": p.ID, "model": "m", "content": "hi"})
	last := ev[len(ev)-1]
	if last.name != "error" || !strings.Contains(last.data["message"].(string), "neither /responses nor /chat/completions") {
		t.Fatalf("plain 404 on responses: %+v", last)
	}
}

func TestChatFlow(t *testing.T) {
	e := setup(t)
	var p provider
	e.do("POST", "/providers", map[string]any{"name": "Fake", "baseUrl": e.base}, &p)
	var c chat
	e.do("POST", "/chats", map[string]any{"providerId": p.ID, "model": "fake-small", "systemPrompt": "Be brief."}, &c)
	if c.ContextWindow != config.DefaultContextWindow {
		t.Fatalf("new chat context window: got %d, want %d", c.ContextWindow, config.DefaultContextWindow)
	}
	send := func(extra map[string]any) []event {
		body := map[string]any{"providerId": p.ID, "model": "fake-small"}
		for k, v := range extra {
			body[k] = v
		}
		return e.send(c.ID, body)
	}

	ev := send(map[string]any{"content": "hello there"})
	if ev[0].name != "user" || ev[len(ev)-1].name != "done" {
		t.Fatalf("events: %+v", ev)
	}
	var streamed strings.Builder
	for _, x := range ev {
		if x.name == "token" {
			streamed.WriteString(x.data["delta"].(string))
		}
	}
	done := ev[len(ev)-1].data
	content := done["message"].(map[string]any)["content"].(string)
	if streamed.String() != content {
		t.Fatalf("streamed %q, saved %q", streamed.String(), content)
	}
	if done["title"] != "hello there" {
		t.Fatalf("title: %v", done["title"])
	}
	if !strings.Contains(content, "received 1 messages") {
		t.Fatalf("reply: %q", content)
	}
	// The system prompt goes in `instructions`, and nothing is stored upstream.
	last, _ := e.fake.Last()
	// With tools on (the default), guidance on using them follows the prompt.
	if instr, _ := last["instructions"].(string); !strings.HasPrefix(instr, "Be brief.\n\nYou can call tools.") {
		t.Fatalf("instructions: %q", instr)
	}
	if last["store"] != false || last["stream"] != true || last["model"] != "fake-small" {
		t.Fatalf("request to provider: %v", last)
	}

	send(map[string]any{"content": "second"})
	var got chat
	e.do("GET", "/chats/"+c.ID, nil, &got)
	if roles := rolesOf(got.Messages); roles != "user,assistant,user,assistant" || got.Title != "A short summary" {
		t.Fatalf("after second: %s %q", roles, got.Title)
	}
	last, _ = e.fake.Last()
	if items := last["input"].([]any); len(items) != 3 || items[1].(map[string]any)["role"] != "assistant" {
		t.Fatalf("history sent: %v", items)
	}

	// regenerate replaces the last reply
	ev = send(map[string]any{"regenerate": true})
	if ev[0].name != "token" {
		t.Fatalf("regenerate events: %+v", ev[0])
	}
	var after chat
	e.do("GET", "/chats/"+c.ID, nil, &after)
	if len(after.Messages) != 4 {
		t.Fatalf("after regenerate: %d messages", len(after.Messages))
	}

	// editing the first message drops everything after it
	send(map[string]any{"content": "edited", "editMessageId": got.Messages[0].ID})
	e.do("GET", "/chats/"+c.ID, nil, &after)
	if len(after.Messages) != 2 || after.Messages[0].Content != "edited" {
		t.Fatalf("after edit: %+v", after.Messages)
	}

	// upstream failure becomes an error event, user message kept
	ev = send(map[string]any{"content": "please FAIL"})
	if x := ev[len(ev)-1]; x.name != "error" || !strings.Contains(x.data["message"].(string), "500") ||
		!strings.Contains(x.data["message"].(string), "model exploded") {
		t.Fatalf("failure: %+v", x)
	}
	e.do("GET", "/chats/"+c.ID, nil, &after)
	if after.Messages[len(after.Messages)-1].Role != "user" {
		t.Fatal("expected the user message to be kept")
	}

	// a failure partway through keeps the partial reply, marked stopped
	ev = send(map[string]any{"content": "MIDFAIL now"})
	if x := ev[len(ev)-1]; x.name != "error" || !strings.Contains(x.data["message"].(string), "lost the thread") {
		t.Fatalf("mid-stream failure: %+v", x)
	}
	e.do("GET", "/chats/"+c.ID, nil, &after)
	if m := after.Messages[len(after.Messages)-1]; m.Role != "assistant" || !m.Stopped || m.Content == "" {
		t.Fatalf("partial reply: %+v", m)
	}

	// soft delete and restore
	e.do("DELETE", "/chats/"+c.ID, nil, nil)
	var chats []chat
	e.do("GET", "/chats", nil, &chats)
	if len(chats) != 0 {
		t.Fatal("chat still listed after delete")
	}
	e.do("POST", "/chats/"+c.ID+"/restore", nil, nil)
	e.do("GET", "/chats", nil, &chats)
	if len(chats) != 1 {
		t.Fatal("chat not restored")
	}
}

// slowReply sends a message the fake answers slowly and reads the reply's
// stream until two tokens have arrived. The stream is left open.
func (e *env) slowReply(chatID, providerID string) *http.Response {
	e.t.Helper()
	b, _ := json.Marshal(map[string]any{"providerId": providerID, "model": "fake-small", "content": "go SLOW"})
	res, err := http.Post(e.api.URL+"/api/chats/"+chatID+"/messages", "application/json", bytes.NewReader(b))
	if err != nil {
		e.t.Fatal(err)
	}
	e.t.Cleanup(func() { res.Body.Close() })
	sc, tokens := bufio.NewScanner(res.Body), 0
	for tokens < 2 && sc.Scan() {
		if sc.Text() == "event: token" {
			tokens++
		}
	}
	return res
}

// chatOf loads a chat as a client that has just opened it would.
func (e *env) chatOf(id string) (got struct {
	Running  bool
	Messages []message
}) {
	e.t.Helper()
	e.do("GET", "/chats/"+id, nil, &got)
	return got
}

func TestStopSavesPartialReply(t *testing.T) {
	e := setup(t)
	var p provider
	e.do("POST", "/providers", map[string]any{"name": "Fake", "baseUrl": e.base}, &p)
	var c chat
	e.do("POST", "/chats", map[string]any{}, &c)
	res := e.slowReply(c.ID, p.ID)

	// A chat writes one reply at a time: nothing else may change it meanwhile.
	for _, body := range []map[string]any{{"content": "another"}, {"regenerate": true}} {
		body["providerId"], body["model"] = p.ID, "fake-small"
		if status := e.do("POST", "/chats/"+c.ID+"/messages", body, nil); status != 409 {
			t.Fatalf("sending %v while a reply is running: %d", body, status)
		}
	}
	var list []map[string]any
	if e.do("GET", "/chats", nil, &list); list[0]["running"] != true {
		t.Fatalf("listed chat while running: %v", list[0])
	}

	// Stopping answers once the partial reply is saved.
	if status := e.do("POST", "/chats/"+c.ID+"/stop", nil, nil); status != 204 {
		t.Fatalf("stop: %d", status)
	}
	got := e.chatOf(c.ID)
	if got.Running || len(got.Messages) != 2 {
		t.Fatalf("after stop: %+v", got)
	}
	m := got.Messages[1]
	if !m.Stopped || m.Content == "" || strings.Contains(m.Content, "Done.") {
		t.Fatalf("stopped reply: %+v", m)
	}
	ev := parseSSE(t, res.Body)
	if last := ev[len(ev)-1]; last.name != "stopped" || last.data["message"].(map[string]any)["id"] != m.ID {
		t.Fatalf("last event: %+v", last)
	}
	if status := e.do("POST", "/chats/"+c.ID+"/stop", nil, nil); status != 204 {
		t.Fatalf("stopping again: %d", status)
	}
}

// Hanging up doesn't stop the reply: it is written and saved all the same,
// and a client that comes back picks it up where it is.
func TestReplyOutlivesClient(t *testing.T) {
	e := setup(t)
	var p provider
	e.do("POST", "/providers", map[string]any{"name": "Fake", "baseUrl": e.base}, &p)
	var c chat
	e.do("POST", "/chats", map[string]any{}, &c)
	e.slowReply(c.ID, p.ID).Body.Close()

	if got := e.chatOf(c.ID); !got.Running || len(got.Messages) != 1 {
		t.Fatalf("after hanging up: %+v", got)
	}
	res, err := http.Get(e.api.URL + "/api/chats/" + c.ID + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	ev := e.withoutTitle(parseSSE(t, res.Body))
	if res.StatusCode != 200 || len(ev) < 4 || ev[0].name != "token" || ev[1].name != "token" || ev[len(ev)-1].name != "done" {
		t.Fatalf("attached stream: %d %+v", res.StatusCode, ev)
	}
	var replayed strings.Builder
	for _, x := range ev[:len(ev)-1] {
		replayed.WriteString(x.data["delta"].(string))
	}
	got := e.chatOf(c.ID)
	if got.Running || len(got.Messages) != 2 {
		t.Fatalf("after the reply: %+v", got)
	}
	if m := got.Messages[1]; m.Stopped || !strings.HasSuffix(m.Content, "Done.") || m.Content != replayed.String() {
		t.Fatalf("reply: %+v, replayed %q", m, replayed.String())
	}
	if status := e.do("GET", "/chats/"+c.ID+"/stream", nil, nil); status != 204 {
		t.Fatalf("attaching to a finished reply: %d", status)
	}
}

// The SDK reads OPENAI_API_KEY and friends from the environment; none of it
// may reach a provider that has no key of its own.
func TestEnvironmentKeyNotLeaked(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-from-env")
	t.Setenv("OPENAI_ORG_ID", "org-env")
	t.Setenv("OPENAI_BASE_URL", "http://127.0.0.1:1/v1")
	e := setup(t)
	var p provider
	e.do("POST", "/providers", map[string]any{"name": "Local", "baseUrl": e.base}, &p)
	var c chat
	e.do("POST", "/chats", map[string]any{}, &c)
	ev := e.send(c.ID, map[string]any{"providerId": p.ID, "model": "m", "content": "hi"})
	if ev[len(ev)-1].name != "done" {
		t.Fatalf("events: %+v", ev[len(ev)-1])
	}
	_, hdr := e.fake.Last()
	if hdr.Get("Authorization") != "" || hdr.Get("OpenAI-Organization") != "" {
		t.Fatalf("environment credentials leaked: %v", hdr)
	}
}

func TestValidation(t *testing.T) {
	e := setup(t)
	var c chat
	e.do("POST", "/chats", map[string]any{}, &c)
	cases := []struct {
		method, path string
		body         any
		want         int
	}{
		{"POST", "/chats", map[string]any{"temperature": 3}, 422},
		{"POST", "/chats", map[string]any{"contextWindow": 10}, 422},
		{"PATCH", "/chats/" + c.ID, map[string]any{"title": ""}, 422},
		{"PATCH", "/chats/nope", map[string]any{"title": "x"}, 404},
		{"GET", "/chats/nope", nil, 404},
		{"DELETE", "/chats/nope", nil, 404},
		{"POST", "/chats/" + c.ID + "/messages", map[string]any{"providerId": "nope", "model": "m", "content": "hi"}, 400},
		{"POST", "/chats/" + c.ID + "/messages", map[string]any{"content": "hi"}, 422},
		{"GET", "/nothing-here", nil, 404},
	}
	for _, tc := range cases {
		var detail struct{ Detail string }
		if code := e.do(tc.method, tc.path, tc.body, &detail); code != tc.want || detail.Detail == "" {
			t.Errorf("%s %s: got %d %q, want %d", tc.method, tc.path, code, detail.Detail, tc.want)
		}
	}
	// null providerId clears it; null title is ignored
	var p provider
	e.do("POST", "/providers", map[string]any{"name": "Fake", "baseUrl": e.base}, &p)
	var raw map[string]any
	e.do("PATCH", "/chats/"+c.ID, map[string]any{"providerId": p.ID, "model": "x"}, &raw)
	e.do("PATCH", "/chats/"+c.ID, map[string]any{"providerId": nil, "title": nil}, &raw)
	if raw["providerId"] != nil || raw["model"] != "x" || raw["title"] != "New chat" {
		t.Fatalf("patch with nulls: %v", raw)
	}
}

// The frontend relies on these exact keys (it pushes onto chat.messages, so
// "messages" must be [] for a new chat, never missing or null).
// keys lists an object's keys in order.
func keys(m map[string]any) string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return strings.Join(ks, ",")
}

func TestResponseShapes(t *testing.T) {
	e := setup(t)
	const chatKeys = "contextWindow,createdAt,favorite,id,model,providerId,running,systemPrompt,temperature,title,updatedAt"

	var created map[string]any
	e.do("POST", "/chats", map[string]any{}, &created)
	if msgs, ok := created["messages"].([]any); !ok || len(msgs) != 0 {
		t.Fatalf("new chat messages = %#v, want []", created["messages"])
	}
	if got := keys(created); got != "contextWindow,createdAt,favorite,id,messages,model,providerId,running,systemPrompt,temperature,title,updatedAt" {
		t.Fatalf("created chat keys: %s", got)
	}
	id := created["id"].(string)

	var fetched map[string]any
	e.do("GET", "/chats/"+id, nil, &fetched)
	if msgs, ok := fetched["messages"].([]any); !ok || len(msgs) != 0 {
		t.Fatalf("fetched empty chat messages = %#v, want []", fetched["messages"])
	}

	var list []map[string]any
	e.do("GET", "/chats", nil, &list)
	if got := keys(list[0]); got != chatKeys {
		t.Fatalf("listed chat keys: %s", got)
	}
	var patched map[string]any
	e.do("PATCH", "/chats/"+id, map[string]any{"title": "x"}, &patched)
	if got := keys(patched); got != chatKeys {
		t.Fatalf("patched chat keys: %s", got)
	}
	// A favourite keeps its place in the list: updatedAt stays as it was.
	var starred map[string]any
	e.do("PATCH", "/chats/"+id, map[string]any{"favorite": true}, &starred)
	e.do("GET", "/chats", nil, &list)
	if starred["favorite"] != true || list[0]["favorite"] != true || starred["updatedAt"] != created["updatedAt"] {
		t.Fatalf("favourite chat: %v, listed %v, created %v", starred, list[0], created)
	}

	var p map[string]any
	e.do("POST", "/providers", map[string]any{"name": "Fake", "baseUrl": e.base}, &p)
	if got := keys(p); got != "baseUrl,hasKey,id,name" {
		t.Fatalf("provider keys: %s", got)
	}
	ev := e.send(id, map[string]any{"providerId": p["id"], "model": "m", "content": "hi"})
	if got := keys(ev[0].data); got != "content,createdAt,id,model,role,stopped" {
		t.Fatalf("message keys: %s", got)
	}
}

func rolesOf(ms []message) string {
	r := make([]string, len(ms))
	for i, m := range ms {
		r[i] = m.Role
	}
	return strings.Join(r, ",")
}

func TestServesEmbeddedFrontend(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	srv := httptest.NewServer(server.New(db, frontend.Files, "*"))
	defer srv.Close()
	get := func(path string) string {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("GET %s: %d", path, res.StatusCode)
		}
		return string(body)
	}
	index := get("/")
	if !strings.Contains(index, `<div id="root">`) {
		t.Fatalf("index.html has no React root:\n%s", index)
	}
	// Vite hashes asset names per build, so follow the script tag rather than hard-coding it.
	_, after, ok := strings.Cut(index, `src="./assets/`)
	script, _, _ := strings.Cut(after, `"`)
	if !ok || !strings.HasSuffix(script, ".js") || get("/assets/"+script) == "" {
		t.Errorf("bundle script %q not served", script)
	}
	var health map[string]any
	res, _ := http.Get(srv.URL + "/api/health")
	_ = json.NewDecoder(res.Body).Decode(&health)
	res.Body.Close()
	if health["app"] != config.AppID {
		t.Errorf("health: %v", health)
	}
}
