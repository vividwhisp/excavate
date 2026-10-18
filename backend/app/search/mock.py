import asyncio

from app.search.search import Result


class MockSearch:
    def __init__(self) -> None:
        pass

    async def search(self, ctx, query: str) -> list[Result]:
        await asyncio.sleep(0.3)
        return [
            Result(url="https://example.com/go-intro", title="Introduction to Go programming", snippet="A beginner's guide to the Go programming language.", position=1),
            Result(url="https://example.com/go-apis", title="Building REST APIs with Go", snippet="How to build production REST APIs using chi.", position=2),
            Result(url="https://example.com/postgres-docs", title="PostgreSQL documentation", snippet="Official PostgreSQL manual and reference.", position=3),
            Result(url="https://example.com/redis-docs", title="Redis in-memory data store", snippet="The open-source, in-memory data structure store.", position=4),
        ]
