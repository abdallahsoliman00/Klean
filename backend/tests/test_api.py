import json
import socket
import subprocess
import sys
import time
from pathlib import Path

import httpx
import pytest
from fastapi.testclient import TestClient

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
from switchboard.app import create_app  # noqa: E402
from switchboard.context import build_messages  # noqa: E402


def free_port():
    s = socket.socket(); s.bind(("127.0.0.1", 0)); p = s.getsockname()[1]; s.close(); return p


@pytest.fixture(scope="module")
def upstream():
    port = free_port()
    proc = subprocess.Popen([sys.executable, str(Path(__file__).parent / "fake_openai.py"), str(port)])
    base = f"http://127.0.0.1:{port}/v1"
    for _ in range(50):
        try:
            httpx.get(base + "/models"); break
        except httpx.HTTPError:
            time.sleep(0.1)
    yield base
    proc.terminate()


@pytest.fixture
def client(tmp_path):
    return TestClient(create_app(tmp_path / "t.db"))


def sse(text):
    out = []
    for block in text.strip().split("\n\n"):
        ev = dict(line.split(": ", 1) for line in block.split("\n"))
        out.append((ev["event"], json.loads(ev["data"])))
    return out


def test_provider_crud_and_models(client, upstream):
    r = client.post("/api/providers", json={"name": "Fake", "baseUrl": upstream + "/", "apiKey": "k"})
    assert r.status_code == 201
    p = r.json()
    assert p["baseUrl"] == upstream and p["hasKey"] is True and "apiKey" not in p
    models = client.get(f"/api/providers/{p['id']}/models").json()
    assert [m["id"] for m in models] == ["fake-large", "fake-small"]
    client.patch(f"/api/providers/{p['id']}", json={"apiKey": "bad"})
    r = client.get(f"/api/providers/{p['id']}/models")
    assert r.status_code == 502 and "API key" in r.json()["detail"]
    assert client.patch(f"/api/providers/{p['id']}", json={"apiKey": ""}).json()["hasKey"] is False
    client.delete(f"/api/providers/{p['id']}")
    assert client.get("/api/providers").json() == []


def test_unreachable_provider(client):
    p = client.post("/api/providers", json={"name": "Off", "baseUrl": "http://127.0.0.1:1/v1"}).json()
    r = client.get(f"/api/providers/{p['id']}/models")
    assert r.status_code == 502 and "Couldn't reach" in r.json()["detail"]
    assert client.post("/api/providers", json={"name": "x", "baseUrl": "ftp://nope"}).status_code == 422


def test_chat_flow(client, upstream):
    p = client.post("/api/providers", json={"name": "Fake", "baseUrl": upstream}).json()
    c = client.post("/api/chats", json={"providerId": p["id"], "model": "fake-small", "systemPrompt": "Be brief."}).json()
    send = lambda **kw: sse(client.post(f"/api/chats/{c['id']}/messages", json={"providerId": p["id"], "model": "fake-small", **kw}).text)

    ev = send(content="hello there")
    assert ev[0][0] == "user" and ev[-1][0] == "done"
    assert "".join(d["delta"] for e, d in ev if e == "token") == ev[-1][1]["message"]["content"]
    assert ev[-1][1]["title"] == "hello there"
    assert "received 2 messages" in ev[-1][1]["message"]["content"]  # system + user

    send(content="second")
    chat = client.get(f"/api/chats/{c['id']}").json()
    assert [m["role"] for m in chat["messages"]] == ["user", "assistant", "user", "assistant"]
    assert chat["title"] == "hello there"

    # regenerate replaces the last reply
    ev = send(regenerate=True)
    assert ev[0][0] == "token"
    assert len(client.get(f"/api/chats/{c['id']}").json()["messages"]) == 4

    # editing the first message drops everything after it
    first = chat["messages"][0]["id"]
    send(content="edited", editMessageId=first)
    msgs = client.get(f"/api/chats/{c['id']}").json()["messages"]
    assert [m["content"] for m in msgs][0] == "edited" and len(msgs) == 2

    # upstream failure becomes an error event, user message kept
    ev = send(content="please FAIL")
    assert ev[-1][0] == "error" and "500" in ev[-1][1]["message"]
    assert client.get(f"/api/chats/{c['id']}").json()["messages"][-1]["role"] == "user"

    # soft delete and restore
    client.delete(f"/api/chats/{c['id']}")
    assert client.get("/api/chats").json() == []
    client.post(f"/api/chats/{c['id']}/restore")
    assert len(client.get("/api/chats").json()) == 1


def test_context_trimming():
    hist = [{"role": "user" if i % 2 == 0 else "assistant", "content": "x" * 400} for i in range(20)]
    out = build_messages("sys", hist, 1024)  # budget 768 tokens, ~104 per message
    assert out[0] == {"role": "system", "content": "sys"}
    assert 1 < len(out) < 21 and out[-1]["content"] == hist[-1]["content"]
    assert out[1]["role"] == "user"
    huge = [{"role": "user", "content": "y" * 100000}]
    assert len(build_messages("", huge, 1024)) == 1  # latest message always sent
