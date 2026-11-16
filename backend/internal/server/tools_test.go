package server_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"klean/backend/internal/config"
	"klean/backend/internal/fakeprovider"
	"klean/backend/internal/server"
	"klean/backend/internal/store"
	"klean/backend/internal/tools"
)

type part struct {
	Type   string         `json:"type"`
	Text   string         `json:"text"`
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Args   map[string]any `json:"args"`
	Status string         `json:"status"`
	Result map[string]any `json:"result"`
	Error  string         `json:"error"`
}

type artifact struct {
	ID, ChatID, Title, Kind, Language, Content string
	Version                                    int
}

type toolChat struct {
	Messages []struct {
		ID, Role, Content string
		Stopped           bool
		Parts             []part
	}
	Artifacts []artifact
}

// toolEnv is a server with a provider and a chat ready to use.
type toolEnv struct {
	*env
	providerID, chatID string
}

func setupTools(t *testing.T, opts ...server.Option) *toolEnv {
	t.Helper()
	return setupToolsWith(t, fakeprovider.New(), opts...)
}

// setupToolsWith is setupTools with the given fake as the provider.
func setupToolsWith(t *testing.T, fake *fakeprovider.Fake, opts ...server.Option) *toolEnv {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	up := httptest.NewServer(fake.Handler())
	srv := server.New(db, nil, "*", opts...)
	api := httptest.NewServer(srv)
	t.Cleanup(func() { srv.Close(); api.Close(); up.Close() })
	e := &env{t, api, fake, up.URL + "/v1"}
	var p provider
	e.do("POST", "/providers", map[string]any{"name": "Fake", "baseUrl": e.base}, &p)
	var c chat
	e.do("POST", "/chats", map[string]any{"systemPrompt": "Be brief."}, &c)
	return &toolEnv{e, p.ID, c.ID}
}

func (e *toolEnv) say(content string, extra map[string]any) []event {
	e.t.Helper()
	body := map[string]any{"providerId": e.providerID, "model": "fake-small"}
	if content != "" {
		body["content"] = content
	}
	for k, v := range extra {
		body[k] = v
	}
	return e.send(e.chatID, body)
}

func (e *toolEnv) chat() toolChat {
	e.t.Helper()
	var c toolChat
	e.do("GET", "/chats/"+e.chatID, nil, &c)
	return c
}

// names lists the event names, with runs of tokens shown once.
func names(ev []event) string {
	var out []string
	for _, x := range ev {
		if x.name == "token" && len(out) > 0 && out[len(out)-1] == "token" {
			continue
		}
		out = append(out, x.name)
	}
	return strings.Join(out, ",")
}

// as re-decodes a piece of event data into a typed value.
func as[T any](t *testing.T, v any) T {
	t.Helper()
	var out T
	b, _ := json.Marshal(v)
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("decoding %s: %v", b, err)
	}
	return out
}

func find(ev []event, name string) *event {
	for i := range ev {
		if ev[i].name == name {
			return &ev[i]
		}
	}
	return nil
}

// inputTypes summarises a provider request's input: roles and item types in order.
func inputTypes(req map[string]any) string {
	var out []string
	for _, it := range req["input"].([]any) {
		m := it.(map[string]any)
		if role, ok := m["role"].(string); ok {
			out = append(out, role)
		} else {
			out = append(out, m["type"].(string))
		}
	}
	return strings.Join(out, ",")
}

