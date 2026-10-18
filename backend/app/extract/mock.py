import asyncio
from typing import Dict

from app.extract.http_extractor import Page


BODIES: Dict[str, str] = {
    "https://example.com/go-intro": "Go is a statically typed, compiled programming language designed at Google. It is known for its simplicity, fast compilation, built-in concurrency and garbage collection.",
    "https://example.com/go-apis": "Building REST APIs in Go with the chi router gives you a composable, idiomatic HTTP layer. Middleware handles logging, auth and recovery, and handlers stay small and testable.",
    "https://example.com/postgres-docs": "PostgreSQL is a powerful, open source object-relational database system with strong reliability, feature robustness, and performance.",
    "https://example.com/redis-docs": "Redis is an in-memory data structure store used as a database, cache and message broker. It supports data structures such as strings, hashes and lists.",
}


class MockExtractor:
    async def extract(self, ctx, url: str) -> Page:
        await asyncio.sleep(0.4)
        body = BODIES.get(url, f"This page ({url}) describes background material relevant to the query.")
        return Page(url=url, title=url, content=body)
