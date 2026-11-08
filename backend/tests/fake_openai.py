"""A tiny OpenAI-compatible server for tests and offline UI work.
Run: python tests/fake_openai.py 9100   -> base URL http://127.0.0.1:9100/v1
It echoes how many messages it received so tests can check context trimming."""
import asyncio
import json
import sys

import uvicorn
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse, StreamingResponse

app = FastAPI()
LAST = {}


@app.get("/v1/models")
def models(request: Request):
    if request.headers.get("authorization") == "Bearer bad":
        return JSONResponse({"error": {"message": "invalid key"}}, status_code=401)
    return {"object": "list", "data": [{"id": "fake-small", "owned_by": "test"}, {"id": "fake-large", "owned_by": "test"}]}


@app.get("/v1/last")
def last():
    return LAST


@app.post("/v1/chat/completions")
async def chat(request: Request):
    body = await request.json()
    LAST.clear(); LAST.update(body)
    msgs = body["messages"]
    last_user = next((m["content"] for m in reversed(msgs) if m["role"] == "user"), "")
    if "FAIL" in last_user:
        return JSONResponse({"error": {"message": "model exploded"}}, status_code=500)
    slow = "SLOW" in last_user
    text = (f"You said: **{last_user}**. I received {len(msgs)} messages.\n\n"
            "```python\nprint('hello')\n```\n\nDone.")
    words = text.split(" ")

    async def gen():
        for i, w in enumerate(words):
            chunk = {"choices": [{"delta": {"content": w + (" " if i < len(words) - 1 else "")}}]}
            yield f"data: {json.dumps(chunk)}\n\n"
            await asyncio.sleep(0.3 if slow else 0.01)
        yield "data: [DONE]\n\n"
    return StreamingResponse(gen(), media_type="text/event-stream")


if __name__ == "__main__":
    uvicorn.run(app, host="127.0.0.1", port=int(sys.argv[1]) if len(sys.argv) > 1 else 9100, log_level="warning")
