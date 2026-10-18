from typing import Optional

import asyncpg


async def create(conn: asyncpg.Connection, thread_id: str, role: str) -> asyncpg.Record:
    return await conn.fetchrow(
        "INSERT INTO messages (thread_id, role) VALUES ($1, $2) RETURNING id, thread_id, role, content, status, error, created_at",
        thread_id,
        role,
    )


async def get(conn: asyncpg.Connection, message_id: str) -> Optional[asyncpg.Record]:
    return await conn.fetchrow(
        "SELECT id, thread_id, role, content, status, error, created_at FROM messages WHERE id = $1",
        message_id,
    )


async def list_by_thread(conn: asyncpg.Connection, thread_id: str) -> list[asyncpg.Record]:
    return await conn.fetch(
        "SELECT id, thread_id, role, content, status, error, created_at FROM messages WHERE thread_id = $1 ORDER BY created_at",
        thread_id,
    )


async def set_streaming(conn: asyncpg.Connection, message_id: str) -> None:
    await conn.execute("UPDATE messages SET status = 'streaming', error = '' WHERE id = $1", message_id)


async def complete(conn: asyncpg.Connection, message_id: str, content: str) -> None:
    await conn.execute("UPDATE messages SET status = 'complete', content = $2 WHERE id = $1", message_id, content)


async def fail(conn: asyncpg.Connection, message_id: str, error: str) -> None:
    await conn.execute("UPDATE messages SET status = 'error', error = $2 WHERE id = $1", message_id, error)


async def replace_sources(conn: asyncpg.Connection, message_id: str, sources: list) -> None:
    async with conn.transaction():
        await conn.execute("DELETE FROM message_sources WHERE message_id = $1", message_id)
        for s in sources:
            await conn.execute(
                "INSERT INTO message_sources (message_id, url, title, snippet, position) VALUES ($1, $2, $3, $4, $5)",
                message_id,
                s["url"],
                s["title"],
                s["snippet"],
                s["position"],
            )


async def sources_by_message(conn: asyncpg.Connection, message_id: str) -> list[asyncpg.Record]:
    return await conn.fetch(
        "SELECT id, message_id, url, title, snippet, position FROM message_sources WHERE message_id = $1 ORDER BY position",
        message_id,
    )
