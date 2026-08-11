package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"excavate/internal/store"
)

func (s *Server) handleCreateThread(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())

	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if body.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}

	thread, err := s.threads.Create(r.Context(), user.ID, body.Title)
	if err != nil {
		s.logger.Error("create thread", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"thread": thread})
}

func (s *Server) handleListThreads(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())

	threads, err := s.threads.ListByUser(r.Context(), user.ID)
	if err != nil {
		s.logger.Error("list threads", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"threads": threads})
}

func (s *Server) handleGetThread(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	id := chi.URLParam(r, "id")

	thread, err := s.threads.ByID(r.Context(), id, user.ID)
	if err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "thread_not_found")
			return
		}
		s.logger.Error("get thread", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	messages, err := s.messages.ListByThread(r.Context(), thread.ID)
	if err != nil {
		s.logger.Error("list messages", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	// Attach sources to assistant messages.
	for i := range messages {
		if messages[i].Role == "assistant" {
			sources, err := s.messages.SourcesByMessage(r.Context(), messages[i].ID)
			if err != nil {
				s.logger.Error("list sources", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			messages[i].Sources = sources
		}
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"thread":   thread,
		"messages": messages,
	})
}