func TestArtifactRoundTrip(t *testing.T) {
	e := setupTools(t)
	ev := e.say("make an /artifact please", nil)
	if got := names(ev); got != "user,token,tool_start,tool_run,artifact,tool_end,token,done" {
		t.Fatalf("events: %s", got)
	}
	start, run := find(ev, "tool_start").data, find(ev, "tool_run").data
	if start["name"] != "create_artifact" || start["id"] == "" || start["id"] != run["id"] {
		t.Fatalf("tool_start %v, tool_run %v", start, run)
	}
	if run["args"].(map[string]any)["kind"] != "html" {
		t.Fatalf("tool_run args: %v", run["args"])
	}
	art := as[artifact](t, find(ev, "artifact").data["artifact"])
	if art.ID == "" || art.ChatID != e.chatID || art.Kind != "html" || art.Version != 1 || art.Language != "" ||
		!strings.Contains(art.Content, "<button") {
		t.Fatalf("artifact event: %+v", art)
	}
	endData := find(ev, "tool_end").data["part"].(map[string]any)
	if _, leaked := endData["output"]; leaked {
		t.Fatal("the model-facing output was sent to the browser")
	}
	end := as[part](t, endData)
	if end.Status != "done" || end.ID != start["id"] || end.Result["artifactId"] != art.ID ||
		end.Result["version"] != float64(1) || end.Result["title"] != "Demo page" || end.Result["kind"] != "html" {
		t.Fatalf("tool_end: %+v", end)
	}

	// The saved message: text, the call, text; content is the text joined.
	done := ev[len(ev)-1].data["message"].(map[string]any)
	parts := as[[]part](t, done["parts"])
	if len(parts) != 3 || parts[0].Type != "text" || parts[1].Type != "tool" || parts[2].Type != "text" {
		t.Fatalf("parts: %+v", parts)
	}
	if done["content"] != parts[0].Text+parts[2].Text || done["stopped"] != false {
		t.Fatalf("content %q", done["content"])
	}
	got := e.chat()
	if len(got.Artifacts) != 1 || got.Artifacts[0].ID != art.ID {
		t.Fatalf("artifacts in chat: %+v", got.Artifacts)
	}
	if m := got.Messages[1]; len(m.Parts) != 3 || m.Parts[1].Result["artifactId"] != art.ID {
		t.Fatalf("saved parts: %+v", m.Parts)
	}
	if got.Messages[0].Parts != nil {
		t.Fatal("user message has parts")
	}

	// The second request replayed the call and its output right after it.
	reqs := e.fake.Requests()
	if len(reqs) != 2 || inputTypes(reqs[1]) != "user,assistant,function_call,function_call_output" {
		t.Fatalf("requests: %d, second input %s", len(reqs), inputTypes(reqs[len(reqs)-1]))
	}
	items := reqs[1]["input"].([]any)
	fc, fco := items[2].(map[string]any), items[3].(map[string]any)
	if fc["call_id"] != start["id"] || fco["call_id"] != start["id"] || fc["name"] != "create_artifact" ||
		!strings.Contains(fco["output"].(string), art.ID) {
		t.Fatalf("replayed call: %v / %v", fc, fco)
	}
	if _, has := reqs[1]["previous_response_id"]; has {
		t.Fatal("previous_response_id sent")
	}

	// update_artifact: same id, next version, new title and content. The
	// earlier turn's call is replayed from the database.
	ev = e.say("now /update it", nil)
	if got := names(ev); got != "user,token,tool_start,tool_run,artifact,tool_end,token,done" {
		t.Fatalf("update events: %s", got)
	}
	updated := as[artifact](t, find(ev, "artifact").data["artifact"])
	if updated.ID != art.ID || updated.Version != 2 || updated.Title != "Demo page (updated)" ||
		!strings.Contains(updated.Content, "Click me again") {
		t.Fatalf("updated artifact: %+v", updated)
	}
	if end := as[part](t, find(ev, "tool_end").data["part"]); end.Name != "update_artifact" || end.Result["version"] != float64(2) {
		t.Fatalf("update tool_end: %+v", end)
	}
	reqs = e.fake.Requests()
	if got := inputTypes(reqs[2]); got != "user,assistant,function_call,function_call_output,assistant,user" {
		t.Fatalf("history replay: %s", got)
	}
	if got := e.chat(); len(got.Artifacts) != 1 || got.Artifacts[0].Version != 2 {
		t.Fatalf("artifacts after update: %+v", got.Artifacts)
	}

	// Deleting the artifact; updating it afterwards is an error the model is told about.
	if code := e.do("DELETE", "/artifacts/"+art.ID, nil, nil); code != 204 {
		t.Fatalf("delete artifact: %d", code)
	}
	var detail struct{ Detail string }
	if code := e.do("DELETE", "/artifacts/"+art.ID, nil, &detail); code != 404 {
		t.Fatalf("second delete: %d", code)
	}
	ev = e.say("/update again", nil)
	end = as[part](t, find(ev, "tool_end").data["part"])
	if end.Status != "error" || end.Result != nil || !strings.Contains(end.Error, "no artifact") || find(ev, "artifact") != nil {
		t.Fatalf("update of a deleted artifact: %+v", end)
	}
	if ev[len(ev)-1].name != "done" {
		t.Fatalf("turn did not finish: %s", names(ev))
	}
}

