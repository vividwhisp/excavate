from dataclasses import dataclass
import asyncio

from app.sse import Event


@dataclass
class Job:
    thread_id: str
    message_id: str
    query: str


class ResearchQueue:
    def __init__(self) -> None:
        self._q = asyncio.Queue()

    async def enqueue(self, job: Job) -> None:
        await self._q.put(job)

    async def dequeue(self) -> Job:
        return await self._q.get()
