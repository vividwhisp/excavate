from dataclasses import asdict, dataclass, field
from typing import Any, AsyncIterator, Dict, List
import asyncio
import json


@dataclass
class Event:
    type: str  # progress | sources | delta | done | error
    payload: Dict[str, Any] = field(default_factory=dict)


class SSEBroker:
    def __init__(self) -> None:
        self._subs: Dict[str, List[asyncio.Queue[Event | None]]] = {}
        self._buffer: Dict[str, List[Event]] = {}
        self._lock = asyncio.Lock()

    async def publish(self, message_id: str, event: Event) -> None:
        async with self._lock:
            buf = self._buffer.setdefault(message_id, [])
            buf.append(event)
            subs = list(self._subs.get(message_id, []))
        for q in subs:
            await q.put(event)

    async def subscribe(self, message_id: str) -> AsyncIterator[Event]:
        q: asyncio.Queue[Event | None] = asyncio.Queue()
        async with self._lock:
            self._subs.setdefault(message_id, []).append(q)
            replay = list(self._buffer.get(message_id, []))
        try:
            for ev in replay:
                yield ev
            while True:
                ev = await q.get()
                if ev is None:
                    return
                yield ev
                if ev.type == "done" or ev.type == "error":
                    return
        finally:
            async with self._lock:
                subs = self._subs.get(message_id, [])
                if q in subs:
                    subs.remove(q)
                if not subs and message_id in self._subs:
                    # keep buffer small? but terminal event exists
                    pass

    async def close(self, message_id: str) -> None:
        async with self._lock:
            subs = self._subs.get(message_id, [])
            for q in subs:
                await q.put(None)
            self._subs[message_id] = []


def format_frame(ev: Event) -> str:
    data = json.dumps({"type": ev.type, "payload": ev.payload})
    return f"event: message\ndata: {data}\n\n"