func TestCodeArtifact(t *testing.T) {
	e := setupTools(t)
	ev := e.say("/code", nil)
	art := as[artifact](t, find(ev, "artifact").data["artifact"])
	if art.Kind != "code" || art.Language != "python" || art.Title != "hello.py" {
		t.Fatalf("code artifact: %+v", art)
	}
}

func TestAskUserAnswered(t *testing.T) {
	e := setupTools(t)
	ev := e.say("/ask me", nil)
	if got := names(ev); got != "user,token,tool_start,tool_run,tool_end,done" {
		t.Fatalf("events: %s", got)
	}
	parts := as[[]part](t, ev[len(ev)-1].data["message"].(map[string]any)["parts"])
	q := parts[len(parts)-1]
	if q.Name != "ask_user" || q.Status != "pending" || q.Result != nil || len(q.Args["options"].([]any)) != 3 {
		t.Fatalf("pending part: %+v", q)
	}
	if n := len(e.fake.Requests()); n != 1 {
		t.Fatalf("the model was asked again before the answer: %d requests", n)
	}
	if got := e.chat(); got.Messages[1].Parts[1].Status != "pending" || got.Messages[1].Stopped {
		t.Fatalf("saved: %+v", got.Messages[1])
	}

	answer := map[string]any{"answer": map[string]any{"callId": q.ID, "text": "Blue"}}
	ev = e.say("", answer)
	if got := names(ev); got != "answered,token,done" {
		t.Fatalf("answer events: %s", got)
	}
	got := e.chat()
	if roles := fmt.Sprint(got.Messages[0].Role, got.Messages[1].Role, got.Messages[2].Role); len(got.Messages) != 3 ||
		roles != "userassistantassistant" {
		t.Fatalf("messages after answer: %+v", got.Messages)
	}
	a := ev[0].data
	closed := as[part](t, a["part"])
	if a["messageId"] != got.Messages[1].ID || closed.ID != q.ID || closed.Status != "done" || closed.Result["answer"] != "Blue" {
		t.Fatalf("answered: %v", a)
	}
	if p := got.Messages[1].Parts[1]; p.Status != "done" || p.Result["answer"] != "Blue" {
		t.Fatalf("saved answer: %+v", p)
	}
	last, _ := e.fake.Last()
	if inputTypes(last) != "user,assistant,function_call,function_call_output" {
		t.Fatalf("continuation input: %s", inputTypes(last))
	}
	if out := last["input"].([]any)[3].(map[string]any)["output"]; out != "The user answered: Blue" {
		t.Fatalf("output sent to the model: %v", out)
	}
	if !strings.Contains(got.Messages[2].Content, "The user answered: Blue") {
		t.Fatalf("continuation: %q", got.Messages[2].Content)
	}

	// Answering twice is refused.
	var detail struct{ Detail string }
	body := map[string]any{"providerId": e.providerID, "model": "fake-small", "answer": answer["answer"]}
	if code := e.do("POST", "/chats/"+e.chatID+"/messages", body, &detail); code != 409 {
		t.Fatalf("second answer: %d %q", code, detail.Detail)
	}
}

