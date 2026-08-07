package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"excavate/internal/auth"
	"excavate/internal/store"
)

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	user, token, err := s.auth.Register(r.Context(), body.Email, body.Password)
	switch {
	case err == nil:
		s.setSessionCookie(w, token)
		WriteJSON(w, http.StatusCreated, map[string]any{"user": user})
	case errIs(err, auth.ErrPasswordShort, auth.ErrEmailExists):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		s.logger.Error("register failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
	}
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	user, token, err := s.auth.Login(r.Context(), body.Email, body.Password)
	switch {
	case err == nil:
		s.setSessionCookie(w, token)
		WriteJSON(w, http.StatusOK, map[string]any{"user": user})
	case errIs(err, auth.ErrInvalidCreds):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		s.logger.Error("login failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
	}
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if token := s.sessionToken(r); token != "" {
		_ = s.auth.Logout(r.Context(), token)
	}
	s.clearSessionCookie(w)
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.Session.Name,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(s.cfg.Session.TTL),
		HttpOnly: true,
		Secure:   s.cfg.Env == "production",
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.Session.Name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cfg.Env == "production",
		SameSite: http.SameSiteLaxMode,
	})
}

// sessionToken extracts the token from the session cookie, falling back to
// an Authorization: Bearer header for non-browser clients / curl.
func (s *Server) sessionToken(r *http.Request) string {
	if c, err := r.Cookie(s.cfg.Session.Name); err == nil && c.Value != "" {
		return c.Value
	}
	authz := r.Header.Get("Authorization")
	if strings.HasPrefix(authz, "Bearer ") {
		return strings.TrimPrefix(authz, "Bearer ")
	}
	return ""
}

// errIs reports whether err matches any of the target errors (wrapped or not).
func errIs(err error, targets ...error) bool {
	if err == nil {
		return false
	}
	for _, t := range targets {
		if errors.Is(err, t) {
			return true
		}
	}
	return false
}

// --- auth middleware ---

// RequireAuth gates protected routes behind a valid session.
func (s *Server) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := s.sessionToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		userID, err := s.auth.UserIDByToken(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		user, err := s.users.ByID(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

const ctxKeyUser ctxKey = 100

func userFromContext(ctx context.Context) (store.User, bool) {
	u, ok := ctx.Value(ctxKeyUser).(store.User)
	return u, ok
}
