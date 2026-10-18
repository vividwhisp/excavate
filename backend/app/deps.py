from typing import Optional

from fastapi import Depends, HTTPException, Request, status

from app.config import get_settings
from app.sessions import SessionStore


def get_session_store(request: Request) -> SessionStore:
    return request.app.state.sessions


def get_current_user(request: Request, store: SessionStore = Depends(get_session_store)) -> Optional[dict]:
    token = None
    cookie_name = get_settings().session_cookie
    if cookie_name in request.cookies:
        token = request.cookies.get(cookie_name)
    if not token:
        auth = request.headers.get("authorization")
        if auth and auth.startswith("Bearer "):
            token = auth[7:]
    if not token:
        return None
    uid = store.get_user(token)
    if not uid:
        return None
    return {"id": str(uid)}


def require_auth(current_user: Optional[dict] = Depends(get_current_user)) -> dict:
    if not current_user:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="unauthorized")
    return current_user  # type: ignore
