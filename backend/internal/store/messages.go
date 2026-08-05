package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MessageRepo provides persistence for messages and their sources.
type MessageRepo struct {
	pool *pgxpool.Pool
}

func NewMessageRepo(pool *pgxpool.Pool) *MessageRepo {
	return &MessageRepo{pool: pool}
}

func (r *MessageRepo) Create(ctx context.Context, threadID, role string) (Message, error) {
	m := Message{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO messages (thread_id, role) VALUES ($1, $2)
		 RETURNING id, thread_id, role, content, status, error, created_at`,
		threadID, role,
	).Scan(&m.ID, &m.ThreadID, &m.Role, &m.Content, &m.Status, &m.Error, &m.CreatedAt)
	return m, err
}

func (r *MessageRepo) Get(ctx context.Context, id string) (Message, error) {
	m := Message{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, thread_id, role, content, status, error, created_at
		 FROM messages WHERE id = $1`,
		id,
	).Scan(&m.ID, &m.ThreadID, &m.Role, &m.Content, &m.Status, &m.Error, &m.CreatedAt)
	return m, err
}

func (r *MessageRepo) ListByThread(ctx context.Context, threadID string) ([]Message, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, thread_id, role, content, status, error, created_at
		 FROM messages WHERE thread_id = $1 ORDER BY created_at`,
		threadID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ThreadID, &m.Role, &m.Content, &m.Status, &m.Error, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (r *MessageRepo) SetStreaming(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE messages SET status = 'streaming', error = '' WHERE id = $1`, id)
	return err
}

func (r *MessageRepo) Complete(ctx context.Context, id, content string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE messages SET status = 'complete', content = $2 WHERE id = $1`,
		id, content)
	return err
}

func (r *MessageRepo) Fail(ctx context.Context, id, errMsg string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE messages SET status = 'error', error = $2 WHERE id = $1`,
		id, errMsg)
	return err
}

// ReplaceSources wipes then inserts the given sources for a message.
func (r *MessageRepo) ReplaceSources(ctx context.Context, messageID string, sources []Source) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM message_sources WHERE message_id = $1`, messageID); err != nil {
		return err
	}

	for _, s := range sources {
		if _, err := tx.Exec(ctx,
			`INSERT INTO message_sources (message_id, url, title, snippet, position)
			 VALUES ($1, $2, $3, $4, $5)`,
			messageID, s.URL, s.Title, s.Snippet, s.Position,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *MessageRepo) SourcesByMessage(ctx context.Context, messageID string) ([]Source, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, message_id, url, title, snippet, position
		 FROM message_sources WHERE message_id = $1 ORDER BY position`,
		messageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sources := []Source{}
	for rows.Next() {
		var s Source
		if err := rows.Scan(&s.ID, &s.MessageID, &s.URL, &s.Title, &s.Snippet, &s.Position); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}