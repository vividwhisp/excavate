from typing import Optional
from bs4 import BeautifulSoup
import httpx
import re


class Page:
    def __init__(self, url: str, title: str, content: str):
        self.url = url
        self.title = title
        self.content = content


def clean_html_to_text(html: str) -> str:
    soup = BeautifulSoup(html, "lxml")
    for tag in soup(["script", "style", "noscript", "header", "footer", "nav", "aside"]):
        tag.decompose()
    text = soup.get_text("\n")
    text = re.sub(r"[ \t\r\f\v]+", " ", text)
    text = re.sub(r"\n{3,}", "\n\n", text)
    return text.strip()


def extract_title(html: str) -> str:
    m = re.search(r"(?is)<title[^>]*>(.*?)</title>", html)
    if m:
        try:
            soup_t = BeautifulSoup(m.group(1), "html.parser")
            return soup_t.get_text(" ", strip=True)
        except Exception:
            return m.group(1).strip()
    return ""


class HTTPExtractor:
    def __init__(self) -> None:
        pass

    async def extract(self, ctx, url: str) -> Page:
        headers = {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) ExcavateResearch/0.1",
            "Accept": "text/html,application/xhtml+xml",
        }
        async with httpx.AsyncClient(timeout=15, follow_redirects=True) as client:
            resp = await client.get(url, headers=headers)
            if resp.status_code != 200:
                raise Exception(f"fetch {url}: status {resp.status_code}")
            html = resp.text[:2 * 1024 * 1024]
        title = extract_title(html) or url
        content = clean_html_to_text(html)
        return Page(url=url, title=title, content=content)
