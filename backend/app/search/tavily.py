import asyncio
import json
from typing import List

import httpx

from app.search.search import Result


class Tavily:
    def __init__(self, api_key: str) -> None:
        self.api_key = api_key

    async def search(self, ctx, query: str) -> List[Result]:
        headers = {"Content-Type": "application/json", "Authorization": f"Bearer {self.api_key}"}
        payload = {"query": query, "max_results": 8, "search_depth": "basic", "include_answer": False}
        async with httpx.AsyncClient(timeout=20) as client:
            resp = await client.post("https://api.tavily.com/search", headers=headers, json=payload)
            if resp.status_code != 200:
                raise Exception(f"tavily search: status {resp.status_code}")
            data = resp.json()
        results = []
        for i, r in enumerate(data.get("results", [])):
            url = r.get("url") or ""
            if not url:
                continue
            title = r.get("title") or ""
            content = r.get("content") or ""
            snippet = content[:400]
            results.append(Result(url=url, title=title, snippet=snippet, position=i + 1))
        return results
