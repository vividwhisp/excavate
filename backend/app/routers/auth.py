import secrets
from datetime import datetime, timedelta

from fastapi import APIRouter, Depends, HTTPException, Request, status
from fastapi.responses import JSONResponse, Response

from app import auth as auth_utils
from app.config import get_settings
from app.deps import get_session_store, require_auth
from app.store import users as ustore

router = APIRouter(prefix="/api", tags=["auth"])


@router.post("/auth/register")
async def register(request: Request) -> Response:
    try:
        data = await request.json()
    except Exception:
        return JSONResponse({"error": "invalid_json"}, status_code=400)
    email = str(data.get("email", "")).lower().strip()
    password = str(data.get("password", ""))
    if len(password) < 8:
        return JSONResponse({"error": "password must be at least 8 characters"}, status_code=422)
    async with request.app.state.pool.acquire() as conn:
        existing = await ustore.get_by_email(conn, email)
        if existing:
            return JSONResponse({"error": "email already registered"}, status_code=422)
        hashed = auth_utils.hash_password(password)
        user = await ustore.create(conn, email, hashed)
    store = get_session_store(request)
    token = store.create(str(user["id"]))
    settings = get_settings()
    resp = JSONResponse(
        {
            "user": {
                "id": str(user["id"]),
                "email": user["email"],
                "createdAt": user["created_at"].isoformat().replace("+00:00", "Z") if hasattr(user["created_at"], "isoformat") else str(user["created_at"]),
            }
        },
        status_code=201,
    )
    expires = datetime.utcnow() + timedelta(hours=settings.session_ttl_hours)
    resp.set_cookie(
        key=settings.session_cookie,
        value=token,
        httponly=True,
        secure=settings.secure_cookies,
        samesite="lax",
        path="/",
        expires=expires,
    )
    return resp


@router.post("/auth/login")
async def login(request: Request) -> Response:
    try:
        data = await request.json()
    except Exception:
        return JSONResponse({"error": "invalid_json"}, status_code=400)
    email = str(data.get("email", "")).lower().strip()
    password = str(data.get("password", ""))
    async with request.app.state.pool.acquire() as conn:
        user = await ustore.get_by_email(conn, email)
    if not user or not auth_utils.verify_password(password, user["password_hash"]):
        return JSONResponse({"error": "invalid email or password"}, status_code=401)
    store = get_session_store(request)
    token = store.create(str(user["id"]))
    settings = get_settings()
    resp = JSONResponse(
        {
            "user": {
                "id": str(user["id"]),
                "email": user["email"],
                "createdAt": user["created_at"].isoformat().replace("+00:00", "Z") if hasattr(user["created_at"], "isoformat") else str(user["created_at"]),
            }
        },
        status_code=200,
    )
    expires = datetime.utcnow() + timedelta(hours=settings.session_ttl_hours)
    resp.set_cookie(
        key=settings.session_cookie,
        value=token,
        httponly=True,
        secure=settings.secure_cookies,
        samesite="lax",
        path="/",
        expires=expires,
    )
    return resp


@router.post("/auth/logout")
async def logout(request: Request, current_user=Depends(require_auth)) -> Response:
    store = get_session_store(request)
    settings = get_settings()
    token = None
    if settings.session_cookie in request.cookies:
        token = request.cookies.get(settings.session_cookie)
    if token:
        store.delete(token)
    resp = JSONResponse({"status": "ok"}, status_code=200)
    resp.delete_cookie(key=settings.session_cookie, path="/")
    return resp


@router.get("/me")
async def me(current_user=Depends(require_auth), request: Request = None) -> Response:  # type: ignore
    async with request.app.state.pool.acquire() as conn:
        user = await ustore.get_by_id(conn, current_user["id"])
    if not user:
        raise HTTPException(status_code=401, detail="unauthorized")
    return JSONResponse(
        {
            "user": {
                "id": str(user["id"]),
                "email": user["email"],
                "createdAt": user["created_at"].isoformat().replace("+00:00", "Z") if hasattr(user["created_at"], "isoformat") else str(user["created_at"]),
            }
        },
        status_code=200,
    )
