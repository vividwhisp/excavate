package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"

	"excavate/internal/cache"
	"excavate/internal/config"
	"excavate/internal/store"
)

// Server wires all HTTP dependencies and builds the route tree.
type Server struct {
	cfg     config.Config
	logger  *slog.Logger
	store   *store.Store
	cache   *cache.Cache
	rdb     *redis.Client
	handler http.Handler

	// Populated in later milestones.
	sessions *cache.SessionStore
}

func NewServer(cfg config.Config, logger *slog.Logger, st *store.Store, c *cache.Cache) *Server {
	s := &Server{
		cfg:     cfg,
		logger:  logger,
		store:   st,
		cache:   c,
		rdb:     c.Client(),
		sessions: cache.NewSessionStore(c.Client(), cfg.Session.TTL),
	}
	s.handler = s.buildRouter()
	return s
}

func (s *Server) Handler() http.Handler { return s.handler }

func (s *Server) buildRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)          // chi's request id
	r.Use(RequestID)                     // our own, sets header
	r.Use(Logger(s.logger))
	r.Use(Recover(s.logger))
	r.Use(s.cors())

	r.Route("/api", func(r chi.Router) {
		// Milestone 1: liveness/readiness.
		r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
		r.Get("/readyz", s.readyz)
	})

	return r
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.store.Pool.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database_unavailable")
		return
	}
	if err := s.rdb.Ping(ctx).Err(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "redis_unavailable")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) cors() func(http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range s.cfg.CORS {
		allowed[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}