func TestAskUserSkipped(t *testing.T) {
	e := setupTools(t)
	e.say("/ask me", nil)
	ev := e.say("never mind, tell me a joke", nil)
	if got := names(ev); got != "user,token,done" {
		t.Fatalf("events: %s", got)
	}
	got := e.chat()
	if p := got.Messages[1].Parts[1]; p.Status != "done" || p.Result == nil || p.Result["answer"] != "" {
		t.Fatalf("skipped part: %+v", p)
	}
	last, _ := e.fake.Last()
	if inputTypes(last) != "user,assistant,function_call,function_call_output,user" {
		t.Fatalf("input: %s", inputTypes(last))
	}
	if out, _ := last["input"].([]any)[3].(map[string]any)["output"].(string); !strings.Contains(out, "did not answer") {
		t.Fatalf("output sent to the model: %q", out)
	}
}

const ddgPage = `<html><body><div class="results">
<div class="result results_links web-result"><h2 class="result__title">
  <a rel="nofollow" class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fgo.dev%2Fdoc%2F&amp;rut=abc">The <b>Go</b> Programming Language</a></h2>
  <a class="result__snippet" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fgo.dev%2Fdoc%2F">Docs for <b>Go</b> &amp; friends.</a></div>
<div class="result result--ad"><a class="result__a" href="https://duckduckgo.com/y.js?ad_domain=x">An advert</a></div>
<div class="result"><a class="result__a" href="https://example.org/plain">Plain link</a>
  <div class="result__snippet">Second snippet</div></div>
</div></body></html>`

