import asyncio
from pathlib import Path

import asyncpg


async def create_pool(dsn: str) -> asyncpg.Pool:
    return await asyncpg.create_pool(dsn=dsn, min_size=2, max_size=10)


async def migrate(dsn: str) -> None:
    pool = await asyncpg.create_pool(dsn=dsn, min_size=1, max_size=1)
    try:
        async with pool.acquire() as conn:
            migrations_dir = Path(__file__).parent / "migrations"
            up = (migrations_dir / "000001_init.up.sql").read_text()
            await conn.execute(up)
    finally:
        await pool.close()
