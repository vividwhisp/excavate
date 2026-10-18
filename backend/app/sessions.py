from dataclasses import dataclass
from datetime import datetime, timedelta
from threading import RLock
from typing import Optional, Tuple
import secrets


@dataclass
class Session:
    user_id: str
    expires_at: datetime


class SessionStore:
    def __init__(self, ttl_hours: int = 168):
        self._ttl = timedelta(hours=ttl_hours)
        self._store: dict[str, Session] = {}
        self._lock = RLock()

    def create(self, user_id: str) -> str:
        token = secrets.token_urlsafe(32)
        with self._lock:
            self._store[token] = Session(user_id=user_id, expires_at=datetime.utcnow() + self._ttl)
        return token

    def get_user(self, token: str) -> Optional[str]:
        if not token:
            return None
        now = datetime.utcnow()
        with self._lock:
            sess = self._store.get(token)
            if not sess:
                return None
            if sess.expires_at < now:
                del self._store[token]
                return None
            # sliding TTL
            sess.expires_at = now + self._ttl
            return sess.user_id

    def delete(self, token: str) -> None:
        with self._lock:
            self._store.pop(token, None)

    def cleanup(self) -> None:
        now = datetime.utcnow()
        with self._lock:
            for k, v in list(self._store.items()):
                if v.expires_at < now:
                    del self._store[k]
