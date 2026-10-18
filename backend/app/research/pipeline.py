import asyncio

from app.sse import Event
from app.llm.base import SourceContext, Prompt
from app.store import messages as mstore, threads as tstore


class Pipeline:
    def __init__(self, search, extract, llm, db_pool, settings, broker):
        self.search = search
        self.extract = extract
        self.llm = llm
        self.db_pool = db_pool
        self.settings = settings
        self.broker = broker

    async def process(self, job):
        async with self.db_pool.acquire() as conn:
            await mstore.set_streaming(conn, job.message_id)
            await tstore.touch(conn, job.thread_id)

        # SEARCH
        await self.broker.publish(job.message_id, Event(type="progress", payload={"stage": "searching"}))
        try:
            results = await self.search.search(None, job.query)
        except Exception as e:
            await self._fail(job, f"search: {e}")
            return
        results = results[: self.settings.max_search_results]
        sources_db = []
        for r in results:
            sources_db.append({"url": r.url, "title": r.title, "snippet": r.snippet, "position": r.position})
        await self.broker.publish(
            job.message_id,
            Event(type="sources", payload={"sources": [s for s in sources_db]}),
        )

        # EXTRACT
        await self.broker.publish(job.message_id, Event(type="progress", payload={"stage": "extracting"}))
        n = min(self.settings.max_extract_pages, len(results))
        llm_sources: list[SourceContext] = []
        for i in range(n):
            try:
                page = await self.extract.extract(None, results[i].url)
            except Exception as e:
                continue
            per_page = max(1, self.settings.max_context_chars // n if n else 0)
            c = page.content[:per_page]
            llm_sources.append(
                SourceContext(
                    url=page.url,
                    title=results[i].title if results[i].title else page.title,
                    snippet=results[i].snippet,
                    content=c,
                    position=results[i].position,
                )
            )

        # REASON
        await self.broker.publish(job.message_id, Event(type="progress", payload={"stage": "reasoning"}))
        content_full = []
        try:
            async for d in self.llm.stream_answer(None, Prompt(job.query, llm_sources)):
                content_full.append(d.text)
                await self.broker.publish(job.message_id, Event(type="delta", payload={"text": d.text}))
        except Exception as e:
            await self._fail(job, f"answer: {e}")
            return
        full_text = "".join(content_full)

        # PERSIST
        try:
            async with self.db_pool.acquire() as conn:
                await mstore.complete(conn, job.message_id, full_text)
                await mstore.replace_sources(conn, job.message_id, sources_db)
        except Exception as e:
            await self._fail(job, f"persist: {e}")
            return
        await self.broker.publish(job.message_id, Event(type="progress", payload={"stage": "done"}))
        await self.broker.publish(job.message_id, Event(type="done", payload={}))

    async def _fail(self, job, msg: str):
        try:
            async with self.db_pool.acquire() as conn:
                await mstore.fail(conn, job.message_id, msg)
        except Exception:
            pass
        await self.broker.publish(job.message_id, Event(type="error", payload={"message": msg}))



