package store

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// The schema as it was before tools: no parts column, no artifacts, no settings.
const oldSchema = `
CREATE TABLE providers (id TEXT PRIMARY KEY, name TEXT NOT NULL, base_url TEXT NOT NULL, api_key TEXT, created_at INTEGER NOT NULL);
CREATE TABLE chats (
    id TEXT PRIMARY KEY, title TEXT NOT NULL DEFAULT 'New chat',
    provider_id TEXT REFERENCES providers(id) ON DELETE SET NULL, model TEXT,
    system_prompt TEXT NOT NULL DEFAULT '', temperature REAL NOT NULL DEFAULT 0.7,
    context_window INTEGER NOT NULL DEFAULT 8192, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, deleted_at INTEGER);
CREATE TABLE messages (
    seq INTEGER PRIMARY KEY AUTOINCREMENT, id TEXT NOT NULL UNIQUE,
    chat_id TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant')), content TEXT NOT NULL,
    model TEXT, stopped INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL);
INSERT INTO chats (id, title, created_at, updated_at) VALUES ('c1', 'Old chat', 1, 1);
INSERT INTO messages (id, chat_id, role, content, created_at) VALUES ('m1', 'c1', 'user', 'hello', 1);
INSERT INTO messages (id, chat_id, role, content, model, created_at) VALUES ('m2', 'c1', 'assistant', 'hi there', 'gpt', 2);
`

func TestOldDatabaseIsMigrated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	raw, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(oldSchema); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	// Opening twice checks the migration is safe to repeat.
	for i := 0; i < 2; i++ {
		db, err := Open(path)
		if err != nil {
			t.Fatalf("open %d: %v", i, err)
		}
		msgs, err := db.Messages("c1")
		if err != nil || len(msgs) != 2+i || msgs[1].Content != "hi there" || msgs[1].Parts != nil {
			t.Fatalf("open %d: old messages: %+v, %v", i, msgs, err)
		}
		parts := []Part{{Type: PartText, Text: "new"}}
		if m, err := db.AddMessage("c1", "assistant", "new", "m", false, parts); err != nil || len(m.Parts) != 1 {
			t.Fatalf("open %d: adding a message with parts: %v", i, err)
		}
		if _, err := db.CreateArtifact("c1", "t", "code", "go", "x"); err != nil {
			t.Fatalf("open %d: artifacts table: %v", i, err)
		}
		if err := db.SetSetting("k", "v"); err != nil {
			t.Fatalf("open %d: settings table: %v", i, err)
		}
		db.Close()
	}
}

func TestPartsRoundTrip(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	c, _ := db.CreateChat(NewChat{Temperature: 0.7, ContextWindow: 8192})
	parts := []Part{
		{Type: PartText, Text: "Looking."},
		{Type: PartTool, ID: "call_1", Name: "web_search", Args: json.RawMessage(`{"query":"go"}`), Status: StatusDone,
			Result: json.RawMessage(`{"query":"go","results":[]}`), Output: "for the model only"},
		{Type: PartTool, ID: "call_2", Name: "ask_user", Args: json.RawMessage(`{"question":"?"}`), Status: StatusPending},
		{Type: PartTool, ID: "call_3", Name: "fetch_page", Status: StatusError, Error: "nope", Output: "Error: nope"},
	}
	m, err := db.AddMessage(c.ID, "assistant", "Looking.", "m", false, parts)
	if err != nil {
		t.Fatal(err)
	}
	// The database keeps the output; the browser JSON leaves it out.
	if m.Parts[1].Output != "for the model only" || string(m.Parts[1].Result) != `{"query":"go","results":[]}` {
		t.Fatalf("read back: %+v", m.Parts[1])
	}
	b, _ := json.Marshal(m)
	want := `"parts":[{"type":"text","text":"Looking."},` +
		`{"type":"tool","id":"call_1","name":"web_search","args":{"query":"go"},"status":"done","result":{"query":"go","results":[]}},` +
		`{"type":"tool","id":"call_2","name":"ask_user","args":{"question":"?"},"status":"pending","result":null},` +
		`{"type":"tool","id":"call_3","name":"fetch_page","args":{},"status":"error","result":null,"error":"nope"}]`
	if !strings.Contains(string(b), want) {
		t.Fatalf("browser JSON:\n%s", b)
	}
	// Deleting the chat for good takes its artifacts along.
	if _, err := db.CreateArtifact(c.ID, "t", "html", "", "<p>"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.sql.Exec("DELETE FROM chats WHERE id = ?", c.ID); err != nil {
		t.Fatal(err)
	}
	if left, _ := db.Artifacts(c.ID); len(left) != 0 {
		t.Fatalf("artifacts outlived their chat: %+v", left)
	}
}

func TestReplaceTitleKeepsARename(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	c, err := db.CreateChat(NewChat{Title: "provisional"})
	if err != nil {
		t.Fatal(err)
	}
	if changed, err := db.ReplaceTitle(c.ID, "provisional", "summary"); err != nil || !changed {
		t.Fatalf("replace: %v %v", changed, err)
	}
	// The title is no longer the provisional one, as after a rename.
	if changed, err := db.ReplaceTitle(c.ID, "provisional", "late summary"); err != nil || changed {
		t.Fatalf("replaced a title that had changed: %v %v", changed, err)
	}
	if got, _ := db.GetChat(c.ID); got.Title != "summary" {
		t.Fatalf("title: %q", got.Title)
	}
}

func TestDeletingProviderClearsPresets(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	p, err := db.CreateProvider("P", "http://x", "")
	if err != nil {
		t.Fatal(err)
	}
	model := "m"
	preset, err := db.CreatePreset("Mine", ChatConfig{ProviderID: &p.ID, Model: &model, Temperature: 1, ContextWindow: 4096})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteProvider(p.ID); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetPreset(preset.ID)
	if err != nil || got.ProviderID != nil || got.Model == nil || *got.Model != "m" {
		t.Fatalf("preset after provider delete: %+v %v", got, err)
	}
}
