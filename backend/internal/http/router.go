package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"excavate/internal/auth"
	"excavate/internal/cache"
	"excavate/internal/config"
	"excavate/internal/extract"
	"excavate/internal/llm"
	"excavate/internal/research"
	"excavate/internal/search"
	"excavate/internal/store"
)

// Server wires all HTTP dependencies and builds the route tree.
type Server struct {
	cfg      config.Config
	logger   *slog.Logger
	store    *store.Store
	cache    *cache.Cache
	rdb      *redis.Client
	auth     *auth.Service
	users    *store.UserRepo
	threads  *store.ThreadRepo
	messages *store.MessageRepo
	sessions *cache.SessionStore
	pipeline *research.Pipeline
	handler  http.Handler
}

func NewServer(cfg config.Config, logger *slog.Logger, st *store.Store, c *cache.Cache) *Server {
	sessions := cache.NewSessionStore(c.Client(), cfg.Session.TTL)
	users := store.NewUserRepo(st.Pool)
	threads := store.NewThreadRepo(st.Pool)
	messages := store.NewMessageRepo(st.Pool)

	pipeline := research.NewPipeline(
		providerSearch(cfg),
		providerExtract(cfg),
		providerLLM(cfg),
		messages,
		threads,
		c.Client(),
		cfg.App,
		logger,
	)

	s := &Server{
		cfg:      cfg,
		logger:   logger,
		store:    st,
		cache:    c,
		rdb:      c.Client(),
		auth:     auth.NewService(users, sessions),
		users:    users,
		threads:  threads,
		messages: messages,
		sessions: sessions,
		pipeline: pipeline,
	}
	s.handler = s.buildRouter()
	return s
}

// StartWorkers launches the research pipeline workers in the background.
func (s *Server) StartWorkers(ctx context.Context) {
	s.pipeline.Worker(ctx, 1)
}

func (s *Server) Handler() http.Handler { return s.handler }

// providerSearch returns the real Tavily provider when configured, else the
// deterministic mock used by the automated test suites.
func providerSearch(cfg config.Config) search.Provider {
	if researchProvidersEnabled(cfg) {
		return search.NewTavily(cfg.Tavily.APIKey)
	}
	return search.NewMock()
}

func providerExtract(cfg config.Config) extract.Extractor {
	if researchProvidersEnabled(cfg) {
		return extract.NewHTTP()
	}
	return extract.NewMock()
}

func providerLLM(cfg config.Config) llm.Provider {
	if researchProvidersEnabled(cfg) {
		return llm.NewGemini(cfg.Gemini.APIKey, cfg.Gemini.Model)
	}
	return llm.NewMock()
}

// researchProvidersEnabled decides between the real (Tavily + Gemini + HTTP)
// providers and the mocks. RESEARCH_MODE=mock forces mocks for deterministic
// automated tests; RESEARCH_MODE=real forces real providers; the default
// "auto" uses real providers when both API keys are present.
func researchProvidersEnabled(cfg config.Config) bool {
	switch cfg.ResearchMode {
	case "mock":
		return false
	case "real":
		return true
	}
	return cfg.Tavily.APIKey != "" && cfg.Gemini.APIKey != ""
}

func (s *Server) buildRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(RequestID)                     // our own, sets header
	r.Use(Logger(s.logger))
	r.Use(Recover(s.logger))
	r.Use(s.cors())

	r.Route("/api", func(r chi.Router) {
		// Liveness/readiness.
		r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
		r.Get("/readyz", s.readyz)

		// Auth (public).
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", s.handleRegister)
			r.Post("/login", s.handleLogin)
		})

		// Authenticated routes.
		r.Group(func(r chi.Router) {
			r.Use(s.RequireAuth)
			r.Post("/auth/logout", s.handleLogout)
			r.Get("/me", s.handleMe)

			r.Get("/threads", s.handleListThreads)
			r.Post("/threads", s.handleCreateThread)
			r.Get("/threads/{id}", s.handleGetThread)

			r.Post("/messages", s.handlePostMessage)
			r.Get("/research/stream", s.handleStream)
		})
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