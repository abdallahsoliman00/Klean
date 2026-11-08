"""HTTP API. Everything lives under /api; the frontend folder is optionally served at /."""
from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Optional

from fastapi import APIRouter, FastAPI, HTTPException, Request, Response
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel, Field

from . import llm
from .context import build_messages, make_title
from .db import Database


# ---------------- request bodies ----------------
class ProviderIn(BaseModel):
    name: str = Field(min_length=1, max_length=80)
    baseUrl: str = Field(min_length=1)
    apiKey: Optional[str] = None


class ProviderPatch(BaseModel):
    name: Optional[str] = Field(default=None, min_length=1, max_length=80)
    baseUrl: Optional[str] = None
    apiKey: Optional[str] = None  # "" clears the key, omitted keeps it


class ChatIn(BaseModel):
    providerId: Optional[str] = None
    model: Optional[str] = None
    title: Optional[str] = None
    systemPrompt: str = ""
    temperature: float = Field(default=0.7, ge=0, le=2)
    contextWindow: int = Field(default=8192, ge=512, le=2_000_000)


class ChatPatch(BaseModel):
    providerId: Optional[str] = None
    model: Optional[str] = None
    title: Optional[str] = Field(default=None, min_length=1, max_length=200)
    systemPrompt: Optional[str] = None
    temperature: Optional[float] = Field(default=None, ge=0, le=2)
    contextWindow: Optional[int] = Field(default=None, ge=512, le=2_000_000)


class SendIn(BaseModel):
    content: Optional[str] = None
    providerId: str
    model: str
    editMessageId: Optional[str] = None
    regenerate: bool = False


def _clean_url(url: str) -> str:
    url = url.strip().rstrip("/")
    if not url.startswith(("http://", "https://")):
        raise HTTPException(422, "The base URL needs to start with http:// or https://")
    return url


def _sse(event: str, data) -> str:
    return f"event: {event}\ndata: {json.dumps(data, ensure_ascii=False)}\n\n"


def create_app(db_path: str | Path, frontend_dir: str | Path | None = None) -> FastAPI:
    db = Database(db_path)
    app = FastAPI(title="Switchboard", docs_url="/api/docs", openapi_url="/api/openapi.json")
    origins = [o.strip() for o in os.environ.get("SWITCHBOARD_CORS", "*").split(",") if o.strip()]
    app.add_middleware(CORSMiddleware, allow_origins=origins, allow_methods=["*"], allow_headers=["*"])
    api = APIRouter(prefix="/api")

    @api.get("/health")
    def health():
        return {"ok": True}

    # ---------------- providers ----------------
    @api.get("/providers")
    def list_providers():
        return db.list_providers()

    @api.post("/providers", status_code=201)
    def create_provider(body: ProviderIn):
        return db.create_provider(body.name.strip(), _clean_url(body.baseUrl), body.apiKey)

    @api.patch("/providers/{pid}")
    def update_provider(pid: str, body: ProviderPatch):
        if not db.get_provider(pid):
            raise HTTPException(404, "Provider not found.")
        url = _clean_url(body.baseUrl) if body.baseUrl is not None else None
        return db.update_provider(pid, body.name.strip() if body.name else None, url, body.apiKey)

    @api.delete("/providers/{pid}", status_code=204)
    def delete_provider(pid: str):
        db.delete_provider(pid)
        return Response(status_code=204)

    @api.get("/providers/{pid}/models")
    async def provider_models(pid: str):
        p = db.get_provider(pid, with_key=True)
        if not p:
            raise HTTPException(404, "Provider not found.")
        try:
            return await llm.list_models(p["baseUrl"], p["apiKey"])
        except llm.UpstreamError as e:
            raise HTTPException(502, str(e))

    # ---------------- chats ----------------
    @api.get("/chats")
    def list_chats():
        return db.list_chats()

    @api.post("/chats", status_code=201)
    def create_chat(body: ChatIn):
        return db.create_chat(**body.model_dump())

    @api.get("/chats/{cid}")
    def get_chat(cid: str):
        chat = db.get_chat(cid, with_messages=True)
        if not chat:
            raise HTTPException(404, "Chat not found.")
        return chat

    @api.patch("/chats/{cid}")
    def update_chat(cid: str, body: ChatPatch):
        if not db.get_chat(cid):
            raise HTTPException(404, "Chat not found.")
        return db.update_chat(cid, body.model_dump(exclude_unset=True))

    @api.delete("/chats/{cid}", status_code=204)
    def delete_chat(cid: str):
        if not db.delete_chat(cid):
            raise HTTPException(404, "Chat not found.")
        return Response(status_code=204)

    @api.post("/chats/{cid}/restore")
    def restore_chat(cid: str):
        chat = db.restore_chat(cid)
        if not chat:
            raise HTTPException(404, "Chat not found.")
        return chat

    @api.post("/chats/{cid}/messages")
    async def send_message(cid: str, body: SendIn, request: Request):
        """Streams Server-Sent Events: `user` (saved user message), `token` ({delta}),
        then `done` ({message, title}) or `error` ({message})."""
        chat = db.get_chat(cid)
        if not chat:
            raise HTTPException(404, "Chat not found.")
        provider = db.get_provider(body.providerId, with_key=True)
        if not provider:
            raise HTTPException(400, "That provider no longer exists. Pick another model.")
        if not body.regenerate and not (body.content or "").strip():
            raise HTTPException(422, "Message is empty.")

        if body.editMessageId and not db.truncate_from(cid, body.editMessageId):
            raise HTTPException(404, "The message being edited was not found.")
        if body.regenerate:
            db.drop_trailing_assistant(cid)
        user_msg = None if body.regenerate else db.add_message(cid, "user", body.content.strip())

        title = None
        patch = {"providerId": body.providerId, "model": body.model}
        if chat["title"] == "New chat" and user_msg:
            title = patch["title"] = make_title(user_msg["content"])
        chat = db.update_chat(cid, patch, touch=True)

        history = db.messages(cid)
        if not history or history[-1]["role"] != "user":
            raise HTTPException(409, "There's no message to answer.")
        messages = build_messages(chat["systemPrompt"], history, chat["contextWindow"])

        async def events():
            parts: list[str] = []
            outcome = "cancelled"  # becomes done / stopped / error
            stream = llm.stream_chat(provider["baseUrl"], provider["apiKey"], body.model,
                                     messages, chat["temperature"])
            try:
                if user_msg:
                    yield _sse("user", user_msg)
                async for delta in stream:
                    if await request.is_disconnected():  # user pressed stop or closed the tab
                        outcome = "stopped"
                        break
                    parts.append(delta)
                    yield _sse("token", {"delta": delta})
                else:
                    outcome = "done"
                    msg = db.add_message(cid, "assistant", "".join(parts), model=body.model)
                    yield _sse("done", {"message": msg, "title": title})
            except llm.UpstreamError as e:
                outcome = "error"
                yield _sse("error", {"message": str(e)})
            finally:
                await stream.aclose()  # closes the upstream request too
                if outcome != "done" and parts:
                    db.add_message(cid, "assistant", "".join(parts), model=body.model, stopped=True)

        return StreamingResponse(events(), media_type="text/event-stream",
                                 headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"})

    app.include_router(api)

    if frontend_dir and Path(frontend_dir).is_dir():
        app.mount("/", StaticFiles(directory=str(frontend_dir), html=True), name="frontend")

    return app
