import asyncio
from typing import List, AsyncIterator

from app.llm.base import Prompt, Delta


class MockLLM:
    def __init__(self) -> None:
        pass

    def build_answer(self, prompt: Prompt) -> str:
        lines = []
        lines.append(f"## About \"{prompt.query}\"\n\n")
        if prompt.sources:
            s0 = prompt.sources[0]
            lines.append(
                f"Based on the available sources, {prompt.query} is best understood through a few key points. "
                f"According to {s0.title}, {s0.content}\n\n"
            )
        lines.append("Key findings:\n\n")
        for i, s in enumerate(prompt.sources, 1):
            lines.append(f"- {s.title} reports: {s.snippet}[{i}]\n")
        if len(prompt.sources) > 1:
            lines.append(
                f"\nTaken together, sources agree that {prompt.query} is a well-studied topic, "
                f"with {prompt.sources[0].title} and {prompt.sources[1].title} providing the most directly relevant detail. "
                f"Further reading is available in the source list below.\n\n"
            )
        lines.append("## Sources\n\n")
        for i, s in enumerate(prompt.sources, 1):
            lines.append(f"[{i}] [{s.title}]({s.url})\n")
        return "".join(lines)

    async def stream_answer(self, ctx, prompt: Prompt) -> AsyncIterator[Delta]:
        answer = self.build_answer(prompt)
        parts = split_parts(answer)
        for p in parts:
            if ctx is not None:
                try:
                    if asyncio.iscoroutinefunction(ctx.cancelled):
                        pass  # can't check easily; proceed
                except Exception:
                    pass
            await asyncio.sleep(0.06)
            yield Delta(p)


def split_parts(answer: str) -> List[str]:
    parts = []
    cur = []
    for word in answer.split():
        cur.append(word)
        chunk = " ".join(cur) + " "
        if chunk and (word.endswith(".") or word.endswith(":") or len(chunk) > 120):
            parts.append(chunk)
            cur = []
    if cur:
        parts.append(" ".join(cur) + " ")
    return parts
