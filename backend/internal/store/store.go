// Package store keeps providers, chats, messages and their images, artifacts,
// presets, skills and settings in SQLite.
package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"klean/backend/internal/config"

	_ "modernc.org/sqlite" // pure-Go driver, registers "sqlite"
)

var schema = fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS providers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    base_url    TEXT NOT NULL,
    api_key     TEXT,
    created_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS chats (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL DEFAULT '%[1]s',
    provider_id     TEXT REFERENCES providers(id) ON DELETE SET NULL,
    model           TEXT,
    system_prompt   TEXT NOT NULL DEFAULT '',
    temperature     REAL NOT NULL DEFAULT %[2]v,
    context_window  INTEGER NOT NULL DEFAULT %[3]d,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    deleted_at      INTEGER,
    favorite        INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS chats_updated ON chats(updated_at DESC);

CREATE TABLE IF NOT EXISTS messages (
    seq         INTEGER PRIMARY KEY AUTOINCREMENT,
    id          TEXT NOT NULL UNIQUE,
    chat_id     TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    role        TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    content     TEXT NOT NULL,
    model       TEXT,
    stopped     INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS messages_chat ON messages(chat_id, seq);

-- The pictures attached to a user message, in the order they were attached.
CREATE TABLE IF NOT EXISTS images (
    id          TEXT PRIMARY KEY,
    message_id  TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    media_type  TEXT NOT NULL,
    data        BLOB NOT NULL,
    position    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS images_message ON images(message_id, position);

-- after_seq is the newest message of the chat when the artifact was made, so
-- editing or regenerating can remove the artifacts of the replies it removes.
CREATE TABLE IF NOT EXISTS artifacts (
    id          TEXT PRIMARY KEY,
    chat_id     TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    kind        TEXT NOT NULL,
    language    TEXT NOT NULL DEFAULT '',
    content     TEXT NOT NULL,
    version     INTEGER NOT NULL DEFAULT 1,
    after_seq   INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS artifacts_chat ON artifacts(chat_id, created_at);

-- Every version an artifact has had, the current one included.
CREATE TABLE IF NOT EXISTS artifact_versions (
    artifact_id  TEXT NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    version      INTEGER NOT NULL,
    title        TEXT NOT NULL,
    content      TEXT NOT NULL,
    created_at   INTEGER NOT NULL,
    PRIMARY KEY (artifact_id, version)
);

CREATE TABLE IF NOT EXISTS settings (
    key    TEXT PRIMARY KEY,
    value  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS presets (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    provider_id     TEXT REFERENCES providers(id) ON DELETE SET NULL,
    model           TEXT,
    system_prompt   TEXT NOT NULL DEFAULT '',
    temperature     REAL NOT NULL DEFAULT %[2]v,
    context_window  INTEGER NOT NULL DEFAULT %[3]d,
    created_at      INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS skills (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL UNIQUE COLLATE NOCASE,
    description   TEXT NOT NULL,
    instructions  TEXT NOT NULL,
    enabled       INTEGER NOT NULL DEFAULT 1,
    created_at    INTEGER NOT NULL
);
`, config.DefaultChatTitle, config.DefaultTemperature, config.DefaultContextWindow)

// Columns added after the first release. CREATE TABLE IF NOT EXISTS leaves an
// existing table as it is, so these are added to older databases on open.
var addedColumns = []struct{ table, column, decl string }{
	{"messages", "parts", "TEXT"},
	{"chats", "favorite", "INTEGER NOT NULL DEFAULT 0"},
}

type Provider struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	HasKey  bool   `json:"hasKey"`
	APIKey  string `json:"-"` // never sent to the browser
}

type Chat struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	ProviderID    *string `json:"providerId"`
	Model         *string `json:"model"`
	SystemPrompt  string  `json:"systemPrompt"`
	Temperature   float64 `json:"temperature"`
	ContextWindow int     `json:"contextWindow"`
	CreatedAt     int64   `json:"createdAt"`
	UpdatedAt     int64   `json:"updatedAt"`
	Favorite      bool    `json:"favorite"`
	// Running is whether a reply is being written. It isn't stored: the
	// server, which knows, fills it in.
	Running bool `json:"running"`
}

// ChatWithMessages is a chat as the thread view loads it. "messages" is
// always present, as [] for a new chat: the frontend pushes onto it.
type ChatWithMessages struct {
	*Chat
	Messages []*Message `json:"messages"`
}

// Message.Content is always the plain text of the message. Parts is set on
// assistant replies and holds the text and tool calls in the order produced;
// it is absent for user messages and for rows written before tools existed.
// Images is set on user messages that have pictures attached.
type Message struct {
	ID        string  `json:"id"`
	Role      string  `json:"role"`
	Content   string  `json:"content"`
	Model     *string `json:"model"`
	Stopped   bool    `json:"stopped"`
	CreatedAt int64   `json:"createdAt"`
	Parts     []Part  `json:"parts,omitempty"`
	Images    []Image `json:"images,omitempty"`
}

const (
	PartText = "text"
	PartTool = "tool"

	StatusPending = "pending" // an ask_user call waiting for the answer
	StatusDone    = "done"
	StatusError   = "error"
)

// Part is one piece of an assistant reply: a run of text or a tool call.
type Part struct {
	Type string
	Text string // text parts

	// tool parts
	ID     string          // the call id, echoed back to the model with the output
	Name   string          // tool name
	Args   json.RawMessage // the arguments object
	Status string
	Result json.RawMessage // what the UI shows; nil while pending or on error
	Error  string
	// Output is what the model was told the call returned. It is saved so later
	// requests can replay the call, and is not sent to the browser.
	Output string
}

// MarshalJSON is the shape the browser gets.
func (p Part) MarshalJSON() ([]byte, error) {
	if p.Type == PartText {
		return json.Marshal(struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{p.Type, p.Text})
	}
	return json.Marshal(struct {
		Type   string          `json:"type"`
		ID     string          `json:"id"`
		Name   string          `json:"name"`
		Args   json.RawMessage `json:"args"`
		Status string          `json:"status"`
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error,omitempty"`
	}{p.Type, p.ID, p.Name, orRaw(p.Args, "{}"), p.Status, orRaw(p.Result, "null"), p.Error})
}

func orRaw(v json.RawMessage, fallback string) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage(fallback)
	}
	return v
}

// dbPart is how a part is saved: the browser shape plus the model-facing output.
type dbPart struct {
	Type   string          `json:"type"`
	Text   string          `json:"text,omitempty"`
	ID     string          `json:"id,omitempty"`
	Name   string          `json:"name,omitempty"`
	Args   json.RawMessage `json:"args,omitempty"`
	Status string          `json:"status,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
	Output string          `json:"output,omitempty"`
}

// encodeParts returns NULL for a message without parts.
func encodeParts(parts []Part) any {
	if len(parts) == 0 {
		return nil
	}
	out := make([]dbPart, len(parts))
	for i, p := range parts {
		out[i] = dbPart(p)
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func decodeParts(raw sql.NullString) ([]Part, error) {
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	var in []dbPart
	if err := json.Unmarshal([]byte(raw.String), &in); err != nil {
		return nil, fmt.Errorf("decoding message parts: %w", err)
	}
	out := make([]Part, len(in))
	for i, p := range in {
		out[i] = Part(p)
	}
	return out, nil
}

// Artifact is a document, page or file the model wrote for the side panel.
type Artifact struct {
	ID        string `json:"id"`
	ChatID    string `json:"chatId"`
	Title     string `json:"title"`
	Kind      string `json:"kind"`     // html, svg, markdown, code or report
	Language  string `json:"language"` // "" unless kind is code
	Content   string `json:"content"`
	Version   int    `json:"version"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// ArtifactVersion is an artifact as it was at one version.
type ArtifactVersion struct {
	Version   int    `json:"version"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"createdAt"`
}

// LibraryArtifact is an artifact as the library lists it: no content, which
// can be large, and the title of its chat.
type LibraryArtifact struct {
	ID        string `json:"id"`
	ChatID    string `json:"chatId"`
	ChatTitle string `json:"chatTitle"`
	Title     string `json:"title"`
	Kind      string `json:"kind"`
	Language  string `json:"language"`
	Version   int    `json:"version"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// ChatPatch holds the columns to change; nil fields are left alone.
type ChatPatch struct {
	Title         *string
	ProviderID    **string // set to a nil *string to clear
	Model         **string
	SystemPrompt  *string
	Temperature   *float64
	ContextWindow *int
	Favorite      *bool
}

type DB struct{ sql *sql.DB }

func NowMs() int64 { return time.Now().UnixMilli() }

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)",
		path, config.DBBusyTimeout.Milliseconds())
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// One connection: SQLite allows one writer anyway, and this avoids "database is locked".
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(schema); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}
	if err := migrate(sqlDB); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("updating schema: %w", err)
	}
	cutoff := NowMs() - config.DeletedRetention.Milliseconds()
	if _, err := sqlDB.Exec("DELETE FROM chats WHERE deleted_at IS NOT NULL AND deleted_at < ?", cutoff); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return &DB{sqlDB}, nil
}

func (d *DB) Close() error { return d.sql.Close() }

// migrate adds the columns an older database is missing, and gives the
// artifacts made before version history their current version as its start.
func migrate(db *sql.DB) error {
	for _, c := range addedColumns {
		has, err := hasColumn(db, c.table, c.column)
		if err != nil {
			return err
		}
		if has {
			continue
		}
		if _, err := db.Exec("ALTER TABLE " + c.table + " ADD COLUMN " + c.column + " " + c.decl); err != nil {
			return err
		}
	}
	_, err := db.Exec(recordVersions + " WHERE id NOT IN (SELECT artifact_id FROM artifact_versions)")
	return err
}

func hasColumn(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query("SELECT name FROM pragma_table_info(?)", table)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// ---------------- providers ----------------

func scanProvider(row interface{ Scan(...any) error }) (*Provider, error) {
	var p Provider
	var key sql.NullString
	if err := row.Scan(&p.ID, &p.Name, &p.BaseURL, &key); err != nil {
		return nil, err
	}
	p.APIKey, p.HasKey = key.String, key.String != ""
	return &p, nil
}

const providerCols = "id, name, base_url, api_key"

func (d *DB) ListProviders() ([]*Provider, error) {
	rows, err := d.sql.Query("SELECT " + providerCols + " FROM providers ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Provider{}
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetProvider returns nil, nil when there is no such provider.
func (d *DB) GetProvider(id string) (*Provider, error) {
	p, err := scanProvider(d.sql.QueryRow("SELECT "+providerCols+" FROM providers WHERE id = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (d *DB) CreateProvider(name, baseURL, apiKey string) (*Provider, error) {
	id := newID()
	_, err := d.sql.Exec("INSERT INTO providers (id, name, base_url, api_key, created_at) VALUES (?,?,?,?,?)",
		id, name, baseURL, nullable(apiKey), NowMs())
	if err != nil {
		return nil, err
	}
	return d.GetProvider(id)
}

// UpdateProvider changes the non-nil fields. An empty apiKey clears the key.
func (d *DB) UpdateProvider(id string, name, baseURL, apiKey *string) (*Provider, error) {
	var sets []string
	var vals []any
	if name != nil {
		sets, vals = append(sets, "name = ?"), append(vals, *name)
	}
	if baseURL != nil {
		sets, vals = append(sets, "base_url = ?"), append(vals, *baseURL)
	}
	if apiKey != nil {
		sets, vals = append(sets, "api_key = ?"), append(vals, nullable(*apiKey))
	}
	if len(sets) > 0 {
		vals = append(vals, id)
		if _, err := d.sql.Exec("UPDATE providers SET "+strings.Join(sets, ", ")+" WHERE id = ?", vals...); err != nil {
			return nil, err
		}
	}
	return d.GetProvider(id)
}

func (d *DB) DeleteProvider(id string) error {
	_, err := d.sql.Exec("DELETE FROM providers WHERE id = ?", id)
	return err
}

// ---------------- chats ----------------

const chatCols = "id, title, provider_id, model, system_prompt, temperature, context_window, created_at, updated_at, favorite"

func scanChat(row interface{ Scan(...any) error }) (*Chat, error) {
	var c Chat
	var pid, model sql.NullString
	if err := row.Scan(&c.ID, &c.Title, &pid, &model, &c.SystemPrompt, &c.Temperature,
		&c.ContextWindow, &c.CreatedAt, &c.UpdatedAt, &c.Favorite); err != nil {
		return nil, err
	}
	if pid.Valid {
		c.ProviderID = &pid.String
	}
	if model.Valid {
		c.Model = &model.String
	}
	return &c, nil
}

func (d *DB) ListChats() ([]*Chat, error) {
	rows, err := d.sql.Query("SELECT " + chatCols + " FROM chats WHERE deleted_at IS NULL ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Chat{}
	for rows.Next() {
		c, err := scanChat(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetChat returns nil, nil for a missing or deleted chat.
func (d *DB) GetChat(id string) (*Chat, error) {
	c, err := scanChat(d.sql.QueryRow("SELECT "+chatCols+" FROM chats WHERE id = ? AND deleted_at IS NULL", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// GetChatWithMessages returns nil, nil for a missing or deleted chat.
func (d *DB) GetChatWithMessages(id string) (*ChatWithMessages, error) {
	c, err := d.GetChat(id)
	if err != nil || c == nil {
		return nil, err
	}
	msgs, err := d.Messages(id) // never nil, so it encodes as []
	if err != nil {
		return nil, err
	}
	return &ChatWithMessages{c, msgs}, nil
}

type NewChat struct {
	ProviderID    *string
	Model         *string
	Title         string
	SystemPrompt  string
	Temperature   float64
	ContextWindow int
}

func (d *DB) CreateChat(n NewChat) (*ChatWithMessages, error) {
	id, ts := newID(), NowMs()
	title := n.Title
	if title == "" {
		title = config.DefaultChatTitle
	}
	_, err := d.sql.Exec("INSERT INTO chats ("+chatCols+") VALUES (?,?,?,?,?,?,?,?,?,?)",
		id, title, n.ProviderID, n.Model, n.SystemPrompt, n.Temperature, n.ContextWindow, ts, ts, false)
	if err != nil {
		return nil, err
	}
	return d.GetChatWithMessages(id)
}

// UpdateChat applies the patch; touch also bumps updated_at.
func (d *DB) UpdateChat(id string, p ChatPatch, touch bool) (*Chat, error) {
	var sets []string
	var vals []any
	add := func(col string, v any) { sets, vals = append(sets, col+" = ?"), append(vals, v) }
	if p.Title != nil {
		add("title", *p.Title)
	}
	if p.ProviderID != nil {
		add("provider_id", *p.ProviderID)
	}
	if p.Model != nil {
		add("model", *p.Model)
	}
	if p.SystemPrompt != nil {
		add("system_prompt", *p.SystemPrompt)
	}
	if p.Temperature != nil {
		add("temperature", *p.Temperature)
	}
	if p.ContextWindow != nil {
		add("context_window", *p.ContextWindow)
	}
	if p.Favorite != nil {
		add("favorite", *p.Favorite)
	}
	if touch {
		add("updated_at", NowMs())
	}
	if len(sets) > 0 {
		vals = append(vals, id)
		if _, err := d.sql.Exec("UPDATE chats SET "+strings.Join(sets, ", ")+" WHERE id = ?", vals...); err != nil {
			return nil, err
		}
	}
	return d.GetChat(id)
}

// ReplaceTitle sets the title only while it is still from, so a title the
// user chose in the meantime is kept. It reports whether the title changed.
func (d *DB) ReplaceTitle(id, from, to string) (bool, error) {
	res, err := d.sql.Exec("UPDATE chats SET title = ? WHERE id = ? AND title = ? AND deleted_at IS NULL", to, id, from)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// DeleteChat soft-deletes; it reports whether a live chat was found.
func (d *DB) DeleteChat(id string) (bool, error) {
	res, err := d.sql.Exec("UPDATE chats SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL", NowMs(), id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (d *DB) RestoreChat(id string) (*Chat, error) {
	if _, err := d.sql.Exec("UPDATE chats SET deleted_at = NULL WHERE id = ?", id); err != nil {
		return nil, err
	}
	return d.GetChat(id)
}

// ---------------- messages ----------------

const messageCols = "id, role, content, model, stopped, created_at, parts"

func scanMessage(row interface{ Scan(...any) error }) (*Message, error) {
	var m Message
	var model, parts sql.NullString
	if err := row.Scan(&m.ID, &m.Role, &m.Content, &model, &m.Stopped, &m.CreatedAt, &parts); err != nil {
		return nil, err
	}
	if model.Valid {
		m.Model = &model.String
	}
	var err error
	m.Parts, err = decodeParts(parts)
	return &m, err
}

// queryer is what *sql.DB and *sql.Tx share.
type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// insertMessage appends a message to the chat and returns its id.
func insertMessage(tx *sql.Tx, chatID, role, content, model string, stopped bool, parts []Part) (string, error) {
	id, ts := newID(), NowMs()
	if _, err := tx.Exec("INSERT INTO messages (id, chat_id, role, content, model, stopped, created_at, parts) VALUES (?,?,?,?,?,?,?,?)",
		id, chatID, role, content, nullable(model), stopped, ts, encodeParts(parts)); err != nil {
		return "", err
	}
	_, err := tx.Exec("UPDATE chats SET updated_at = ? WHERE id = ?", ts, chatID)
	return id, err
}

// getMessage reads one message with its images.
func getMessage(q queryer, id string) (*Message, error) {
	m, err := scanMessage(q.QueryRow("SELECT "+messageCols+" FROM messages WHERE id = ?", id))
	if err != nil {
		return nil, err
	}
	images, err := queryImages(q, "i.message_id = ?", id)
	m.Images = images[id]
	return m, err
}

// AddMessage appends a message. parts is nil for user messages.
func (d *DB) AddMessage(chatID, role, content, model string, stopped bool, parts []Part) (*Message, error) {
	return d.addMessage(chatID, role, content, model, stopped, parts, nil)
}

// AddUserMessage appends a user message with the pictures attached to it.
func (d *DB) AddUserMessage(chatID, content string, images []NewImage) (*Message, error) {
	return d.addMessage(chatID, "user", content, "", false, nil, images)
}

func (d *DB) addMessage(chatID, role, content, model string, stopped bool, parts []Part, images []NewImage) (*Message, error) {
	tx, err := d.sql.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	id, err := insertMessage(tx, chatID, role, content, model, stopped, parts)
	if err != nil {
		return nil, err
	}
	if err := insertImages(tx, id, images); err != nil {
		return nil, err
	}
	m, err := getMessage(tx, id)
	if err != nil {
		return nil, err
	}
	return m, tx.Commit()
}

// Messages lists a chat's messages, oldest first. Never nil, so it encodes as [].
func (d *DB) Messages(chatID string) ([]*Message, error) {
	images, err := queryImages(d.sql, "i.message_id IN (SELECT id FROM messages WHERE chat_id = ?)", chatID)
	if err != nil {
		return nil, err
	}
	rows, err := d.sql.Query("SELECT "+messageCols+" FROM messages WHERE chat_id = ? ORDER BY seq", chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		m.Images = images[m.ID]
		out = append(out, m)
	}
	return out, rows.Err()
}

// SetMessageParts replaces a message's parts (used when a question is answered).
func (d *DB) SetMessageParts(messageID string, parts []Part) error {
	_, err := d.sql.Exec("UPDATE messages SET parts = ? WHERE id = ?", encodeParts(parts), messageID)
	return err
}

// dropOrphanArtifacts deletes the artifacts made after the newest message
// that is left, which are the ones the removed replies created. The message
// exceptID, one just added in place of those removed, doesn't count as left.
func dropOrphanArtifacts(tx *sql.Tx, chatID, exceptID string) error {
	_, err := tx.Exec(`DELETE FROM artifacts WHERE chat_id = ? AND after_seq >= COALESCE(
		(SELECT MAX(seq) FROM messages WHERE chat_id = ? AND id != ?), 0)`, chatID, chatID, exceptID)
	return err
}

// ReplaceFrom is an edit: the message and everything after it are deleted,
// along with the artifacts those messages created, and a user message with
// the new content takes its place, keeping the images of the one it replaces.
// It returns nil, nil when the chat has no such message.
func (d *DB) ReplaceFrom(chatID, messageID, content string) (*Message, error) {
	tx, err := d.sql.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var seq int64
	err = tx.QueryRow("SELECT seq FROM messages WHERE id = ? AND chat_id = ?", messageID, chatID).Scan(&seq)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	id, err := insertMessage(tx, chatID, "user", content, "", false, nil)
	if err != nil {
		return nil, err
	}
	// The images move first: deleting the old message would take them with it.
	if _, err := tx.Exec("UPDATE images SET message_id = ? WHERE message_id = ?", id, messageID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec("DELETE FROM messages WHERE chat_id = ? AND seq >= ? AND id != ?", chatID, seq, id); err != nil {
		return nil, err
	}
	if err := dropOrphanArtifacts(tx, chatID, id); err != nil {
		return nil, err
	}
	m, err := getMessage(tx, id)
	if err != nil {
		return nil, err
	}
	return m, tx.Commit()
}

// DropTrailingAssistant removes assistant replies at the end of a chat (used
// when regenerating), along with the artifacts they created.
func (d *DB) DropTrailingAssistant(chatID string) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM messages WHERE chat_id = ? AND role = 'assistant' AND seq > COALESCE(
		(SELECT MAX(seq) FROM messages WHERE chat_id = ? AND role = 'user'), 0)`, chatID, chatID); err != nil {
		return err
	}
	if err := dropOrphanArtifacts(tx, chatID, ""); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------------- artifacts ----------------

const artifactCols = "id, chat_id, title, kind, language, content, version, created_at, updated_at"

func scanArtifact(row interface{ Scan(...any) error }) (*Artifact, error) {
	var a Artifact
	err := row.Scan(&a.ID, &a.ChatID, &a.Title, &a.Kind, &a.Language, &a.Content, &a.Version, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Artifacts lists a chat's artifacts, oldest first. Never nil, so it encodes as [].
func (d *DB) Artifacts(chatID string) ([]*Artifact, error) {
	rows, err := d.sql.Query("SELECT "+artifactCols+" FROM artifacts WHERE chat_id = ? ORDER BY created_at, rowid", chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Artifact{}
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Artifact returns the artifact, or nil, nil when there is none.
func (d *DB) Artifact(id string) (*Artifact, error) {
	a, err := scanArtifact(d.sql.QueryRow("SELECT "+artifactCols+" FROM artifacts WHERE id = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

// LibraryArtifacts lists the artifacts of every chat that isn't deleted,
// newest change first. Never nil, so it encodes as [].
func (d *DB) LibraryArtifacts() ([]*LibraryArtifact, error) {
	rows, err := d.sql.Query(`SELECT a.id, a.chat_id, c.title, a.title, a.kind, a.language, a.version,
		a.created_at, a.updated_at FROM artifacts a JOIN chats c ON c.id = a.chat_id
		WHERE c.deleted_at IS NULL ORDER BY a.updated_at DESC, a.rowid DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*LibraryArtifact{}
	for rows.Next() {
		var a LibraryArtifact
		if err := rows.Scan(&a.ID, &a.ChatID, &a.ChatTitle, &a.Title, &a.Kind, &a.Language, &a.Version,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

// ArtifactVersions lists every version of the artifact, oldest first. It is
// empty only when there is no such artifact. Never nil, so it encodes as [].
func (d *DB) ArtifactVersions(id string) ([]*ArtifactVersion, error) {
	rows, err := d.sql.Query(`SELECT version, title, content, created_at FROM artifact_versions
		WHERE artifact_id = ? ORDER BY version`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*ArtifactVersion{}
	for rows.Next() {
		var v ArtifactVersion
		if err := rows.Scan(&v.Version, &v.Title, &v.Content, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}

// recordVersions copies the current state of artifacts into their history;
// the caller appends the WHERE that picks which.
const recordVersions = `INSERT INTO artifact_versions (artifact_id, version, title, content, created_at)
	SELECT id, version, title, content, updated_at FROM artifacts`

// writeArtifact runs a statement that creates or changes the artifact and
// records the resulting version with it. It reports whether a row was written.
func (d *DB) writeArtifact(id, query string, args ...any) (bool, error) {
	tx, err := d.sql.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(query, args...)
	if err != nil {
		return false, err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return false, err
	}
	if _, err := tx.Exec(recordVersions+" WHERE id = ?", id); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (d *DB) CreateArtifact(chatID, title, kind, language, content string) (*Artifact, error) {
	id, ts := newID(), NowMs()
	_, err := d.writeArtifact(id, `INSERT INTO artifacts (`+artifactCols+`, after_seq) VALUES (?,?,?,?,?,?,1,?,?,
		COALESCE((SELECT MAX(seq) FROM messages WHERE chat_id = ?), 0))`,
		id, chatID, title, kind, language, content, ts, ts, chatID)
	if err != nil {
		return nil, err
	}
	return d.Artifact(id)
}

// UpdateArtifact replaces the content (and the title, when given) and bumps
// the version; the versions before it stay in the history. It returns nil, nil when the chat has no such artifact.
func (d *DB) UpdateArtifact(chatID, id, content string, title *string) (*Artifact, error) {
	found, err := d.writeArtifact(id, `UPDATE artifacts SET content = ?, title = COALESCE(?, title), version = version + 1,
		updated_at = ? WHERE id = ? AND chat_id = ?`, content, title, NowMs(), id, chatID)
	if err != nil || !found {
		return nil, err
	}
	return d.Artifact(id)
}

// DeleteArtifact reports whether the artifact existed.
func (d *DB) DeleteArtifact(id string) (bool, error) {
	res, err := d.sql.Exec("DELETE FROM artifacts WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ---------------- settings ----------------

// Setting returns "" for a key that was never set.
func (d *DB) Setting(key string) (string, error) {
	var v string
	err := d.sql.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func (d *DB) SetSetting(key, value string) error {
	_, err := d.sql.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}
