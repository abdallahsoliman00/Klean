"""SQLite storage for providers, chats and messages."""
from __future__ import annotations

import sqlite3
import time
import uuid
from contextlib import contextmanager
from pathlib import Path

SCHEMA = """
PRAGMA journal_mode = WAL;

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
"""

DELETED_RETENTION_MS = 24 * 3600 * 1000  # deleted chats can be restored for a day


def now_ms() -> int:
    return int(time.time() * 1000)


def new_id() -> str:
    return uuid.uuid4().hex[:12]


class Database:
    def __init__(self, path: str | Path):
        self.path = str(path)
        Path(self.path).parent.mkdir(parents=True, exist_ok=True)
        with self.conn() as c:
            c.executescript(SCHEMA)
            c.execute("DELETE FROM chats WHERE deleted_at IS NOT NULL AND deleted_at < ?",
                      (now_ms() - DELETED_RETENTION_MS,))

    @contextmanager
    def conn(self):
        c = sqlite3.connect(self.path, timeout=10)
        c.row_factory = sqlite3.Row
        c.execute("PRAGMA foreign_keys = ON")
        try:
            yield c
            c.commit()
        finally:
            c.close()

    # ---------------- providers ----------------
    @staticmethod
    def _provider(row, with_key=False) -> dict:
        d = {"id": row["id"], "name": row["name"], "baseUrl": row["base_url"], "hasKey": bool(row["api_key"])}
        if with_key:
            d["apiKey"] = row["api_key"] or ""
        return d

    def list_providers(self) -> list[dict]:
        with self.conn() as c:
            rows = c.execute("SELECT * FROM providers ORDER BY created_at").fetchall()
        return [self._provider(r) for r in rows]

    def get_provider(self, pid: str, with_key=False) -> dict | None:
        with self.conn() as c:
            row = c.execute("SELECT * FROM providers WHERE id = ?", (pid,)).fetchone()
        return self._provider(row, with_key) if row else None

    def create_provider(self, name: str, base_url: str, api_key: str | None) -> dict:
        pid = new_id()
        with self.conn() as c:
            c.execute("INSERT INTO providers (id, name, base_url, api_key, created_at) VALUES (?,?,?,?,?)",
                      (pid, name, base_url, api_key or None, now_ms()))
        return self.get_provider(pid)

    def update_provider(self, pid: str, name: str | None, base_url: str | None, api_key: str | None) -> dict | None:
        sets, vals = [], []
        if name is not None:
            sets.append("name = ?"); vals.append(name)
        if base_url is not None:
            sets.append("base_url = ?"); vals.append(base_url)
        if api_key is not None:  # empty string clears the key
            sets.append("api_key = ?"); vals.append(api_key or None)
        if sets:
            with self.conn() as c:
                c.execute(f"UPDATE providers SET {', '.join(sets)} WHERE id = ?", (*vals, pid))
        return self.get_provider(pid)

    def delete_provider(self, pid: str) -> None:
        with self.conn() as c:
            c.execute("DELETE FROM providers WHERE id = ?", (pid,))

    # ---------------- chats ----------------
    @staticmethod
    def _chat(row) -> dict:
        return {
            "id": row["id"], "title": row["title"], "providerId": row["provider_id"], "model": row["model"],
            "systemPrompt": row["system_prompt"], "temperature": row["temperature"],
            "contextWindow": row["context_window"], "createdAt": row["created_at"], "updatedAt": row["updated_at"],
        }

    @staticmethod
    def _message(row) -> dict:
        return {"id": row["id"], "role": row["role"], "content": row["content"], "model": row["model"],
                "stopped": bool(row["stopped"]), "createdAt": row["created_at"]}

    def list_chats(self) -> list[dict]:
        with self.conn() as c:
            rows = c.execute("SELECT * FROM chats WHERE deleted_at IS NULL ORDER BY updated_at DESC").fetchall()
        return [self._chat(r) for r in rows]

    def get_chat(self, cid: str, with_messages=False) -> dict | None:
        with self.conn() as c:
            row = c.execute("SELECT * FROM chats WHERE id = ? AND deleted_at IS NULL", (cid,)).fetchone()
            if not row:
                return None
            chat = self._chat(row)
            if with_messages:
                msgs = c.execute("SELECT * FROM messages WHERE chat_id = ? ORDER BY seq", (cid,)).fetchall()
                chat["messages"] = [self._message(m) for m in msgs]
        return chat

    def create_chat(self, **fields) -> dict:
        cid, ts = new_id(), now_ms()
        with self.conn() as c:
            c.execute(
                "INSERT INTO chats (id, title, provider_id, model, system_prompt, temperature, context_window, created_at, updated_at)"
                " VALUES (?,?,?,?,?,?,?,?,?)",
                (cid, fields.get("title") or "New chat", fields.get("providerId"), fields.get("model"),
                 fields.get("systemPrompt") or "", fields.get("temperature", 0.7),
                 fields.get("contextWindow", 8192), ts, ts))
        return self.get_chat(cid, with_messages=True)

    _CHAT_COLS = {"title": "title", "providerId": "provider_id", "model": "model", "systemPrompt": "system_prompt",
                  "temperature": "temperature", "contextWindow": "context_window"}

    def update_chat(self, cid: str, patch: dict, touch=False) -> dict | None:
        sets, vals = [], []
        for k, col in self._CHAT_COLS.items():
            if k in patch:
                sets.append(f"{col} = ?"); vals.append(patch[k])
        if touch:
            sets.append("updated_at = ?"); vals.append(now_ms())
        if sets:
            with self.conn() as c:
                c.execute(f"UPDATE chats SET {', '.join(sets)} WHERE id = ?", (*vals, cid))
        return self.get_chat(cid)

    def delete_chat(self, cid: str) -> bool:
        with self.conn() as c:
            cur = c.execute("UPDATE chats SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL", (now_ms(), cid))
        return cur.rowcount > 0

    def restore_chat(self, cid: str) -> dict | None:
        with self.conn() as c:
            c.execute("UPDATE chats SET deleted_at = NULL WHERE id = ?", (cid,))
        return self.get_chat(cid)

    # ---------------- messages ----------------
    def add_message(self, cid: str, role: str, content: str, model: str | None = None, stopped=False) -> dict:
        mid, ts = new_id(), now_ms()
        with self.conn() as c:
            c.execute("INSERT INTO messages (id, chat_id, role, content, model, stopped, created_at) VALUES (?,?,?,?,?,?,?)",
                      (mid, cid, role, content, model, int(stopped), ts))
            c.execute("UPDATE chats SET updated_at = ? WHERE id = ?", (ts, cid))
            row = c.execute("SELECT * FROM messages WHERE id = ?", (mid,)).fetchone()
        return self._message(row)

    def messages(self, cid: str) -> list[dict]:
        with self.conn() as c:
            rows = c.execute("SELECT * FROM messages WHERE chat_id = ? ORDER BY seq", (cid,)).fetchall()
        return [self._message(r) for r in rows]

    def truncate_from(self, cid: str, mid: str) -> bool:
        """Delete a message and everything after it (used when editing)."""
        with self.conn() as c:
            row = c.execute("SELECT seq FROM messages WHERE id = ? AND chat_id = ?", (mid, cid)).fetchone()
            if not row:
                return False
            c.execute("DELETE FROM messages WHERE chat_id = ? AND seq >= ?", (cid, row["seq"]))
        return True

    def drop_trailing_assistant(self, cid: str) -> None:
        """Remove assistant replies at the end of a chat (used when regenerating)."""
        with self.conn() as c:
            while True:
                row = c.execute("SELECT seq, role FROM messages WHERE chat_id = ? ORDER BY seq DESC LIMIT 1", (cid,)).fetchone()
                if not row or row["role"] != "assistant":
                    break
                c.execute("DELETE FROM messages WHERE seq = ?", (row["seq"],))
