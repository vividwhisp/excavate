package http

import (
	"encoding/json"
	"net/http"

	"excavate/internal/research"
	"excavate/internal/sse"
	"excavate/internal/store"
)

func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ThreadID string `json:"threadId"`
		Content  string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if body.Content == "" {
		writeError(w, http.StatusUnprocessableEntity, "content is required")
		return
	}

	user, _ := userFromContext(r.Context())

	// Owner check: the thread must belong to the requesting user.
	thread, err := s.threads.ByID(r.Context(), body.ThreadID, user.ID)
	if err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "thread_not_found")
			return
		}
		s.logger.Error("get thread for message", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	// Persist the user's question.
	userMsg, err := s.messages.Create(r.Context(), thread.ID, "user")
	if err != nil {
		s.logger.Error("create user message", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	if err := s.messages.Complete(r.Context(), userMsg.ID, body.Content); err != nil {
		s.logger.Error("persist question", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	// Reserve the assistant message and hand it to the research queue.
	assistantMsg, err := s.messages.Create(r.Context(), thread.ID, "assistant")
	if err != nil {
		s.logger.Error("create assistant message", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	if err := research.Enqueue(r.Context(), s.rdb, research.Job{
		ThreadID:  thread.ID,
		MessageID: assistantMsg.ID,
		Query:     body.Content,
	}); err != nil {
		s.logger.Error("enqueue research job", "error", err)
		_ = s.messages.Fail(r.Context(), assistantMsg.ID, "failed to enqueue")
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]any{"message": assistantMsg})
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	messageID := r.URL.Query().Get("messageID")
	if messageID == "" {
		writeError(w, http.StatusBadRequest, "messageID is required")
		return
	}

	// Owner check via message → thread → user.
	msg, err := s.messages.Get(r.Context(), messageID)
	if err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "message_not_found")
			return
		}
		s.logger.Error("get message for stream", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	user, _ := userFromContext(r.Context())
	if _, err := s.threads.ByID(r.Context(), msg.ThreadID, user.ID); err != nil {
		writeError(w, http.StatusNotFound, "message_not_found")
		return
	}

	sse.WritePreamble(w)

	s.logger.Debug("stream connect", "message_id", messageID, "status", msg.Status)

	// Late connection: the message already finished. Replay the terminal state
	// instead of hanging forever on a dead channel.
	switch msg.Status {
	case "complete":
		ev, _ := sse.NewEvent(sse.EventDone, research.DonePayload{})
		_ = sse.WriteEvent(w, ev)
		return
	case "error":
		ev, _ := sse.NewEvent(sse.EventError, research.ErrorPayload{Message: msg.Error})
		_ = sse.WriteEvent(w, ev)
		return
	}

	err = sse.Forward(r.Context(), w, r, s.rdb, sse.Channel(messageID))
	s.logger.Debug("stream closed", "message_id", messageID, "err", err)
}
