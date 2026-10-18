from typing import Optional

import asyncpg


async def create(conn: asyncpg.Connection, user_id: str, title: str) -> asyncpg.Record:
    return await conn.fetchrow(
        "INSERT INTO threads (user_id, title) VALUES ($1, $2) RETURNING id, user_id, title, created_at, updated_at",
        user_id,
        title,
    )


async def by_id_and_user(conn: asyncpg.Connection, thread_id: str, user_id: str) -> Optional[asyncpg.Record]:
    return await conn.fetchrow(
        "SELECT id, user_id, title, created_at, updated_at FROM threads WHERE id = $1 AND user_id = $2",
        thread_id,
        user_id,
    )


async def list_by_user(conn: asyncpg.Connection, user_id: str) -> list[asyncpg.Record]:
    return await conn.fetch(
        "SELECT id, user_id, title, created_at, updated_at FROM threads WHERE user_id = $1 ORDER BY updated_at DESC",
        user_id,
    )


async def touch(conn: asyncpg.Connection, thread_id: str) -> None:
    await conn.execute("UPDATE threads SET updated_at = now() WHERE id = $1", thread_id)
