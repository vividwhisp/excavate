from typing import Any, Dict, List


class Source:
    def __init__(self, url: str, title: str, snippet: str, position: int):
        self.url = url
        self.title = title
        self.snippet = snippet
        self.position = position


class SourceContext:
    def __init__(self, url: str, title: str, snippet: str, content: str, position: int):
        self.url = url
        self.title = title
        self.snippet = snippet
        self.content = content
        self.position = position


class Prompt:
    def __init__(self, query: str, sources: List[SourceContext]):
        self.query = query
        self.sources = sources


class Delta:
    def __init__(self, text: str):
        self.text = text
