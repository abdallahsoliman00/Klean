"""Talks to any OpenAI-compatible server: GET {base}/models and POST {base}/chat/completions."""
from __future__ import annotations

import json
from typing import AsyncIterator

import httpx

TIMEOUT = httpx.Timeout(connect=10.0, read=300.0, write=30.0, pool=10.0)


class UpstreamError(Exception):
    pass


def _headers(api_key: str | None) -> dict:
    h = {"Accept": "application/json"}
    if api_key:
        h["Authorization"] = f"Bearer {api_key}"
    return h


def _url(base: str, path: str) -> str:
    return base.rstrip("/") + path


def _explain(status: int, body: bytes, url: str) -> str:
    detail = ""
    try:
        data = json.loads(body)
        err = data.get("error", data) if isinstance(data, dict) else data
        detail = err.get("message") if isinstance(err, dict) else str(err)
    except Exception:
        detail = body.decode("utf-8", "replace")[:300].strip()
    if status == 401:
        return "The provider rejected the API key (401). Check the key in Providers & settings."
    if status == 404:
        return f"{url} returned 404. Check that the base URL ends where the API starts (often /v1)."
    return f"The provider returned {status}" + (f": {detail}" if detail else ".")


def _unreachable(url: str) -> str:
    return f"Couldn't reach {url}. Check that the server is running and the URL is right."


async def list_models(base_url: str, api_key: str | None) -> list[dict]:
    url = _url(base_url, "/models")
    try:
        async with httpx.AsyncClient(timeout=httpx.Timeout(15.0)) as client:
            res = await client.get(url, headers=_headers(api_key))
    except httpx.HTTPError:
        raise UpstreamError(_unreachable(url))
    if res.status_code >= 400:
        raise UpstreamError(_explain(res.status_code, res.content, url))
    try:
        data = res.json()
    except ValueError:
        raise UpstreamError(f"{url} didn't return JSON. Is this an OpenAI-compatible endpoint?")
    items = (data.get("data") or data.get("models") or []) if isinstance(data, dict) else data
    models = []
    for m in items:
        if isinstance(m, str):
            models.append({"id": m, "ownedBy": ""})
        elif isinstance(m, dict) and (m.get("id") or m.get("name")):
            models.append({"id": m.get("id") or m.get("name"), "ownedBy": m.get("owned_by") or ""})
    return sorted(models, key=lambda m: m["id"].lower())


async def stream_chat(base_url: str, api_key: str | None, model: str, messages: list[dict],
                      temperature: float | None) -> AsyncIterator[str]:
    """Yields text deltas as they arrive."""
    url = _url(base_url, "/chat/completions")
    payload = {"model": model, "messages": messages, "stream": True}
    if temperature is not None:
        payload["temperature"] = temperature
    headers = {**_headers(api_key), "Accept": "text/event-stream", "Content-Type": "application/json"}
    try:
        async with httpx.AsyncClient(timeout=TIMEOUT) as client:
            async with client.stream("POST", url, json=payload, headers=headers) as res:
                if res.status_code >= 400:
                    raise UpstreamError(_explain(res.status_code, await res.aread(), url))
                ctype = res.headers.get("content-type", "")
                if "text/event-stream" not in ctype and "json" in ctype:
                    # Server ignored stream=true and sent one JSON response.
                    data = json.loads(await res.aread())
                    text = (data.get("choices") or [{}])[0].get("message", {}).get("content") or ""
                    if text:
                        yield text
                    return
                async for line in res.aiter_lines():
                    line = line.strip()
                    if not line.startswith("data:"):
                        continue
                    data = line[5:].strip()
                    if data == "[DONE]":
                        return
                    try:
                        chunk = json.loads(data)
                    except ValueError:
                        continue
                    if chunk.get("error"):
                        err = chunk["error"]
                        raise UpstreamError(err.get("message") if isinstance(err, dict) else str(err))
                    for choice in chunk.get("choices") or []:
                        delta = (choice.get("delta") or {}).get("content")
                        if delta:
                            yield delta
    except httpx.TimeoutException:
        raise UpstreamError("The provider stopped responding (timed out).")
    except httpx.HTTPError:
        raise UpstreamError(_unreachable(url))
