package server_test

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"klean/backend/internal/config"
	"klean/backend/internal/fakeprovider"
)

// The first bytes of each format are all the type check reads.
var (
	pngBytes  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	jpegBytes = []byte("\xff\xd8\xff\xe0\x00\x10JFIF")
)

func imageOf(mediaType string, data []byte) map[string]any {
	return map[string]any{"mediaType": mediaType, "data": base64.StdEncoding.EncodeToString(data)}
}

type imageRef struct{ ID, MediaType string }

// userImages reads the images of the `user` event's message.
func userImages(t *testing.T, ev []event) []imageRef {
	t.Helper()
	return as[struct{ Images []imageRef }](t, find(ev, "user").data).Images
}

// sentImages lists the data URLs of the images in a Responses request, per input item.
func sentImages(req map[string]any) [][]string {
	var out [][]string
	for _, it := range req["input"].([]any) {
		var urls []string
		parts, _ := it.(map[string]any)["content"].([]any)
		for _, p := range parts {
			if m := p.(map[string]any); m["type"] == "input_image" {
				urls = append(urls, m["image_url"].(string))
			}
		}
		out = append(out, urls)
	}
	return out
}

func TestImagesInMessages(t *testing.T) {
	e := setupTools(t)
	png, jpeg := imageOf("image/png", pngBytes), imageOf("image/jpeg", jpegBytes)

	// A message may be images alone. They are saved, listed on the message and sent to the model.
	ev := e.sendAll(e.chatID, map[string]any{"providerId": e.providerID, "model": "fake-small", "images": []any{png, jpeg}})
	images := userImages(t, ev)
	if len(images) != 2 || images[0].MediaType != "image/png" || images[1].MediaType != "image/jpeg" || images[0].ID == "" {
		t.Fatalf("user event images: %+v", images)
	}
	if done := find(ev, "done"); done.data["title"] != config.ImageChatTitle ||
		!strings.Contains(done.data["message"].(map[string]any)["content"].(string), "and 2 images") {
		t.Fatalf("done: %+v", done.data)
	}
	last, _ := e.fake.Last()
	wantPNG := "data:image/png;base64," + png["data"].(string)
	if got := sentImages(last); len(got) != 1 || len(got[0]) != 2 || got[0][0] != wantPNG {
		t.Fatalf("images sent: %v", got)
	}

	// The bytes are served with their type, and may be cached for good.
	res, err := http.Get(e.api.URL + "/api/images/" + images[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !bytes.Equal(body, jpegBytes) || res.Header.Get("Content-Type") != "image/jpeg" ||
		res.Header.Get("X-Content-Type-Options") != "nosniff" || res.Header.Get("Cache-Control") != config.ImageCacheControl {
		t.Fatalf("image: %d %q %v", res.StatusCode, body, res.Header)
	}
	var detail struct{ Detail string }
	if code := e.do("GET", "/images/nope", nil, &detail); code != 404 || detail.Detail == "" {
		t.Fatalf("missing image: %d", code)
	}

	// Later requests still carry the images of earlier turns, and a text
	// message has no images key at all.
	ev = e.say("what about it?", nil)
	if _, has := find(ev, "user").data["images"]; has {
		t.Fatal("images on a message without any")
	}
	last, _ = e.fake.Last()
	if got := sentImages(last); len(got) != 3 || len(got[0]) != 2 || len(got[2]) != 0 {
		t.Fatalf("images sent with the history: %v", got)
	}
	if stored := e.chatImages(); len(stored) != 4 || len(stored[0]) != 2 || stored[0][0] != images[0] {
		t.Fatalf("stored: %+v", stored)
	}

	// Regenerating reuses the message with its images.
	e.say("", map[string]any{"regenerate": true})
	last, _ = e.fake.Last()
	if got := sentImages(last); len(got) != 3 || len(got[0]) != 2 {
		t.Fatalf("images sent when regenerating: %v", got)
	}

	// An edit changes the text only: the images stay, and any in the body are ignored.
	first := e.chat().Messages[0].ID
	ev = e.say("now with words", map[string]any{"editMessageId": first, "images": []any{png}})
	if got := userImages(t, ev); len(got) != 2 || got[0] != images[0] || got[1] != images[1] {
		t.Fatalf("images after edit: %+v", got)
	}
	last, _ = e.fake.Last()
	if got := sentImages(last); len(got) != 1 || len(got[0]) != 2 {
		t.Fatalf("images sent after edit: %v", got)
	}
	// With images the edited text may be empty; without, it may not.
	if ev = e.say("", map[string]any{"editMessageId": e.chat().Messages[0].ID}); len(userImages(t, ev)) != 2 {
		t.Fatalf("edit to no text: %s", names(ev))
	}
	e.say("plain", nil)
	edit := map[string]any{"providerId": e.providerID, "model": "fake-small", "content": " ", "editMessageId": e.chat().Messages[2].ID}
	if code := e.do("POST", "/chats/"+e.chatID+"/messages", edit, &detail); code != 422 {
		t.Fatalf("edit to nothing: %d", code)
	}
}

// chatImages returns the images of each message of the chat.
func (e *toolEnv) chatImages() [][]imageRef {
	e.t.Helper()
	var c struct{ Messages []struct{ Images []imageRef } }
	e.do("GET", "/chats/"+e.chatID, nil, &c)
	var out [][]imageRef
	for _, m := range c.Messages {
		out = append(out, m.Images)
	}
	return out
}

func TestImageValidation(t *testing.T) {
	e := setupTools(t)
	png := imageOf("image/png", pngBytes)
	var many []any
	for range config.MaxImagesPerMessage + 1 {
		many = append(many, png)
	}
	for name, c := range map[string]struct {
		images []any
		want   string
	}{
		"too many":       {many, "at most 8 images"},
		"type not taken": {[]any{imageOf("image/svg+xml", []byte("<svg/>"))}, "Only PNG, JPEG, WebP and GIF"},
		"not base64":     {[]any{map[string]any{"mediaType": "image/png", "data": "data:image/png;base64,AAAA"}}, "Image 1 couldn't be read"},
		"type disagrees": {[]any{png, imageOf("image/png", jpegBytes)}, "Image 2 isn't a valid image/png file"},
		"not an image":   {[]any{imageOf("image/gif", []byte("<script>alert(1)</script>"))}, "isn't a valid image/gif file"},
		"too large":      {[]any{imageOf("image/png", append(pngBytes, make([]byte, config.MaxImageBytes)...))}, "at most 5 MB"},
	} {
		var detail struct{ Detail string }
		body := map[string]any{"providerId": e.providerID, "model": "fake-small", "content": "look", "images": c.images}
		if code := e.do("POST", "/chats/"+e.chatID+"/messages", body, &detail); code != 422 || !strings.Contains(detail.Detail, c.want) {
			t.Errorf("%s: %d %q", name, code, detail.Detail)
		}
	}
	if msgs := e.chat().Messages; len(msgs) != 0 {
		t.Fatalf("a refused message was saved: %+v", msgs)
	}
	// Images of the largest size fit in the body limit, which the ordinary one is too small for.
	largest := imageOf("image/png", append(pngBytes, make([]byte, config.MaxImageBytes-len(pngBytes))...))
	if 3*len(largest["data"].(string)) <= config.MaxRequestBody {
		t.Fatal("this message would fit in an ordinary request body")
	}
	if ev := e.say("big ones", map[string]any{"images": []any{largest, largest, largest}}); len(userImages(t, ev)) != 3 {
		t.Fatalf("large message: %s", names(ev))
	}

	// A model that can't read images says so in plain words.
	ev := e.say("NOIMAGES", map[string]any{"images": []any{png}})
	if last := ev[len(ev)-1]; last.name != "error" || !strings.Contains(last.data["message"].(string), "can't read images") {
		t.Fatalf("images refused by the model: %+v", last)
	}
}

// A provider without the Responses API works the same through Chat
// Completions: text, tools, images and the title.
func TestCompletionsOnlyProvider(t *testing.T) {
	e := setupToolsWith(t, fakeprovider.NewCompletionsOnly())
	ev := e.sendAll(e.chatID, map[string]any{"providerId": e.providerID, "model": "fake-small", "content": "/artifact please",
		"images": []any{imageOf("image/png", pngBytes)}})
	if got := names(ev); got != "user,token,tool_start,tool_run,artifact,tool_end,token,done,title" {
		t.Fatalf("events: %s", got)
	}
	if ev[len(ev)-1].data["title"] != "A short summary" {
		t.Fatalf("title: %+v", ev[len(ev)-1].data)
	}
	c := e.chat()
	parts := c.Messages[1].Parts
	if len(c.Artifacts) != 1 || c.Artifacts[0].Kind != "html" || len(parts) != 3 || parts[1].Name != "create_artifact" ||
		parts[1].Status != "done" || !strings.Contains(parts[2].Text, "The tool returned") {
		t.Fatalf("artifacts %+v, parts %+v", c.Artifacts, parts)
	}
	// The second round replayed the call, with the image still on the user turn.
	last, _ := e.fake.Last()
	msgs := last["messages"].([]any)
	user := msgs[1].(map[string]any)["content"].([]any)
	if len(msgs) != 5 || msgs[0].(map[string]any)["role"] != "system" || user[1].(map[string]any)["type"] != "image_url" ||
		msgs[4].(map[string]any)["role"] != "tool" {
		t.Fatalf("messages: %v", msgs)
	}
	if title := e.fake.TitleRequests(); len(title) != 1 || title[0]["messages"] == nil {
		t.Fatalf("title requests: %v", title)
	}
	// Three requests so far, and a fourth: only the first tried /responses.
	if ev = e.say("thanks", nil); ev[len(ev)-1].name != "done" {
		t.Fatalf("second message: %+v", ev[len(ev)-1])
	}
	if n := e.fake.ResponsesMissed(); n != 1 {
		t.Fatalf("/responses was tried %d times", n)
	}
}

func TestReport(t *testing.T) {
	e := setupTools(t)
	ev := e.say("/report on the quarter", nil)
	if got := names(ev); got != "user,token,tool_start,tool_run,artifact,tool_end,token,done" {
		t.Fatalf("events: %s", got)
	}
	if start := find(ev, "tool_start"); start.data["name"] != "create_report" {
		t.Fatalf("tool_start: %+v", start.data)
	}
	made := as[artifact](t, find(ev, "artifact").data["artifact"])
	end := as[part](t, find(ev, "tool_end").data["part"])
	if made.Kind != "report" || made.Title != "Quarterly report" || made.Version != 1 || !strings.HasPrefix(made.Content, "<!doctype html>") {
		t.Fatalf("artifact: %+v", made)
	}
	if end.Name != "create_report" || end.Status != "done" || keys(end.Result) != "artifactId,kind,language,title,version" ||
		end.Result["artifactId"] != made.ID || end.Result["kind"] != "report" || end.Result["language"] != "" {
		t.Fatalf("part: %+v", end)
	}
	// update_artifact revises a report like any other artifact.
	e.say("/update it", nil)
	if got := e.chat().Artifacts; len(got) != 1 || got[0].Kind != "report" || got[0].Version != 2 {
		t.Fatalf("after update: %+v", got)
	}
	// With the artifacts group off there is no report tool.
	e.say("/report again", map[string]any{"tools": []string{"web"}})
	if got := e.chat().Artifacts; len(got) != 1 {
		t.Fatalf("report made with artifacts off: %+v", got)
	}
}

func TestArtifactLibraryAndVersions(t *testing.T) {
	e := setupTools(t)
	var library []map[string]any
	if e.do("GET", "/artifacts", nil, &library); library == nil || len(library) != 0 {
		t.Fatalf("empty library = %#v, want []", library)
	}
	e.say("/artifact please", nil)
	e.say("/update it", nil)
	id := e.chat().Artifacts[0].ID

	e.do("GET", "/artifacts", nil, &library)
	if len(library) != 1 || keys(library[0]) != "chatId,chatTitle,createdAt,id,kind,language,title,updatedAt,version" ||
		library[0]["id"] != id || library[0]["chatId"] != e.chatID || library[0]["version"] != 2.0 {
		t.Fatalf("library: %+v", library)
	}
	var versions []map[string]any
	e.do("GET", "/artifacts/"+id+"/versions", nil, &versions)
	if len(versions) != 2 || keys(versions[0]) != "content,createdAt,title,version" ||
		versions[0]["version"] != 1.0 || versions[0]["title"] != "Demo page" ||
		versions[1]["version"] != 2.0 || versions[1]["title"] != "Demo page (updated)" ||
		versions[0]["content"] == versions[1]["content"] {
		t.Fatalf("versions: %+v", versions)
	}
	if status := e.do("GET", "/artifacts/missing/versions", nil, nil); status != 404 {
		t.Fatalf("versions of a missing artifact: %d", status)
	}
}

type skill struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
	Enabled      bool   `json:"enabled"`
}

func TestSkillCRUD(t *testing.T) {
	e := setup(t)
	var raw map[string]any
	body := map[string]any{"name": " tone ", "description": " How to write to customers. ", "instructions": "Be warm.\n"}
	if code := e.do("POST", "/skills", body, &raw); code != 201 || keys(raw) != "description,enabled,id,instructions,name" {
		t.Fatalf("create: %d %v", code, raw)
	}
	tone := as[skill](t, raw)
	if tone.Name != "tone" || tone.Description != "How to write to customers." || tone.Instructions != "Be warm.\n" || !tone.Enabled {
		t.Fatalf("created: %+v", tone)
	}
	var report skill
	e.do("POST", "/skills", map[string]any{"name": "a-report", "description": "d", "instructions": "i", "enabled": false}, &report)
	var list []skill
	if e.do("GET", "/skills", nil, &list); len(list) != 2 || list[0].Name != "a-report" || list[0].Enabled || list[1] != tone {
		t.Fatalf("list: %+v", list)
	}

	// A patch changes only what it carries.
	var patched skill
	if code := e.do("PATCH", "/skills/"+tone.ID, map[string]any{"enabled": false}, &patched); code != 200 ||
		patched.Enabled || patched.Name != "tone" || patched.Instructions != "Be warm.\n" {
		t.Fatalf("patch: %d %+v", code, patched)
	}
	// Keeping its own name is not a clash.
	if code := e.do("PATCH", "/skills/"+tone.ID, map[string]any{"name": "tone", "description": "New."}, &patched); code != 200 || patched.Description != "New." {
		t.Fatalf("patch with own name: %d %+v", code, patched)
	}

	long := strings.Repeat("a", config.SkillNameMaxChars+1)
	for name, bad := range map[string]map[string]any{
		"no name":            {"description": "d", "instructions": "i"},
		"blank name":         {"name": "  ", "description": "d", "instructions": "i"},
		"upper case":         {"name": "Tone-2", "description": "d", "instructions": "i"},
		"spaces":             {"name": "my skill", "description": "d", "instructions": "i"},
		"long name":          {"name": long, "description": "d", "instructions": "i"},
		"taken":              {"name": "tone", "description": "d", "instructions": "i"},
		"no description":     {"name": "x", "instructions": "i"},
		"long description":   {"name": "x", "description": strings.Repeat("d", config.SkillDescriptionMaxChars+1), "instructions": "i"},
		"no instructions":    {"name": "x", "description": "d"},
		"blank instructions": {"name": "x", "description": "d", "instructions": " \n"},
		"long instructions":  {"name": "x", "description": "d", "instructions": strings.Repeat("i", config.SkillInstructionsMaxChars+1)},
	} {
		var detail struct{ Detail string }
		if code := e.do("POST", "/skills", bad, &detail); code != 422 || detail.Detail == "" {
			t.Errorf("%s: %d %q", name, code, detail.Detail)
		}
	}
	if code := e.do("PATCH", "/skills/"+report.ID, map[string]any{"name": "tone"}, nil); code != 422 {
		t.Fatalf("renamed onto another skill: %d", code)
	}
	if code := e.do("PATCH", "/skills/"+tone.ID, map[string]any{"instructions": ""}, nil); code != 422 {
		t.Fatalf("emptied instructions: %d", code)
	}

	if code := e.do("DELETE", "/skills/"+tone.ID, nil, nil); code != 204 {
		t.Fatalf("delete: %d", code)
	}
	if code := e.do("DELETE", "/skills/"+tone.ID, nil, nil); code != 404 {
		t.Fatalf("delete again: %d", code)
	}
	if code := e.do("PATCH", "/skills/"+tone.ID, map[string]any{"enabled": true}, nil); code != 404 {
		t.Fatalf("patch a deleted skill: %d", code)
	}
	if e.do("GET", "/skills", nil, &list); len(list) != 1 {
		t.Fatalf("after delete: %+v", list)
	}
}

func TestUseSkill(t *testing.T) {
	e := setupTools(t)
	const secret = "Always sign off with 'Yours, Klean'."
	var tone, off skill
	e.do("POST", "/skills", map[string]any{"name": "tone", "description": "How to write to customers.", "instructions": secret}, &tone)
	e.do("POST", "/skills", map[string]any{"name": "retired", "description": "Old rules.", "instructions": "x", "enabled": false}, &off)

	// With every tool group off, use_skill is still offered, and the
	// enabled skills are listed in the instructions without their content.
	ev := e.say("/skill tone", map[string]any{"tools": []string{}})
	if got := names(ev); got != "user,token,tool_start,tool_run,tool_end,token,done" {
		t.Fatalf("events: %s", got)
	}
	reqs := e.fake.Requests()
	first := reqs[len(reqs)-2]
	offered := first["tools"].([]any)
	instr := first["instructions"].(string)
	if len(offered) != 1 || offered[0].(map[string]any)["name"] != "use_skill" {
		t.Fatalf("offered: %v", offered)
	}
	if !strings.HasPrefix(instr, "Be brief.\n\n") || !strings.Contains(instr, "\n- tone: How to write to customers.") ||
		strings.Contains(instr, "retired") || strings.Contains(instr, secret) || strings.Contains(instr, "create_artifact") {
		t.Fatalf("instructions: %q", instr)
	}

	// The browser learns which skill was loaded; the model gets the instructions.
	end := as[part](t, find(ev, "tool_end").data["part"])
	if end.Name != "use_skill" || end.Status != "done" || end.Args["name"] != "tone" || keys(end.Result) != "name" || end.Result["name"] != "tone" {
		t.Fatalf("part: %+v", end)
	}
	for _, x := range ev {
		if b := as[map[string]any](t, x.data); strings.Contains(strings.Join(texts(b), " "), "Yours, Klean") && x.name != "token" && x.name != "done" {
			t.Fatalf("%s carries the instructions: %v", x.name, x.data)
		}
	}
	items := reqs[len(reqs)-1]["input"].([]any)
	if out := items[len(items)-1].(map[string]any); out["type"] != "function_call_output" || out["output"] != secret {
		t.Fatalf("tool output: %v", out)
	}
	saved := e.chat().Messages[1].Parts[1]
	if saved.Name != "use_skill" || saved.Result["name"] != "tone" || len(saved.Result) != 1 {
		t.Fatalf("saved part: %+v", saved)
	}

	// A disabled or unknown skill is an error the model is told about.
	for _, name := range []string{"retired", "nonsense"} {
		ev = e.say("/skill "+name, nil)
		end = as[part](t, find(ev, "tool_end").data["part"])
		if end.Status != "error" || !strings.Contains(end.Error, "The enabled skills are: tone") || ev[len(ev)-1].name != "done" {
			t.Fatalf("%s: %+v, then %s", name, end, ev[len(ev)-1].name)
		}
	}

	// With no skill enabled the tool and the listing are gone.
	e.do("PATCH", "/skills/"+tone.ID, map[string]any{"enabled": false}, nil)
	e.say("hello", map[string]any{"tools": []string{}})
	last, _ := e.fake.Last()
	if _, has := last["tools"]; has || last["instructions"] != "Be brief." {
		t.Fatalf("without skills: tools=%v instructions=%q", last["tools"], last["instructions"])
	}
}

// texts collects every string inside a decoded JSON value.
func texts(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		var out []string
		for _, item := range x {
			out = append(out, texts(item)...)
		}
		return out
	case map[string]any:
		var out []string
		for _, item := range x {
			out = append(out, texts(item)...)
		}
		return out
	}
	return nil
}
