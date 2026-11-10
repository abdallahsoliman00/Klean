// Package store keeps providers, chats and messages in SQLite.
package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go driver, registers "sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS providers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    base_url    TEXT NOT NULL,
    api_key     TEXT,
    created_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS chats (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL DEFAULT 'New chat',
    provider_id     TEXT REFERENCES providers(id) ON DELETE SET NULL,
    model           TEXT,
    system_prompt   TEXT NOT NULL DEFAULT '',
    temperature     REAL NOT NULL DEFAULT 0.7,
    context_window  INTEGER NOT NULL DEFAULT 8192,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    deleted_at      INTEGER
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
`

// Deleted chats can be restored for a day.
const deletedRetention = 24 * time.Hour

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
}

// ChatWithMessages is a chat as the thread view loads it. "messages" is
// always present, as [] for a new chat: the frontend pushes onto it.
type ChatWithMessages struct {
	*Chat
	Messages []*Message `json:"messages"`
}

type Message struct {
	ID        string  `json:"id"`
	Role      string  `json:"role"`
	Content   string  `json:"content"`
	Model     *string `json:"model"`
	Stopped   bool    `json:"stopped"`
	CreatedAt int64   `json:"createdAt"`
}

// ChatPatch holds the columns to change; nil fields are left alone.
type ChatPatch struct {
	Title         *string
	ProviderID    **string // set to a nil *string to clear
	Model         **string
	SystemPrompt  *string
	Temperature   *float64
	ContextWindow *int
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
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)"
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
	cutoff := NowMs() - deletedRetention.Milliseconds()
	if _, err := sqlDB.Exec("DELETE FROM chats WHERE deleted_at IS NOT NULL AND deleted_at < ?", cutoff); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return &DB{sqlDB}, nil
}

func (d *DB) Close() error { return d.sql.Close() }

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

const chatCols = "id, title, provider_id, model, system_prompt, temperature, context_window, created_at, updated_at"

func scanChat(row interface{ Scan(...any) error }) (*Chat, error) {
	var c Chat
	var pid, model sql.NullString
	if err := row.Scan(&c.ID, &c.Title, &pid, &model, &c.SystemPrompt, &c.Temperature,
		&c.ContextWindow, &c.CreatedAt, &c.UpdatedAt); err != nil {
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
		title = "New chat"
	}
	_, err := d.sql.Exec("INSERT INTO chats ("+chatCols+") VALUES (?,?,?,?,?,?,?,?,?)",
		id, title, n.ProviderID, n.Model, n.SystemPrompt, n.Temperature, n.ContextWindow, ts, ts)
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

const messageCols = "id, role, content, model, stopped, created_at"

func scanMessage(row interface{ Scan(...any) error }) (*Message, error) {
	var m Message
	var model sql.NullString
	if err := row.Scan(&m.ID, &m.Role, &m.Content, &model, &m.Stopped, &m.CreatedAt); err != nil {
		return nil, err
	}
	if model.Valid {
		m.Model = &model.String
	}
	return &m, nil
}

func (d *DB) AddMessage(chatID, role, content, model string, stopped bool) (*Message, error) {
	id, ts := newID(), NowMs()
	tx, err := d.sql.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("INSERT INTO messages (id, chat_id, role, content, model, stopped, created_at) VALUES (?,?,?,?,?,?,?)",
		id, chatID, role, content, nullable(model), stopped, ts); err != nil {
		return nil, err
	}
	if _, err := tx.Exec("UPDATE chats SET updated_at = ? WHERE id = ?", ts, chatID); err != nil {
		return nil, err
	}
	m, err := scanMessage(tx.QueryRow("SELECT "+messageCols+" FROM messages WHERE id = ?", id))
	if err != nil {
		return nil, err
	}
	return m, tx.Commit()
}

func (d *DB) Messages(chatID string) ([]*Message, error) {
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
		out = append(out, m)
	}
	return out, rows.Err()
}

// TruncateFrom deletes a message and everything after it (used when editing).
// It reports whether the message was found.
func (d *DB) TruncateFrom(chatID, messageID string) (bool, error) {
	var seq int64
	err := d.sql.QueryRow("SELECT seq FROM messages WHERE id = ? AND chat_id = ?", messageID, chatID).Scan(&seq)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, err = d.sql.Exec("DELETE FROM messages WHERE chat_id = ? AND seq >= ?", chatID, seq)
	return err == nil, err
}

// DropTrailingAssistant removes assistant replies at the end of a chat (used when regenerating).
func (d *DB) DropTrailingAssistant(chatID string) error {
	_, err := d.sql.Exec(`DELETE FROM messages WHERE chat_id = ? AND role = 'assistant' AND seq > COALESCE(
		(SELECT MAX(seq) FROM messages WHERE chat_id = ? AND role = 'user'), 0)`, chatID, chatID)
	return err
}
