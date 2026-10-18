from app.search import base as sb

# match Go shape
__all__ = ["Result"]


class Result:
    def __init__(self, url: str, title: str, snippet: str, position: int):
        self.url = url
        self.title = title
        self.snippet = snippet
        self.position = position
