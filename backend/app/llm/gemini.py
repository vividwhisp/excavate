import asyncio
import json
from typing import List, AsyncIterator

import httpx

from app.llm.base import Prompt, Delta, SourceContext


SYSTEM_PROMPT = (
    "You are Excavate, a precise research assistant. Answer the user's question using ONLY the provided sources, "
    "which are listed as [1], [2], etc. Cite each claim with the source number(s) in brackets, like [1] or [2][3]. "
    "If the sources don't answer the question, say so instead of guessing. Write a clear markdown answer with a short "
    "summary up top, then the detailed findings. End with a \"## Sources\" section listing each source as "
    "\"[n] title (url)\"."
)


def truncate(s: str, max_chars: int) -> str:
    if len(s) <= max_chars:
        return s
    return s[:max_chars]


def build_user_prompt(prompt: Prompt) -> str:
    lines = [f"Question: {prompt.query}\n", "Sources:"]
    for i, sc in enumerate(prompt.sources, start=1):
        lines.append(f"[{i}] {sc.title}")
        lines.append(f"    URL: {sc.url}")
        lines.append(f"    {sc.snippet}")
        if sc.content:
            lines.append(truncate(sc.content, 1500))
    return "\n".join(lines)


class Gemini:
    def __init__(self, api_key: str, model: str) -> None:
        self.api_key = api_key
        self.model = model

    async def stream_answer(self, ctx, prompt: Prompt) -> AsyncIterator[Delta]:
        url = f"https://generativelanguage.googleapis.com/v1beta/models/{self.model}:streamGenerateContent?alt=sse&key={self.api_key}"
        body = {
            "systemInstruction": {"parts": [{"text": SYSTEM_PROMPT}]},
            "contents": [{"role": "user", "parts": [{"text": build_user_prompt(prompt)}]}],
            "generationConfig": {"maxOutputTokens": 4096},
        }
        async with httpx.AsyncClient(timeout=120) as client:
            async with client.stream("POST", url, json=body, headers={"Content-Type": "application/json"}) as resp:
                if resp.status_code != 200:
                    text = await resp.aread()
                    raise Exception(f"gemini: status {resp.status_code}")
                async for line in resp.aiter_lines():
                    if not line:
                        continue
                    if not line.startswith("data:"):
                        continue
                    data = line[5:].lstrip()
                    if not data or data == "[DONE]":
                        continue
                    try:
                        j = json.loads(data)
                    except Exception:
                        continue
                    candidates = j.get("candidates") or []
                    for c in candidates:
                        content = c.get("content") or {}
                        parts = content.get("parts") or []
                        for p in parts:
                            t = p.get("text") or ""
                            if not t.strip():
                                continue
                            yield Delta(t)
