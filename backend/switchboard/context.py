"""Decides which messages go into each request so the chat fits the model's context window."""
from __future__ import annotations

REPLY_SHARE = 0.25  # keep a quarter of the window free for the model's answer
PER_MESSAGE_OVERHEAD = 4


def estimate_tokens(text: str) -> int:
    # Rough and model-agnostic: about four characters per token.
    return (len(text or "") + 3) // 4


def build_messages(system_prompt: str, history: list[dict], context_window: int) -> list[dict]:
    """Newest messages first until the budget is used; the system prompt is always kept,
    and the latest message is always sent even if it alone is over budget."""
    budget = int(context_window * (1 - REPLY_SHARE))
    used = estimate_tokens(system_prompt) if system_prompt else 0
    picked: list[dict] = []
    for m in reversed(history):
        if not m.get("content"):
            continue
        cost = estimate_tokens(m["content"]) + PER_MESSAGE_OVERHEAD
        if picked and used + cost > budget:
            break
        picked.append(m)
        used += cost
    picked.reverse()
    # A conversation shouldn't open with an orphaned assistant reply.
    while picked and picked[0]["role"] == "assistant" and len(picked) > 1:
        picked.pop(0)
    out = [{"role": "system", "content": system_prompt}] if system_prompt else []
    out += [{"role": m["role"], "content": m["content"]} for m in picked]
    return out


def make_title(text: str, limit: int = 42) -> str:
    t = " ".join((text or "").split())
    return t if len(t) <= limit else t[:limit].rstrip() + "…"
