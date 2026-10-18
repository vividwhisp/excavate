from typing import Optional

import asyncpg


async def get_by_email(conn: asyncpg.Connection, email: str) -> Optional[asyncpg.Record]:
    return await conn.fetchrow(
        "SELECT id, email, password_hash, created_at FROM users WHERE email = $1",
        email,
    )


async def get_by_id(conn: asyncpg.Connection, user_id: str) -> Optional[asyncpg.Record]:
    return await conn.fetchrow(
        "SELECT id, email, password_hash, created_at FROM users WHERE id = $1",
        user_id,
    )


async def create(conn: asyncpg.Connection, email: str, password_hash: str) -> asyncpg.Record:
    return await conn.fetchrow(
        "INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, password_hash, created_at",
        email,
        password_hash,
    )