// webStub stands in for DuckDuckGo, Brave and an ordinary web page.
func webStub(t *testing.T) (*tools.Web, *httptest.Server) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ddg", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("q") == "" {
			w.WriteHeader(400)
			return
		}
		if strings.Contains(r.FormValue("q"), "blocked") {
			w.WriteHeader(202)
			fmt.Fprint(w, `<html><form id="challenge-form" class="anomaly-modal"></form></html>`)
			return
		}
		fmt.Fprint(w, ddgPage)
	})
	mux.HandleFunc("/brave", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Subscription-Token") != "brave-key" {
			w.WriteHeader(401)
			return
		}
		writeJSON(w, map[string]any{"web": map[string]any{"results": []map[string]any{
			{"title": "From <strong>Brave</strong>", "url": "https://brave.example/1", "description": "q=" + r.URL.Query().Get("q")},
		}}})
	})
	mux.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><head><title>A  page</title><style>p{color:red}</style></head><body>
			<nav>Menu</nav><script>var secret = 1</script><h1>Heading</h1><p>First <b>para</b>graph.</p>`+
			strings.Repeat("<p>Filler text for the body of the page.</p>", 20)+`</body></html>`)
	})
	stub := httptest.NewServer(mux)
	t.Cleanup(stub.Close)
	web := tools.NewWeb()
	web.DuckDuckGoURL, web.BraveURL, web.TavilyURL = stub.URL+"/ddg", stub.URL+"/brave", stub.URL+"/tavily"
	web.AllowPrivate = true
	return web, stub
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func TestWebSearchAndFetch(t *testing.T) {
	web, stub := webStub(t)
	e := setupTools(t, server.WithWeb(web))

	ev := e.say("/search golang docs", nil)
	if got := names(ev); got != "user,token,tool_start,tool_run,tool_end,token,done" {
		t.Fatalf("events: %s", got)
	}
	end := as[part](t, find(ev, "tool_end").data["part"])
	results := as[[]tools.Result](t, end.Result["results"])
	if end.Status != "done" || end.Result["query"] != "golang docs" || len(results) != 2 {
		t.Fatalf("search part: %+v", end)
	}
	if results[0] != (tools.Result{Title: "The Go Programming Language", URL: "https://go.dev/doc/", Snippet: "Docs for Go & friends."}) ||
		results[1].URL != "https://example.org/plain" || results[1].Snippet != "Second snippet" {
		t.Fatalf("results: %+v", results)
	}
	last, _ := e.fake.Last()
	items := last["input"].([]any)
	if out := items[len(items)-1].(map[string]any)["output"].(string); !strings.Contains(out, "1. The Go Programming Language\n   https://go.dev/doc/") {
		t.Fatalf("model-facing results: %q", out)
	}

	// A challenge page is reported as a tool error, and the turn still completes.
	ev = e.say("/search blocked", nil)
	end = as[part](t, find(ev, "tool_end").data["part"])
	if end.Status != "error" || end.Result != nil || !strings.Contains(end.Error, "DuckDuckGo turned the search away") ||
		ev[len(ev)-1].name != "done" {
		t.Fatalf("blocked search: %+v (%s)", end, names(ev))
	}

	// Settings: default, validation, and the key is write-only.
	var st map[string]any
	e.do("GET", "/settings", nil, &st)
	if len(st) != 3 || st["searchProvider"] != "duckduckgo" || st["hasSearchKey"] != false || st["chatDefaults"] != nil {
		t.Fatalf("default settings: %v", st)
	}
	if code := e.do("PATCH", "/settings", map[string]any{"searchProvider": "bing"}, &st); code != 422 {
		t.Fatalf("unknown provider accepted: %d", code)
	}
	e.do("PATCH", "/settings", map[string]any{"searchProvider": "brave"}, &st)
	ev = e.say("/search no key yet", nil)
	if end = as[part](t, find(ev, "tool_end").data["part"]); end.Status != "error" || !strings.Contains(end.Error, "no API key") {
		t.Fatalf("brave without a key: %+v", end)
	}
	st = nil
	if code := e.do("PATCH", "/settings", map[string]any{"searchKey": "brave-key"}, &st); code != 200 ||
		len(st) != 3 || st["searchProvider"] != "brave" || st["hasSearchKey"] != true {
		t.Fatalf("settings after key: %v", st)
	}
	ev = e.say("/search with brave", nil)
	end = as[part](t, find(ev, "tool_end").data["part"])
	results = as[[]tools.Result](t, end.Result["results"])
	if end.Status != "done" || len(results) != 1 || results[0].Title != "From Brave" || results[0].Snippet != "q=with brave" {
		t.Fatalf("brave search: %+v", end)
	}
	st = nil
	e.do("PATCH", "/settings", map[string]any{"searchKey": "", "searchProvider": nil}, &st)
	if st["hasSearchKey"] != false || st["searchProvider"] != "brave" {
		t.Fatalf("settings after clearing the key: %v", st)
	}

	// fetch_page
	ev = e.say("/fetch "+stub.URL+"/page", nil)
	end = as[part](t, find(ev, "tool_end").data["part"])
	if end.Status != "done" || end.Result["title"] != "A page" || end.Result["url"] != stub.URL+"/page" || end.Result["chars"].(float64) < 100 {
		t.Fatalf("fetch part: %+v", end)
	}
	last, _ = e.fake.Last()
	items = last["input"].([]any)
	out := items[len(items)-1].(map[string]any)["output"].(string)
	if !strings.Contains(out, "Heading\nFirst paragraph.") || strings.Contains(out, "secret") || strings.Contains(out, "color:red") ||
		strings.Contains(out, "Menu") {
		t.Fatalf("page text: %q", out)
	}
}

// Without the test hook, the model cannot make the server fetch local addresses.
func TestFetchRefusesLocalAddresses(t *testing.T) {
	_, stub := webStub(t)
	e := setupTools(t) // the real web tools
	ev := e.say("/fetch "+stub.URL+"/page", nil)
	end := as[part](t, find(ev, "tool_end").data["part"])
	if end.Status != "error" || !strings.Contains(end.Error, "private network") {
		t.Fatalf("local fetch: %+v", end)
	}
}

func TestToolGroups(t *testing.T) {
	e := setupTools(t)
	// tools: [] sends no tools and no guidance, and the trigger word is plain text.
	ev := e.say("an /artifact please", map[string]any{"tools": []string{}})
	if got := names(ev); got != "user,token,done" {
		t.Fatalf("events: %s", got)
	}
	last, _ := e.fake.Last()
	if _, has := last["tools"]; has || !strings.HasSuffix(last["instructions"].(string), ".\n\nBe brief.") {
		t.Fatalf("request with tools off: tools=%v instructions=%q", last["tools"], last["instructions"])
	}
	// Only the enabled groups are offered and described.
	e.say("hello", map[string]any{"tools": []string{"web"}})
	last, _ = e.fake.Last()
	offered := last["tools"].([]any)
	instr := last["instructions"].(string)
	if len(offered) != 2 || offered[0].(map[string]any)["name"] != "web_search" || offered[1].(map[string]any)["name"] != "fetch_page" {
		t.Fatalf("offered: %v", offered)
	}
	if !strings.Contains(instr, "web_search") || strings.Contains(instr, "create_artifact") || strings.Contains(instr, "ask_user") {
		t.Fatalf("guidance: %q", instr)
	}
	var detail struct{ Detail string }
	body := map[string]any{"providerId": e.providerID, "model": "m", "content": "hi", "tools": []string{"shell"}}
	if code := e.do("POST", "/chats/"+e.chatID+"/messages", body, &detail); code != 422 {
		t.Fatalf("unknown group: %d", code)
	}

	// A chat that used tools still works with tools off: calls are left out of the replay.
	e.say("/artifact", nil)
	e.say("and now?", map[string]any{"tools": []string{}})
	last, _ = e.fake.Last()
	if got := inputTypes(last); strings.Contains(got, "function_call") {
		t.Fatalf("calls replayed with tools off: %s", got)
	}
}

func TestProviderWithoutToolSupport(t *testing.T) {
	e := setupTools(t)
	ev := e.say("NOTOOLS hello", nil)
	msg, _ := ev[len(ev)-1].data["message"].(string)
	if ev[len(ev)-1].name != "error" || !strings.Contains(msg, "doesn't appear to support tools") ||
		!strings.Contains(msg, "Turn tools off in the composer") {
		t.Fatalf("got %s: %q", ev[len(ev)-1].name, msg)
	}
	ev = e.say("", map[string]any{"regenerate": true, "tools": []string{}})
	if ev[len(ev)-1].name != "done" {
		t.Fatalf("with tools off: %+v", ev[len(ev)-1])
	}
}

func TestRegenerateAndEditDeleteArtifacts(t *testing.T) {
	e := setupTools(t)
	e.say("/svg one", nil)
	keep := e.chat().Artifacts[0]
	e.say("/artifact two", nil)
	before := e.chat()
	if len(before.Artifacts) != 2 {
		t.Fatalf("artifacts: %+v", before.Artifacts)
	}
	// Regenerating the last reply removes its artifact; the new reply makes its own.
	e.say("", map[string]any{"regenerate": true})
	after := e.chat()
	if len(after.Artifacts) != 2 || after.Artifacts[0].ID != keep.ID || after.Artifacts[1].ID == before.Artifacts[1].ID {
		t.Fatalf("after regenerate: %+v", after.Artifacts)
	}
	// Editing the second message into plain text leaves only the first artifact.
	e.say("no tools this time", map[string]any{"editMessageId": after.Messages[2].ID})
	if got := e.chat(); len(got.Artifacts) != 1 || got.Artifacts[0].ID != keep.ID || len(got.Messages) != 4 {
		t.Fatalf("after edit: %d artifacts, %d messages", len(got.Artifacts), len(got.Messages))
	}
	// Editing the first message removes them all.
	e.say("start over", map[string]any{"editMessageId": after.Messages[0].ID})
	if got := e.chat(); len(got.Artifacts) != 0 {
		t.Fatalf("after editing the first message: %+v", got.Artifacts)
	}
}

func TestToolRoundCap(t *testing.T) {
	e := setupTools(t)
	ev := e.say("/loop forever", nil)
	last := ev[len(ev)-1]
	if last.name != "done" {
		t.Fatalf("ended with %s: %v", last.name, last.data)
	}
	// every round, then one request without tools for the final answer
	if n := len(e.fake.Requests()); n != config.MaxToolRounds+1 {
		t.Fatalf("%d requests to the provider, want %d", n, config.MaxToolRounds+1)
	}
	msg := last.data["message"].(map[string]any)
	if !strings.Contains(msg["content"].(string), config.ToolLimitPrompt) || msg["stopped"] != false {
		t.Fatalf("message: %v", msg["content"])
	}
	calls := 0
	for _, p := range as[[]part](t, msg["parts"]) {
		if p.Type == "tool" && p.Status == "done" {
			calls++
		}
	}
	if calls != config.MaxToolRounds || len(e.chat().Artifacts) != config.MaxToolRounds {
		t.Fatalf("%d calls, %d artifacts", calls, len(e.chat().Artifacts))
	}
	// Each request carried every earlier call of the turn.
	if got := inputTypes(e.fake.Requests()[2]); got != "user,assistant,function_call,function_call_output,assistant,function_call,function_call_output" {
		t.Fatalf("third request: %s", got)
	}
}

// A reply that reaches the web gets the larger round limit.
func TestWebToolRoundCap(t *testing.T) {
	web, _ := webStub(t)
	e := setupTools(t, server.WithWeb(web))
	ev := e.say("/webloop forever", nil)
	last := ev[len(ev)-1]
	if last.name != "done" {
		t.Fatalf("ended with %s: %v", last.name, last.data)
	}
	// every round, then one request without tools for the final answer
	if n := len(e.fake.Requests()); n != config.MaxWebSearchToolRounds+1 {
		t.Fatalf("%d requests to the provider, want %d", n, config.MaxWebSearchToolRounds+1)
	}
	msg := last.data["message"].(map[string]any)
	if !strings.Contains(msg["content"].(string), config.ToolLimitPrompt) {
		t.Fatalf("message: %v", msg["content"])
	}
	calls := 0
	for _, p := range as[[]part](t, msg["parts"]) {
		if p.Type == "tool" && p.Status == "done" {
			calls++
		}
	}
	if calls != config.MaxWebSearchToolRounds {
		t.Fatalf("%d calls, want %d", calls, config.MaxWebSearchToolRounds)
	}
}

// Stopping after a tool ran keeps the finished call in the saved reply.
func TestStopMidTurnKeepsFinishedCalls(t *testing.T) {
	e := setupTools(t)
	b, _ := json.Marshal(map[string]any{"providerId": e.providerID, "model": "fake-small", "content": "/artifact but SLOW"})
	res, err := http.Post(e.api.URL+"/api/chats/"+e.chatID+"/messages", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	sc, seenEnd := bufio.NewScanner(res.Body), false
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		if sc.Text() == "event: tool_end" {
			seenEnd = true
		} else if seenEnd && sc.Text() == "event: token" {
			break
		}
	}
	if status := e.do("POST", "/chats/"+e.chatID+"/stop", nil, nil); status != 204 {
		t.Fatalf("stop: %d", status)
	}

	got := e.chat()
	if len(got.Messages) != 2 {
		t.Fatalf("partial reply not saved: %+v", got.Messages)
	}
	m := got.Messages[1]
	if !m.Stopped || len(m.Parts) < 2 || m.Parts[1].Status != "done" || m.Parts[1].Name != "create_artifact" || len(got.Artifacts) != 1 {
		t.Fatalf("stopped reply: %+v", m)
	}
	if strings.Contains(m.Content, "Done.") {
		t.Fatalf("reply was not cut short: %q", m.Content)
	}
}
